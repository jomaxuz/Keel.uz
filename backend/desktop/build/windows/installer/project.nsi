Unicode true

####
## ⚠️ **This file must keep its UTF-8 BOM.**
##
## `Unicode true` decides what the *installer* speaks, not what the compiler
## reads. Without a BOM, makensis reads the source in the build machine's ANSI
## codepage — so every Cyrillic letter here arrives as two mojibake characters,
## and it does so **silently**: the build succeeds, the installer runs, and the
## only symptom is a wizard nobody can read. That shipped once.
##
## Most editors preserve a BOM once it is there. If a diff ever shows the first
## line as plain `Unicode true` with no EF BB BF before it, that is the bug
## coming back: `file project.nsi` must say "UTF-8 Unicode (with BOM) text".
####

####
## Установщик Keel.
##
## Wails' template with three changes, each noted where it is. wails_tools.nsh
## beside this file is Wails' own and is copied unmodified — it is the part that
## installs the WebView2 runtime, and editing it would be editing the one thing
## that makes a clean Windows 10 able to run this at all.
##
## Build:  wails build -nsis    (NSIS must be on PATH)
## Output: build\bin\keel-amd64-installer.exe
####
####
## Please note: Template replacements don't work in this file. They are provided with default defines like
## mentioned underneath.
## If the keyword is not defined, "wails_tools.nsh" will populate them with the values from ProjectInfo.
## If they are defined here, "wails_tools.nsh" will not touch them. This allows to use this project.nsi manually
## from outside of Wails for debugging and development of the installer.
##
## For development first make a wails nsis build to populate the "wails_tools.nsh":
## > wails build --target windows/amd64 --nsis
## Then you can call makensis on this file with specifying the path to your binary:
## For a AMD64 only installer:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app.exe
## For a ARM64 only installer:
## > makensis -DARG_WAILS_ARM64_BINARY=..\..\bin\app.exe
## For a installer with both architectures:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app-amd64.exe -DARG_WAILS_ARM64_BINARY=..\..\bin\app-arm64.exe
####
## The following information is taken from the ProjectInfo file, but they can be overwritten here.
####
## !define INFO_PROJECTNAME    "MyProject" # Default "{{.Name}}"
## !define INFO_COMPANYNAME    "MyCompany" # Default "{{.Info.CompanyName}}"
## !define INFO_PRODUCTNAME    "MyProduct" # Default "{{.Info.ProductName}}"
## !define INFO_PRODUCTVERSION "1.0.0"     # Default "{{.Info.ProductVersion}}"
## !define INFO_COPYRIGHT      "Copyright" # Default "{{.Info.Copyright}}"
###
## !define PRODUCT_EXECUTABLE  "Application.exe"      # Default "${INFO_PROJECTNAME}.exe"
## !define UNINST_KEY_NAME     "UninstKeyInRegistry"  # Default "${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"
####
## !define REQUEST_EXECUTION_LEVEL "admin"            # Default "admin"  see also https://nsis.sourceforge.io/Docs/Chapter4.html
####
## Include the wails tools
####
## ⚠️ **Pinned, because the product was renamed and these are stored.**
##
## `UNINST_KEY_NAME` defaults to company+product, which after the rename would
## be "KeelKeel" — and, worse, a *different* key from the one every installed
## copy already wrote. Windows would list two Keels in Programs & Features and
## the old entry would point at files this installer is about to replace.
## Pinning it also means the next rename cannot silently orphan an install.
!define UNINST_KEY_NAME "Keel"

## The install path, spelled out for the same reason: company\product is
## `Keel\Keel` now, and a folder repeating itself is a folder somebody assumes
## is a mistake and deletes.
!define KEEL_INSTALL_DIR "$PROGRAMFILES64\Keel"

## Where the previous name put everything. ⚠️ Kept as a constant rather than
## typed into the two places that need it: this is the one string that must not
## drift, because both uses of it *delete* a directory.
!define KEEL_OLD_DIR "$PROGRAMFILES64\Keel\Keel Kassa"

!include "wails_tools.nsh"

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"
## GetParent, for resolving C:\Users without building a path out of "..".
!include "FileFunc.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"

