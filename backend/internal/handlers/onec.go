package handlers

// ---- The door an accountant's 1C knocks on ----
//
// ⚠️ **Everything here is answered in plain text, and the first word is the
// whole protocol.** 1C reads `success`, `progress` or `failure` from the first
// line and nothing else; a JSON error — the shape every other endpoint in this
// codebase returns — is read by 1C as a failure with our JSON as its reason,
// which is what the accountant then reads out over the phone. So this file's
// handlers deliberately do **not** use httpx.
//
// ⚠️ **Basic auth with its own login, never a panel account.** 1C stores the
// password in a settings form on an office machine, in clear text, and shows it
// on screen. A login that also opened the panel would put the whole restaurant
// behind a password that lives in a text field somebody's colleague can read.
//
// ⚠️ **The session cookie is checked but the credentials are accepted on every
// request too.** Some configurations send the cookie, some forget it, some send
// Basic auth every time; the protocol allows all three, and a server that
// insisted on the cookie would work with one accountant's 1C and not the next
// one's — with no error either side could act on.
//
// The protocol, read and copied: docs/vendor/1c-exchange.md.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/onec"
)

// oneCSettings loads the door's settings. A missing document is a closed door.
func (h *Handler) oneCSettings(ctx context.Context) *models.OneCSettings {
	var s models.OneCSettings
	if err := h.Store.OneCSettings.FindOne(ctx, bson.M{}).Decode(&s); err != nil {
		return &models.OneCSettings{}
	}
	return &s
}

// OneCExchange is the whole protocol: one endpoint, two parameters.
//
// ⚠️ **Not behind the panel's JWT.** 1C has no way to hold one — it sends Basic
// auth, which is what the published protocol specifies — so this route is
// mounted outside the admin group and does its own authentication. That makes
// it the one endpoint in the product where a wrong answer is a data leak rather
// than a broken screen, which is why the credentials are checked before the
// mode is even read.
func (h *Handler) OneCExchange(w http.ResponseWriter, r *http.Request) {
	s := h.oneCSettings(r.Context())
	if !s.Enabled || s.Login == "" || s.PasswordHash == "" {
		// ⚠️ **"failure" with a reason, not 401.** A 401 makes 1C pop a
		// password box at the accountant, who then types a correct password
		// into a door that is switched off — and concludes the password is
		// wrong. The reason is the answer they need.
		oneCFail(w, "1C almashinuvi yoqilmagan")
		return
	}
	if !h.oneCAuthorised(r, s) {
		// Here a 401 is right: the credentials genuinely are wrong, and the
		// password box is what 1C should show.
		w.Header().Set("WWW-Authenticate", `Basic realm="Keel"`)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "failure\nlogin yoki parol noto'g'ri\n")
		return
	}

	kind := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	mode := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mode")))
	h.noteOneC(r.Context(), kind, mode, "")

	switch mode {
	case "checkauth":
		name, value := h.oneCCookie(s)
		// Three lines, in this order: 1C reads them positionally.
		oneCText(w, "success\n"+name+"\n"+value+"\n")
	case "init":
		// ⚠️ **`zip=no` is honest rather than lazy.** Accepting zip and then
		// failing to unpack one would be a silent half-import; the files an
		// accounting exchange carries are small enough that the difference is
		// not worth a mode that only breaks on the customer with the largest
		// catalogue.
		oneCText(w, "zip=no\nfile_limit=8388608\n")
	case "file", "import":
		// ⚠️ **The catalogue half is the stock module, reached through a door
		// the module gate does not watch.** That gate is a filter on
		// `/admin/…` paths (modulegate.go); this endpoint is deliberately
		// outside it because 1C cannot hold a panel token — so the same rule is
		// applied here by hand, or a restaurant that never bought the stockroom
		// could fill it from an office machine. Selling *sales* to 1C stays
		// open: reports are in the price for everybody.
		if !h.subscription(r.Context()).Has(models.ModStock) {
			oneCFail(w, "ombor moduli tarifingizga kirmaydi")
			return
		}
		if mode == "file" {
			h.oneCFile(w, r)
		} else {
			h.oneCImport(w, r)
		}
	case "query":
		h.oneCQuery(w, r, s)
	case "success":
		// ⚠️ **Nothing is deleted or marked here.** 1C says "I have them"; our
		// documents are the restaurant's own sales and deliveries, which exist
		// whether or not an accountant has imported them. Marking them exported
		// would create a second meaning for "this sale happened" and a support
		// question the first time somebody re-imports a month.
		oneCText(w, "success\n")
	case "deactivate", "complete":
		oneCText(w, "success\n")
	default:
		oneCFail(w, "noma'lum rejim: "+mode)
	}
}

