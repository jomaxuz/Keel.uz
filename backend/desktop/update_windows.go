//go:build windows

package main

// ---- Keeping itself current ----
//
// ⚠️ **A till nobody updates is a till that stays broken.** These monoblocks
// stand in restaurants with no IT, often behind a router nobody can reach from
// outside; "we pushed a fix" means nothing until the machine on the counter
// takes it. Left to people, an update is a visit — which is why the first
// question about any bug here has been "which build is that one on".
//
// Windows makes the mechanism awkward and the awkwardness is worth writing
// down, because every shortcut around it is a worse trade:
//
//   - The app lives in Program Files and runs as an ordinary user, so it
//     **cannot replace its own files**. Self-update by overwriting the exe is
//     off the table without elevation.
//   - Asking for elevation shows a UAC prompt. On a counter, at eight in the
//     evening, that is a dialog nobody understands standing between a guest and
//     their bill — and if the cashier is a standard user it is a password
//     nobody there has.
//   - So the elevation is arranged **once, at install time**, by the installer,
//     which is already elevated: it registers a scheduled task that runs with
//     the highest available privileges. The app can start that task without a
//     prompt. See build/windows/installer/project.nsi.
//
// ⚠️ **Downloading and applying are deliberately separate.** The download can
// happen whenever; applying kills the till, and a till must not be replaced
// while somebody is ringing up a table. So the staged installer waits, and the
// app applies it only when the screen is locked — which on a monoblock is
// every few minutes, and always overnight.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Where Keel publishes what the current build is. ⚠️ Not the restaurant's own
// server: the binary is ours, and shipping it to every tenant VPS would mean a
// hundred copies to keep in step — see control/internal/handlers/tillrelease.go.
const updateManifestURL = "https://keel.uz/internal/till/release"

// How often a paired till asks. ⚠️ Slow on purpose: a release reaches the
// counters within a working day, and the check costs a request from every
// monoblock in the country. Anything faster buys nothing — nobody is waiting on
// a till update the way they wait on a page load.
const updateCheckEvery = 6 * time.Hour

// The task the installer registers so this can elevate without a prompt.
const updateTaskName = "KeelKassaUpdate"

// ⚠️ **A cap, because this runs unattended on a restaurant's wifi.** A truncated
// or endless download would otherwise sit there holding a connection all night;
// the checksum catches the corruption, and this catches the hang.
const updateMaxBytes = 200 << 20

type releaseManifest struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Notes   string `json:"notes"`
}

// updateDir is where a downloaded installer waits to be applied.
//
// ⚠️ **ProgramData, not the user's temp.** The scheduled task that applies it
// runs with different privileges and possibly a different profile, and a path
// under one user's AppData is a path it may not be able to read. ProgramData is
// the one place both halves of this can agree on.
func updateDir() string {
	base := os.Getenv("PROGRAMDATA")
	if base == "" {
		base = `C:\ProgramData`
	}
	return filepath.Join(base, "Keel", "update")
}

func stagedPath() string     { return filepath.Join(updateDir(), "pending.exe") }
func stagedMetaPath() string { return filepath.Join(updateDir(), "pending.json") }

// staged is what is waiting to be installed, written beside the installer so
// the applying process can check the same things the downloading one did.
type staged struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	// How many times applying it has been started. See applyStagedUpdateAtBoot.
	Attempts int `json:"attempts"`
}

func writeStaged(s staged) {
	meta, err := json.Marshal(s)
	if err != nil {
		return
	}
	_ = os.WriteFile(stagedMetaPath(), meta, 0o644)
}

