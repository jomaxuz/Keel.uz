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
	res, err := pairClient().Do(req)
	if err != nil {
		return fmt.Errorf("qurilma kalitini olib bo'lmadi")
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode == http.StatusForbidden {
		return fmt.Errorf("bu hisobda shu filialga ruxsat yo'q")
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

	cfg := settings{
		Address:    a.pairing.address,
		Server:     a.pairing.base,
		Token:      got.Token,
		BranchID:   got.BranchID,
		BranchName: got.BranchName,
	}
	if err := saveSettings(cfg); err != nil {
		// ⚠️ Reported rather than swallowed. A pairing that worked but was not
		// written is a till that sells all evening and asks to be set up again
		// tomorrow morning, with nobody able to say why.
		return fmt.Errorf("sozlamani saqlab bo'lmadi: %w", err)
	}
	a.cfg = cfg
	log.Printf("qurilma %s filialiga bog'landi", cfg.BranchName)
	a.startAgent()
	return nil
}
