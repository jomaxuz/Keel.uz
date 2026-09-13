//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Pairing a monoblock to a branch.
//
// ⚠️ **An administrator signs in once, and the machine never holds that
// credential again.** The alternative — a username and a password kept on the
// till so it can re-authenticate — is how a restaurant ends up with one panel
// login shared by everybody and written on the wall beside the screen. What is
// kept instead is a branch device token: it opens the till and the print queue
// and nothing else, it expires in a year, and one click in the panel revokes
// every one this branch ever issued (branch.TillVersion).
//
// This is the same shape the kiosk screen next door already uses, and the same
// reasoning: the machine belongs to the branch, the people identify themselves
// with four digits afterwards (handlers/tillpin.go).

// pairSession is the administrator's session, alive only between the two button
// presses of the setup screen.
type pairSession struct {
	// address is what was typed, kept so the saved pairing records it as well
	// as the resolved base — the two answer different questions later ("what
	// did somebody enter here?" and "where does this till call?").
	address string
	base    string
	token   string
}

// BranchView is one branch the signed-in administrator may pair this till to.
type BranchView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ConnectResult is what the setup screen shows after a successful sign-in.
type ConnectResult struct {
	Server   string       `json:"server"`
	Branches []BranchView `json:"branches"`
}

func pairClient() *http.Client { return &http.Client{Timeout: 20 * time.Second} }

// Connect signs in to a restaurant and lists the branches this administrator
// may bind a till to.
//
// ⚠️ It does not save anything. Sign-in only proves the person is allowed to
// choose; the choice is a separate press, because binding the till to the wrong
// branch sends its receipts to another kitchen and its sales to another report,
// and neither is visible from this screen.
func (a *App) Connect(address, username, password string) (ConnectResult, error) {
	var out ConnectResult
	base := apiBase(address)
	if base == "" {
		return out, fmt.Errorf("restoran manzilini kiriting")
	}

	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req, err := http.NewRequestWithContext(a.ctx, http.MethodPost,
		base+"/admin/login", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := pairClient().Do(req)
	if err != nil {
		// ⚠️ Named as an address problem, because that is what it almost always
		// is: a typed slug that does not exist resolves to nothing and fails
		// here, and "login yoki parol xato" would send somebody to reset a
		// password that was never the problem.
		return out, fmt.Errorf("serverga ulanib bo'lmadi (%s) — manzilni tekshiring", base)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode == http.StatusUnauthorized {
		return out, fmt.Errorf("login yoki parol xato")
	}
	if res.StatusCode >= 300 {
		return out, fmt.Errorf("server javobi: %d", res.StatusCode)
	}
	var login struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &login); err != nil || login.Token == "" {
		return out, fmt.Errorf("serverdan tushunarsiz javob keldi")
	}

	a.pairing = pairSession{address: address, base: base, token: login.Token}

	branches, err := a.branches()
	if err != nil {
		return out, err
	}
	if len(branches) == 0 {
		// A manager with no branch, or an account that cannot see any: saying
		// so is more use than an empty list somebody stares at.
		return out, fmt.Errorf("bu hisobda filial yo'q — ega hisobi bilan kiring")
	}
	return ConnectResult{Server: base, Branches: branches}, nil
}

func (a *App) branches() ([]BranchView, error) {
	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet,
		a.pairing.base+"/admin/branches", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.pairing.token)
	res, err := pairClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("filiallar ro'yxatini olib bo'lmadi")
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("filiallar ro'yxati: server %d", res.StatusCode)
	}
	var rows []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("filiallar ro'yxatini o'qib bo'lmadi")
	}
	out := make([]BranchView, 0, len(rows))
	for _, b := range rows {
		out = append(out, BranchView{ID: b.ID, Name: b.Name})
	}
	return out, nil
}

