package deej

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
	"go.uber.org/zap"
)

// ConnectionState describes where the serial connection to the board currently stands
type ConnectionState string

const (
	ConnectionConnected    ConnectionState = "connected"
	ConnectionReconnecting ConnectionState = "reconnecting" // was connected, lost it, trying again
	ConnectionDisconnected ConnectionState = "disconnected" // no board found (yet)
	ConnectionBusy         ConnectionState = "busy"         // the port exists but another program holds it
)

// ConnectionStatus is a snapshot of the serial connection, shared with the tray and the UI
type ConnectionStatus struct {
	State     ConnectionState `json:"state"`
	Port      string          `json:"port"`
	Device    string          `json:"device"`
	Attempt   int             `json:"attempt"`
	Since     time.Time       `json:"since"`
	LastError string          `json:"lastError"`
	LastEvent string          `json:"lastEvent"`
}

// SliderMoveEvent represents a single slider move captured by deej
type SliderMoveEvent struct {
	SliderID     int
	PercentValue float32
}

// SliderReading is the latest known position of a slider, before (Raw) and after (Value) noise filtering
type SliderReading struct {
	Index int     `json:"index"`
	Raw   float32 `json:"raw"`
	Value float32 `json:"value"`
}

// SerialIO keeps a connection to the arduino board alive: it finds the port, reconnects whenever
// the board goes away, parses its lines and turns them into filtered slider move events
type SerialIO struct {
	deej   *Deej
	logger *zap.SugaredLogger

	lock         sync.Mutex
	status       ConnectionStatus
	started      bool
	stopChannel  chan struct{}
	reconnectNow chan struct{}

	// set while a port is open, so Stop can unblock a pending read
	port serial.Port

	// the configured port and baud rate of the latest connection attempt
	attemptedPort string
	attemptedBaud int

	// slider state, only touched by the connection goroutine (and guarded by lock for readers)
	lastKnownNumSliders int
	pendingNumSliders   int
	filters             []*sliderFilter
	readings            []SliderReading

	// the latest "connected/disconnected at" event, which slider detection adds to
	lastEventBase string

	sliderMoveConsumers []chan SliderMoveEvent
	readingConsumers    []chan []SliderReading
	statusConsumers     []chan ConnectionStatus
}

var expectedLinePattern = regexp.MustCompile(`^\d{1,4}(\|\d{1,4})*\r?\n$`)

const (
	readTimeout = 250 * time.Millisecond

	// the board resets when the port opens; give it a moment before treating silence as a problem
	bootGracePeriod = 3 * time.Second
)

// usb vendor ids of boards (and usb-serial chips) commonly used for deej, best match first
var knownBoardVendorIDs = []string{
	"2341", // arduino
	"2a03", // arduino.org
	"1a86", // wch ch340 clones
	"0403", // ftdi
	"10c4", // silicon labs cp210x
	"2e8a", // raspberry pi pico
	"239a", // adafruit
	"1b4f", // sparkfun
}

// NewSerialIO creates a SerialIO instance that uses the provided deej
// instance's connection info to establish communications with the arduino chip
func NewSerialIO(deej *Deej, logger *zap.SugaredLogger) (*SerialIO, error) {
	logger = logger.Named("serial")

	sio := &SerialIO{
		deej:         deej,
		logger:       logger,
		stopChannel:  make(chan struct{}),
		reconnectNow: make(chan struct{}, 1),
		status: ConnectionStatus{
			State: ConnectionDisconnected,
			Since: time.Now(),
		},
	}

	logger.Debug("Created serial i/o instance")

	// respond to config changes
	sio.setupOnConfigReload()

	return sio, nil
}

// Start launches the connection manager, which keeps trying to reach the board until Stop is called
func (sio *SerialIO) Start() {
	sio.lock.Lock()
	defer sio.lock.Unlock()

	if sio.started {
		return
	}

	sio.started = true
	go sio.manageConnection()
}

