//go:build windows

package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// ⚠️ `all:` keeps files Wails needs that begin with `_` or `.`; without it the
// build succeeds and the window comes up blank, which reads as a broken app
// rather than a missing directive.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// ---- The elevated half of an update ----
	//
	// ⚠️ **Checked before anything else, and it never opens a window.** The
	// scheduled task the installer registered runs this same binary with this
	// flag, at administrator level, to apply a staged installer — the only way
	// a till in Program Files can replace its own files without a UAC prompt on
	// a counter. See update_windows.go.
	if len(os.Args) > 1 && os.Args[1] == "--apply-update" {
		os.Exit(RunStagedInstaller())
	}

	app := NewApp()

	// ⚠️ **Frameless and genuinely fullscreen.** Maximised was the earlier
	// choice and it was wrong on the hardware: a maximised window stops above
	// the taskbar, so a monoblock showed a strip of Windows along the bottom
	// all evening — the thing an appliance must not do, and the first thing
	// anybody notices.
	//
	// The reason it was maximised was to keep Alt+Tab as a way out of a wedged
	// app. That still holds, so the escape is written down instead of designed
	// around: **Alt+F4** closes the window even when this program has stopped
	// listening, because Windows sends it, and it belongs in the install notes.
	// A screen that has only lost its controls is covered by Ctrl+Shift+Q.
	err := wails.Run(&options.App{
		Title:            "Keel",
		Width:            1280,
		Height:           800,
		Frameless:        true,
		WindowStartState: options.Fullscreen,
		AssetServer:      &assetserver.Options{Assets: assets, Middleware: app.proxy},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind:             []any{app},
		Windows: &windows.Options{
			// ⚠️ **Set per machine, not guessed once here.** Monoblocks differ
			// in resolution and in the Windows display scaling somebody left on
			// them, and both land on the webview — so the same build reads
			// comfortably on one counter and enormous on the next. A number in
			// the pairing file is adjustable by whoever is standing in front of
			// the screen; a constant in this binary needs a release.
			ZoomFactor: app.cfg.zoom(),
			// See settings.GPU: the driver decides which way this helps.
			WebviewGpuIsDisabled: app.cfg.GPU == "off",
			// ⚠️ The zoom is ours to set, not the cashier's to change by
			// accident. Ctrl+scroll on a touch screen is one careless swipe,
			// and a till at 300% mid-service is a till nobody can use.
			IsZoomControlEnabled: false,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
