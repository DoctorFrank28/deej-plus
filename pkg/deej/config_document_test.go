package deej

import (
	"strings"
	"testing"
)

const sampleConfig = `# process names are case-insensitive
slider_mapping:
  0: master
  1:
    - chrome.exe
    - brave.exe

  5: mic

# set this to true if you want the controls inverted (i.e. top is 0%, bottom is 100%)
invert_sliders: false

# settings for connecting to the arduino board
com_port: COM3
baud_rate: 9600
`

func TestConfigDocumentKeepsCommentsWhenEditing(t *testing.T) {
	doc, err := parseConfigDocument([]byte(sampleConfig))
	if err != nil {
		t.Fatal(err)
	}

	doc.setBool(configKeyInvertSliders, true)
	doc.setString(configKeyCOMPort, comPortAuto)
	doc.setSliderMapping(map[int][]string{0: {"master"}, 1: {"chrome.exe"}, 2: {"spotify.exe", "vlc.exe"}})
	doc.setSliderNames(map[int]string{0: "Master", 1: "", 2: "Musica"})
	doc.setNoiseFilter(filterParams{Deadband: 0.004, Smoothing: 0.5, EdgeSnap: 0.01})

	out, err := doc.bytes()
	if err != nil {
		t.Fatal(err)
	}

	text := string(out)
	for _, want := range []string{
		"# process names are case-insensitive",
		"# set this to true if you want the controls inverted",
		"invert_sliders: true",
		"com_port: auto",
		"1: chrome.exe",
		"- spotify.exe",
		"slider_names:",
		"2: Musica",
		"deadband: 0.4",
		"edge_snap: 1",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in:\n%s", want, text)
		}
	}

	if !strings.Contains(text, "    - vlc.exe\n\n# set this to true") || !strings.Contains(text, "com_port: auto\nbaud_rate: 9600\n\nslider_names:") {
		t.Errorf("expected blank lines between sections only:\n%s", text)
	}

	if strings.Contains(text, "1: \"\"") {
		t.Errorf("empty slider names should be dropped:\n%s", text)
	}

	if _, err := parseConfigDocument(out); err != nil {
		t.Fatalf("result is not valid yaml: %v", err)
	}
}