// oneCAuthorised checks the login 1C was given.
func (h *Handler) oneCAuthorised(r *http.Request, s *models.OneCSettings) bool {
	user, pass, ok := r.BasicAuth()
	if ok && strings.EqualFold(strings.TrimSpace(user), s.Login) {
		if bcrypt.CompareHashAndPassword([]byte(s.PasswordHash), []byte(pass)) == nil {
			return true
		}
	}
	// The cookie from `checkauth`, for the requests that carry it instead.
	if c, err := r.Cookie(oneCCookieName); err == nil {
		name, value := h.oneCCookie(s)
		_ = name
		if hmac.Equal([]byte(c.Value), []byte(value)) {
			return true
		}
	}
	return false
}

const oneCCookieName = "keel_1c"

// oneCCookie is the session marker handed out by `checkauth`.
//
// ⚠️ **Derived rather than stored, and derived from the password hash.** A
// value kept in the database would need a table, an expiry and a cleanup; a
// value derived from the credentials is invalidated the moment they change,
// which is exactly when it should be. It is an HMAC so that knowing the login
// name does not let anybody compute it.
func (h *Handler) oneCCookie(s *models.OneCSettings) (string, string) {
	mac := hmac.New(sha256.New, []byte(h.Cfg.JWTSecret))
	mac.Write([]byte("1c|" + s.Login + "|" + s.PasswordHash))
	return oneCCookieName, hex.EncodeToString(mac.Sum(nil))[:32]
}

func oneCText(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, body)
}

// oneCFail answers the way 1C expects a refusal.
//
// ⚠️ **HTTP 200 with `failure` in the body.** The protocol carries its own
// result in the text; a 500 makes 1C report a network problem, which sends the
// accountant to their IT person instead of to the sentence that says what is
// actually wrong.
func oneCFail(w http.ResponseWriter, why string) {
	oneCText(w, "failure\n"+why+"\n")
}

// oneCFile receives an uploaded exchange file and keeps it in memory.
//
// ⚠️ **Held in memory rather than written to disk**, and capped. The upload and
// the `import` that follows are two requests from the same 1C seconds apart;
// a file on disk would be a temporary file to clean up, a permission to get
// right in a container, and a way for one customer's exchange to fill a volume
// shared by every tenant on the box.
func (h *Handler) oneCFile(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("filename"))
	raw, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		oneCFail(w, "faylni o'qib bo'lmadi")
		return
	}
	h.oneCMu.Lock()
	if h.oneCFiles == nil {
		h.oneCFiles = map[string][]byte{}
	}
	// ⚠️ Appended, because 1C splits a file that exceeds `file_limit` into
	// several `file` requests with the same name. Replacing would keep only the
	// last slice — a catalogue that imports its final fifty products and drops
	// the rest, with `success` reported for all of them.
	h.oneCFiles[name] = append(h.oneCFiles[name], raw...)
	h.oneCMu.Unlock()
	oneCText(w, "success\n")
}

// oneCImport reads an uploaded catalogue into the stock list.
//
// ⚠️ **Products are created and renamed; nothing is ever deleted.** A
// nomenclature that arrives short of one line is far more often a filter in 1C
// than a product a restaurant has stopped selling — and a delete here would
// take a technical card, a shelf balance and a year of purchase history with
// it.
func (h *Handler) oneCImport(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("filename"))
	h.oneCMu.Lock()
	raw := h.oneCFiles[name]
	delete(h.oneCFiles, name)
	h.oneCMu.Unlock()
	if len(raw) == 0 {
		oneCFail(w, "fayl topilmadi: "+name)
		return
	}
	items, err := onec.Parse(raw)
	if err != nil {
		h.noteOneC(r.Context(), "catalog", "import", err.Error())
		oneCFail(w, err.Error())
		return
	}
	written, err := h.oneCWriteProducts(r.Context(), items)
	if err != nil {
		h.noteOneC(r.Context(), "catalog", "import", err.Error())
		oneCFail(w, err.Error())
		return
	}
	_, _ = h.Store.OneCSettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": bson.M{"lastImported": written}}, options.Update().SetUpsert(true))
	oneCText(w, "success\n")
}