// Stop shuts down the connection manager and closes the port, if open
func (sio *SerialIO) Stop() {
	sio.lock.Lock()
	defer sio.lock.Unlock()

	if !sio.started {
		return
	}

	sio.started = false
	close(sio.stopChannel)

	if sio.port != nil {
		sio.port.Close()
	}
}

// Reconnect drops the current connection (if any) and immediately tries to connect again
func (sio *SerialIO) Reconnect() {
	select {
	case sio.reconnectNow <- struct{}{}:
	default:
	}
}

// Status returns the current connection status
func (sio *SerialIO) Status() ConnectionStatus {
	sio.lock.Lock()
	defer sio.lock.Unlock()

	return sio.status
}

// Readings returns the latest known position of every slider
func (sio *SerialIO) Readings() []SliderReading {
	sio.lock.Lock()
	defer sio.lock.Unlock()

	return append([]SliderReading(nil), sio.readings...)
}

// SubscribeToSliderMoveEvents returns an unbuffered channel that receives
// a sliderMoveEvent struct every time a slider moves
func (sio *SerialIO) SubscribeToSliderMoveEvents() chan SliderMoveEvent {
	ch := make(chan SliderMoveEvent)
	sio.sliderMoveConsumers = append(sio.sliderMoveConsumers, ch)

	return ch
}

// SubscribeToReadings returns a channel that receives every slider reading (filtered or not).
// Slow consumers miss readings instead of blocking serial i/o
func (sio *SerialIO) SubscribeToReadings() chan []SliderReading {
	ch := make(chan []SliderReading, 1)
	sio.readingConsumers = append(sio.readingConsumers, ch)

	return ch
}

// SubscribeToStatus returns a channel that receives the connection status whenever it changes
func (sio *SerialIO) SubscribeToStatus() chan ConnectionStatus {
	ch := make(chan ConnectionStatus, 4)
	sio.statusConsumers = append(sio.statusConsumers, ch)

	return ch
}

func (sio *SerialIO) setupOnConfigReload() {
	configReloadedChannel := sio.deej.config.SubscribeToChanges()

	// the session map re-acquires sessions on reload too; re-sending volumes before it's done would be lost
	const resendDelay = 200 * time.Millisecond

	go func() {
		for range configReloadedChannel {
			sio.applyFilterParams()

			connInfo := sio.deej.config.ConnectionInfo

			sio.lock.Lock()
			changed := connInfo.COMPort != sio.attemptedPort || connInfo.BaudRate != sio.attemptedBaud
			sio.lock.Unlock()

			if changed {
				sio.logger.Infow("Detected change in connection parameters, renewing connection",
					"comPort", connInfo.COMPort, "baudRate", connInfo.BaudRate)

				sio.Reconnect()

				continue
			}

			// mappings may have changed: push every slider's current value again
			go func() {
				<-time.After(resendDelay)
				sio.resendAll()
			}()
		}
	}()
}

