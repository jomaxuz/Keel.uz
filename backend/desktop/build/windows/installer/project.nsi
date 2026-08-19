Unicode true

####
## Keel Kassa uchun o'rnatuvchi.
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

## ---- The words, in the language the app is in -----------------------------
##
## ⚠️ NSIS ships no Uzbek, so the English slot is used and every visible string
## is replaced. A restaurant installing a till in Uzbek should not meet an
## English wizard on the way in.
!define MUI_WELCOMEPAGE_TITLE "Keel Kassa"
!define MUI_WELCOMEPAGE_TEXT "Bu dastur restoran kassasi va zali uchun.$\r$\n$\r$\nO'rnatilgach birinchi ochilishda restoran manzili va ega yoki menejer logini so'raladi, so'ng filial tanlanadi. Undan keyin faqat PIN.$\r$\n$\r$\nDavom etish uchun 'Keyingi' ni bosing."
!define MUI_FINISHPAGE_TITLE "Kassa o'rnatildi"
!define MUI_FINISHPAGE_TEXT "Kassa ish stolidagi yorliqdan ochiladi va bundan keyin kompyuter yoqilganda o'zi ishga tushadi.$\r$\n$\r$\nYopish uchun: Alt+F4 yoki Ctrl+Shift+Q."

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

## ⚠️ **One language slot, and every string in it is replaced.** Adding Russian
## beside it would be worse than it sounds: NSIS picks by system locale, so a
## Russian Windows — most of these monoblocks — would get the stock Russian file
## and none of the Uzbek below. One slot means what is written here is what is
## shown, on every machine.
!insertmacro MUI_LANGUAGE "English"

## The wizard's own words. NSIS has no Uzbek language file; these override the
## English one, which is why the slot above must stay the only one.
LangString ^SetupCaption     ${LANG_ENGLISH} "Keel Kassa — o'rnatish"
LangString ^UninstallCaption ${LANG_ENGLISH} "Keel Kassa — o'chirish"
LangString ^BackBtn          ${LANG_ENGLISH} "< Orqaga"
LangString ^NextBtn          ${LANG_ENGLISH} "Keyingi >"
LangString ^InstallBtn       ${LANG_ENGLISH} "O'rnatish"
LangString ^UninstallBtn     ${LANG_ENGLISH} "O'chirish"
LangString ^CancelBtn        ${LANG_ENGLISH} "Bekor qilish"
LangString ^CloseBtn         ${LANG_ENGLISH} "Yopish"
LangString ^ShowDetailsBtn   ${LANG_ENGLISH} "Tafsilotlar"
LangString ^ClickInstall     ${LANG_ENGLISH} "O'rnatishni boshlash uchun 'O'rnatish' ni bosing."
LangString ^ClickUninstall   ${LANG_ENGLISH} "O'chirishni boshlash uchun 'O'chirish' ni bosing."
LangString ^Completed        ${LANG_ENGLISH} "Tayyor"

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
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
  !endif
!else
  InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
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
    DetailPrint "Ishlab turgan kassa yopilmoqda..."
    nsExec::Exec 'taskkill /F /IM "${PRODUCT_EXECUTABLE}" /T'
    Pop $0
    Sleep 500

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

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # Remove the WebView2 DataPath

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"
    Delete "$SMSTARTUP\${INFO_PRODUCTNAME}.lnk"

    ## ⚠️ **The pairing in %PROGRAMDATA%\Keel is deliberately left behind.**
    ## Most uninstalls are a support person reinstalling a fixed version, and
    ## wiping it there turns a two-minute reinstall into a call to the owner for
    ## panel credentials. A machine actually being retired is handled the way
    ## this system already handles it: rotate the branch key in the panel, which
    ## kills every till token that branch ever issued (branch.TillVersion).

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
SectionEnd
