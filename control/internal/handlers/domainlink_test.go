package handlers

import (
	"crypto/subtle"
	"testing"
)

// Connecting a domain is the one place a customer's own server gets to change
// something at the edge, and getting it wrong means asking a certificate
// authority for a name we do not own. The two rules that hold that line are
// pinned here; the DNS check itself needs a resolver and is exercised against
// a real domain, not in a unit test.

// A token issued for one restaurant must be worthless at another. Without
// this, any tenant container — every one of which holds a valid token — could
// name a different slug and move somebody else's domains.
func TestLinkTokenIsBoundToOneTenant(t *testing.T) {
	const secret = "s3cret"
	a := LinkToken(secret, "osh")
	b := LinkToken(secret, "somsa")
	if a == b {
		t.Fatal("two tenants share a token")
	}
	if a != LinkToken(secret, "osh") {
		t.Fatal("token is not stable — a container would lose access on restart")
	}
	// A different control secret invalidates every token, which is what makes
	// rotating the secret a working revocation.
	if a == LinkToken("other", "osh") {
		t.Fatal("token does not depend on the secret")
	}
	if subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 {
		t.Fatal("distinct tokens compared equal")
	}
}

// The platform's own names are never claimable. A customer can point a CNAME
// at us freely; without this they could then "connect" admin.keel.uz and be
// served for it.
func TestPlatformDomainsAreNotClaimable(t *testing.T) {
	refused := []string{"keel.uz", "www.keel.uz", "admin.keel.uz", "osh.keel.uz"}
	for _, d := range refused {
		if !isPlatformDomain(d, "keel.uz") {
			t.Errorf("%s should be refused as a platform domain", d)
		}
	}
	allowed := []string{"osh.uz", "keel.uz.example.com", "notkeel.uz", "mykeel.uz"}
	for _, d := range allowed {
		if isPlatformDomain(d, "keel.uz") {
			t.Errorf("%s is a customer domain and must be allowed", d)
		}
	}
}

func TestAnyMatchNeedsOneAddressInCommon(t *testing.T) {
	// A host may legitimately answer on both IPv4 and IPv6; the owner only has
	// to get one of them right.
	if !anyMatch([]string{"1.2.3.4", "::1"}, []string{"::1"}) {
		t.Error("a single shared address should be enough")
	}
	if anyMatch([]string{"1.2.3.4"}, []string{"5.6.7.8"}) {
		t.Error("unrelated addresses must not match")
	}
	if anyMatch(nil, []string{"1.2.3.4"}) {
		t.Error("a domain with no records must not pass")
	}
	// The dangerous one: an empty expectation must never mean "anything goes".
	if anyMatch([]string{"1.2.3.4"}, nil) {
		t.Error("an unresolvable server address must not accept every domain")
	}
}
