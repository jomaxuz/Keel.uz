package handlers

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Taking the whole business away, as one archive.
//
// See models.ExportGrant for why this is gated by the platform rather than
// being a permanent button: the archive is the most dangerous object the system
// can produce, and the gate turns copying a restaurant's entire customer base
// into a dated, attributable act instead of a silent one.
//
// Three rules shape the code below:
//
//   - **The permission is checked here, every time, against the clock.** The
//     panel hides the button when there is no grant, but hiding is not a
//     control: the request is the thing that has to be refused.
//   - **Nothing that could be used to act as the restaurant leaves.** Not the
//     payment keys, not the SMS password, not the till token, not one password
//     hash. Two independent guards enforce it — an allowlist of collections and
//     a redaction pass over every key name — because either alone fails the
//     day somebody adds a field.
//   - **The archive is readable by whoever receives it.** JSON with real dates
//     and a README, not a Mongo dump: the point is leaving for another system,
//     and a BSON archive is only usable by this one.

// exportCollections is the allowlist — the first of the two guards.
//
// ⚠️ **Written as "what may leave", never as "what may not".** The list of
// secrets grows every time a provider is added; the list of business data does
// not. A denylist would have quietly started exporting `sms_settings` the day
// that collection was created, and nothing would have failed.
//
// Deliberately absent: `payment_settings`, `sms_settings`, `pbx_settings`,
// `pos_settings` (credentials, all four), and `phone_code` (live login codes —
// exporting one is handing over the ability to sign in as a customer).
//
// A plain list of names rather than a list of collection handles, so the guard
// itself can be asserted in a test without a database — the thing being
// protected here is *which names are on the list*, and a test that cannot run
// without Mongo is a test that stops running.
func exportCollections() []string {
	return []string{
		"restaurant",
		"brand",
		"branch",
		"category",
		"menu_item",
		"order",
		"payment", // the ledger: amounts and states, no keys
		"user",
		"feedback",
		"reservation",
		"promotion",
		"loyalty_txn",
		"courier",
		"courier_settlement",
		"staff",
		"shift",
		"staff_payment",
		"call",
		"cash_shift",
		"cash_entry",
		"delivery_provider",
		"pos_mapping",
		"admin_user",
		"admin_log",
		"visit",
	}
}

// exportSecretKey decides whether a field name is a credential.
//
// The second guard, and a **pattern** rather than a list on purpose: a list
// covers the secrets that exist today, and the failure being defended against
// is the one added next year by somebody who never read this file. Every
// credential in this system is named for what it is — password, secret, token,
// key, hash — so the shape holds even for fields nobody has written yet.
//
// It over-matches slightly (a dish called "hash browns" would keep its name;
// only *field names* are tested), and that trade is deliberate: a missing
// column in an export is a support question, a leaked key is not recoverable.
func exportSecretKey(key string) bool {
	k := strings.ToLower(key)
	for _, bad := range []string{
		"password", "secret", "token", "credential", "signature", "jwt", "otp",
		// Bare "key", not "apikey": Payme's field is called exactly `key` and
		// Click's is `secretKey`, so a list of compound names would have missed
		// the shortest and most dangerous one. Nothing in this schema is called
		// "key" for an innocent reason.
		"key",
		// Same shape: `passwordHash`, `codeHash`. A hash is not usable as a
		// password, but it is usable against every other place that person
		// reused it — which is not ours to hand out.
		"hash",
	} {
		if strings.Contains(k, bad) {
			return true
		}
	}
	return false
}

// exportValue walks a decoded document, dropping credentials and turning Mongo
// types into something another system can read.
//
// Dates matter more than they look: left alone, `primitive.DateTime` marshals
// as a bare millisecond integer, and an export whose every timestamp is
// 1785312000000 is an export somebody has to write a script to understand —
// which is exactly the friction this feature exists to remove.
func exportValue(v any) any {
	switch t := v.(type) {
	case bson.M:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if exportSecretKey(k) {
				continue
			}
			out[k] = exportValue(val)
		}
		return out
	case bson.D:
		out := make(map[string]any, len(t))
		for _, e := range t {
			if exportSecretKey(e.Key) {
				continue
			}
			out[e.Key] = exportValue(e.Value)
		}
		return out
	case bson.A:
		out := make([]any, 0, len(t))
		for _, e := range t {
			out = append(out, exportValue(e))
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, e := range t {
			out = append(out, exportValue(e))
		}
		return out
	case primitive.DateTime:
		// Local, like every other date a person reads here: the archive is
		// opened by the restaurant, not by a UTC-aware pipeline.
		return t.Time().In(time.Local).Format(time.RFC3339)
	case primitive.ObjectID:
		return t.Hex()
	case primitive.Binary:
		// Nothing in this schema stores meaningful binary, and a base64 blob in
		// an export is noise at best.
		return nil
	default:
		return v
	}
}

