package deej

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/DoctorFrank28/deej-plus/pkg/deej/util"
)

// CanonicalConfig provides application-wide access to configuration fields,
// as well as loading/file watching logic for deej's configuration file
type CanonicalConfig struct {
	SliderMapping *sliderMap
	SliderNames   map[int]string

	ConnectionInfo struct {
		COMPort  string
		BaudRate int
	}

	AutoReconnect      bool
	ReconnectInterval  time.Duration
	SerialTimeout      time.Duration
	NotifyOnDisconnect bool

	InvertSliders  bool
	SliderMaxValue int

	NoiseReductionLevel string
	customFilter        filterParams

	UIPort        int
	OpenUIOnStart bool

	Language         string // as configured: "auto" or a language code
	resolvedLanguage string // the language messages are shown in

	logger             *zap.SugaredLogger
	notifier           Notifier
	stopWatcherChannel chan bool

	reloadConsumers []chan bool

	// serializes loads and writes, which can come from the file watcher and the UI at the same time
	loadLock sync.Mutex

	// hash of the last file contents deej wrote itself, so the watcher doesn't reload them twice
	lastWrittenHash [32]byte

	userConfig     *viper.Viper
	internalConfig *viper.Viper
}

const (
	userConfigFilepath     = "config.yaml"
	internalConfigFilepath = "preferences.yaml"

	userConfigName     = "config"
	internalConfigName = "preferences"

	userConfigPath = "."

	configType = "yaml"

	configKeySliderMapping       = "slider_mapping"
	configKeySliderNames         = "slider_names"
	configKeyInvertSliders       = "invert_sliders"
	configKeyCOMPort             = "com_port"
	configKeyBaudRate            = "baud_rate"
	configKeyNoiseReductionLevel = "noise_reduction"
	configKeyNoiseFilter         = "noise_filter"
	configKeySliderMaxValue      = "slider_max_value"
	configKeyAutoReconnect       = "auto_reconnect"
	configKeyReconnectInterval   = "reconnect_interval"
	configKeySerialTimeout       = "serial_timeout"
	configKeyNotifyOnDisconnect  = "notify_on_disconnect"
	configKeyUIPort              = "ui_port"
	configKeyOpenUIOnStart       = "open_ui_on_start"
	configKeyLanguage            = "language"

	comPortAuto = "auto"

	defaultCOMPort           = comPortAuto
	defaultBaudRate          = 9600
	defaultSliderMaxValue    = 1023
	defaultReconnectInterval = 2 // seconds
	defaultUIPort            = 47823
)

// has to be defined as a non-constant because we're using path.Join
var internalConfigPath = path.Join(".", logDirectory)

var defaultSliderMapping = func() *sliderMap {
	emptyMap := newSliderMap()
	emptyMap.set(0, []string{masterSessionName})

	return emptyMap
}()

// NewConfig creates a config instance for the deej object and sets up viper instances for deej's config files
func NewConfig(logger *zap.SugaredLogger, notifier Notifier) (*CanonicalConfig, error) {
	logger = logger.Named("config")

	cc := &CanonicalConfig{
		logger:             logger,
		notifier:           notifier,
		reloadConsumers:    []chan bool{},
		stopWatcherChannel: make(chan bool),
		Language:           languageAuto,
		resolvedLanguage:   resolveLanguage(languageAuto),
	}

	// distinguish between the user-provided config (config.yaml) and the internal config (logs/preferences.yaml)
	userConfig := viper.New()
	userConfig.SetConfigName(userConfigName)
	userConfig.SetConfigType(configType)
	userConfig.AddConfigPath(userConfigPath)

	userConfig.SetDefault(configKeySliderMapping, map[string][]string{})
	userConfig.SetDefault(configKeyInvertSliders, false)
	userConfig.SetDefault(configKeyCOMPort, defaultCOMPort)
	userConfig.SetDefault(configKeyBaudRate, defaultBaudRate)
	userConfig.SetDefault(configKeySliderMaxValue, defaultSliderMaxValue)
	userConfig.SetDefault(configKeyAutoReconnect, true)
	userConfig.SetDefault(configKeyReconnectInterval, defaultReconnectInterval)
	userConfig.SetDefault(configKeySerialTimeout, 0)
	userConfig.SetDefault(configKeyNotifyOnDisconnect, true)
	userConfig.SetDefault(configKeyUIPort, defaultUIPort)
	userConfig.SetDefault(configKeyOpenUIOnStart, false)
	userConfig.SetDefault(configKeyLanguage, languageAuto)

	internalConfig := viper.New()
	internalConfig.SetConfigName(internalConfigName)
	internalConfig.SetConfigType(configType)
	internalConfig.AddConfigPath(internalConfigPath)

	cc.userConfig = userConfig
	cc.internalConfig = internalConfig

	logger.Debug("Created config instance")

	return cc, nil
}

