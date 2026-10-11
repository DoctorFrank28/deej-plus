package deej

import (
	"fmt"
	"strings"
)

const (
	languageAuto     = "auto"
	fallbackLanguage = "en"
)

// messages holds every user-facing string deej shows outside the UI (tray, notifications, connection
// events, API errors). adding a language means adding one more map; missing keys fall back to english
var messages = map[string]map[string]string{
	"en": {
		"tray.status.connected":    "● Connected · %s",
		"tray.status.reconnecting": "◌ Reconnecting…",
		"tray.status.busy":         "○ %s is in use by another program",
		"tray.status.disconnected": "○ Arduino not connected",
		"tray.open":                "Open deej plus",
		"tray.open.tip":            "Mixer, assignments and settings",
		"tray.reconnect":           "Reconnect now",
		"tray.reconnect.tip":       "Close and reopen the serial port",
		"tray.editConfig":          "Edit config.yaml",
		"tray.editConfig.tip":      "Open the configuration file with Notepad",
		"tray.refresh":             "Re-scan audio sessions",
		"tray.refresh.tip":         "Reload audio apps if something's stuck",
		"tray.startup":             "Start with Windows",
		"tray.startup.tip":         "Start deej plus when you sign in",
		"tray.quit":                "Quit",
		"tray.quit.tip":            "Stop deej plus and quit",

		"notify.disconnected.title":   "Arduino disconnected",
		"notify.disconnected.body":    "deej plus will pick it up again as soon as it's back.",
		"notify.reconnected.title":    "Arduino reconnected",
		"notify.reconnected.body":     "%s: volumes are under control again.",
		"notify.configReloaded.title": "Configuration reloaded!",
		"notify.configReloaded.body":  "Your changes have been applied.",
		"notify.configMissing.title":  "Can't find configuration!",
		"notify.configMissing.body":   "%s must be in the same directory as deej plus. Please re-launch",
		"notify.configInvalid.title":  "Invalid configuration!",
		"notify.configInvalid.body":   "Please make sure %s is in a valid YAML format.",
		"notify.configError.title":    "Error loading configuration!",
		"notify.configError.body":     "Please check the deej plus logs for more details.",

		"event.connected":    "Connected at %s",
		"event.disconnected": "Disconnected at %s",
		"event.sliders":      "%s · %d sliders detected",

		"error.noBoard":     "No Arduino connected",
		"error.busy":        "%s is open in another program",
		"error.notFound":    "%s not found",
		"error.read":        "read error: %v",
		"error.timeout":     "no data for %s",
		"error.readTimeout": "can't set the read timeout: %v",
		"reason.manual":     "reconnect requested",
		"reason.stopped":    "deej plus is closing",
		"api.badRequest":    "invalid request: %v",
		"api.startup":       "start with Windows: %v",
		"api.baud":          "invalid baud rate: %d",
		"api.filterLevel":   "unknown filter level: %s",
		"api.maxValue":      "invalid max value: %d",
		"api.sliderIndex":   "invalid slider index: %d",
		"api.language":      "unknown language: %s",
		"api.openUnknown":   "don't know how to open %q",
	},
	"it": {
		"tray.status.connected":    "● Connesso · %s",
		"tray.status.reconnecting": "◌ Riconnessione…",
		"tray.status.busy":         "○ %s occupata da un altro programma",
		"tray.status.disconnected": "○ Arduino non collegato",
		"tray.open":                "Apri deej plus",
		"tray.open.tip":            "Mixer, assegnazioni e impostazioni",
		"tray.reconnect":           "Ricollega ora",
		"tray.reconnect.tip":       "Chiudi e riapri la porta seriale",
		"tray.editConfig":          "Modifica config.yaml",
		"tray.editConfig.tip":      "Apri il file di configurazione con il Blocco note",
		"tray.refresh":             "Riscansiona le sessioni audio",
		"tray.refresh.tip":         "Ricarica le app audio se qualcosa non risponde",
		"tray.startup":             "Avvia con Windows",
		"tray.startup.tip":         "Avvia deej plus all'accesso",
		"tray.quit":                "Esci",
		"tray.quit.tip":            "Chiudi deej plus",

		"notify.disconnected.title":   "Arduino scollegato",
		"notify.disconnected.body":    "deej plus lo riprende da solo appena torna.",
		"notify.reconnected.title":    "Arduino ricollegato",
		"notify.reconnected.body":     "%s: i volumi sono di nuovo sotto controllo.",
		"notify.configReloaded.title": "Configurazione ricaricata",
		"notify.configReloaded.body":  "Le modifiche sono attive.",
		"notify.configMissing.title":  "Configurazione non trovata",
		"notify.configMissing.body":   "%s deve stare nella stessa cartella di deej plus. Riavvialo.",
		"notify.configInvalid.title":  "Configurazione non valida",
		"notify.configInvalid.body":   "Controlla che %s sia in formato YAML valido.",
		"notify.configError.title":    "Errore nel caricare la configurazione",
		"notify.configError.body":     "Trovi i dettagli nel log di deej plus.",

		"event.connected":    "Collegato alle %s",
		"event.disconnected": "Scollegato alle %s",
		"event.sliders":      "%s · %d slider rilevati",

		"error.noBoard":     "Nessun Arduino collegato",
		"error.busy":        "%s è aperta da un altro programma",
		"error.notFound":    "%s non trovata",
		"error.read":        "errore di lettura: %v",
		"error.timeout":     "nessun dato da %s",
		"error.readTimeout": "impossibile impostare il timeout di lettura: %v",
		"reason.manual":     "riconnessione richiesta",
		"reason.stopped":    "deej plus in chiusura",
		"api.badRequest":    "richiesta non valida: %v",
		"api.startup":       "avvio con Windows: %v",
		"api.baud":          "velocità non valida: %d",
		"api.filterLevel":   "livello di filtro sconosciuto: %s",
		"api.maxValue":      "valore massimo non valido: %d",
		"api.sliderIndex":   "indice di slider non valido: %d",
		"api.language":      "lingua sconosciuta: %s",
		"api.openUnknown":   "non so cosa aprire: %q",
	},
}

// resolveLanguage turns a configured language ("auto", "it", "it-IT"...) into one deej has messages for
func resolveLanguage(configured string) string {
	candidates := []string{configured}
	if configured == "" || strings.EqualFold(configured, languageAuto) {
		candidates = systemLanguages()
	}

	for _, candidate := range candidates {
		base := strings.ToLower(strings.SplitN(strings.ReplaceAll(candidate, "_", "-"), "-", 2)[0])
		if _, ok := messages[base]; ok {
			return base
		}
	}

	return fallbackLanguage
}

func supportedLanguage(lang string) bool {
	_, ok := messages[lang]
	return ok || lang == languageAuto
}

// translate formats a message in the given language
func translate(lang string, key string, args ...any) string {
	format, ok := messages[lang][key]
	if !ok {
		if format, ok = messages[fallbackLanguage][key]; !ok {
			format = key
		}
	}

	if len(args) == 0 {
		return format
	}

	return fmt.Sprintf(format, args...)
}

// T formats a message in the configured language
func (cc *CanonicalConfig) T(key string, args ...any) string {
	return translate(cc.resolvedLanguage, key, args...)
}