## ⚠️ **BMP, never PNG, and at exactly these sizes.** NSIS reads these as
## bitmaps and says nothing when it cannot — the same silence that shipped an
## installer wearing Wails' logo. MUI fixes the sizes: 164x314 for the side
## panel, 150x57 for the header strip. Anything else is stretched without
## complaint.
!define MUI_WELCOMEFINISHPAGE_BITMAP "welcome.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "welcome.bmp"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_RIGHT
!define MUI_HEADERIMAGE_BITMAP "header.bmp"

!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.

## ---- The words, in the language the room actually reads --------------------
##
## ⚠️ **Russian, and it is the language file rather than a pile of overrides.**
## This wizard used to sit in the English slot with every visible string
## replaced by an Uzbek one — which worked for the strings we thought to
## replace and left the rest ("Installing", "Please wait", the uninstall
## prompts, every error) in English. NSIS ships a complete Russian file and no
## Uzbek one, so moving to Russian both matches the monoblocks these run on and
## makes the untranslated remainder correct instead of English.
!define MUI_WELCOMEPAGE_TITLE "Keel"
!define MUI_WELCOMEPAGE_TEXT "Программа для кассы и зала ресторана.$\r$\n$\r$\nПри первом запуске нужно указать адрес ресторана и войти под владельцем или менеджером, затем выбрать филиал. Дальше — только PIN-код.$\r$\n$\r$\nНажмите «Далее», чтобы продолжить."
!define MUI_FINISHPAGE_TITLE "Касса установлена"
!define MUI_FINISHPAGE_TEXT "Касса открывается ярлыком на рабочем столе и дальше запускается сама при включении компьютера.$\r$\n$\r$\nЗакрыть: Alt+F4 или Ctrl+Shift+Q."

## ⚠️ **The finish page does not offer to launch it.** This installer runs as
## administrator, so anything it starts is elevated too — and the first launch
## creates the WebView2 data folder, which would then belong to Administrator
## and refuse the ordinary user who opens the till tomorrow morning. The
## shortcut is the safe path, and it is one click away.

!insertmacro MUI_PAGE_WELCOME

## ⚠️ **No folder chooser.** Nobody setting up a monoblock has an opinion about
## where a till lives, and the question only creates two ways for it to be
## somewhere unexpected — one of them a network drive that is not mounted at
## boot, which turns the autostart shortcut into a dead link. Program Files,
## always.
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_INSTFILES # Uinstalling page

## ⚠️ **One language slot, still.** NSIS picks by system locale when several are
## offered, so listing two would mean a machine set to English gets the English
## file and none of the wording below. One slot means what is written here is
## what every monoblock shows, whatever its Windows is set to.
!insertmacro MUI_LANGUAGE "Russian"

## ⚠️ **Only the caption is overridden now.** Everything the previous version
## had to spell out by hand — Back, Next, Install, Cancel, "please wait", the
## uninstall prompts — is already correct in Russian.nlf, and a hand-written
## copy of it is a second translation to keep in step with nothing. The caption
## is here because it carries the product's name rather than a generic verb.
LangString ^SetupCaption     ${LANG_RUSSIAN} "Keel — установка"
LangString ^UninstallCaption ${LANG_RUSSIAN} "Keel — удаление"

## ⚠️ **Signing goes here when there is a certificate**, and until then Windows
## shows "Windows protected your PC" on this installer — more forcefully than on
## a bare .exe, because this one asks to install. pos-reja.md §9 has it as an
## open question; it is the last thing between here and handing this to a
## restaurant that did not buy it from a person standing next to them.
#!uninstfinalize 'signtool sign /fd sha256 /tr http://timestamp.digicert.com /td sha256 "%1"'
#!finalize 'signtool sign /fd sha256 /tr http://timestamp.digicert.com /td sha256 "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
!ifdef WAILS_INSTALL_SCOPE
  !if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
  !else
    InstallDir "${KEEL_INSTALL_DIR}"
  !endif
!else
  InstallDir "${KEEL_INSTALL_DIR}"
!endif # Default installing folder ($PROGRAMFILES is Program Files folder).
ShowInstDetails show # This will always show the installation details.

