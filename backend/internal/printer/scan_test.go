package printer

import (
	"context"
	"net"
	"testing"
	"time"
)

// ⚠️ **What is worth sealing is the arithmetic, not the network.** A scan that
// enumerates the wrong addresses fails as "no printers found", which reads as
// there being none — and the person setting up a till has no way to tell that
// apart from a printer that is switched off.

func TestTheAddressesInASlashTwentyFour(t *testing.T) {
	_, n, err := net.ParseCIDR("192.168.123.20/24")
	if err != nil {
		t.Fatal(err)
	}
	n.IP = net.ParseIP("192.168.123.20").To4()
	got := hosts(n)

	// 254 usable, less this machine's own.
	if len(got) != 253 {
		t.Fatalf("got %d addresses, want 253", len(got))
	}
	for _, bad := range []string{"192.168.123.0", "192.168.123.255", "192.168.123.20"} {
		for _, x := range got {
			if x == bad {
				// ⚠️ The network and broadcast addresses answer nothing and
				// cost a timeout each; this machine's own would answer if the
				// till itself ever listened on 9100, and offering the till as
				// its own printer is a loop nobody would debug.
				t.Fatalf("%s should not be scanned", bad)
			}
		}
	}
	if got[0] != "192.168.123.1" {
		t.Fatalf("first = %s, want the gateway's usual address", got[0])
	}
}

func TestSomethingListeningIsFoundAndSilenceIsNot(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if !answers(ctx, "127.0.0.1", port) {
		t.Fatal("a listening socket was not found")
	}
	// ⚠️ A closed port refuses immediately; a silent host is what the timeout
	// is for. Both must come back false rather than as an error somebody sees.
	if answers(ctx, "127.0.0.1", 1) {
		t.Fatal("a closed port was reported as a printer")
	}
}

func TestAddressesComeBackInOrder(t *testing.T) {
	// Read by somebody comparing them against a self-test slip, so 2 sorts
	// before 10 rather than after it.
	if !less("192.168.1.2", "192.168.1.10") {
		t.Fatal("addresses are being sorted as text")
	}
}
