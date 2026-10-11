# Changelog

## v1.0.0 — 2026-10-11

First release of deej plus, forked from deej v0.9.10.

**Connection**
- Reconnects automatically when the board is unplugged, the PC sleeps, or the port is busy, with a notification on disconnect and on reconnect.
- `com_port: auto` finds the board by its USB id.
- Optional `serial_timeout` treats a silent link as dead (for sketches that send a heartbeat).
- Ignores the partial line read when connecting mid-transmission, which could briefly set volumes on the wrong sliders.
- Only one instance runs; starting it again shows the running one's window.

**Window**
- New local window (tray → Open deej plus) with Mixer, Assignments and Settings, saving straight into `config.yaml` with its comments.
- Slider names (`slider_names`), start with Windows, open on start, language.

**Sliders**
- New noise filter with hysteresis: no flicker at rest, 1% steps when moving, exact 0% and 100%. Presets `low`, `default`, `high` and `custom` (`noise_filter`).
- `slider_max_value` for sketches that don't send 1023 at the top.
- New sketch `deej-6-sliders-filtered` with on-board averaging, change-only sending and a heartbeat.

**Other**
- English and Italian for the tray, notifications and window.
- New icon and Windows installer (per user, migrates an existing deej config).
- Builds with current Go; GitHub Actions builds the installer and drafts releases from tags.
