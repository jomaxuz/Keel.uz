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

	// ⚠️ **Frameless, and maximised rather than truly fullscreen.**
	// pos-reja.md §2 asks for a frameless window filling the screen, and for a
	// way out when the app is wedged — a monoblock that has to be unplugged is
	// the failure that section is written against. Those two pull against each
	// other: real fullscreen also takes Alt+Tab and Win+D away, so the only
	// remaining escape is the power button, which is exactly what was to be
	// avoided. Maximised and frameless looks the same to the cashier — no title
	// bar, no chrome, the whole screen — while Windows keeps its own way to
	// reach a stuck program.
	//
	// ⚠️ The close and drag controls now have to be drawn by the screen
	// (--wails-draggable), and there is deliberately no minimise: behind the
	// till there is nothing, and a minimised till is a support call that begins
	// "the cash register disappeared".
	err := wails.Run(&options.App{
		Title:            "Keel Kassa",
		Width:            1280,
		Height:           800,
		Frameless:        true,
		WindowStartState: options.Maximised,
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind:             []any{app},
	})
	if err != nil {
		log.Fatal(err)
	}
}