// ---- The grant ----

func (h *Handler) exportGrant(ctx context.Context) *models.ExportGrant {
	var g models.ExportGrant
	err := h.Store.DB.Collection("export_grant").
		FindOne(ctx, bson.M{"_id": models.ExportGrantID}).Decode(&g)
	if err != nil {
		return nil
	}
	return &g
}

// AdminExportStatus tells the panel whether to show the section at all.
//
// Returns the grant's terms rather than a bare boolean: an owner who sees
// "download your data" with no explanation reasonably wonders who decided that,
// and the answer — who opened it, why, until when — is the reassuring part.
func (h *Handler) AdminExportStatus(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	g := h.exportGrant(r.Context())
	if !g.Allowed(time.Now()) {
		// Deliberately the same answer for "never granted" and "expired": the
		// panel has one thing to do in both cases, and the difference is the
		// platform's business, not a hint to whoever is holding the session.
		httpx.JSON(w, http.StatusOK, map[string]any{"allowed": false})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"allowed":   true,
		"reason":    g.Reason,
		"expiresAt": g.ExpiresAt,
		"downloads": len(g.Downloads),
	})
}

// AdminExportArchive streams the whole business as one zip.
//
// Owner only. A manager runs a kitchen; taking the customer base out of the
// building is not a shift-level decision, and the account most likely to be
// shared is exactly the one that must not be able to do this.
func (h *Handler) AdminExportArchive(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	ctx := r.Context()
	now := time.Now()
	g := h.exportGrant(ctx)
	if !g.Allowed(now) {
		httpx.Error(w, http.StatusForbidden,
			"ma'lumotlarni yuklab olishga ruxsat berilmagan yoki muddati tugagan")
		return
	}

	admin, err := h.adminUser(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	var rest models.Restaurant
	_ = h.Store.Restaurant.FindOne(ctx, bson.M{}).Decode(&rest)
	stamp := now.Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.zip", exportSlug(rest.Name), stamp)

	// Headers before the first byte of the body: once the zip starts there is
	// no way to send a status code, so everything that can be refused has been
	// refused above.
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	// Never cached anywhere: this is the one response in the system that must
	// not sit in a proxy, a browser cache or a service worker.
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	counter := &countingWriter{w: w}
	zw := zip.NewWriter(counter)
	files := 0

	if f, err := zw.Create("README.txt"); err == nil {
		fmt.Fprint(f, exportReadme(rest.Name, now))
		files++
	}

	for _, name := range exportCollections() {
		_, err := h.writeCollection(ctx, zw, name)
		if err != nil {
			// Logged, not returned: the response is already a zip. A missing
			// file inside an archive that opens is recoverable; a truncated
			// archive that looks complete is not, so the run continues and the
			// README says how to tell.
			log.Printf("export: %s: %v", name, err)
			continue
		}
		files++
	}

	files += h.writeUploads(zw)

	if err := zw.Close(); err != nil {
		log.Printf("export: close: %v", err)
		return
	}

	// Recorded after the fact, with what actually went out.
	//
	// ⚠️ Written even though the response has already been sent, and to two
	// places: the tenant's own audit log (which the restaurant can read) and
	// the grant document (which the console reads without opening anybody's
	// panel). "Was it downloaded?" is the first question after a leak, and it
	// has to be answerable after the grant expires.
	h.recordExport(context.WithoutCancel(ctx), admin.Username, counter.n, files)
	h.logAction(r, ActDataExport, "settings", models.ExportGrantID,
		"Ma'lumotlarni yuklab olish", fmt.Sprintf("%d fayl, %d bayt", files, counter.n))
}

// writeCollection streams one collection as a JSON array.
//
// Streamed document by document rather than marshalled whole: a restaurant with
// two years of orders holds more of them than this process should hold in
// memory at once, and the failure mode of finding that out in production is the
// container being killed mid-download.
func (h *Handler) writeCollection(ctx context.Context, zw *zip.Writer, name string) (int, error) {
	cur, err := h.Store.DB.Collection(name).
		Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	f, err := zw.Create("data/" + name + ".json")
	if err != nil {
		return 0, err
	}
	if _, err := io.WriteString(f, "[\n"); err != nil {
		return 0, err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("  ", "  ")
	n := 0
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			continue
		}
		if n > 0 {
			if _, err := io.WriteString(f, ",\n"); err != nil {
				return n, err
			}
		}
		if _, err := io.WriteString(f, "  "); err != nil {
			return n, err
		}
		if err := enc.Encode(exportValue(doc)); err != nil {
			return n, err
		}
		n++
	}
	_, err = io.WriteString(f, "]\n")
	return n, err
}