// startUpdater polls for a new build in the background.
//
// ⚠️ Started only on a **paired** till. An unpaired machine is somebody's
// laptop halfway through a setup, and updating it under them is a surprise
// nobody asked for.
func (a *App) startUpdater(ctx context.Context) {
	go func() {
		// A quiet first minute: a monoblock that has just booted is opening a
		// webview, loading a menu and talking to its own server, and a download
		// competing with that is the one thing anybody would notice.
		select {
		case <-ctx.Done():
			return
		case <-time.After(90 * time.Second):
		}
		for {
			if err := a.checkForUpdate(ctx); err != nil {
				// ⚠️ Logged and nothing else. A till whose update check failed
				// must go on selling: the restaurant's internet being down is
				// not a reason to put anything on the screen in front of a
				// guest.
				log.Printf("update: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(updateCheckEvery):
			}
		}
	}()
}

// checkForUpdate asks what the current build is and stages it if it is newer.
func (a *App) checkForUpdate(ctx context.Context) error {
	rel, err := fetchManifest(ctx)
	if err != nil {
		return err
	}
	if !newerVersion(Version, rel.Version) {
		return nil
	}
	// Already downloaded and waiting — do not fetch it twice a day until
	// somebody locks the screen.
	if cur, err := readStaged(); err == nil && cur.Version == rel.Version {
		return nil
	}
	log.Printf("update: %s → %s, downloading", Version, rel.Version)
	if err := download(ctx, rel); err != nil {
		return err
	}
	log.Printf("update: %s staged, waiting for the screen to be idle", rel.Version)
	return nil
}

func fetchManifest(ctx context.Context) (releaseManifest, error) {
	var rel releaseManifest
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updateManifestURL, nil)
	if err != nil {
		return rel, err
	}
	// ⚠️ Says which build is asking. Not for the till's benefit — for ours,
	// when a release is out and half the counters have not taken it.
	req.Header.Set("User-Agent", "KeelKassa/"+Version)
	res, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return rel, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return rel, fmt.Errorf("release manifest: %s", res.Status)
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&rel); err != nil {
		return rel, err
	}
	if rel.Version == "" || rel.URL == "" || len(rel.SHA256) != 64 {
		return rel, errors.New("release manifest is incomplete")
	}
	return rel, nil
}

// download fetches the installer and refuses it unless it hashes to what the
// manifest promised.
//
// ⚠️ **The checksum is the whole of the trust here.** This machine will run the
// downloaded file as administrator; a download nobody checked is a download
// somebody on the same café wifi can substitute. Written to a temporary name
// and only then renamed, so a half-finished download can never be found and
// executed by the applying half.
func download(ctx context.Context, rel releaseManifest) error {
	if err := os.MkdirAll(updateDir(), 0o755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rel.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "KeelKassa/"+Version)
	res, err := (&http.Client{Timeout: 30 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("installer: %s", res.Status)
	}

	tmp := stagedPath() + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	sum := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, sum), io.LimitReader(res.Body, updateMaxBytes))
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if n >= updateMaxBytes {
		_ = os.Remove(tmp)
		return errors.New("installer is implausibly large — refused")
	}
	if got := hex.EncodeToString(sum.Sum(nil)); !strings.EqualFold(got, rel.SHA256) {
		_ = os.Remove(tmp)
		return fmt.Errorf("checksum mismatch: got %s", got)
	}
	if err := os.Rename(tmp, stagedPath()); err != nil {
		return err
	}
	writeStaged(staged{Version: rel.Version, SHA256: rel.SHA256})
	return nil
}

func readStaged() (staged, error) {
	var s staged
	raw, err := os.ReadFile(stagedMetaPath())
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, err
	}
	if s.Version == "" || len(s.SHA256) != 64 {
		return s, errors.New("staged update is incomplete")
	}
	return s, nil
}

