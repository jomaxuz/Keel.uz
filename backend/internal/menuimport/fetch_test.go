package menuimport

import (
	"net"
	"testing"
)

// ⚠️ **This container sits on a Docker network beside Mongo, the control plane
// and every other tenant's backend**, and this feature exists to accept a
// pasted address. `http://mongo:27017` is not a hypothetical attack, it is one
// paste into the box.
func TestPrivateAndInternalAddressesAreRefused(t *testing.T) {
	for _, ip := range []string{
		"127.0.0.1",
		"10.0.0.5",
		"172.17.0.2", // the Docker bridge, where the neighbours are
		"192.168.1.10",
		"169.254.169.254", // cloud metadata — the most valuable thing an SSRF reaches
		"100.64.0.1",
		"::1",
		"fe80::1",
		"::ffff:127.0.0.1", // the usual way round a v4-only check
		"::ffff:10.0.0.5",
	} {
		if publicIP(net.ParseIP(ip)) {
			t.Fatalf("%s was treated as a public address", ip)
		}
	}
}

// ⚠️ **The other half, and it caught a real one.** `::ffff:0:0/96` was in the
// block list because it looked like it closed the IPv4-mapped hole. Go
// normalises that prefix to `0.0.0.0/0`, so it blocked every address on the
// internet — a check that reviews cleanly and refuses the entire feature, with
// the symptom being "import never works for any site".
func TestRealAddressesAreAllowed(t *testing.T) {
	for _, ip := range []string{
		"8.8.8.8",
		"1.1.1.1",
		"93.184.216.34",
		"213.230.106.10", // uz
		"2001:4860:4860::8888",
	} {
		if !publicIP(net.ParseIP(ip)) {
			t.Fatalf("%s was refused as if it were internal", ip)
		}
	}
}

// ⚠️ A hostname with no dot is a Docker service name — `mongo`,
// `keel-control`, `keel-<slug>` — and no public site has one. Checked by name
// as well as by address because it does not have to resolve to anything this
// code would recognise for the request to be a bad idea.
func TestServiceNamesAreRefusedByName(t *testing.T) {
	for _, raw := range []string{
		"http://mongo:27017/",
		"http://localhost:8080/menu",
		"http://keel-control:9000/internal/report",
		"file:///etc/passwd",
		"gopher://example.com/",
	} {
		if _, _, err := Fetch(t.Context(), raw); err == nil {
			t.Fatalf("%s was fetched", raw)
		}
	}
}
