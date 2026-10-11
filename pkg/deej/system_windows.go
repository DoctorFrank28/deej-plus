package deej

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName = "deej-plus"

	singleInstanceMutexName = `Local\deej-plus-single-instance`
)

// keeps the single instance mutex alive for the lifetime of the process
var singleInstanceHandle windows.Handle

// acquireSingleInstance returns false if another deej is already running for this user
func acquireSingleInstance() (bool, error) {
	name, err := windows.UTF16PtrFromString(singleInstanceMutexName)
	if err != nil {
		return false, err
	}

	handle, err := windows.CreateMutex(nil, false, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		windows.CloseHandle(handle)
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("create single instance mutex: %w", err)
	}

	singleInstanceHandle = handle

	return true, nil
}

// startWithSystemEnabled reports whether deej is registered to start when the user logs in
func startWithSystemEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(runValueName)

	return err == nil
}

// setStartWithSystem registers (or unregisters) deej to start when the user logs in
func setStartWithSystem(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open run key: %w", err)
	}
	defer key.Close()

	if !enabled {
		if err := key.DeleteValue(runValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf("delete run value: %w", err)
		}

		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	return key.SetStringValue(runValueName, fmt.Sprintf(`"%s"`, exe))
}

// openAppWindow shows the UI in a chromeless Edge window, falling back to the default browser
func openAppWindow(url string) error {
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(os.Getenv("ProgramFiles"), `Microsoft\Edge\Application\msedge.exe`),
	}

	for _, edge := range candidates {
		if _, err := os.Stat(edge); err == nil {
			return startDetached(edge, "--app="+url, "--window-size=1120,780")
		}
	}

	return startDetached("rundll32", "url.dll,FileProtocolHandler", url)
}

// revealInExplorer opens an explorer window with the given file selected
func revealInExplorer(path string) error {
	return startDetached("explorer.exe", "/select,"+path)
}

// openTextFile opens a file with notepad
func openTextFile(path string) error {
	return startDetached("notepad.exe", path)
}

func startDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}

	return cmd.Process.Release()
}

// systemLanguages returns the user's preferred UI languages, e.g. ["it-IT", "en-US"]
func systemLanguages() []string {
	languages, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err != nil {
		return nil
	}

	return languages
}
