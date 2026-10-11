package deej

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

//go:embed ui
var uiFiles embed.FS

// uiServer serves the control panel on 127.0.0.1 and streams live state to it
type uiServer struct {
	deej   *Deej
	logger *zap.SugaredLogger
	port   int

	clientsLock sync.Mutex
	clients     map[chan sseMessage]struct{}
}

type sseMessage struct {
	event string
	data  []byte
}

type apiSlider struct {
	Index   int      `json:"index"`
	Name    string   `json:"name"`
	Raw     float32  `json:"raw"`
	Value   float32  `json:"value"`
	Targets []string `json:"targets"`
}

// apiFilter is the noise filter as the UI shows it: deadband and edge snap in percent
type apiFilter struct {
	Deadband  float64 `json:"deadband"`
	Smoothing float64 `json:"smoothing"`
	EdgeSnap  float64 `json:"edgeSnap"`
}

type apiSettings struct {
	ComPort            string    `json:"comPort"`
	BaudRate           int       `json:"baudRate"`
	InvertSliders      bool      `json:"invertSliders"`
	AutoReconnect      bool      `json:"autoReconnect"`
	ReconnectInterval  float64   `json:"reconnectInterval"`
	NotifyOnDisconnect bool      `json:"notifyOnDisconnect"`
	NoiseReduction     string    `json:"noiseReduction"`
	Filter             apiFilter `json:"filter"`
	SliderMaxValue     int       `json:"sliderMaxValue"`
	StartWithWindows   bool      `json:"startWithWindows"`
	OpenUIOnStart      bool      `json:"openUiOnStart"`
	Language           string    `json:"language"`
	ConfigPath         string    `json:"configPath"`
}

type apiState struct {
	Version    string           `json:"version"`
	Connection ConnectionStatus `json:"connection"`
	Sliders    []apiSlider      `json:"sliders"`
	Settings   apiSettings      `json:"settings"`
}

type settingsUpdate struct {
	ComPort            *string    `json:"comPort"`
	BaudRate           *int       `json:"baudRate"`
	InvertSliders      *bool      `json:"invertSliders"`
	AutoReconnect      *bool      `json:"autoReconnect"`
	ReconnectInterval  *float64   `json:"reconnectInterval"`
	NotifyOnDisconnect *bool      `json:"notifyOnDisconnect"`
	NoiseReduction     *string    `json:"noiseReduction"`
	Filter             *apiFilter `json:"filter"`
	SliderMaxValue     *int       `json:"sliderMaxValue"`
	StartWithWindows   *bool      `json:"startWithWindows"`
	OpenUIOnStart      *bool      `json:"openUiOnStart"`
	Language           *string    `json:"language"`
}

type mappingUpdate struct {
	Sliders []struct {
		Index   int      `json:"index"`
		Name    string   `json:"name"`
		Targets []string `json:"targets"`
	} `json:"sliders"`
}

const sliderEventInterval = 50 * time.Millisecond

func newUIServer(deej *Deej, logger *zap.SugaredLogger) *uiServer {
	return &uiServer{
		deej:    deej,
		logger:  logger.Named("ui"),
		port:    deej.config.UIPort,
		clients: map[chan sseMessage]struct{}{},
	}
}

func (s *uiServer) url() string {
	return fmt.Sprintf("http://127.0.0.1:%d/", s.port)
}