// oneCWriteProducts creates or updates one ingredient per line of a catalogue.
//
// ⚠️ **Matched by name, and the name is normalised the same way the electronic
// invoice import normalises it.** Two matching rules for "is this the same
// product" would eventually disagree, and the disagreement would be a
// duplicated shelf row that nobody can merge.
func (h *Handler) oneCWriteProducts(ctx context.Context, items []onec.Product) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	cur, err := h.Store.Ingredients.Find(ctx, bson.M{})
	if err != nil {
		return 0, err
	}
	var have []models.Ingredient
	_ = cur.All(ctx, &have)
	byName := map[string]primitive.ObjectID{}
	for _, in := range have {
		byName[normalizeEDIName(in.Name)] = in.ID
	}
	// Every ingredient of a chain belongs to the brand, and an exchange is one
	// company's: the first brand is the one an accountant means.
	brand := h.firstBrandID(ctx)

	now := time.Now()
	written := 0
	for _, p := range items {
		name := clampText(p.Name, 120)
		if name == "" {
			continue
		}
		unit := oneCUnit(p.Unit)
		if id, ok := byName[normalizeEDIName(name)]; ok {
			set := bson.M{"updatedAt": now}
			// ⚠️ **A price of zero does not overwrite a price.** An offers file
			// without prices is ordinary — the accountant exported the
			// nomenclature only — and reading its silence as "free" would zero
			// the cost of every dish the ingredient goes into.
			if p.Price > 0 {
				set["price"] = int(p.Price + 0.5)
			}
			if _, err := h.Store.Ingredients.UpdateOne(ctx,
				bson.M{"_id": id}, bson.M{"$set": set}); err != nil {
				return written, err
			}
			written++
			continue
		}
		res, err := h.Store.Ingredients.InsertOne(ctx, models.Ingredient{
			BrandID: brand, Name: name, Unit: unit,
			Price: int(p.Price + 0.5), Note: "1C", UpdatedAt: now,
		})
		if err != nil {
			return written, err
		}
		byName[normalizeEDIName(name)] = oidOf(res.InsertedID)
		written++
	}
	return written, nil
}

// oneCUnit maps 1C's unit names onto the three this product buys in.
//
// ⚠️ **Anything unknown becomes a piece rather than an error.** A unit we do
// not recognise is a translation problem; refusing the product over it would
// lose the row entirely, and a piece is the assumption a person makes too.
func oneCUnit(u string) string {
	switch strings.ToLower(strings.TrimSpace(u)) {
	case "кг", "kg", "килограмм":
		return "kg"
	case "л", "l", "литр":
		return "l"
	}
	return "pcs"
}

// oneCQuery answers with the documents 1C asked for.
func (h *Handler) oneCQuery(w http.ResponseWriter, r *http.Request, s *models.OneCSettings) {
	days := s.Days
	if days <= 0 {
		days = models.OneCDays
	}
	from := time.Now().In(time.Local).AddDate(0, 0, -days)
	docs, err := h.oneCDocuments(r.Context(), s, from)
	if err != nil {
		h.noteOneC(r.Context(), "sale", "query", err.Error())
		oneCFail(w, err.Error())
		return
	}
	_, _ = h.Store.OneCSettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": bson.M{"lastExported": len(docs)}}, options.Update().SetUpsert(true))
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(onec.Documents(docs))
}

