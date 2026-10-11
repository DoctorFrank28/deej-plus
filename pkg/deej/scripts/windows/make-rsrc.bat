@ECHO OFF

REM embeds the manifest and the app icon into the windows executable.
REM the icon is drawn by tools/genicons (go run ./tools/genicons), which also updates assets\logo.ico

SET "DEEJ_ROOT=%~dp0..\..\..\.."

ECHO Creating pkg/deej/cmd/rsrc_windows.syso
go run github.com/akavel/rsrc@v0.10.2 -arch amd64 -manifest "%DEEJ_ROOT%\pkg\deej\assets\deej.manifest" -ico "%DEEJ_ROOT%\pkg\deej\assets\logo.ico" -o "%DEEJ_ROOT%\pkg\deej\cmd\rsrc_windows.syso"
IF ERRORLEVEL 1 ECHO Failed to create rsrc_windows.syso