// writeUploads copies the images in.
//
// They are the half of the data that cannot be re-entered: a menu without its
// photographs is a menu somebody has to shoot again, and a restaurant leaving
// for another system needs the pictures more than it needs the order history.
func (h *Handler) writeUploads(zw *zip.Writer) int {
	root := h.Cfg.UploadDir
	if strings.TrimSpace(root) == "" {
		return 0
	}
	n := 0
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		// Regular files only: a symlink here would copy whatever it points at
		// into an archive that leaves the building, and /etc/passwd is a link
		// away from anything writable.
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer in.Close()
		out, err := zw.Create("uploads/" + filepath.ToSlash(rel))
		if err != nil {
			return nil
		}
		if _, err := io.Copy(out, in); err != nil {
			return nil
		}
		n++
		return nil
	})
	return n
}

func (h *Handler) recordExport(ctx context.Context, by string, size int64, files int) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err := h.Store.DB.Collection("export_grant").UpdateOne(ctx,
		bson.M{"_id": models.ExportGrantID},
		bson.M{"$push": bson.M{"downloads": bson.M{
			"$each": bson.A{models.ExportDownload{
				At: time.Now(), By: by, Bytes: size, Files: files,
			}},
			// Capped, oldest dropped: the useful part is "did this happen and
			// when", and an unbounded array in a settings document is a slow
			// way to make that document unreadable.
			"$slice": -50,
		}}},
	)
	if err != nil {
		log.Printf("export: record: %v", err)
	}
}

// countingWriter measures what actually reached the client, which is the only
// size worth recording — the zip's own accounting would report what we meant
// to send.
type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

func exportSlug(name string) string {
	s := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r >= 'A' && r <= 'Z':
			return r + 32
		case r == ' ' || r == '-' || r == '_':
			return '-'
		default:
			return -1
		}
	}, name)
	s = strings.Trim(s, "-")
	if s == "" {
		return "restoran"
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

func exportReadme(name string, at time.Time) string {
	return `MA'LUMOTLAR ARXIVI
==================

Restoran:  ` + name + `
Olingan:   ` + at.Format("2006-01-02 15:04") + `

ICHIDA NIMA BOR
---------------
data/*.json  — har bir bo'lim alohida fayl: menyu, buyurtmalar, mijozlar,
               bronlar, kuryerlar, ishchilar, kassa, qo'ng'iroqlar va h.k.
               Sanalar odam o'qiydigan ko'rinishda (masalan 2026-08-08T13:20:00+05:00),
               id'lar esa matn ko'rinishida.
uploads/     — menyu va restoran rasmlari, asl fayl nomlari bilan.
               data/menu_item.json dagi imageUrl shu fayllarga ishora qiladi.

NIMA YO'Q, VA NEGA
------------------
Arxivda hech qanday kalit, parol yoki token yo'q: to'lov tizimlari
(Payme/Click/Uzum/ATMOS) kalitlari, SMS shlyuzi paroli, kassa (POS) tokeni,
telefoniya kaliti va barcha parol hash'lari ataylab chiqarilmagan. Sabab
oddiy: bu fayl bir necha qo'ldan o'tadi, ular esa sizning nomingizdan pul
qabul qilish yoki SMS yuborish imkonini beradi. Yangi tizimda ularni
qaytadan kiritasiz — har biri o'z kabinetingizda turibdi.

Bir martalik SMS kodlari ham yo'q (ular bir necha daqiqa yashaydi).

TO'LIQ EMASMI?
--------------
Har bir data/*.json fayli to'g'ri yopilgan JSON massiv bo'lishi kerak, ya'ni
oxirgi belgisi "]". Agar biror fayl shu bilan tugamasa, yuklab olish uzilgan —
arxivni qaytadan yuklab oling.

SAVOLLAR
--------
Bu arxiv sizniki. Boshqa tizimga ko'chirishda formatga oid savol chiqsa,
Keel qo'llab-quvvatlash xizmatiga murojaat qiling.
`
}
