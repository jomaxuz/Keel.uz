//go:build windows

package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// The on-screen keyboard — Windows', and why it is no longer used.
//
// ⚠️ **The till draws its own, and two keyboards are worse than none.** This
// file existed because a monoblock has no keyboard and Windows will not offer
// one in desktop mode, so a cashier tapping "comment" got a text field, a
// cursor and no way to type. That was true, and it was solved twice: the shared
// screens grew `components/till/OnScreenKeyboard`, and this kept raising
// TabTip. Both answered the same tap, and Windows' won — the wrong shape for a
// 1024×768 counter, covering the bottom third of the screen including the
// button being reached for, in whatever language Windows was installed in, and
// looking nothing like the application it covers.
//
// ⚠️ **Kept rather than deleted, and unwired rather than kept quietly.**
// `ShowKeyboard` is still bound, because a machine whose webview cannot render
// our keyboard has nothing else — and because deleting it would take the
// registry note with it, which is the thing somebody will need if a till is
// ever run on hardware where the shared keyboard does not fit. Nothing calls
// it: the focus handler is gone from main.tsx and `enableTouchKeyboard` is no
// longer called at startup.

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
