//go:build !windows

package printer

// Everywhere else there is no spooler, and that is not a failure: the agent
// runs on Linux in development and on the server in tests, where a printer is
// a socket or a device node. Send falls straight through to the path.
func spoolPrint(string, []byte) error { return errNoSpooler }

// List has nothing to enumerate without a spooler.
//
// ⚠️ Nil rather than a stub entry: the caller offers this list to a person, and
// an invented "Default printer" row would be a choice that cannot print.
func List() []Installed { return nil }

// defaultPrinter is the same answer for the same reason.
func defaultPrinter() string { return "" }
