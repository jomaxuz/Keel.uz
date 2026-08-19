package handlers

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/auth"
)

const testSecret = "test-secret-not-a-real-one"

func bearer(t *testing.T, id primitive.ObjectID, role string, ver int) string {
	t.Helper()
	tok, err := auth.GenerateLong(testSecret, id.Hex(), role, ver, time.Hour)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return "Bearer " + tok
}

// ⚠️ The till application authenticates to the agent endpoints with the device
// token it was paired with, instead of a fiscal relay token it cannot obtain
// without revoking somebody else's. That makes this the door to a branch's
// print and filing queue, so what it refuses matters more than what it accepts.
func TestTillDeviceTokenRejectsOtherRoles(t *testing.T) {
	id := primitive.NewObjectID()

	// ⚠️ Every screen in the restaurant holds a JWT signed with this same
	// secret. If the role were not checked, a guest's phone token would open
	// the branch's print queue — and nothing about the code would look wrong.
	for _, role := range []string{"till", "staff", "kiosk", "user", "owner", "manager", "courier"} {
		if _, _, ok := tillDeviceToken(testSecret, bearer(t, id, role, 0)); ok {
			t.Fatalf("role %q was accepted as a till device", role)
		}
	}

	if _, _, ok := tillDeviceToken(testSecret, bearer(t, id, "tilldevice", 0)); !ok {
		t.Fatal("a real device token was refused")
	}
}

func TestTillDeviceTokenRejectsRubbish(t *testing.T) {
	id := primitive.NewObjectID()
	cases := map[string]string{
		"empty":            "",
		"no scheme":        "abc.def.ghi",
		"wrong scheme":     "Basic " + "abc.def.ghi",
		"not a token":      "Bearer hello",
		"signed elsewhere": "Bearer " + mustToken(t, "another-secret", id),
	}
	for name, header := range cases {
		if _, _, ok := tillDeviceToken(testSecret, header); ok {
			t.Fatalf("%s was accepted", name)
		}
	}
}

func mustToken(t *testing.T, secret string, id primitive.ObjectID) string {
	t.Helper()
	tok, err := auth.GenerateLong(secret, id.Hex(), "tilldevice", 0, time.Hour)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return tok
}

// ⚠️ The version is what makes a stolen monoblock stop working, and it is
// checked against the branch — which needs a database, so it cannot be
// exercised here. What can be checked is that the freshness number reaches the
// caller at all: an earlier draft returned only the branch id, and with it the
// revocation counter would have been decoration on this route while working
// everywhere else.
func TestTillDeviceTokenCarriesTheVersion(t *testing.T) {
	id := primitive.NewObjectID()
	_, ver, ok := tillDeviceToken(testSecret, bearer(t, id, "tilldevice", 7))
	if !ok {
		t.Fatal("device token refused")
	}
	if ver != 7 {
		t.Fatalf("version %d, want 7 — the revocation check has nothing to compare", ver)
	}
}
