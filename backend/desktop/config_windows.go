//go:build windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// settings is what this machine remembers about the restaurant it belongs to.
//
// ⚠️ **Written by the application, not by a person.** The first version of this
// was a JSON file somebody filled in by hand, which meant visiting every branch
// of every restaurant with a text editor — and a mistyped slug fails silently,
// as a till that simply never prints. Now the setup screen writes it.
type settings struct {
	// Address is what was typed on the setup screen: a slug ("osh") or a full
	// host ("kassa.restoran.uz").
	Address string `json:"address"`
	// Server is the resolved API base, stored rather than derived again so a
	// change in how slugs expand cannot orphan a till that is already working.
	Server string `json:"server"`
	// Token is the branch device token (role "tilldevice"), good for a year and
	// revocable from the panel.
	Token      string `json:"token"`
	BranchID   string `json:"branchId"`
	BranchName string `json:"branchName"`
	Verbose    bool   `json:"verbose"`
	// Zoom scales the whole screen. 1 is the design's own size; below 1 fits
	// more on a small or heavily-scaled display.
	//
	// ⚠️ **Zero means 1, not "invisible".** Every till paired before this field
	// existed has no value for it, and reading that as a scale factor would
	// collapse the screen to nothing on machines that are working today. Same
	// rule as everywhere else in this codebase: the zero value is today's
	// behaviour.
	Zoom float64 `json:"zoom"`
}

// zoom is the scale to render at, with the sane bounds applied.
//
// ⚠️ Clamped, because this is a hand-edited file: a stray zero, a decimal comma
// read as nothing, or a fat-fingered 10 all produce a screen nobody can use and
// no way to fix it from inside the app.
func (s settings) zoom() float64 {
	if s.Zoom < 0.5 || s.Zoom > 2 {
		return 1
	}
	return s.Zoom
}

func (s settings) paired() bool { return s.Server != "" && s.Token != "" }

// configDir is %PROGRAMDATA%\Keel.
//
// ⚠️ **Not beside the executable.** A program installed under Program Files
// cannot write next to itself without elevation, so the setup screen would fail
// to save on exactly the machines this is built for. ProgramData is the
// directory Windows provides for state shared by every user of the machine,
// which is what a till's pairing is — the monoblock belongs to the branch, not
// to whoever happens to be logged in.
func configDir() string {
	if base := os.Getenv("ProgramData"); base != "" {
		return filepath.Join(base, "Keel")
	}
	return filepath.Join(exeDir(), "config")
}

func configPath() string { return filepath.Join(configDir(), "till.json") }

// loadSettings reads the pairing, falling back to the hand-written file this
// replaced.
//
// ⚠️ The legacy path is still read, because a till set up by hand before the
// setup screen existed is a till in a restaurant that is selling. It is never
// written back to.
func loadSettings() settings {
	var s settings
	if raw, err := os.ReadFile(configPath()); err == nil {
		_ = json.Unmarshal(raw, &s)
	}
	if !s.paired() {
		if raw, err := os.ReadFile(filepath.Join(exeDir(), "till.json")); err == nil {
			var legacy struct {
				Server  string `json:"server"`
				Token   string `json:"token"`
				Verbose bool   `json:"verbose"`
			}
			if json.Unmarshal(raw, &legacy) == nil {
				s.Server, s.Token, s.Verbose = legacy.Server, legacy.Token, legacy.Verbose
			}
		}
	}
	// Environment wins, because it is what a support call can change without
	// asking somebody in a restaurant to edit JSON over the phone.
	if v := os.Getenv("KEEL_SERVER"); v != "" {
		s.Server = v
	}
	if v := os.Getenv("KEEL_AGENT_TOKEN"); v != "" {
		s.Token = v
	}
	return s
}

func saveSettings(s settings) error {
	if err := os.MkdirAll(configDir(), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	// ⚠️ Written to a temporary file and renamed. A half-written pairing is a
	// till that starts, reads broken JSON and asks to be set up again — with a
	// queue of unsent sales behind it.
	tmp := configPath() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, configPath())
}

// apiBase turns what was typed into an API root.
//
// ⚠️ **A bare word is a Keel subdomain, deliberately.** Every tenant is given
// <slug>.keel.uz at provisioning (control/handlers/tenants.go), and that
// address is ours: it cannot expire, move registrar or go unpaid the way a
// restaurant's own domain can. A till pointed at a customer domain stops
// selling the day that domain lapses, and the symptom — no receipts — looks
// nothing like the cause.
//
// A dotted address is taken as typed, for the restaurant that wants its own.
func apiBase(address string) string {
	a := strings.TrimSpace(strings.ToLower(address))
	a = strings.TrimPrefix(strings.TrimPrefix(a, "https://"), "http://")
	a = strings.TrimSuffix(a, "/")
	a = strings.TrimSuffix(a, "/api/v1")
	if a == "" {
		return ""
	}
	if !strings.Contains(a, ".") {
		a += ".keel.uz"
	}
	return "https://" + a + "/api/v1"
}