// start listens in the background. failing to listen only disables the UI
func (s *uiServer) start() {
	static, err := fs.Sub(uiFiles, "ui")
	if err != nil {
		s.logger.Errorw("Failed to load embedded UI files", "error", err)
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/ports", s.handlePorts)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/mapping", s.handleMapping)
	mux.HandleFunc("/api/reconnect", s.handleReconnect)
	mux.HandleFunc("/api/open", s.handleOpen)
	mux.Handle("/", noCache(http.FileServer(http.FS(static))))

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", s.port))
	if err != nil {
		s.logger.Warnw("Can't listen for the UI, it won't be available", "port", s.port, "error", err)
		return
	}

	server := &http.Server{
		Handler:           s.guard(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go s.forwardLiveUpdates()

	go func() {
		s.logger.Infow("Serving UI", "url", s.url())

		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Warnw("UI server stopped", "error", err)
		}
	}()
}

// guard only lets the local UI talk to deej: the host must be ours (no dns rebinding) and every
// change must carry a header a cross-site form can't add
func (s *uiServer) guard(next http.Handler) http.Handler {
	allowedHosts := map[string]bool{
		fmt.Sprintf("127.0.0.1:%d", s.port): true,
		fmt.Sprintf("localhost:%d", s.port): true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowedHosts[r.Host] {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			origin := r.Header.Get("Origin")
			if r.Header.Get("X-Deej") != "1" || (origin != "" && !allowedHosts[strings.TrimPrefix(origin, "http://")]) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *uiServer) state() apiState {
	cfg := s.deej.config
	readings := s.deej.serial.Readings()

	indexes := map[int]bool{}
	for _, reading := range readings {
		indexes[reading.Index] = true
	}
	cfg.SliderMapping.iterate(func(idx int, _ []string) { indexes[idx] = true })
	for idx := range cfg.SliderNames {
		indexes[idx] = true
	}

	sorted := make([]int, 0, len(indexes))
	for idx := range indexes {
		sorted = append(sorted, idx)
	}
	sort.Ints(sorted)

	sliders := make([]apiSlider, 0, len(sorted))
	for _, idx := range sorted {
		targets, _ := cfg.SliderMapping.get(idx)
		slider := apiSlider{Index: idx, Name: cfg.SliderNames[idx], Targets: append([]string{}, targets...)}

		if idx < len(readings) {
			slider.Raw = readings[idx].Raw
			slider.Value = readings[idx].Value
		}

		sliders = append(sliders, slider)
	}

	params := cfg.FilterParams()

	return apiState{
		Version:    s.deej.version,
		Connection: s.deej.serial.Status(),
		Sliders:    sliders,
		Settings: apiSettings{
			ComPort:            cfg.ConnectionInfo.COMPort,
			BaudRate:           cfg.ConnectionInfo.BaudRate,
			InvertSliders:      cfg.InvertSliders,
			AutoReconnect:      cfg.AutoReconnect,
			ReconnectInterval:  cfg.ReconnectInterval.Seconds(),
			NotifyOnDisconnect: cfg.NotifyOnDisconnect,
			NoiseReduction:     cfg.NoiseReductionLevel,
			Filter:             apiFilter{Deadband: params.Deadband * 100, Smoothing: params.Smoothing, EdgeSnap: params.EdgeSnap * 100},
			SliderMaxValue:     cfg.SliderMaxValue,
			StartWithWindows:   startWithSystemEnabled(),
			OpenUIOnStart:      cfg.OpenUIOnStart,
			Language:           cfg.Language,
			ConfigPath:         cfg.Path(),
		},
	}
}

func (s *uiServer) handleState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.state())
}

func (s *uiServer) handleSessions(w http.ResponseWriter, r *http.Request) {

	// windows core audio calls want a stable os thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// pick up apps opened since the last refresh (rate limited inside)
	s.deej.sessions.refreshSessions(false)

	writeJSON(w, http.StatusOK, map[string]any{"sessions": s.deej.sessions.snapshot()})
}

func (s *uiServer) handlePorts(w http.ResponseWriter, r *http.Request) {
	ports, err := ListPorts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ports": ports})
}

func (s *uiServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var update settingsUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(s.deej.config.T("api.badRequest", err)))
		return
	}

	if update.StartWithWindows != nil {
		if err := setStartWithSystem(*update.StartWithWindows); err != nil {
			writeError(w, http.StatusInternalServerError, errors.New(s.deej.config.T("api.startup", err)))
			return
		}
	}

	err := s.deej.config.Update(func(doc *configDocument) error {
		if update.ComPort != nil {
			port := strings.TrimSpace(*update.ComPort)
			if port == "" {
				port = comPortAuto
			}
			doc.setString(configKeyCOMPort, port)
		}

		if update.BaudRate != nil {
			if *update.BaudRate < 300 || *update.BaudRate > 2000000 {
				return errors.New(s.deej.config.T("api.baud", *update.BaudRate))
			}
			doc.setNumber(configKeyBaudRate, float64(*update.BaudRate))
		}

		if update.InvertSliders != nil {
			doc.setBool(configKeyInvertSliders, *update.InvertSliders)
		}

		if update.AutoReconnect != nil {
			doc.setBool(configKeyAutoReconnect, *update.AutoReconnect)
		}

		if update.ReconnectInterval != nil {
			doc.setNumber(configKeyReconnectInterval, clampFloat(*update.ReconnectInterval, 0.5, 60))
		}

		if update.NotifyOnDisconnect != nil {
			doc.setBool(configKeyNotifyOnDisconnect, *update.NotifyOnDisconnect)
		}

		if update.NoiseReduction != nil {
			level := strings.ToLower(*update.NoiseReduction)
			if _, preset := noiseReductionPresets[level]; !preset && level != noiseReductionCustom {
				return errors.New(s.deej.config.T("api.filterLevel", level))
			}
			doc.setString(configKeyNoiseReductionLevel, level)
		}

		if update.Filter != nil {
			doc.setNoiseFilter(filterParams{
				Deadband:  clampFloat(update.Filter.Deadband, 0, 5) / 100,
				Smoothing: clampFloat(update.Filter.Smoothing, 0, 0.9),
				EdgeSnap:  clampFloat(update.Filter.EdgeSnap, 0, 10) / 100,
			})
		}

		if update.SliderMaxValue != nil {
			if *update.SliderMaxValue < 1 {
				return errors.New(s.deej.config.T("api.maxValue", *update.SliderMaxValue))
			}
			doc.setNumber(configKeySliderMaxValue, float64(*update.SliderMaxValue))
		}

		if update.OpenUIOnStart != nil {
			doc.setBool(configKeyOpenUIOnStart, *update.OpenUIOnStart)
		}

		if update.Language != nil {
			lang := strings.ToLower(strings.TrimSpace(*update.Language))
			if !supportedLanguage(lang) {
				return errors.New(s.deej.config.T("api.language", lang))
			}
			doc.setString(configKeyLanguage, lang)
		}

		return nil
	})

	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	state := s.state()
	s.broadcastJSON("state", state)
	writeJSON(w, http.StatusOK, state)
}

