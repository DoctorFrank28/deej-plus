package deej

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"gopkg.in/yaml.v3"
)

// configDocument is the user config file as a yaml tree, so deej can change single keys
// without losing the comments and layout people put in their config
type configDocument struct {
	root *yaml.Node
}

func parseConfigDocument(contents []byte) (*configDocument, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(contents, &doc); err != nil {
		return nil, err
	}

	// an empty file has no document node yet
	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("config root must be a mapping")
	}

	return &configDocument{root: &doc}, nil
}

func (d *configDocument) bytes() ([]byte, error) {
	var buf bytes.Buffer

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	if err := encoder.Encode(d.root); err != nil {
		return nil, err
	}

	if err := encoder.Close(); err != nil {
		return nil, err
	}

	return spaceTopLevelEntries(buf.Bytes()), nil
}

// spaceTopLevelEntries puts back the blank lines yaml.v3 drops: before a commented section
// and after a nested block, so the file keeps reading like the one people wrote
func spaceTopLevelEntries(contents []byte) []byte {
	lines := bytes.Split(contents, []byte("\n"))
	result := make([][]byte, 0, len(lines)+16)

	for i, line := range lines {
		if i > 0 && len(line) > 0 && line[0] != ' ' && line[0] != '-' {
			previous := lines[i-1]
			previousIsComment := len(previous) > 0 && previous[0] == '#'
			previousIsNested := len(previous) > 0 && (previous[0] == ' ' || previous[0] == '-')

			startsBlock := bytes.HasSuffix(line, []byte(":"))

			if !previousIsComment && (line[0] == '#' || previousIsNested || startsBlock) {
				result = append(result, []byte{})
			}
		}

		result = append(result, line)
	}

	return bytes.Join(result, []byte("\n"))
}

func (d *configDocument) mapping() *yaml.Node {
	return d.root.Content[0]
}

// set replaces the value of a top-level key (adding it at the end if missing), keeping the key's comments
func (d *configDocument) set(key string, value *yaml.Node) {
	setInMapping(d.mapping(), key, value)
}

func (d *configDocument) setString(key string, value string) {
	d.set(key, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
}

func (d *configDocument) setBool(key string, value bool) {
	d.set(key, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(value)})
}

func (d *configDocument) setNumber(key string, value float64) {
	d.set(key, numberNode(value))
}

// setNoiseFilter writes the custom filter, as percentages
func (d *configDocument) setNoiseFilter(params filterParams) {
	node := existingOrNewMapping(d.mapping(), configKeyNoiseFilter)
	setInMapping(node, "deadband", numberNode(params.Deadband*100))
	setInMapping(node, "smoothing", numberNode(params.Smoothing))
	setInMapping(node, "edge_snap", numberNode(params.EdgeSnap*100))
	d.set(configKeyNoiseFilter, node)
}

// setSliderMapping rewrites slider_mapping. a single target stays a plain value, more become a list
func (d *configDocument) setSliderMapping(mapping map[int][]string) {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	for _, idx := range sortedKeys(mapping) {
		targets := mapping[idx]
		key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(idx)}

		var value *yaml.Node
		if len(targets) == 1 {
			value = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: targets[0]}
		} else {
			value = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			for _, target := range targets {
				value.Content = append(value.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: target})
			}
		}

		node.Content = append(node.Content, key, value)
	}

	d.set(configKeySliderMapping, node)
}

// setSliderNames rewrites slider_names, dropping empty names (and the whole key if none are left)
func (d *configDocument) setSliderNames(names map[int]string) {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	for _, idx := range sortedKeys(names) {
		if names[idx] == "" {
			continue
		}

		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(idx)},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: names[idx]})
	}

	if len(node.Content) == 0 {
		removeFromMapping(d.mapping(), configKeySliderNames)
		return
	}

	d.set(configKeySliderNames, node)
}

func setInMapping(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			old := mapping.Content[i+1]
			value.LineComment = old.LineComment
			if value.Kind == yaml.ScalarNode {
				value.HeadComment = old.HeadComment
			}
			value.FootComment = old.FootComment
			mapping.Content[i+1] = value

			return
		}
	}

	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value)
}

func removeFromMapping(mapping *yaml.Node, key string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}

func existingOrNewMapping(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key && mapping.Content[i+1].Kind == yaml.MappingNode {
			return mapping.Content[i+1]
		}
	}

	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
}

func numberNode(value float64) *yaml.Node {
	formatted := strconv.FormatFloat(value, 'f', -1, 64)

	tag := "!!float"
	if value == float64(int64(value)) {
		tag = "!!int"
	}

	return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: formatted}
}

func sortedKeys[V any](m map[int]V) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Ints(keys)

	return keys
}
