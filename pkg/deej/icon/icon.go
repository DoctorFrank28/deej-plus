// Package icon holds deej's icons. They're drawn by tools/genicons: edit that and run
// `go run ./tools/genicons` from the repo root instead of editing the .ico files.
package icon

import _ "embed"

// DeejLogo is the deej logo; used for notifications and the tray icon
//
//go:embed logo.ico
var DeejLogo []byte

// EditConfig is the icon of the edit config menu option
//
//go:embed edit-config.ico
var EditConfig []byte

// RefreshSessions is the icon of the reconnect menu option
//
//go:embed refresh.ico
var RefreshSessions []byte
