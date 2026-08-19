//go:build windows

package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// The on-screen keyboard.
//
// ⚠️ **A monoblock has no keyboard, and Windows will not offer one by default.**
// The touch keyboard appears automatically in tablet mode; a till runs in
// desktop mode, where Windows waits to be asked. So a cashier tapping "comment"
// gets a text field, a blinking cursor and no way to type into it — which is
// not a missing feature, it is a screen that looks broken.
//
// Two mechanisms, because neither is reliable alone:
//
//  1. **EnableDesktopModeAutoInvoke** — the supported setting, and the one that
//     makes Windows itself raise the keyboard on any focused field, including
//     ones we never thought about. Per-user, so no elevation is needed.
//  2. **Launching TabTip.exe** on focus, for the machines where the setting has
//     not taken effect (it is read at sign-in) or where touch is not reported.

// enableTouchKeyboard asks Windows to raise the touch keyboard on focus.
//
// ⚠️ Written to HKCU, which needs no administrator — an installer that demands
// elevation is an installer somebody runs once, badly, and never again.
// Failure is logged and ignored: the fallback below still works, and a till
// that refused to start because a registry write failed would be a worse
// outcome than a keyboard that needs a tap.
func enableTouchKeyboard() {
	cmd := exec.Command("reg", "add",
		`HKCU\SOFTWARE\Microsoft\TabletTip\1.7`,
		"/v", "EnableDesktopModeAutoInvoke", "/t", "REG_DWORD", "/d", "1", "/f")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("ekran klaviaturasi sozlanmadi: %v (%s)", err, out)
		return
	}
	// ⚠️ Said in the log because it only takes effect at the next sign-in, and
	// "I set it and nothing happened" is otherwise indistinguishable from a
	// broken write. The fallback covers this session.
	log.Print("ekran klaviaturasi yoqildi (to'liq kuchga keyingi kirishda kiradi)")
}

// tabTip is where Windows keeps the touch keyboard.
func tabTip() string {
	base := os.Getenv("CommonProgramFiles")
	if base == "" {
		base = `C:\Program Files\Common Files`
	}
	return filepath.Join(base, "microsoft shared", "ink", "TabTip.exe")
}

// ShowKeyboard raises the on-screen keyboard.
//
// ⚠️ **Called from the screen on focus, and deliberately silent about failure.**
// A cashier who tapped a comment box is not helped by an error about a missing
// executable; if the keyboard does not come up, the setting above is what has
// to be fixed, and it is fixed by somebody else, later.
func (a *App) ShowKeyboard() {
	path := tabTip()
	if _, err := os.Stat(path); err != nil {
		return
	}
	cmd := exec.Command(path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		log.Printf("klaviaturani ochib bo'lmadi: %v", err)
		return
	}
	// ⚠️ Released rather than waited on: TabTip outlives this call by design,
	// and a goroutine blocked on Wait would hold a zombie for the whole shift.
	_ = cmd.Process.Release()
}