func (s *uiServer) handleMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var update mappingUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(s.deej.config.T("api.badRequest", err)))
		return
	}

	mapping := map[int][]string{}
	names := map[int]string{}

	for _, slider := range update.Sliders {
		if slider.Index < 0 || slider.Index > 63 {
			writeError(w, http.StatusBadRequest, errors.New(s.deej.config.T("api.sliderIndex", slider.Index)))
			return
		}

		targets := []string{}
		for _, target := range slider.Targets {
			if target = strings.TrimSpace(target); target != "" {
				targets = append(targets, target)
			}
		}

		if len(targets) > 0 {
			mapping[slider.Index] = targets
		}

		names[slider.Index] = strings.TrimSpace(slider.Name)
	}

	err := s.deej.config.Update(func(doc *configDocument) error {
		doc.setSliderMapping(mapping)
		doc.setSliderNames(names)

		return nil
	})

	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	state := s.state()
	s.broadcastJSON("state", state)
	writeJSON(w, http.StatusOK, state)
}

func (s *uiServer) handleReconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	s.logger.Info("Reconnect requested from the UI")
	s.deej.serial.Reconnect()
	w.WriteHeader(http.StatusNoContent)
}

func (s *uiServer) handleOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		What string `json:"what"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var err error
	switch body.What {
	case "configFolder":
		err = revealInExplorer(s.deej.config.Path())
	case "log":
		var logPath string
		logPath, err = filepath.Abs(filepath.Join(logDirectory, logFilename))
		if err == nil {
			err = openTextFile(logPath)
		}
	default:
		err = errors.New(s.deej.config.T("api.openUnknown", body.What))
	}

	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *uiServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")

	client := make(chan sseMessage, 16)

	s.clientsLock.Lock()
	s.clients[client] = struct{}{}
	s.clientsLock.Unlock()

	defer func() {
		s.clientsLock.Lock()
		delete(s.clients, client)
		s.clientsLock.Unlock()
	}()

	// start every stream with the full picture
	if data, err := json.Marshal(s.state()); err == nil {
		writeSSE(w, sseMessage{event: "state", data: data})
		flusher.Flush()
	}

	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-client:
			writeSSE(w, msg)
			flusher.Flush()
		case <-keepAlive.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

// forwardLiveUpdates pushes slider readings (throttled), connection changes and config reloads to every client
func (s *uiServer) forwardLiveUpdates() {
	readings := s.deej.serial.SubscribeToReadings()
	statuses := s.deej.serial.SubscribeToStatus()
	reloads := s.deej.config.SubscribeToChanges()

	ticker := time.NewTicker(sliderEventInterval)
	defer ticker.Stop()

	var latest []SliderReading

	for {
		select {
		case r := <-readings:
			latest = r
		case <-ticker.C:
			if latest != nil {
				s.broadcastJSON("sliders", map[string]any{"sliders": latest})
				latest = nil
			}
		case status := <-statuses:
			s.broadcastJSON("connection", status)
		case <-reloads:
			s.broadcastJSON("state", s.state())
		}
	}
}

func (s *uiServer) broadcastJSON(event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		s.logger.Warnw("Failed to encode UI event", "event", event, "error", err)
		return
	}

	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	for client := range s.clients {
		select {
		case client <- sseMessage{event: event, data: data}:
		default:
			// a stuck client misses updates rather than slowing everyone down
		}
	}
}

func writeSSE(w http.ResponseWriter, msg sseMessage) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.event, msg.data)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}

	if v > max {
		return max
	}

	return v
}
