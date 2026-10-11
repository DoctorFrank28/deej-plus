@ECHO OFF

REM builds deej plus in release mode and packs it into releases\deej-plus-setup-<version>.exe
REM usage: build-installer.bat 1.0.0

IF "%1"=="" GOTO NOVERSION

SET "DEEJ_ROOT=%~dp0..\..\..\.."
PUSHD "%DEEJ_ROOT%"
SET "DEEJ_ROOT=%CD%"
POPD

SET "RELEASE_DIR=%DEEJ_ROOT%\releases\v%1"
MKDIR "%RELEASE_DIR%" 2> NUL

for /f "delims=" %%a in ('git rev-list -1 --abbrev-commit HEAD') do @set GIT_COMMIT=%%a

ECHO Building deej plus v%1 (%GIT_COMMIT%)...
go build -o "%RELEASE_DIR%\deej-plus.exe" -ldflags "-H=windowsgui -s -w -X main.gitCommit=%GIT_COMMIT% -X main.versionTag=v%1 -X main.buildType=release" "%DEEJ_ROOT%\pkg\deej\cmd"
IF %ERRORLEVEL% NEQ 0 GOTO BUILDERROR

SET "ISCC=%LOCALAPPDATA%\Programs\Inno Setup 6\ISCC.exe"
IF NOT EXIST "%ISCC%" SET "ISCC=%ProgramFiles(x86)%\Inno Setup 6\ISCC.exe"
IF NOT EXIST "%ISCC%" GOTO NOISCC

ECHO Packing the installer...
"%ISCC%" /Q /DAppVersion=%1 "/DReleaseDir=%RELEASE_DIR%" "%~dp0installer.iss"
IF %ERRORLEVEL% NEQ 0 GOTO BUILDERROR

ECHO Done: releases\deej-plus-setup-%1.exe
GOTO DONE

:NOVERSION
ECHO Usage: build-installer.bat ^<version^>, e.g. build-installer.bat 1.0.0
EXIT /B 1

:NOISCC
ECHO Inno Setup 6 not found. Install it with: winget install JRSoftware.InnoSetup
EXIT /B 1

:BUILDERROR
ECHO Failed to build the installer, see above.
EXIT /B 1

:DONE