// Pair binds this machine to a branch and starts the relay loop.
//
// ⚠️ The administrator's session is dropped as soon as the token is in hand,
// whether or not the save succeeded — leaving it alive so a retry is cheaper
// would mean a panel credential sitting in memory on an unattended machine for
// the rest of the day.
func (a *App) Pair(branchID string) error {
	if a.pairing.token == "" {
		return fmt.Errorf("avval tizimga kiring")
	}
	defer func() { a.pairing = pairSession{} }()

	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet,
		a.pairing.base+"/admin/branches/"+branchID+"/till-token", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.pairing.token)
	// ⚠️ Which machine is being bound, recorded on the row the server is about
	// to write. It is the answer to "which of these five is the one in the
	// kitchen" months later, and this request is the only moment the panel and
	// the monoblock are the same computer.
	setTillHost(req.Header)
	res, err := pairClient().Do(req)
	if err != nil {
		return fmt.Errorf("qurilma kalitini olib bo'lmadi")
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode == http.StatusForbidden {
		return fmt.Errorf("bu hisobda shu filialga ruxsat yo'q")
	}
	// ⚠️ **The one refusal that is not a fault**, and it arrived here as
	// "qurilma kaliti: server 402" — a number, on the screen of somebody
	// standing in front of a new monoblock with nothing to act on. The plan's
	// register limit is full; the server already says so in words and sends the
	// count with it, and this is the only place that can put those in front of
	// the person who just pressed the button. See handlers/tilldevices.go.
	if res.StatusCode == http.StatusPaymentRequired {
		return fmt.Errorf("%s", capMessage(raw))
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("qurilma kaliti: server %d", res.StatusCode)
	}
	var got struct {
		Token      string `json:"token"`
		BranchID   string `json:"branchId"`
		BranchName string `json:"branchName"`
	}
	if err := json.Unmarshal(raw, &got); err != nil || got.Token == "" {
		return fmt.Errorf("qurilma kaliti kelmadi")
	}

	// ⚠️ **The machine's own settings are carried over, not rebuilt.** This
	// used to construct a fresh `settings{}`, which silently cleared the
	// printer choice, the zoom and the GPU switch — every setting that exists
	// because somebody stood in front of this monoblock and fixed something.
	// Re-pairing is an ordinary event (a branch renamed, a token rotated, a
	// machine moved), and the symptom would be a till that starts printing to
	// the wrong printer for a reason nobody could connect to what they just did.
	// The same trap as AdminUpdateBranch and `soldOut`.
	cfg := a.cfg
	cfg.Address = a.pairing.address
	cfg.Server = a.pairing.base
	cfg.Token = got.Token
	cfg.BranchID = got.BranchID
	cfg.BranchName = got.BranchName
	if err := saveSettings(cfg); err != nil {
		// ⚠️ Reported rather than swallowed. A pairing that worked but was not
		// written is a till that sells all evening and asks to be set up again
		// tomorrow morning, with nobody able to say why.
		return fmt.Errorf("sozlamani saqlab bo'lmadi: %w", err)
	}
	a.cfg = cfg
	log.Printf("qurilma %s filialiga bog'landi", cfg.BranchName)
	a.startAgent()
	// ⚠️ **Here, at the end of setup, because this is the one moment somebody is
	// waiting anyway.** A machine installed in the morning has its whole menu on
	// disk before the first guest, rather than filling in tile by tile through
	// the lunch rush.
	a.warmImages()
	return nil
}

// Unpair takes this machine out of service, from the till's own screen.
//
// ⚠️ **Because a URL is not a way out of this window.** The screen used to end
// by navigating to `/staff/login`, which is correct in a browser and wrong
// here: the till is a bundled application, so the navigation left it and the
// monoblock was showing a bare webview pointed at localhost. In a restaurant
// that is a black screen with an address bar nobody can act on, at the exact
// moment somebody has just retired the machine and needs it to be obvious what
// happens next. The application has its own way back — the setup screen — and
// this is what returns to it.
//
// ⚠️ **The pairing is cleared, everything else stays.** A machine being handed
// to another branch keeps the printer plugged into it, the zoom somebody set
// for its screen, and the GPU switch. Those are facts about the hardware, not
// about the restaurant that was using it.
//
// ⚠️ **The server is told first, by the screen, and this is only the local
// half.** The order matters and is explained where the button lives: releasing
// the device row before the token is thrown away is what keeps the register
// slot from being counted against a machine nobody can find any more.
func (a *App) Unpair() error {
	cfg := a.cfg
	cfg.Token = ""
	cfg.BranchID = ""
	cfg.BranchName = ""
	if err := saveSettings(cfg); err != nil {
		return fmt.Errorf("sozlamani saqlab bo'lmadi: %w", err)
	}
	a.cfg = cfg
	// ⚠️ The relay is stopped by dropping the flag rather than the context: the
	// context belongs to the whole application and cancelling it would take the
	// updater with it. The loop itself exits on its next failed poll — its token
	// is gone — and startAgent refuses to run a second one meanwhile.
	a.agentOn = false
	log.Print("qurilma filialdan ajratildi")
	return nil
}