// Path returns the absolute path of the user config file
func (cc *CanonicalConfig) Path() string {
	abs, err := filepath.Abs(userConfigFilepath)
	if err != nil {
		return userConfigFilepath
	}

	return abs
}

// FilterParams returns the noise filter parameters for the configured noise reduction level
func (cc *CanonicalConfig) FilterParams() filterParams {
	if cc.NoiseReductionLevel == noiseReductionCustom {
		return cc.customFilter
	}

	if params, ok := noiseReductionPresets[cc.NoiseReductionLevel]; ok {
		return params
	}

	return noiseReductionPresets[noiseReductionDefault]
}

// Load reads deej's config files from disk and tries to parse them
func (cc *CanonicalConfig) Load() error {
	cc.loadLock.Lock()
	defer cc.loadLock.Unlock()

	return cc.load()
}

func (cc *CanonicalConfig) load() error {
	cc.logger.Debugw("Loading config", "path", userConfigFilepath)

	// make sure it exists
	if !util.FileExists(userConfigFilepath) {
		cc.logger.Warnw("Config file not found", "path", userConfigFilepath)
		cc.notifier.Notify(cc.T("notify.configMissing.title"), cc.T("notify.configMissing.body", userConfigFilepath))

		return fmt.Errorf("config file doesn't exist: %s", userConfigFilepath)
	}

	// load the user config
	if err := cc.userConfig.ReadInConfig(); err != nil {
		cc.logger.Warnw("Viper failed to read user config", "error", err)

		// if the error is yaml-format-related, show a sensible error. otherwise, show 'em to the logs
		if strings.Contains(err.Error(), "yaml:") {
			cc.notifier.Notify(cc.T("notify.configInvalid.title"), cc.T("notify.configInvalid.body", userConfigFilepath))
		} else {
			cc.notifier.Notify(cc.T("notify.configError.title"), cc.T("notify.configError.body"))
		}

		return fmt.Errorf("read user config: %w", err)
	}

	// load the internal config - this doesn't have to exist, so it can error
	if err := cc.internalConfig.ReadInConfig(); err != nil {
		cc.logger.Debugw("Viper failed to read internal config", "error", err, "reminder", "this is fine")
	}

	// canonize the configuration with viper's helpers
	if err := cc.populateFromVipers(); err != nil {
		cc.logger.Warnw("Failed to populate config fields", "error", err)
		return fmt.Errorf("populate config fields: %w", err)
	}

	cc.logger.Info("Loaded config successfully")
	cc.logger.Infow("Config values",
		"sliderMapping", cc.SliderMapping,
		"connectionInfo", cc.ConnectionInfo,
		"invertSliders", cc.InvertSliders,
		"noiseReduction", cc.NoiseReductionLevel)

	return nil
}

// SubscribeToChanges allows external components to receive updates when the config is reloaded
func (cc *CanonicalConfig) SubscribeToChanges() chan bool {
	c := make(chan bool)
	cc.reloadConsumers = append(cc.reloadConsumers, c)

	return c
}

