//go:build windows

package main

import (
	"context"
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
	// agentOn guards against a second relay loop; see startAgent.
	agentOn bool
	// pairing holds the administrator's session for the length of the setup
	// screen only. ⚠️ Never written to disk: it is a full panel credential, and
	// the whole point of the device token is that a monoblock does not keep
	// one. It dies with the process, which is the correct lifetime for
	// something used once between two button presses.
	pairing pairSession
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
	a.cfg = loadSettings()
	a.startAgent()
}

// startAgent runs the relay loop for the branch this machine is paired with.
//
// ⚠️ **The agent runs inside the till, because it is the same machine.**
// pos-reja.md §2: the app grows out of the agent rather than shipping beside
// it. Two programs means two things to install, two to update, and two to be
// running the wrong version of.
//
// ⚠️ Safe to call twice. Pairing starts it, and a machine that was already
// paired started it at boot; without the guard, setting up a till a second time
// would leave two loops racing for the same jobs.
func (a *App) startAgent() {
	if !a.cfg.paired() || a.agentOn {
		return
	}
	a.agentOn = true
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

// Status is what the setup and till screens ask for on load.
type Status struct {
	Paired     bool   `json:"paired"`
	BranchName string `json:"branchName"`
	Server     string `json:"server"`
	Agent      bool   `json:"agent"`
	Platform   string `json:"platform"`
	ConfigPath string `json:"configPath"`
}

// Status tells the screen whether this machine belongs to a branch yet.
func (a *App) Status() Status {
	return Status{
		Paired:     a.cfg.paired(),
		BranchName: a.cfg.BranchName,
		Server:     a.cfg.Server,
		Agent:      a.agentOn,
		Platform:   runtime.GOOS + "/" + runtime.GOARCH,
		ConfigPath: configPath(),
	}
}