func (sio *SerialIO) manageConnection() {
	attempt := 0
	everConnected := false
	notifiedDisconnect := false

	for {
		if sio.stopped() {
			return
		}

		sio.lock.Lock()
		sio.attemptedPort = sio.deej.config.ConnectionInfo.COMPort
		sio.attemptedBaud = sio.deej.config.ConnectionInfo.BaudRate
		sio.lock.Unlock()

		portName, device, err := sio.resolvePort()

		var port serial.Port
		if err == nil {
			port, err = serial.Open(portName, &serial.Mode{BaudRate: sio.deej.config.ConnectionInfo.BaudRate})
		}

		if err != nil {
			attempt++

			state := ConnectionDisconnected
			if everConnected {
				state = ConnectionReconnecting
			}

			var portErr *serial.PortError
			if errors.As(err, &portErr) && (portErr.Code() == serial.PortBusy || portErr.Code() == serial.PermissionDenied) {
				state = ConnectionBusy
			}

			if attempt == 1 || attempt%30 == 0 {
				sio.logger.Infow("Can't open serial port, will retry", "port", portName, "attempt", attempt, "error", err)
			}

			sio.setStatus(func(s *ConnectionStatus) {
				s.State = state
				s.Port = portName
				s.Device = device
				s.Attempt = attempt
				s.LastError = sio.describeSerialError(portName, err)
			})

			if !sio.waitBeforeRetry() {
				return
			}

			continue
		}

		namedLogger := sio.logger.Named(strings.ToLower(portName))
		namedLogger.Infow("Connected", "device", device, "baudRate", sio.deej.config.ConnectionInfo.BaudRate)

		if notifiedDisconnect {
			sio.deej.notifier.Notify(sio.t("notify.reconnected.title"), sio.t("notify.reconnected.body", portName))
			notifiedDisconnect = false
		}

		attempt = 0
		everConnected = true

		sio.lock.Lock()
		sio.port = port
		sio.lock.Unlock()

		sio.setStatus(func(s *ConnectionStatus) {
			s.State = ConnectionConnected
			s.Port = portName
			s.Device = device
			s.Attempt = 0
			s.LastError = ""
			s.LastEvent = sio.t("event.connected", time.Now().Format("15:04"))
		})
		sio.lastEventBase = sio.Status().LastEvent

		reason := sio.readUntilFailure(namedLogger, port)

		sio.lock.Lock()
		sio.port = nil
		sio.lock.Unlock()

		if err := port.Close(); err != nil {
			namedLogger.Debugw("Failed to close serial port", "error", err)
		}

		if sio.stopped() {
			return
		}

		namedLogger.Warnw("Serial connection lost", "reason", reason)

		shownReason := reason
		if reason == reasonManualReconnect {
			shownReason = sio.t("reason.manual")
		}

		sio.setStatus(func(s *ConnectionStatus) {
			s.State = ConnectionReconnecting
			s.LastError = shownReason
			s.LastEvent = sio.t("event.disconnected", time.Now().Format("15:04"))
		})
		sio.lastEventBase = sio.Status().LastEvent

		if sio.deej.config.NotifyOnDisconnect && reason != reasonManualReconnect {
			sio.deej.notifier.Notify(sio.t("notify.disconnected.title"), sio.t("notify.disconnected.body"))
			notifiedDisconnect = true
		}

		// a manual reconnect retries right away, anything else waits a little first
		if reason != reasonManualReconnect && !sio.waitBeforeRetry() {
			return
		}
	}
}

const (
	reasonManualReconnect = "manual"
	reasonStopped         = "stopped"
)

// readUntilFailure reads lines until the port fails, goes silent for too long, or we're asked to stop/reconnect.
// it returns a human readable reason
func (sio *SerialIO) readUntilFailure(logger *zap.SugaredLogger, port serial.Port) string {
	if err := port.SetReadTimeout(readTimeout); err != nil {
		return sio.t("error.readTimeout", err)
	}

	connectedAt := time.Now()
	lastLine := connectedAt
	buf := make([]byte, 256)
	pending := []byte{}

	sio.lock.Lock()
	sio.pendingNumSliders = 0
	sio.lock.Unlock()

	for {
		select {
		case <-sio.stopChannel:
			return reasonStopped
		case <-sio.reconnectNow:
			return reasonManualReconnect
		default:
		}

		n, err := port.Read(buf)
		if err != nil {
			if sio.stopped() {
				return reasonStopped
			}

			return sio.t("error.read", err)
		}

		if n > 0 {
			pending = append(pending, buf[:n]...)

			for {
				idx := indexOfByte(pending, '\n')
				if idx < 0 {
					break
				}

				line := string(pending[:idx+1])
				pending = pending[idx+1:]
				lastLine = time.Now()

				if sio.deej.Verbose() {
					logger.Debugw("Read new line", "line", line)
				}

				sio.handleLine(logger, line)
			}

			// garbage without newlines shouldn't grow forever
			if len(pending) > 1024 {
				pending = pending[:0]
			}
		}

		timeout := sio.deej.config.SerialTimeout
		if timeout > 0 && time.Since(connectedAt) > bootGracePeriod && time.Since(lastLine) > timeout {
			return sio.t("error.timeout", timeout)
		}
	}
}

