; Inno Setup script for doc-html-translate - a single universal setup.exe that runs on
; both x86 and x64 Windows and installs the matching architecture. Per-user by default
; (no admin / no UAC): {autopf} resolves to %LocalAppData%\Programs under PrivilegesRequired=lowest.
;
; NOT built by hand - scripts/build-installer.ps1 builds both architectures, stages the
; payload, and calls ISCC with these /D defines:
;   AppVersion  -> YY.MMDD.HHmm  (four-part is not needed here, unlike MSIX)
;   Staging     -> payload root (x64\, x86\, tessdata\, icon, LICENSE, README.md)
;   Output      -> where the compiled setup.exe is written
;
; File associations are intentionally NOT hard-coded here. The app owns that logic
; (internal/windowsreg): the optional "openwith" task runs `doc-html-translate.exe
; -register-openwith`, which non-destructively adds the app to the "Open with" list and
; the right-click "Convert to HTML" menu without hijacking any file type's default. The
; [Registry] sweep below removes those HKCU entries on uninstall so nothing dangles.

#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif
#ifndef Staging
  #define Staging "..\temp\installer-staging"
#endif
#ifndef Output
  #define Output "..\dist"
#endif

#define AppName "doc-html-translate"
#define AppExe "doc-html-ui.exe"
#define CliExe "doc-html-translate.exe"