// applyStagedUpdateAtBoot installs what is waiting, before the window opens.
//
// ⚠️ **At startup, and nowhere else.** Applying an update closes the till — the
// installer has to, because Windows will not replace a running executable — so
// the only safe moment is one where nobody can be mid-sale. Boot is that
// moment, and it is not rare: a restaurant's monoblock is switched on every
// morning and again after every power cut, which is how restaurants reboot.
//
// The alternatives were both worse. Applying on a timer replaces the till while
// a table is being rung up. Applying when the screen locks sounds better and
// is not: the lock screen is served by the restaurant's own web frontend
// through the proxy, so the till UI would have to reach back into this
// program's bindings — coupling a tenant's page to the desktop build, and
// leaving updates broken on any tenant running an older frontend.
//
// ⚠️ **Two attempts, then it stops trying.** An installer that fails every time
// would otherwise turn every boot into a restart loop on a machine somebody is
// trying to sell from. After two the file is left where it is — for whoever
// comes to look — and the till simply comes up on the build it has.
func (a *App) applyStagedUpdateAtBoot() bool {
	s, err := readStaged()
	if err != nil {
		return false
	}
	if s.Attempts >= 2 {
		log.Printf("update: %s failed twice, leaving it alone", s.Version)
		return false
	}
	// ⚠️ Re-hashed at the last moment. The file has been sitting in a
	// world-readable folder since it was downloaded, and this is the step that
	// hands it to a process running as administrator.
	if err := verifyStaged(s); err != nil {
		log.Printf("update: staged installer rejected: %v", err)
		clearStaged()
		return false
	}
	// ⚠️ Counted **before** the attempt, not after. The attempt ends with this
	// process being killed by the installer it started, so anything written
	// afterwards is written by a process that may no longer exist — and a
	// counter that only increments on a clean failure never increments on the
	// failure that matters.
	s.Attempts++
	writeStaged(s)

	log.Printf("update: applying %s", s.Version)
	if err := runUpdateTask(); err != nil {
		log.Printf("update: %v", err)
		return false
	}
	// Nothing below here is guaranteed to run: the task's first act is to close
	// this process so its files can be replaced.
	return true
}

func verifyStaged(s staged) error {
	f, err := os.Open(stagedPath())
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	sum := sha256.New()
	if _, err := io.Copy(sum, io.LimitReader(f, updateMaxBytes)); err != nil {
		return err
	}
	if got := hex.EncodeToString(sum.Sum(nil)); !strings.EqualFold(got, s.SHA256) {
		return fmt.Errorf("checksum mismatch: got %s", got)
	}
	return nil
}

func clearStaged() {
	_ = os.Remove(stagedPath())
	_ = os.Remove(stagedMetaPath())
}

// runUpdateTask starts the elevated task the installer registered.
//
// ⚠️ **This is the only way this app gets administrator rights, and it never
// asks for them.** The task was created at install time by a process that
// already had them; starting it needs none. Without this the update would need
// a UAC prompt on a counter, which is a dialog a cashier will either dismiss or
// call somebody about — and either way the till stays on the old build.
func runUpdateTask() error {
	cmd := exec.Command("schtasks", "/Run", "/TN", updateTaskName)
	// No console window: this runs while a fullscreen till is on screen, and a
	// black box flashing over it looks like something went wrong.
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks /Run %s: %v: %s",
			updateTaskName, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// RunStagedInstaller is the elevated half, run by the scheduled task.
//
// ⚠️ **The same binary, a different mode.** A separate updater executable would
// be a second thing to sign, ship and keep in step — and the first time they
// disagreed about where the staged file lives, updates would stop with nothing
// to see. Reached with `keel.exe --apply-update`; see main_windows.go.
func RunStagedInstaller() int {
	s, err := readStaged()
	if err != nil {
		return 0
	}
	if err := verifyStaged(s); err != nil {
		log.Printf("apply: %v", err)
		clearStaged()
		return 1
	}
	// `/S` is NSIS's silent switch. The installer's own first act is to close
	// the running till — see project.nsi, which has to do it because Windows
	// will not replace a running executable.
	cmd := exec.Command(stagedPath(), "/S")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		// ⚠️ **The staged file is kept on a failure.** A half-applied install
		// is exactly when somebody needs to see what was being applied, and
		// deleting the evidence would leave a broken till and no answer.
		log.Printf("apply: installer failed: %v", err)
		return 1
	}
	clearStaged()
	// ⚠️ **Restarted through Explorer, not started directly.** This process is
	// elevated; anything it launches inherits that, and an elevated webview
	// writes its data folder as Administrator — after which the ordinary user
	// who opens the till tomorrow morning is refused. Handing the path to the
	// shell makes the new process the shell's child, at the shell's privileges,
	// which is the level the till is meant to run at.
	if exe := installedExe(); exe != "" {
		relaunch := exec.Command("explorer.exe", exe)
		relaunch.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = relaunch.Start()
	}
	return 0
}

// installedExe is where the till lives after the installer has run.
//
// ⚠️ Read from this process's own path rather than assumed, because the
// installer may have been given a different scope: the NSI supports a per-user
// install, and a hard-coded Program Files path would relaunch nothing on those
// machines and leave the counter dark.
func installedExe() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	return p
}
