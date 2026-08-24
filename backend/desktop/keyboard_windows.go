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

// disableTouchKeyboard tells Windows to stop raising its own keyboard.
//
// ⚠️ **This exists because an earlier build turned it on, and turning that build
// off does not turn the setting off.** Every till installed before the till drew
// its own keyboard has `EnableDesktopModeAutoInvoke = 1` written into its
// registry, and Windows reads that at sign-in and keeps obeying it — so a
// machine updated to a version that never asks still raised the system keyboard,
// which is exactly the "sometimes ours, sometimes Windows'" a counter reports.
// Removing the call was not enough; the write has to be undone.
//
// ⚠️ **Set to 0 rather than deleted.** A missing value means "Windows decides",
// and on a machine Windows has decided is a tablet that means the keyboard comes
// back. Zero is the only answer that stays answered.
//
// ⚠️ HKCU, so no administrator is needed. Failure is logged and ignored: a till
// that refused to start over a registry write would be a worse outcome than a
// second keyboard.
func disableTouchKeyboard() {
	cmd := exec.Command("reg", "add",
		`HKCU\SOFTWARE\Microsoft\TabletTip\1.7`,
		"/v", "EnableDesktopModeAutoInvoke", "/t", "REG_DWORD", "/d", "0", "/f")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("windows klaviaturasi o'chirilmadi: %v (%s)", err, out)
		return
	}
	log.Print("windows ekran klaviaturasi o'chirildi")
	hideTabTip()
}

// hideTabTip closes the system keyboard if it is already on screen.
//
// ⚠️ **The registry setting takes effect at the next sign-in**, so on the very
// shift a till is updated the old keyboard is still up and still popping over
// the bottom of the screen. This closes the one that is running now; the setting
// stops the next one.
//
// ⚠️ Failure is not reported and must not be: on most machines there is nothing
// to close, and `taskkill` says so with an exit code that means nothing here.
func hideTabTip() {
	cmd := exec.Command("taskkill", "/IM", "TabTip.exe", "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
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
