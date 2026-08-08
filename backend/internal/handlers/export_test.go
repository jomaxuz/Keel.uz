package handlers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The export is the one response that carries everything at once, so the tests
// here are about what must **not** be in it. Each corresponds to a way a real
// credential could reach a file that leaves the building.

func TestExportDropsEverySecretShape(t *testing.T) {
	// Field names taken from the actual settings documents, plus two that do
	// not exist yet — the point of matching on shape rather than on a list is
	// that a credential added next year is caught without anybody remembering
	// to come back here.
	secrets := []string{
		"password", "passwordHash", "key", "testKey", "secretKey",
		"consumerKey", "consumerSecret", "apiKey", "apiToken", "kioskSecret",
		"jwtSecret", "codeHash", "merchantToken", "webhookToken",
		"refreshToken", "clientSecret",
	}
	for _, k := range secrets {
		if !exportSecretKey(k) {
			t.Errorf("exportSecretKey(%q) = false — bu maydon arxivga tushadi", k)
		}
	}

	// And the ordinary business fields must survive, or the export is empty of
	// the thing it exists for.
	for _, k := range []string{
		"name", "price", "phone", "address", "total", "status", "items",
		"createdAt", "imageUrl", "posProductId", "deliveryFee",
	} {
		if exportSecretKey(k) {
			t.Errorf("exportSecretKey(%q) = true — kerakli maydon tushib qoldi", k)
		}
	}
}

func TestExportValueRedactsNestedAndArrays(t *testing.T) {
	// Shaped like the real payment settings document: the secrets are never at
	// the top level, they are two maps down inside a provider block.
	doc := bson.M{
		"name": "Osh Markazi",
		"payme": bson.M{
			"merchantId": "abc",
			"key":        "SUPER-SECRET",
			"nested":     bson.M{"testKey": "ALSO-SECRET"},
		},
		"providers": bson.A{
			bson.M{"name": "Yandex", "apiToken": "TOKEN"},
		},
	}
	out, err := json.Marshal(exportValue(doc))
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, leaked := range []string{"SUPER-SECRET", "ALSO-SECRET", "TOKEN"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("arxivda sir qoldi (%s): %s", leaked, got)
		}
	}
	if !strings.Contains(got, "Osh Markazi") || !strings.Contains(got, "Yandex") {
		t.Fatalf("kerakli ma'lumot yo'qoldi: %s", got)
	}
}

func TestExportValueMakesDatesReadable(t *testing.T) {
	// A bare millisecond integer is technically the same information and
	// practically a script somebody has to write, which is the friction this
	// whole feature exists to remove.
	at := time.Date(2026, 8, 8, 13, 20, 0, 0, time.Local)
	got := exportValue(bson.M{"createdAt": primitive.NewDateTimeFromTime(at)})
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("kutilmagan tur: %T", got)
	}
	s, ok := m["createdAt"].(string)
	if !ok || !strings.HasPrefix(s, "2026-08-08T13:20:00") {
		t.Fatalf("sana o'qilmaydigan ko'rinishda: %#v", m["createdAt"])
	}
}

func TestExportValueTurnsIDsIntoText(t *testing.T) {
	id := primitive.NewObjectID()
	m := exportValue(bson.M{"_id": id}).(map[string]any)
	if m["_id"] != id.Hex() {
		t.Fatalf("id matn bo'lmadi: %#v", m["_id"])
	}
}

func TestExportGrantExpiryFailsClosed(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.Local)

	// The case the whole design turns on: a grant somebody opened for a
	// migration and nobody closed. It has to stop working by itself.
	expired := &models.ExportGrant{Enabled: true, ExpiresAt: now.Add(-time.Minute)}
	if expired.Allowed(now) {
		t.Error("muddati o'tgan ruxsat hali ham ishlayapti")
	}

	// No document at all — a restaurant nobody ever granted anything.
	var missing *models.ExportGrant
	if missing.Allowed(now) {
		t.Error("ruxsatsiz yuklab olish mumkin bo'lib qoldi")
	}

	off := &models.ExportGrant{Enabled: false, ExpiresAt: now.Add(time.Hour)}
	if off.Allowed(now) {
		t.Error("o'chirilgan ruxsat ishlayapti")
	}

	live := &models.ExportGrant{Enabled: true, ExpiresAt: now.Add(time.Hour)}
	if !live.Allowed(now) {
		t.Error("haqiqiy ruxsat ishlamadi")
	}
}

// The allowlist is the other half of the guard, and the collections missing
// from it are the point of it.
func TestExportAllowlistExcludesCredentialCollections(t *testing.T) {
	names := map[string]bool{}
	for _, name := range exportCollections() {
		names[name] = true
	}
	for _, banned := range []string{
		"payment_settings", "sms_settings", "pbx_settings", "pos_settings",
		"phone_code",
	} {
		if names[banned] {
			t.Errorf("%s eksport ro'yxatida — bu kolleksiya butunlay kalitlardan iborat", banned)
		}
	}
	for _, wanted := range []string{"order", "menu_item", "user", "restaurant"} {
		if !names[wanted] {
			t.Errorf("%s eksport ro'yxatida yo'q — mijoz o'z ma'lumotini olmaydi", wanted)
		}
	}
}
