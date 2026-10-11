package deej

import (
	"errors"
	"os"
	"os/exec"
)

var errUnsupported = errors.New("not supported on this platform")

func acquireSingleInstance() (bool, error) {
	return true, nil
}

func startWithSystemEnabled() bool {
	return false
}

func setStartWithSystem(enabled bool) error {
	return errUnsupported
}

func openAppWindow(url string) error {
	return exec.Command("xdg-open", url).Start()
}

func revealInExplorer(path string) error {
	return exec.Command("xdg-open", path).Start()
}

func openTextFile(path string) error {
	return exec.Command("xdg-open", path).Start()
}

func systemLanguages() []string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(key); value != "" {
			return []string{value}
		}
	}

	return nil
}