// oneCDocuments collects what an accountant posts: the sales and the deliveries.
//
// ⚠️ **Both, and as two kinds of document rather than one list of numbers.** A
// month's takings without the month's purchases is half a ledger — the half
// that makes a business look enormously profitable — and an accountant who has
// to type the other half by hand is doing the work this integration exists to
// remove.
func (h *Handler) oneCDocuments(
	ctx context.Context, s *models.OneCSettings, from time.Time,
) ([]onec.Doc, error) {
	out := []onec.Doc{}

	orderFilter := bson.M{
		"createdAt": bson.M{"$gte": from},
		"status":    models.StatusDelivered,
	}
	if !s.BranchID.IsZero() {
		orderFilter["branchId"] = s.BranchID
	}
	cur, err := h.Store.Orders.Find(ctx, orderFilter,
		options.Find().SetLimit(5000).SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var orders []models.Order
	if err := cur.All(ctx, &orders); err != nil {
		return nil, err
	}
	for _, o := range orders {
		d := onec.Doc{
			ID:        o.ID.Hex(),
			Operation: onec.OpSale,
			Number:    o.Number,
			// ⚠️ `.In(time.Local)` — Mongo hands every time back in UTC, and a
			// document dated 19:00 the previous day lands in the wrong month
			// four evenings out of five (CLAUDE.md's second timezone trap).
			At:      o.CreatedAt.In(time.Local),
			Total:   float64(o.Total),
			Comment: oneCChannel(o),
		}
		for _, it := range o.Items {
			d.Lines = append(d.Lines, onec.Line{
				ID:    it.MenuItemID.Hex(),
				Name:  it.Name,
				Unit:  "pcs",
				Qty:   float64(it.Qty),
				Price: float64(it.Price),
				Sum:   float64(it.Price * it.Qty),
			})
		}
		out = append(out, d)
	}

	purchaseFilter := bson.M{"at": bson.M{"$gte": from}}
	if !s.BranchID.IsZero() {
		purchaseFilter["branchId"] = s.BranchID
	}
	pcur, err := h.Store.Purchases.Find(ctx, purchaseFilter,
		options.Find().SetLimit(5000).SetSort(bson.D{{Key: "at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var purchases []models.Purchase
	if err := pcur.All(ctx, &purchases); err != nil {
		return nil, err
	}
	stock := h.oneCIngredients(ctx)
	for _, p := range purchases {
		d := onec.Doc{
			ID:           p.ID.Hex(),
			Operation:    onec.OpPurchase,
			Number:       p.ID.Hex()[18:],
			At:           p.At.In(time.Local),
			Total:        float64(p.Total),
			Counterparty: p.Supplier,
			Comment:      p.Note,
		}
		for _, l := range p.Lines {
			d.Lines = append(d.Lines, onec.Line{
				ID:   l.IngredientID.Hex(),
				Name: stock[l.IngredientID].name,
				// ⚠️ **The ingredient's own unit, not "piece".** A delivery of
				// 12.5 written as 12.5 pieces of flour is a quantity an
				// accountant cannot reconcile with the invoice in their hand,
				// and 1C prints the unit on every line of the document.
				Unit:  stock[l.IngredientID].unit,
				Qty:   l.Qty,
				Price: float64(l.Price),
				Sum:   float64(l.Sum()),
			})
		}
		out = append(out, d)
	}
	return out, nil
}

// oneCChannel says how the sale was taken, in one word.
//
// ⚠️ Kept as a comment on the document rather than as an operation of its own:
// 1C's chart of operations has no idea what a Telegram order is, and inventing
// an operation name would land the sale in a part of the accountant's books
// they never asked for.
func oneCChannel(o models.Order) string {
	if o.Type != "" {
		return o.Type
	}
	return ""
}

// oneCIngredient is what a delivery line needs about what was delivered.
type oneCIngredient struct{ name, unit string }

// oneCIngredients is the whole stock list by id.
//
// ⚠️ Its own function rather than `ingredientNames` next door: that one takes
// the ids it needs, which is right for a recipe diff and wrong here — an
// exchange walks a month of deliveries and would ask the database once per
// document.
func (h *Handler) oneCIngredients(ctx context.Context) map[primitive.ObjectID]oneCIngredient {
	out := map[primitive.ObjectID]oneCIngredient{}
	cur, err := h.Store.Ingredients.Find(ctx, bson.M{})
	if err != nil {
		return out
	}
	var rows []models.Ingredient
	_ = cur.All(ctx, &rows)
	for _, in := range rows {
		out[in.ID] = oneCIngredient{name: in.Name, unit: in.Unit}
	}
	return out
}

func (h *Handler) firstBrandID(ctx context.Context) primitive.ObjectID {
	var b models.Brand
	if err := h.Store.Brands.FindOne(ctx, bson.M{},
		options.FindOne().SetSort(bson.D{{Key: "sortOrder", Value: 1}})).Decode(&b); err != nil {
		return primitive.NilObjectID
	}
	return b.ID
}

// noteOneC records what the last knock was, so the panel can answer "is it
// working?" without anybody opening 1C.
func (h *Handler) noteOneC(ctx context.Context, kind, mode, failure string) {
	set := bson.M{"lastSeenAt": time.Now(), "lastType": kind, "lastMode": mode}
	// ⚠️ Cleared on a good request rather than left standing: a week-old error
	// beside a working exchange is the same lie as a stale `provisionStatus`.
	set["lastError"] = failure
	_, _ = h.Store.OneCSettings.UpdateOne(ctx, bson.M{}, bson.M{"$set": set},
		options.Update().SetUpsert(true))
}

// ---- The panel's side of the same settings ----

// AdminGetOneC returns the door's settings and its address.
func (h *Handler) AdminGetOneC(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	s := h.oneCSettings(r.Context())
	days := s.Days
	if days <= 0 {
		days = models.OneCDays
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"enabled":     s.Enabled,
		"login":       s.Login,
		"hasPassword": s.PasswordHash != "",
		"branchId":    s.BranchID,
		"days":        days,
		// ⚠️ **The address is built here rather than typed by the owner.** It
		// is the one field an accountant must paste exactly, and an owner
		// asked to assemble it from a base URL and a path will get it wrong
		// once in three — with a 404 that reads as "the integration does not
		// work".
		"url":          strings.TrimRight(h.Cfg.PublicBaseURL, "/") + "/api/v1/1c/exchange",
		"lastSeenAt":   s.LastSeenAt,
		"lastType":     s.LastType,
		"lastMode":     s.LastMode,
		"lastError":    s.LastError,
		"lastImported": s.LastImported,
		"lastExported": s.LastExported,
	})
}

type oneCSettingsRequest struct {
	Enabled  bool   `json:"enabled"`
	Login    string `json:"login"`
	Password string `json:"password"`
	BranchID string `json:"branchId"`
	Days     int    `json:"days"`
}

// AdminUpdateOneC saves the door's settings.
func (h *Handler) AdminUpdateOneC(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req oneCSettingsRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	login := clampText(req.Login, 60)
	current := h.oneCSettings(r.Context())
	if req.Enabled && login == "" {
		httpx.Error(w, http.StatusBadRequest, "login kerak")
		return
	}
	if req.Enabled && req.Password == "" && current.PasswordHash == "" {
		httpx.Error(w, http.StatusBadRequest, "parol kerak")
		return
	}
	days := req.Days
	if days <= 0 || days > 366 {
		days = models.OneCDays
	}
	set := bson.M{
		"enabled":   req.Enabled,
		"login":     login,
		"days":      days,
		"updatedAt": time.Now(),
	}
	if b, err := objectID(req.BranchID); err == nil && !b.IsZero() {
		set["branchId"] = b
	} else {
		set["branchId"] = primitive.NilObjectID
	}
	// ⚠️ An empty password keeps the stored one — the same rule as every other
	// credential on these pages.
	if pw := strings.TrimSpace(req.Password); pw != "" {
		if len(pw) < 8 {
			httpx.Error(w, http.StatusBadRequest, "parol kamida 8 belgidan iborat bo'lsin")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		set["passwordHash"] = string(hash)
	}
	if _, err := h.Store.OneCSettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": set}, options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, "onec.settings", "onec", "", login, "")
	h.AdminGetOneC(w, r)
}

// AdminOneCExport hands the accountant the same XML by hand.
//
// ⚠️ **Because half of them will never switch the automatic exchange on.** An
// accountant who is handed a file can open it in 1C today; one who is asked to
// configure an exchange first will do it next month. The file is identical, so
// nothing here is a second implementation — it is the same builder with a
// download attached.
func (h *Handler) AdminOneCExport(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	s := h.oneCSettings(r.Context())
	days := s.Days
	if days <= 0 {
		days = models.OneCDays
	}
	// ⚠️ A bad or missing number falls back to the configured window rather
	// than refusing: this is a download button on a settings page, and an error
	// about a query parameter is an answer to nobody's question.
	if n, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && n > 0 && n <= 366 {
		days = n
	}
	from := time.Now().In(time.Local).AddDate(0, 0, -days)
	docs, err := h.oneCDocuments(r.Context(), s, from)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	name := fmt.Sprintf("keel-1c-%s.xml", time.Now().In(time.Local).Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	_, _ = w.Write(onec.Documents(docs))
}
