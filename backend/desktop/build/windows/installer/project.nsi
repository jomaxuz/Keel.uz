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
# !define MUI_WELCOMEFINISHPAGE_BITMAP "resources\leftimage.bmp" #Include this to add a bitmap on the left side of the Welcome Page. Must be a size of 164x314
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.

!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.
# !insertmacro MUI_PAGE_LICENSE "resources\eula.txt" # Adds a EULA page to the installer
!insertmacro MUI_PAGE_DIRECTORY # In which folder install page.
!insertmacro MUI_PAGE_INSTFILES # Installing page.
!insertmacro MUI_PAGE_FINISH # Finished installation page.

!insertmacro MUI_UNPAGE_INSTFILES # Uinstalling page

## ⚠️ Russian as well as English. NSIS picks by the system locale and shows no
## chooser, so a Russian Windows — which is most of the monoblocks sold here —
## gets a readable installer and everything else falls back to English. NSIS
## ships no Uzbek, which is why the application is in Uzbek and its installer is
## not; adding a chooser instead would put a dialog nobody needs in front of the
## one screen this has.
!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "Russian"

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
