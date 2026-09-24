// Package netguard keeps requests this server makes on somebody else's behalf
// off the network it lives on.
//
// Two callers so far: the menu importer (a URL the owner pastes) and webhooks
// (a URL the owner registers, called again and again for as long as it stays
// registered). On a shared host both are a way to reach `mongo`, the console's
// control API, a neighbouring tenant's container or the cloud metadata service
// — and the person typing the URL does not have to mean any harm for it to be
// a bad idea.
package netguard

import (
	"errors"
	"net"
	"strings"
	"syscall"
)

// ErrBlocked is the answer for an address that is not on the public internet.
var ErrBlocked = errors.New("bu manzil ochiq internetda emas")

// PublicIP reports whether ip is an ordinary public internet address.
func PublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return false
	}
	// ⚠️ Named explicitly because `IsPrivate` does not cover them and one of
	// them is the cloud metadata service — the single most valuable thing an
	// SSRF can reach on a rented server.
	for _, n := range extraBlocked {
		if n.Contains(ip) {
			return false
		}
	}
	return true
}

var extraBlocked = func() []*net.IPNet {
	var out []*net.IPNet
	for _, cidr := range []string{
		"169.254.0.0/16", // link-local, and 169.254.169.254 with it
		"100.64.0.0/10",  // carrier-grade NAT
		"192.0.0.0/24",   // IETF protocol assignments
		"198.18.0.0/15",  // benchmarking
		// ⚠️ **No `::ffff:0:0/96` here, though it looks like it belongs.** Go
		// normalises that prefix to `0.0.0.0/0`, so adding it blocks every
		// address on the internet — the check passes its own review and refuses
		// the entire feature. IPv4-mapped addresses are already handled: net.IP
		// stores them in a form the v4 checks above read correctly.
	} {
		if _, n, err := net.ParseCIDR(cidr); err == nil {
			out = append(out, n)
		}
	}
	return out
}()

// PublicHost refuses a host name that names something beside us rather than
// something on the internet, before any lookup is made.
//
// ⚠️ Checked by name as well as by address: `localhost` and the Docker service
// names beside us do not have to resolve to something PublicIP would recognise
// for the request to be a bad idea.
func PublicHost(host string) error {
	lower := strings.ToLower(strings.TrimSuffix(host, "."))
	if lower == "" || lower == "localhost" || strings.HasSuffix(lower, ".localhost") ||
		strings.HasSuffix(lower, ".internal") || strings.HasSuffix(lower, ".local") {
		return ErrBlocked
	}
	if ip := net.ParseIP(lower); ip != nil {
		if !PublicIP(ip) {
			return ErrBlocked
		}
		return nil
	}
	// ⚠️ A hostname with no dot is a Docker service name — `mongo`,
	// `keel-control`, `keel-<slug>` — which is exactly the set this is here to
	// keep out, and no public site has one.
	if !strings.Contains(lower, ".") {
		return ErrBlocked
	}
	return nil
}

// Control is a net.Dialer Control hook that refuses a connection to anything
// but a public address.
//
// ⚠️ **At dial time, on the address actually being connected to.** Resolving
// the name first and checking the answer is a race: a hostile DNS server can
// answer the check with a public address and the connection a second later
// with 127.0.0.1 (DNS rebinding). Here there is no second lookup to win.
func Control(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return ErrBlocked
	}
	ip := net.ParseIP(host)
	if ip == nil || !PublicIP(ip) {
		return ErrBlocked
	}
	return nil
}
