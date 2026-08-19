//go:build windows

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"restaurant-backend/internal/agent"
)

// App is what the screen can call.
type App struct {
	ctx  context.Context
	stop context.CancelFunc
	cfg  settings
}

// settings is how this machine is told which restaurant it belongs to.
//
// ⚠️ **A file beside the executable, not a build-time constant.** One binary is
// installed in every restaurant; a server address compiled in would mean a
// build per customer, and the customer whose build was skipped finds out when
// the till cannot sell.
type settings struct {
	// Server is the Keel API root, e.g. https://restoran.example.uz/api/v1.
	Server string `json:"server"`
	// Token is the agent token from the panel (Settings → Fiscal register).
	Token string `json:"token"`
	// Verbose logs every job, not only the failures.
	Verbose bool `json:"verbose"`
}

func NewApp() *App { return &App{} }

// exeDir is where the settings and the log live: beside the program.
//
// ⚠️ Not the working directory. A shortcut in the Startup folder runs with the
// working directory set to somewhere else entirely, so a relative path finds
// nothing — and the failure is a till that starts, looks fine and never prints.
func exeDir() string {
	p, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(p)
}

func (a *App) load() {
	raw, err := os.ReadFile(filepath.Join(exeDir(), "till.json"))
	if err == nil {
		_ = json.Unmarshal(raw, &a.cfg)
	}
	// Environment wins, because it is what a support call can change without
	// asking somebody in a restaurant to edit JSON over the phone.
	if v := os.Getenv("KEEL_SERVER"); v != "" {
		a.cfg.Server = v
	}
	if v := os.Getenv("KEEL_AGENT_TOKEN"); v != "" {
		a.cfg.Token = v
	}
}

// openLog points the standard logger at a file beside the executable.
//
// ⚠️ **A GUI program has no console**, so every line the agent writes would go
// nowhere — including the one that says the token was refused, which is the
// single most likely thing to be wrong on the day this is installed.
func openLog() {
	f, err := os.OpenFile(filepath.Join(exeDir(), "till.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(f)
}

func (a *App) startup(ctx context.Context) {
	a.ctx, a.stop = context.WithCancel(ctx)
	openLog()
	a.load()

	// ⚠️ **The agent runs inside the till, because it is the same machine.**
	// pos-reja.md §2: the app grows out of the agent rather than shipping
	// beside it. Two programs means two things to install, two to update and
	// two to be running the wrong version of.
	if a.cfg.Server == "" || a.cfg.Token == "" {
		log.Print("till.json to'ldirilmagan: agent ishga tushmadi (server/token yo'q)")
		return
	}
	go agent.Run(a.ctx, agent.Config{
		Base:    a.cfg.Server,
		Token:   a.cfg.Token,
		Verbose: a.cfg.Verbose,
		Log:     log.Printf,
	})
}

// ⚠️ **The loop is stopped on the way out, not left to the process exiting.**
// A print job taken off the queue and never reported is a job the server still
// believes is waiting — and the next start prints it again. The kitchen finds
// out, not us.
func (a *App) shutdown(context.Context) {
	if a.stop != nil {
		a.stop()
	}
	log.Print("kassa ilovasi yopildi")
}

// Quit closes the window. Bound because the frame is gone: with no title bar
// there is no close button but the one the screen draws.
func (a *App) Quit() { wruntime.Quit(a.ctx) }

// Env tells the screen what it is running on, which is how it knows whether the
// Go side is reachable at all.
func (a *App) Env() map[string]string {
	return map[string]string{
		"platform": runtime.GOOS,
		"arch":     runtime.GOARCH,
		"server":   a.cfg.Server,
		"agent":    boolText(a.cfg.Server != "" && a.cfg.Token != ""),
		"exeDir":   exeDir(),
	}
}

func boolText(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