[Setup]
; Stable per-product AppId - keeps upgrades/uninstall linked across versions. Do not change.
AppId={{E8B4F1C7-2A9D-4E63-9F1B-7C3A5D8E2B04}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher=SZA
AppPublisherURL=https://sza.od.ua
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
; Per-user, no admin. {autopf} -> %LocalAppData%\Programs, {autoprograms} -> per-user Start menu.
PrivilegesRequired=lowest
; Universal package: install x64 binaries on 64-bit Windows, x86 on 32-bit Windows.
ArchitecturesInstallIn64BitMode=x64compatible
ArchitecturesAllowed=x86compatible or x64compatible
OutputDir={#Output}
OutputBaseFilename=doc-html-translate-setup-{#AppVersion}
SetupIconFile={#Staging}\doc-html-translate.ico
UninstallDisplayIcon={app}\{#AppExe}
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
ChangesAssociations=yes

[Languages]
Name: "en"; MessagesFile: "compiler:Default.isl"
Name: "ru"; MessagesFile: "compiler:Languages\Russian.isl"
Name: "uk"; MessagesFile: "compiler:Languages\Ukrainian.isl"

[CustomMessages]
en.OpenWithTask=Add to the "Open with" list and the right-click "Convert to HTML" menu
ru.OpenWithTask=Добавить в список "Открыть с помощью" и пункт "Convert to HTML" в контекстное меню
uk.OpenWithTask=Додати до списку "Відкрити за допомогою" та пункт "Convert to HTML" у контекстне меню
en.ChromeExtTask=Install the Chrome browser extension (Document Page Translator)
ru.ChromeExtTask=Установить расширение для браузера Chrome (Document Page Translator)
uk.ChromeExtTask=Встановити розширення для браузера Chrome (Document Page Translator)
en.EdgeExtTask=Install the Microsoft Edge browser extension (Document Page Translator)
ru.EdgeExtTask=Установить расширение для браузера Microsoft Edge (Document Page Translator)
uk.EdgeExtTask=Встановити розширення для браузера Microsoft Edge (Document Page Translator)

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; Flags: unchecked
Name: "openwith"; Description: "{cm:OpenWithTask}"
; Optional browser extensions. Only shown when the matching browser is detected (Check: below);
; opt-in (unchecked). These write a per-user "external extension" registry pointer (HKCU, no admin);
; the browser then installs the extension from its store on next launch and shows its own "enable?"
; prompt. Removed on uninstall via the [Registry] uninsdeletekey entries.
Name: "chromeext"; Description: "{cm:ChromeExtTask}"; Flags: unchecked; Check: ChromeInstalled
Name: "edgeext"; Description: "{cm:EdgeExtTask}"; Flags: unchecked; Check: EdgeInstalled

[Files]
; --- x64 binaries (64-bit Windows) ---
Source: "{#Staging}\x64\{#AppExe}"; DestDir: "{app}"; Flags: ignoreversion; Check: Is64BitInstallMode
Source: "{#Staging}\x64\{#CliExe}"; DestDir: "{app}"; Flags: ignoreversion; Check: Is64BitInstallMode
; --- x86 binaries (32-bit Windows) ---
Source: "{#Staging}\x86\{#AppExe}"; DestDir: "{app}"; Flags: ignoreversion; Check: not Is64BitInstallMode
Source: "{#Staging}\x86\{#CliExe}"; DestDir: "{app}"; Flags: ignoreversion; Check: not Is64BitInstallMode
; --- shared payload (architecture-independent) ---
Source: "{#Staging}\tessdata\*"; DestDir: "{app}\tessdata"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#Staging}\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#Staging}\README.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExe}"
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#AppExe}"; Tasks: desktopicon

[Run]
; Non-destructive association registration (HKCU, no admin). Scriptable path in app.go:
; prints its result and exits, so runhidden is safe.
Filename: "{app}\{#CliExe}"; Parameters: "-register-openwith"; Flags: runhidden nowait; Tasks: openwith; StatusMsg: "Registering file associations..."
; Offer to launch the GUI when the wizard finishes.
Filename: "{app}\{#AppExe}"; Description: "{cm:LaunchProgram,{#AppName}}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
; Release the default-handler association if the user had opted into it (harmless otherwise).
Filename: "{app}\{#CliExe}"; Parameters: "-unregister"; Flags: runhidden; RunOnceId: "unregister"

[Registry]
; --- Optional browser-extension pointers (per-user, HKCU, no admin) -------------------------
; Written only when the "chromeext"/"edgeext" task is selected (that task is only offered when
; the browser is detected). The browser reads this "external extension" key on next launch,
; installs the extension from its own store, and prompts the user to enable it. uninsdeletekey
; removes the pointer on uninstall (the browser then drops the externally-installed extension).
; Extension ids: Chrome item nmcckamdocainafmmompkbmelkpbnmic; Edge CRX anokfnnfiboaccbbpfkdaphejnkkhajh.
Root: HKCU; Subkey: "Software\Google\Chrome\Extensions\nmcckamdocainafmmompkbmelkpbnmic"; ValueType: string; ValueName: "update_url"; ValueData: "https://clients2.google.com/service/update2/crx"; Flags: uninsdeletekey; Tasks: chromeext
Root: HKCU; Subkey: "Software\Microsoft\Edge\Extensions\anokfnnfiboaccbbpfkdaphejnkkhajh"; ValueType: string; ValueName: "update_url"; ValueData: "https://edge.microsoft.com/extensionwebstorebase/v1/crx"; Flags: uninsdeletekey; Tasks: edgeext

; Sweep the HKCU entries written by -register-openwith / -register-contextmenu on uninstall.
; dontcreatekey = never created by the installer (the app writes them); uninsdeletekey = remove on uninstall.
Root: HKCU; Subkey: "Software\Classes\Applications\{#CliExe}"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\doc-html-translate"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.epub\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.pdf\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.txt\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.md\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.fb2\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.rtf\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.html\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.htm\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.mobi\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.azw3\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.cbz\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.cbr\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.cb7\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\SystemFileAssociations\.cbt\shell\dochtmltranslate.convert"; Flags: dontcreatekey uninsdeletekey

[Code]
{ Browser detection - the extension tasks are offered only when the matching browser is present.
  Chrome: App Paths (HKLM or HKCU per-user install) or the HKCU product key.
  Edge:   App Paths (system-wide on Windows 10/11). }
function ChromeInstalled: Boolean;
begin
  Result := RegKeyExists(HKLM, 'SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe') or
            RegKeyExists(HKCU, 'SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe') or
            RegKeyExists(HKCU, 'Software\Google\Chrome');
end;

function EdgeInstalled: Boolean;
begin
  Result := RegKeyExists(HKLM, 'SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\msedge.exe') or
            RegKeyExists(HKCU, 'SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\msedge.exe');
end;