// WatchConfigFileChanges starts watching for configuration file changes
// and attempts reloading the config when they happen
func (cc *CanonicalConfig) WatchConfigFileChanges() {
	cc.logger.Debugw("Starting to watch user config file for changes", "path", userConfigFilepath)

	const (
		minTimeBetweenReloadAttempts = time.Millisecond * 500
		delayBetweenEventAndReload   = time.Millisecond * 50
	)

	lastAttemptedReload := time.Now()

	// establish watch using viper as opposed to doing it ourselves, though our internal cooldown is still required
	cc.userConfig.WatchConfig()
	cc.userConfig.OnConfigChange(func(event fsnotify.Event) {

		// when we get a write event...
		if event.Op&fsnotify.Write == fsnotify.Write {

			now := time.Now()

			// ... check if it's not a duplicate (many editors will write to a file twice)
			if lastAttemptedReload.Add(minTimeBetweenReloadAttempts).Before(now) {

				// wait a bit to let the editor actually flush the new file contents to disk
				<-time.After(delayBetweenEventAndReload)

				// deej's own writes are reloaded right away by the writer
				if cc.isOwnWrite() {
					cc.logger.Debug("Config file change was written by deej itself, skipping reload")
					return
				}

				cc.logger.Debugw("Config file modified, attempting reload", "event", event)

				if err := cc.Load(); err != nil {
					cc.logger.Warnw("Failed to reload config file", "error", err)
				} else {
					cc.logger.Info("Reloaded config successfully")
					cc.notifier.Notify(cc.T("notify.configReloaded.title"), cc.T("notify.configReloaded.body"))

					cc.onConfigReloaded()
				}

				// don't forget to update the time
				lastAttemptedReload = now
			}
		}
	})

	// wait till they stop us
	<-cc.stopWatcherChannel
	cc.logger.Debug("Stopping user config file watcher")
	cc.userConfig.OnConfigChange(nil)
}

// StopWatchingConfigFile signals our filesystem watcher to stop
func (cc *CanonicalConfig) StopWatchingConfigFile() {
	cc.stopWatcherChannel <- true
}

// Update edits the user config file in place (keeping its comments), then reloads it
func (cc *CanonicalConfig) Update(edit func(doc *configDocument) error) error {
	cc.loadLock.Lock()

	contents, err := os.ReadFile(userConfigFilepath)
	if err != nil {
		cc.loadLock.Unlock()
		return fmt.Errorf("read config file: %w", err)
	}

	doc, err := parseConfigDocument(contents)
	if err != nil {
		cc.loadLock.Unlock()
		return fmt.Errorf("parse config file: %w", err)
	}

	if err := edit(doc); err != nil {
		cc.loadLock.Unlock()
		return err
	}

	updated, err := doc.bytes()
	if err != nil {
		cc.loadLock.Unlock()
		return fmt.Errorf("serialize config file: %w", err)
	}

	if bytes.Equal(updated, contents) {
		cc.loadLock.Unlock()
		return nil
	}

	cc.lastWrittenHash = sha256.Sum256(updated)

	if err := os.WriteFile(userConfigFilepath, updated, 0o644); err != nil {
		cc.loadLock.Unlock()
		return fmt.Errorf("write config file: %w", err)
	}

	cc.logger.Info("Config file updated from the UI")

	err = cc.load()
	cc.loadLock.Unlock()

	if err != nil {
		return fmt.Errorf("reload config after update: %w", err)
	}

	cc.onConfigReloaded()

	return nil
}

func (cc *CanonicalConfig) isOwnWrite() bool {
	contents, err := os.ReadFile(userConfigFilepath)
	if err != nil {
		return false
	}

	cc.loadLock.Lock()
	defer cc.loadLock.Unlock()

	return sha256.Sum256(contents) == cc.lastWrittenHash
}