Function .onInit
   !insertmacro wails.checkArchitecture
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    ## ⚠️ **Close the till before overwriting it.** Windows will not replace a
    ## running executable, and NSIS reports that as a file error two thirds of
    ## the way through — leaving a half-installed till on a counter, which is
    ## the worst possible moment for it. This is not an edge case: every update
    ## after the first arrives on a machine where the till is open, because the
    ## till is always open.
    ##
    ## nsExec ships with NSIS, so this adds no plugin to install. The result is
    ## discarded: "no such process" is the ordinary answer on a first install.
    DetailPrint "Закрываем запущенную кассу..."
    nsExec::Exec 'taskkill /F /IM "${PRODUCT_EXECUTABLE}" /T'
    Pop $0
    Sleep 500

    ## ⚠️ **The previous name's installation, removed here.** This build renamed
    ## the product from "Keel Kassa" to "Keel", which moves the install folder
    ## and the shortcuts. Without this, an upgraded monoblock keeps the old
    ## directory in Program Files, an old desktop icon that still launches the
    ## old build, and an old Startup shortcut — so after the next reboot the
    ## counter comes back on the *previous* version and every update lands on a
    ## copy nobody is running.
    ##
    ## Deleting a directory is the one thing in this file worth being literal
    ## about, so the path is a constant and it is the full path, never a
    ## variable that could be empty.
    DetailPrint "Удаляем прежнюю версию..."
    RMDir /r "${KEEL_OLD_DIR}"
    Delete "$SMPROGRAMS\Keel Kassa.lnk"
    Delete "$DESKTOP\Keel Kassa.lnk"
    Delete "$SMSTARTUP\Keel Kassa.lnk"
    ## The old entry in Programs & Features, which points at files that have
    ## just stopped existing. Both hives: Wails writes HKCU, older scopes HKLM.
    DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\KeelKeel Kassa"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\KeelKeel Kassa"

    SetOutPath $INSTDIR

    !insertmacro wails.files

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    ## ⚠️ **Starts with Windows.** A till is not an application somebody chooses
    ## to open; it is what the machine is for. After a power cut — which is the
    ## normal way a restaurant reboots — the monoblock has to come back to the
    ## till on its own, because the person who would know to click an icon is
    ## the one standing at the counter with guests waiting.
    ##
    ## A Startup shortcut rather than a registry Run key or a service: it is
    ## visible in a folder anybody can open, and removing it needs no tools.
    ## pos-reja.md §2 lists autostart among the things we write ourselves.
    CreateShortCut "$SMSTARTUP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    ## ---- The task that lets the till update itself --------------------
    ##
    ## ⚠️ **Registered here because this is the one moment we are already
    ## administrator.** A till in Program Files cannot replace its own files,
    ## and asking for elevation later means a UAC dialog on a counter at eight
    ## in the evening — which a cashier either dismisses or telephones somebody
    ## about, and either way the machine stays on the old build. The task runs
    ## the same executable with `--apply-update`, at the highest privileges the
    ## account has; starting it needs none.
    ##
    ## ⚠️ **`/SC ONCE` in the past, and that is deliberate.** schtasks demands a
    ## schedule; this task is never meant to fire on its own, only when the app
    ## asks for it by name. A date that has already gone is the plainest way to
    ## say "never, unless told".
    ##
    ## ⚠️ Recreated on every install (`/F`), because the path changes with the
    ## install scope and a task pointing at the old one fails silently — which
    ## looks exactly like updates having stopped working.
    DetailPrint "Автообновление настраивается..."
    nsExec::Exec 'schtasks /Create /F /TN "KeelKassaUpdate" /SC ONCE /SD 01/01/2020 /ST 00:00 /RL HIGHEST /TR "\"$INSTDIR\${PRODUCT_EXECUTABLE}\" --apply-update"'
    Pop $0

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext

    ## The same reason as above: a running till holds its own file open.
    nsExec::Exec 'taskkill /F /IM "${PRODUCT_EXECUTABLE}" /T'
    Pop $0
    Sleep 500

    ## ---- Everything, and the reason it is everything --------------------
    ##
    ## ⚠️ **An uninstall now leaves nothing behind, which is a reversal.** This
    ## used to keep the pairing in %PROGRAMDATA%\Keel so a support person could
    ## reinstall without asking the owner for panel credentials. That is a real
    ## convenience and it was the wrong trade: what stays behind is a **branch
    ## token and a signed-in session**, on a machine that is being removed from
    ## the restaurant — sold, returned, sent for repair, or handed to somebody
    ## else. "Uninstalled" has to mean the till is gone, or the word is a lie on
    ## the one screen where it matters.
    ##
    ## The reinstall case is not lost, it is just no longer free: pairing again
    ## is the setup screen, which is the same three answers it took the first
    ## time.
    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"
    Delete "$SMSTARTUP\${INFO_PRODUCTNAME}.lnk"
    ## The previous name's shortcuts, for a machine that was upgraded rather
    ## than freshly installed.
    Delete "$SMPROGRAMS\Keel Kassa.lnk"
    Delete "$DESKTOP\Keel Kassa.lnk"
    Delete "$SMSTARTUP\Keel Kassa.lnk"

    ## The task points at a program that is about to stop existing.
    ##
    ## ⚠️ The task name is **not** renamed with the product. It is a stored
    ## identifier that the running binary looks up by name (update_windows.go),
    ## and renaming it would leave the old task on every machine already
    ## installed — pointing at a deleted executable, firing forever.
    nsExec::Exec 'schtasks /Delete /F /TN "KeelKassaUpdate"'
    Pop $0

    ## ⚠️ **The pairing and the staged update, both.** `$APPDATA` resolves to
    ## C:\ProgramData here because `wails.setShellContext` put us in all-users
    ## context — which is easy to misread, so: this is %PROGRAMDATA%\Keel, the
    ## machine-wide folder holding till.json (server address and branch token)
    ## and the downloaded installer.
    DetailPrint "Удаляем данные приложения..."
    RMDir /r "$APPDATA\Keel"

    ## ---- The signed-in session, in every user profile --------------------
    ##
    ## ⚠️ **This is the part that was actually missing.** The webview keeps its
    ## cookies and local storage — the staff token, the till token, the device
    ## binding — under the *user's* Roaming AppData. The old line here read
    ## `$AppData\${PRODUCT_EXECUTABLE}`, but under all-users shell context
    ## `$APPDATA` is ProgramData, so it deleted a path that never existed and
    ## the login survived the uninstall completely. Nothing failed; the folder
    ## was simply still there.
    ##
    ## An elevated uninstaller cannot ask "which user runs the till" — it may
    ## not be the one who launched it — so every profile is swept. A till is a
    ## single-account machine in practice, and on any machine this loop is a
    ## handful of directories that mostly do not exist.
    ## ⚠️ **`GetParent`, not "$PROFILE\..".** Every path below is handed to
    ## `RMDir /r`, and a recursive delete is the one place where a path that
    ## merely *usually* resolves is not good enough. GetParent returns
    ## "C:\Users" as a real path with no traversal in it.
    ${GetParent} "$PROFILE" $R0
    ## And if it somehow came back empty, do nothing at all rather than build a
    ## path that starts at the drive root.
    StrCmp $R0 "" keel_profiles_skip
    FindFirst $R1 $R2 "$R0\*"
    keel_profiles:
        StrCmp $R2 "" keel_profiles_done
        StrCmp $R2 "." keel_profiles_next
        StrCmp $R2 ".." keel_profiles_next
        ## Both names: a machine upgraded from "Keel Kassa" has a webview
        ## folder under the old executable name as well.
        RMDir /r "$R0\$R2\AppData\Roaming\${PRODUCT_EXECUTABLE}"
        RMDir /r "$R0\$R2\AppData\Roaming\keel.exe"
        RMDir /r "$R0\$R2\AppData\Roaming\Keel"
        RMDir /r "$R0\$R2\AppData\Local\Keel"
    keel_profiles_next:
        FindNext $R1 $R2
        Goto keel_profiles
    keel_profiles_done:
    FindClose $R1
    ## ⚠️ The empty-parent case lands here instead, past the FindClose: closing
    ## a handle that was never opened is the kind of tidy-looking line that
    ## works until the day it does not.
    keel_profiles_skip:

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
SectionEnd