func (sio *SerialIO) handleLine(logger *zap.SugaredLogger, line string) {

	// this function receives an unsanitized line which is guaranteed to end with LF,
	// but most lines will end with CRLF. it may also have garbage instead of
	// deej-formatted values, so we must check for that! just ignore bad ones
	if !expectedLinePattern.MatchString(line) {
		return
	}

	line = strings.TrimRight(line, "\r\n")

	// split on pipe (|), this gives a slice of numerical strings between "0" and the board's max value
	splitLine := strings.Split(line, "|")
	numSliders := len(splitLine)

	// connecting in the middle of a line yields a shorter (but well formed) one, so a new slider
	// count is only trusted once two consecutive lines agree on it
	if numSliders != sio.lastKnownNumSliders {
		if numSliders != sio.pendingNumSliders {
			sio.pendingNumSliders = numSliders
			return
		}

		logger.Infow("Detected sliders", "amount", numSliders)

		sio.lock.Lock()
		sio.lastKnownNumSliders = numSliders
		sio.filters = make([]*sliderFilter, numSliders)
		sio.readings = make([]SliderReading, numSliders)
		params := sio.deej.config.FilterParams()
		for idx := range sio.filters {
			sio.filters[idx] = newSliderFilter(params)
			sio.readings[idx].Index = idx
		}
		sio.lock.Unlock()

		sio.setStatus(func(s *ConnectionStatus) {
			s.LastEvent = sio.t("event.sliders", sio.lastEventBase, numSliders)
		})
	}

	maxValue := float32(sio.deej.config.SliderMaxValue)
	moveEvents := []SliderMoveEvent{}

	sio.lock.Lock()
	for sliderIdx, stringValue := range splitLine {
		number, _ := strconv.Atoi(stringValue)

		// turns out the first line could come out dirty sometimes (i.e. "4558|925|41|643|220")
		if float32(number) > maxValue*1.05 {
			sio.lock.Unlock()
			logger.Debugw("Got malformed line from serial, ignoring", "line", line)
			return
		}

		raw := clamp01(float32(number) / maxValue)
		if sio.deej.config.InvertSliders {
			raw = 1 - raw
		}

		value, changed := sio.filters[sliderIdx].update(raw)
		sio.readings[sliderIdx].Raw = raw
		sio.readings[sliderIdx].Value = value

		if changed {
			moveEvents = append(moveEvents, SliderMoveEvent{SliderID: sliderIdx, PercentValue: value})

			if sio.deej.Verbose() {
				logger.Debugw("Slider moved", "event", moveEvents[len(moveEvents)-1])
			}
		}
	}
	readings := append([]SliderReading(nil), sio.readings...)
	sio.lock.Unlock()

	for _, consumer := range sio.readingConsumers {
		select {
		case consumer <- readings:
		default:
		}
	}

	sio.deliverMoveEvents(moveEvents)
}

func (sio *SerialIO) deliverMoveEvents(moveEvents []SliderMoveEvent) {
	for _, consumer := range sio.sliderMoveConsumers {
		for _, moveEvent := range moveEvents {
			consumer <- moveEvent
		}
	}
}

// resendAll emits a move event for every slider with its current value
func (sio *SerialIO) resendAll() {
	sio.lock.Lock()
	moveEvents := make([]SliderMoveEvent, 0, len(sio.readings))
	for _, reading := range sio.readings {
		moveEvents = append(moveEvents, SliderMoveEvent{SliderID: reading.Index, PercentValue: reading.Value})
	}
	sio.lock.Unlock()

	sio.deliverMoveEvents(moveEvents)
}

func (sio *SerialIO) applyFilterParams() {
	params := sio.deej.config.FilterParams()

	sio.lock.Lock()
	defer sio.lock.Unlock()

	for _, filter := range sio.filters {
		filter.params = params
	}
}