func (cc *CanonicalConfig) populateFromVipers() error {

	// merge the slider mappings from the user and internal configs
	cc.SliderMapping = sliderMapFromConfigs(
		cc.userConfig.GetStringMapStringSlice(configKeySliderMapping),
		cc.internalConfig.GetStringMapStringSlice(configKeySliderMapping),
	)

	cc.SliderNames = map[int]string{}
	for key, name := range cc.userConfig.GetStringMapString(configKeySliderNames) {
		if idx, err := strconv.Atoi(key); err == nil {
			cc.SliderNames[idx] = name
		}
	}

	// get the rest of the config fields - viper saves us a lot of effort here
	cc.ConnectionInfo.COMPort = cc.userConfig.GetString(configKeyCOMPort)

	cc.ConnectionInfo.BaudRate = cc.userConfig.GetInt(configKeyBaudRate)
	if cc.ConnectionInfo.BaudRate <= 0 {
		cc.logger.Warnw("Invalid baud rate specified, using default value",
			"key", configKeyBaudRate,
			"invalidValue", cc.ConnectionInfo.BaudRate,
			"defaultValue", defaultBaudRate)

		cc.ConnectionInfo.BaudRate = defaultBaudRate
	}

	cc.AutoReconnect = cc.userConfig.GetBool(configKeyAutoReconnect)
	cc.NotifyOnDisconnect = cc.userConfig.GetBool(configKeyNotifyOnDisconnect)

	interval := cc.userConfig.GetFloat64(configKeyReconnectInterval)
	if interval < 0.5 {
		interval = defaultReconnectInterval
	}
	cc.ReconnectInterval = time.Duration(interval * float64(time.Second))

	timeout := cc.userConfig.GetFloat64(configKeySerialTimeout)
	if timeout < 0 {
		timeout = 0
	}
	cc.SerialTimeout = time.Duration(timeout * float64(time.Second))

	cc.InvertSliders = cc.userConfig.GetBool(configKeyInvertSliders)

	cc.SliderMaxValue = cc.userConfig.GetInt(configKeySliderMaxValue)
	if cc.SliderMaxValue <= 0 {
		cc.SliderMaxValue = defaultSliderMaxValue
	}

	cc.NoiseReductionLevel = strings.ToLower(cc.userConfig.GetString(configKeyNoiseReductionLevel))
	if cc.NoiseReductionLevel == "" {
		cc.NoiseReductionLevel = noiseReductionDefault
	}

	// noise_filter values are percentages in the file, fractions in memory
	defaults := noiseReductionPresets[noiseReductionDefault]
	cc.customFilter = filterParams{
		Deadband:  percentOr(cc.userConfig, configKeyNoiseFilter+".deadband", defaults.Deadband),
		Smoothing: cc.userConfig.GetFloat64(configKeyNoiseFilter + ".smoothing"),
		EdgeSnap:  percentOr(cc.userConfig, configKeyNoiseFilter+".edge_snap", defaults.EdgeSnap),
	}

	cc.UIPort = cc.userConfig.GetInt(configKeyUIPort)
	if cc.UIPort <= 0 || cc.UIPort > 65535 {
		cc.UIPort = defaultUIPort
	}

	cc.OpenUIOnStart = cc.userConfig.GetBool(configKeyOpenUIOnStart)

	cc.Language = strings.ToLower(strings.TrimSpace(cc.userConfig.GetString(configKeyLanguage)))
	if cc.Language == "" {
		cc.Language = languageAuto
	}
	cc.resolvedLanguage = resolveLanguage(cc.Language)

	cc.logger.Debug("Populated config fields from vipers")

	return nil
}

func percentOr(v *viper.Viper, key string, fallback float64) float64 {
	if !v.IsSet(key) {
		return fallback
	}

	return v.GetFloat64(key) / 100
}

func (cc *CanonicalConfig) onConfigReloaded() {
	cc.logger.Debug("Notifying consumers about configuration reload")

	for _, consumer := range cc.reloadConsumers {
		consumer <- true
	}
}
