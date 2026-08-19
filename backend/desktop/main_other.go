//go:build !windows

// Command desktop is the Keel till application: the till and dining-room
// screens in a window, on the one machine that can reach the receipt printer.
//
// ⚠️ **This stub is why `go build ./...` still passes on Linux.** A package
// whose files are all excluded by build constraints is not skipped by `./...`
// — it is an error ("build constraints exclude all Go files"), so a
// Windows-only program with no counterpart here would turn CI red on a
// directory it was never asked to build. Same shape as printer/spool_other.go,
// and for the same reason: the platform split belongs in the files, not in a
// note telling people which command to avoid.
package main

import "fmt"

func main() {
	fmt.Println("keel-till: Windows dasturi. Build: wails build (Windows VM).")
}
