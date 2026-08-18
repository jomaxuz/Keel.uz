//go:build !windows

package printer

// Everywhere else there is no spooler, and that is not a failure: the agent
// runs on Linux in development and on the server in tests, where a printer is
// a socket or a device node. Send falls straight through to the path.
func spoolPrint(string, []byte) error { return errNoSpooler }
