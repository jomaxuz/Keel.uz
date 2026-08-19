//go:build windows

package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// ⚠️ `all:` keeps files Wails needs that begin with `_` or `.`; without it the
// build succeeds and the window comes up blank, which reads as a broken app
// rather than a missing directive.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	// ⚠️ **Not frameless yet, and that is deliberate.** pos-reja.md §2 settles
	// on a frameless fullscreen window with our own buttons — right for a
	// monoblock, wrong for the week we are spending on whether the printer
	// answers. A window that cannot be moved or closed by ordinary means costs
	// a reboot every time the app is wedged, and this milestone will wedge it.
	// The frame comes off when the till screen is what is being tested.
	err := wails.Run(&options.App{
		Title:       "Keel Kassa",
		Width:       1280,
		Height:      800,
		AssetServer: &assetserver.Options{Assets: assets},
		OnStartup:   app.startup,
		OnShutdown:  app.shutdown,
		Bind:        []any{app},
	})
	if err != nil {
		log.Fatal(err)
	}
}
