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
	// Pictures kept on this machine, so a menu grid does not fill in tile by
	// tile on a restaurant's connection. See imagecache_windows.go.
	cache *imageCache
	// pairing holds the administrator's session for the length of the setup
	// screen only. ⚠️ Never written to disk: it is a full panel credential, and
	// the whole point of the device token is that a monoblock does not keep
	// one. It dies with the process, which is the correct lifetime for
	// something used once between two button presses.
	pairing pairSession
}

// NewApp reads the log and the pairing before the window is built.
//
// ⚠️ **Before, because the window is built from them.** The zoom and the GPU
// switch are passed to wails.Run as values, so anything loaded in OnStartup
// arrives after the decision has been taken: a till.json saying `"gpu": "off"`
// or `"zoom": 0.75` was read, kept, shown by Status — and had no effect on the
// window. Both settings exist for the machine that is misbehaving in front of
// somebody, and the remedy the README gives for a stuttering or wedged screen
// was inert on every installed till.
func NewApp() *App {
	openLog()
	return &App{cfg: loadSettings(), cache: newImageCache()}
}

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

// openLog points the standard logger at a file in %PROGRAMDATA%\Keel.
//
// ⚠️ **A GUI program has no console**, so every line the agent writes would go
// nowhere — including the one that says the token was refused, which is the
// single most likely thing to be wrong on the day this is installed.
//
// ⚠️ **Beside the pairing, not beside the executable.** The installer puts the
// program in Program Files, which a cashier's account cannot write to — so the
// log silently did not exist on exactly the machines it was written for, and
// the first question about an installed till ("what does the log say?") had no
// answer. Same reason till.json moved there.
func openLog() {
	if err := os.MkdirAll(configDir(), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(configDir(), "till.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(f)
}

func (a *App) startup(ctx context.Context) {
	a.ctx, a.stop = context.WithCancel(ctx)
	// ⚠️ **Undoing what an earlier build wrote.** The till draws its own
	// keyboard; Windows' was still being raised on machines where a previous
	// version had switched the setting on, because removing the call that wrote
	// it does not unwrite it. See keyboard_windows.go.
	disableTouchKeyboard()
	// ⚠️ The pairing is already loaded (NewApp) and must not be read again here:
	// re-reading would be harmless today and wrong the moment the setup screen
	// has written a file this process has not adopted.
	a.startAgent()
	// ⚠️ Only on a paired machine. An unpaired one is somebody's laptop
	// halfway through a setup, and replacing the binary under them is a
	// surprise nobody asked for — see update_windows.go.
	if a.cfg.paired() {
		// ⚠️ **Applied at boot, before anything is sold.** If this succeeds the
		// process is about to be closed by the installer it just started, so
		// there is no point starting the poller behind it.
		if a.applyStagedUpdateAtBoot() {
			return
		}
		a.startUpdater(a.ctx)
		// ⚠️ After the updater and never before the window: this downloads the
		// whole menu's photographs, and a till that spent its first minute on
		// pictures instead of opening would have traded one visible wait for a
		// worse one.
		a.warmImages()
	}
}

// TillVersion is what build this is, for the screen to show and to decide
// whether an update is worth applying.
//
// ⚠️ Bound rather than baked into the frontend: the two are built together but
// shipped as one binary, and a version string duplicated in JavaScript is one
// that gets forgotten on the release where it matters.
func (a *App) TillVersion() string { return Version }

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
	// ⚠️ A clean close checkpoints the WAL. Skipping it is not a loss — the
	// next start recovers — but it turns every ordinary shutdown into the
	// recovery path, and then the recovery path is never the exceptional one
	// anybody notices going wrong.
	closeStore()
	log.Print("kassa ilovasi yopildi")
}

// Quit closes the window. Bound because the frame is gone: with no title bar
// there is no close button but the one the screen draws.
func (a *App) Quit() { wruntime.Quit(a.ctx) }

// DeviceToken hands the screen the branch token this machine was paired with.
//
// ⚠️ **Given to the screen rather than injected by the proxy**, because the till
// alternates between this and the unlocked person's token per call (lib/api.ts,
// tillAuth) — that alternation is what makes a void carry the name of whoever
// is standing there. The browser till keeps the same token in localStorage for
// the same reason; this is where it comes from when there is no panel link to
// open.
func (a *App) DeviceToken() string { return a.cfg.Token }

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