// resolvePort turns the configured port ("auto" or a name like COM3) into a port name and a device description
func (sio *SerialIO) resolvePort() (string, string, error) {
	configured := strings.TrimSpace(sio.deej.config.ConnectionInfo.COMPort)
	ports, enumErr := enumerator.GetDetailedPortsList()

	if configured != "" && !strings.EqualFold(configured, comPortAuto) {
		for _, p := range ports {
			if strings.EqualFold(p.Name, configured) {
				return p.Name, portDescription(p), nil
			}
		}

		return configured, "", nil
	}

	if enumErr != nil {
		return "", "", fmt.Errorf("list serial ports: %w", enumErr)
	}

	if best := bestBoardPort(ports); best != nil {
		return best.Name, portDescription(best), nil
	}

	return "", "", errNoBoardFound
}

var errNoBoardFound = errors.New("no arduino-like board found")

// AvailablePort describes a serial port for the UI's port picker
type AvailablePort struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsArduino   bool   `json:"isArduino"`
}

// ListPorts returns the serial ports currently present on the system
func ListPorts() ([]AvailablePort, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("list serial ports: %w", err)
	}

	best := bestBoardPort(ports)
	result := make([]AvailablePort, 0, len(ports))

	for _, p := range ports {
		result = append(result, AvailablePort{
			Name:        p.Name,
			Description: portDescription(p),
			IsArduino:   best != nil && p.Name == best.Name,
		})
	}

	return result, nil
}

func bestBoardPort(ports []*enumerator.PortDetails) *enumerator.PortDetails {
	for _, vid := range knownBoardVendorIDs {
		for _, p := range ports {
			if p.IsUSB && strings.EqualFold(p.VID, vid) {
				return p
			}
		}
	}

	return nil
}

func portDescription(p *enumerator.PortDetails) string {
	description := strings.TrimSpace(p.Product)

	// windows friendly names end with the port, e.g. "Arduino Uno (COM3)"
	description = strings.TrimSpace(strings.TrimSuffix(description, fmt.Sprintf("(%s)", p.Name)))

	if description == "" && p.IsUSB {
		description = fmt.Sprintf("USB %s:%s", strings.ToUpper(p.VID), strings.ToUpper(p.PID))
	}

	return description
}

func (sio *SerialIO) describeSerialError(portName string, err error) string {
	if errors.Is(err, errNoBoardFound) {
		return sio.t("error.noBoard")
	}

	var portErr *serial.PortError
	if errors.As(err, &portErr) {
		switch portErr.Code() {
		case serial.PortBusy, serial.PermissionDenied:
			return sio.t("error.busy", portName)
		case serial.PortNotFound:
			return sio.t("error.notFound", portName)
		}
	}

	return err.Error()
}

// waitBeforeRetry sleeps for the reconnect interval (or until a manual reconnect is asked for).
// it returns false if deej is stopping
func (sio *SerialIO) waitBeforeRetry() bool {
	var retry <-chan time.Time
	if sio.deej.config.AutoReconnect {
		retry = time.After(sio.deej.config.ReconnectInterval)
	}

	select {
	case <-sio.stopChannel:
		return false
	case <-sio.reconnectNow:
		return true
	case <-retry:
		return true
	}
}

func (sio *SerialIO) stopped() bool {
	select {
	case <-sio.stopChannel:
		return true
	default:
		return false
	}
}

func (sio *SerialIO) setStatus(mutate func(*ConnectionStatus)) {
	sio.lock.Lock()
	previous := sio.status
	mutate(&sio.status)
	if sio.status.State != previous.State {
		sio.status.Since = time.Now()
	}
	status := sio.status
	sio.lock.Unlock()

	if status == previous {
		return
	}

	for _, consumer := range sio.statusConsumers {
		select {
		case consumer <- status:
		default:
		}
	}
}

func (sio *SerialIO) t(key string, args ...any) string {
	return sio.deej.config.T(key, args...)
}

func indexOfByte(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}

	return -1
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}

	if v > 1 {
		return 1
	}

	return v
}
