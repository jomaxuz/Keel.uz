//go:build windows

package main

import (
	"net/http"
	"os"
	"strings"
)

// What this computer is called, so the panel can name it.
//
// ⚠️ **Because "Kassa 1" names a row, not a machine.** The panel's list of
// bound registers had one free-text name per row and nothing else, and a
// manager at their plan's limit has to decide which one to unbind — from a list
// where every row looks alike and the only other fact is a date that is today
// for all of them. The computer name is what is written on the machine's own
// login screen, so it is the one string that can be matched against something
// standing in a building.
//
// ⚠️ Sent on every call, not stored: a monoblock renamed after it was paired
// would otherwise keep answering to a name nobody uses any more.
func machineName() string {
	// COMPUTERNAME is what Windows itself shows; the hostname is the same
	// string on any machine that has not been given a different DNS name, and
	// the fallback matters on the ones that have.
	if n := strings.TrimSpace(os.Getenv("COMPUTERNAME")); n != "" {
		return n
	}
	if n, err := os.Hostname(); err == nil {
		return strings.TrimSpace(n)
	}
	return ""
}

// setTillHost stamps the computer name on an outgoing request, when there is
// one to stamp. Nothing is sent when the name cannot be read — an empty header
// would be indistinguishable from a browser, and the panel reads that absence
// as "no machine is behind this row".
func setTillHost(h http.Header) {
	if name := machineName(); name != "" {
		h.Set("X-Till-Host", name)
	}
}
