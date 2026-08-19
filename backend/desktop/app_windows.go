//go:build windows

package main

import (
	"context"
	"runtime"
)

// App is what the screen can call. It is deliberately almost empty.
//
// ⚠️ **Printing does not belong here.** The obvious binding — hand the window a
// list of receipt lines and let it encode and send them — would put a second
// decision about paper in the codebase: the server already assembles
// escpos.Options from the branch's printer record (charset, cut, full cut,
// drawer, feed lines) and queues the finished bytes as a print_job. Encoding
// them again here means the two drift, and the drift is found by a guest
// holding a receipt that came out of the wrong printer with the wrong cut.
//
// So the app takes the same route cmd/fiscalagent already takes: ask the server
// for the next job, write its bytes to the printer, report what happened. The
// bytes are decided in one place, by the code that knows which printer this is.
type App struct {
	ctx context.Context
	// stop ends the agent loop when the window closes.
	stop context.CancelFunc
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx, a.stop = context.WithCancel(ctx)
}

// ⚠️ **The loop is stopped on the way out, not left to the process exiting.**
// A print job taken off the queue and never reported is a job the server still
// believes is waiting — and the next start prints it again. The kitchen finds
// out, not us.
func (a *App) shutdown(context.Context) {
	if a.stop != nil {
		a.stop()
	}
}

// Env is what the screen needs to know about the machine it is running on. It
// exists so the first build proves the binding layer end to end: the window can
// call Go, and Go can answer.
func (a *App) Env() map[string]string {
	return map[string]string{
		"platform": runtime.GOOS,
		"arch":     runtime.GOARCH,
	}
}
