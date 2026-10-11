; deej plus installer (Inno Setup 6). Built by build-installer.bat, which passes:
;   /DAppVersion=1.2.3   /DReleaseDir=<folder holding deej-plus.exe>
; Per-user install: no admin rights needed, and deej plus can save its own config.yaml.

#ifndef AppVersion
  #define AppVersion "0.0.0-dev"
#endif

#ifndef ReleaseDir
  #error ReleaseDir must point to the folder with the built deej-plus.exe
#endif

#define RepoRoot "..\..\..\.."

[Setup]
AppId={{B4D1E7A0-3C5F-4E8B-A2D9-7F60C1E83B24}
AppName=deej plus
AppVersion={#AppVersion}
AppPublisher=DoctorFrank28
AppPublisherURL=https://github.com/DoctorFrank28/deej-plus
AppSupportURL=https://github.com/DoctorFrank28/deej-plus/issues
DefaultDirName={autopf}\deej-plus
DefaultGroupName=deej plus
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
OutputDir={#RepoRoot}\releases
OutputBaseFilename=deej-plus-setup-{#AppVersion}
SetupIconFile={#RepoRoot}\pkg\deej\assets\logo.ico
UninstallDisplayIcon={app}\deej-plus.exe
UninstallDisplayName=deej plus
WizardStyle=modern
Compression=lzma2
SolidCompression=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
CloseApplications=no

[Languages]
Name: "en"; MessagesFile: "compiler:Default.isl"
Name: "it"; MessagesFile: "compiler:Languages\Italian.isl"

[CustomMessages]
en.TaskStartup=Start deej plus when I sign in to Windows
it.TaskStartup=Avvia deej plus all'accesso a Windows
en.TaskDesktop=Create a desktop shortcut
it.TaskDesktop=Crea un collegamento sul desktop
en.RunDeej=Start deej plus and open its window
it.RunDeej=Avvia deej plus e apri la sua finestra

[Tasks]
Name: "startup"; Description: "{cm:TaskStartup}"
Name: "desktop"; Description: "{cm:TaskDesktop}"; Flags: unchecked

[Files]
Source: "{#ReleaseDir}\deej-plus.exe"; DestDir: "{app}"; Flags: ignoreversion

; coming from the original deej? bring its config along (onlyifdoesntexist keeps the one of an earlier deej plus)
Source: "{commonpf64}\deej\config.yaml"; DestDir: "{app}"; Flags: external skipifsourcedoesntexist onlyifdoesntexist uninsneveruninstall
Source: "{localappdata}\Programs\deej\config.yaml"; DestDir: "{app}"; Flags: external skipifsourcedoesntexist onlyifdoesntexist uninsneveruninstall
Source: "{#RepoRoot}\pkg\deej\scripts\misc\default-config.yaml"; DestDir: "{app}"; DestName: "config.yaml"; Flags: onlyifdoesntexist uninsneveruninstall

Source: "{#RepoRoot}\arduino\deej-6-sliders-filtered\deej-6-sliders-filtered.ino"; DestDir: "{app}\arduino\deej-6-sliders-filtered"; Flags: ignoreversion
Source: "{#RepoRoot}\arduino\deej-5-sliders-vanilla\deej-5-sliders-vanilla.ino"; DestDir: "{app}\arduino\deej-5-sliders-vanilla"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\deej plus"; Filename: "{app}\deej-plus.exe"; Parameters: "--open"; WorkingDir: "{app}"
Name: "{autodesktop}\deej plus"; Filename: "{app}\deej-plus.exe"; Parameters: "--open"; WorkingDir: "{app}"; Tasks: desktop

[Registry]
; the same value the app's own "start with windows" switch writes, so the two stay in sync
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "deej-plus"; ValueData: """{app}\deej-plus.exe"""; Flags: uninsdeletevalue; Tasks: startup

[Run]
Filename: "{app}\deej-plus.exe"; Parameters: "--open"; WorkingDir: "{app}"; Description: "{cm:RunDeej}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "{sys}\taskkill.exe"; Parameters: "/IM deej-plus.exe /F"; Flags: runhidden; RunOnceId: "StopDeej"

[UninstallDelete]
Type: filesandordirs; Name: "{app}\logs"

[Code]
// a running deej plus can't have its exe replaced, and a running original deej would keep the
// serial port busy: stop both first
function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
begin
  Exec(ExpandConstant('{sys}\taskkill.exe'), '/IM deej-plus.exe /F', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Exec(ExpandConstant('{sys}\taskkill.exe'), '/IM deej.exe /F', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Sleep(500);
  Result := '';
end;
