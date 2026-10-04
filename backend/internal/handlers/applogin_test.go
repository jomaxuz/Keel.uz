package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// ⚠️ **The Keel app's device keys must never be the older apps' keys.** A
// person moving from Keel Waiter to Keel on the same phone would otherwise be
// refused as "bound to another phone" — by the other app on that very phone.
func TestKeelDeviceKeysAreTheirOwn(t *testing.T) {
	old := map[string]bool{"owner": true, "waiter": true, "team": true, "courier": true}
	seen := map[string]bool{}
	for _, kind := range []string{"admin", "staff", "courier"} {
		k := keelAppKey(kind)
		if k == "" || old[k] || seen[k] {
			t.Fatalf("keel key for %s is %q — empty, shared with an older app, or reused", kind, k)
		}
		seen[k] = true
	}
}

// Against a real database: one username that is both an employee and a
// courier opens both, each bound under its own key, and a second phone is
// refused for both without locking anybody out of the other app.
func TestAppLoginOpensEveryAccountAndBindsEachOnce(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("sir"), bcrypt.MinCost)
	staffID, courierID := primitive.NewObjectID(), primitive.NewObjectID()
	if _, err := h.Store.Staff.InsertOne(ctx, models.Staff{
		ID: staffID, BranchID: branch, Name: "Aziz", Username: "aziz",
		PasswordHash: string(hash), IsActive: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Store.Couriers.InsertOne(ctx, models.Courier{
		ID: courierID, Name: "Aziz", Username: "aziz", PasswordHash: string(hash), IsActive: true,
	}); err != nil {
		t.Fatal(err)
	}

	login := func(device string) (int, []appAccount) {
		body, _ := json.Marshal(map[string]string{
			"username": "aziz", "password": "sir", "deviceId": device, "app": "keel",
		})
		rec := httptest.NewRecorder()
		h.AppLogin(rec, httptest.NewRequest(http.MethodPost, "/app/login", bytes.NewReader(body)))
		var out struct {
			Accounts []appAccount `json:"accounts"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out.Accounts
	}

	code, accs := login("phone-1")
	if code != http.StatusOK || len(accs) != 2 || accs[0].Token == "" || accs[1].Token == "" {
		t.Fatalf("first sign-in: %d %+v", code, accs)
	}
	for kind, key := range map[string]string{"staff": "keel-staff", "courier": "keel-courier"} {
		n, _ := h.Store.LoginDevices.CountDocuments(ctx, bson.M{"kind": kind, "app": key, "deviceId": "phone-1"})
		if n != 1 {
			t.Fatalf("%s was not bound under %s", kind, key)
		}
	}

	// A second phone: both accounts refused, so the request is refused.
	if code, _ := login("phone-2"); code != http.StatusConflict {
		t.Fatalf("a second phone was let in: %d", code)
	}

	// Wrong password: nothing opens.
	body, _ := json.Marshal(map[string]string{"username": "aziz", "password": "no"})
	rec := httptest.NewRecorder()
	h.AppLogin(rec, httptest.NewRequest(http.MethodPost, "/app/login", bytes.NewReader(body)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("a wrong password was answered with %d", rec.Code)
	}
}
