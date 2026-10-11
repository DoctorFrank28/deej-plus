package deej

import (
	"fmt"

	"github.com/getlantern/systray"

	"github.com/DoctorFrank28/deej-plus/pkg/deej/icon"
	"github.com/DoctorFrank28/deej-plus/pkg/deej/util"
)

func (d *Deej) initializeTray(onDone func()) {
	logger := d.logger.Named("tray")

	onReady := func() {
		logger.Debug("Tray instance ready")

		systray.SetTemplateIcon(icon.DeejLogo, icon.DeejLogo)
		systray.SetTitle(appName)
		systray.SetTooltip(appName)

		t := d.config.T

		status := systray.AddMenuItem(d.statusLabel(d.serial.Status()), "")
		status.Disable()

		systray.AddSeparator()

		openUI := systray.AddMenuItem(t("tray.open"), t("tray.open.tip"))

		reconnect := systray.AddMenuItem(t("tray.reconnect"), t("tray.reconnect.tip"))
		reconnect.SetIcon(icon.RefreshSessions)

		editConfig := systray.AddMenuItem(t("tray.editConfig"), t("tray.editConfig.tip"))
		editConfig.SetIcon(icon.EditConfig)

		refreshSessions := systray.AddMenuItem(t("tray.refresh"), t("tray.refresh.tip"))

		systray.AddSeparator()

		startWithWindows := systray.AddMenuItem(t("tray.startup"), t("tray.startup.tip"))
		if startWithSystemEnabled() {
			startWithWindows.Check()
		}

		if d.version != "" {
			systray.AddSeparator()
			versionInfo := systray.AddMenuItem(d.version, "")
			versionInfo.Disable()
		}

		systray.AddSeparator()
		quit := systray.AddMenuItem(t("tray.quit"), t("tray.quit.tip"))

		// titles follow the language setting, which can change while deej runs
		retitle := func() {
			for _, item := range []struct {
				menuItem *systray.MenuItem
				key      string
			}{
				{openUI, "tray.open"}, {reconnect, "tray.reconnect"}, {editConfig, "tray.editConfig"},
				{refreshSessions, "tray.refresh"}, {startWithWindows, "tray.startup"}, {quit, "tray.quit"},
			} {
				item.menuItem.SetTitle(t(item.key))
				item.menuItem.SetTooltip(t(item.key + ".tip"))
			}

			status.SetTitle(d.statusLabel(d.serial.Status()))
		}

		statuses := d.serial.SubscribeToStatus()
		reloads := d.config.SubscribeToChanges()

		// wait on things to happen
		go func() {
			for {
				select {

				case s := <-statuses:
					status.SetTitle(d.statusLabel(s))
					systray.SetTooltip(fmt.Sprintf("%s · %s", appName, d.statusLabel(s)))

				// the ui may have changed the language or toggled start with windows
				case <-reloads:
					retitle()

					if startWithSystemEnabled() {
						startWithWindows.Check()
					} else {
						startWithWindows.Uncheck()
					}

				case <-quit.ClickedCh:
					logger.Info("Quit menu item clicked, stopping")

					d.signalStop()

				case <-openUI.ClickedCh:
					logger.Info("Open UI menu item clicked")
					d.showUI()

				case <-reconnect.ClickedCh:
					logger.Info("Reconnect menu item clicked")
					d.serial.Reconnect()

				case <-startWithWindows.ClickedCh:
					enable := !startWithWindows.Checked()
					if err := setStartWithSystem(enable); err != nil {
						logger.Warnw("Failed to change start with Windows", "error", err)
					} else if enable {
						startWithWindows.Check()
					} else {
						startWithWindows.Uncheck()
					}

				// edit config
				case <-editConfig.ClickedCh:
					logger.Info("Edit config menu item clicked, opening config for editing")

					editor := "notepad.exe"
					if util.Linux() {
						editor = "gedit"
					}

					if err := util.OpenExternal(logger, editor, userConfigFilepath); err != nil {
						logger.Warnw("Failed to open config file for editing", "error", err)
					}

				// refresh sessions
				case <-refreshSessions.ClickedCh:
					logger.Info("Refresh sessions menu item clicked, triggering session map refresh")

					// performance: the reason that forcing a refresh here is okay is that users can't spam the
					// right-click -> select-this-option sequence at a rate that's meaningful to performance
					d.sessions.refreshSessions(true)
				}
			}
		}()

		// actually start the main runtime
		onDone()
	}

	onExit := func() {
		logger.Debug("Tray exited")
	}

	// start the tray icon
	logger.Debug("Running in tray")
	systray.Run(onReady, onExit)
}

func (d *Deej) stopTray() {
	d.logger.Debug("Quitting tray")
	systray.Quit()
}

func (d *Deej) statusLabel(s ConnectionStatus) string {
	switch s.State {
	case ConnectionConnected:
		return d.config.T("tray.status.connected", s.Port)
	case ConnectionReconnecting:
		return d.config.T("tray.status.reconnecting")
	case ConnectionBusy:
		return d.config.T("tray.status.busy", s.Port)
	default:
		return d.config.T("tray.status.disconnected")
	}
}
