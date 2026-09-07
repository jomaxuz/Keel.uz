package handlers

// ---- Electronic documents: the post a company already receives ----
//
// ⚠️ **The whole feature is one sentence: a delivery that arrived with an
// electronic invoice should not be retyped.** Every restaurant with a supplier
// that issues ЭСФ already has the delivery — twenty lines, quantities, prices,
// VAT — sitting in Didox on the morning the crates come through the door. Until
// now a storekeeper opened that in one browser tab and typed it into ours in
// another, which is where the shelf and the invoice quietly stop agreeing:
// one line skipped, one price rounded, one delivery entered twice after a
// distraction.
//
// ⚠️ **Our copy is never the truth, and the screen says so.** Didox holds the
// document and its signatures; this file keeps a mirror so the panel can list
// and link without dialling the operator per screen. Anything about a *status*
// is as fresh as the last pull.
//
// ⚠️ **Nothing here signs, accepts or rejects a document**, and that is a
// decision rather than an omission — see internal/didox and docs/vendor/didox.md.
// A signature is made by an E-IMZO key on somebody's own computer. A button
// here that flipped a status without one would leave the document unsigned at
// the operator and absent from the tax return, while our screen showed it done.
// The panel links to didox.uz and says whose job it is.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/didox"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// tinDigits keeps the digits of a typed tax number.
//
// ⚠️ A tax number is compared, never displayed from this: "302 936 161" and
// "302936161" are one company, and a comparison that says otherwise puts a
// delivery on nobody's account. (`digitsOnly` next door is the phone
// normaliser's regexp — a second name rather than a shared one, because the two
// answer different questions and a phone's rules are not a СТИР's.)
func tinDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ediSettings loads the connection. A missing document is not an error: it is
// an install whose owner has never opened the page.
func (h *Handler) ediSettings(ctx context.Context) *models.EDISettings {
	var s models.EDISettings
	if err := h.Store.EDISettings.FindOne(ctx, bson.M{}).Decode(&s); err != nil {
		return &models.EDISettings{}
	}
	return &s
}

var (
	errEDIOff  = errors.New("elektron hujjat aylanishi yoqilmagan")
	errEDIHalf = errors.New("Didox sozlamalari to'liq emas: STIR, parol va hamkor tokeni kerak")
)

// ediClient builds a client with a live session, logging in when the cached
// token has run out.
//
// ⚠️ **The token is cached in the settings document and re-used until it
// expires**, because every login counts towards the operator's own "too many
// attempts" limit — a panel that logged in per page view would lock the
// customer out of didox.uz with a screen that never mentioned it.
func (h *Handler) ediClient(ctx context.Context) (*didox.Client, *models.EDISettings, error) {
	s := h.ediSettings(ctx)
	if !s.Enabled {
		return nil, s, errEDIOff
	}
	if s.TIN == "" || s.PartnerToken == "" || s.Password == "" {
		return nil, s, errEDIHalf
	}
	if s.Token != "" && s.TokenExpires != nil && time.Now().Before(*s.TokenExpires) {
		return didox.New(s.Sandbox, s.PartnerToken, s.Token), s, nil
	}
	c := didox.New(s.Sandbox, s.PartnerToken, "")
	token, err := c.LoginByPassword(ctx, s.TIN, s.Password)
	if err != nil {
		h.noteEDIError(ctx, err)
		return nil, s, err
	}
	c.UserToken = token
	until := time.Now().Add(didox.TokenLife)
	_, _ = h.Store.EDISettings.UpdateOne(ctx, bson.M{},
		bson.M{"$set": bson.M{"token": token, "tokenExpires": until, "lastError": ""}},
		options.Update().SetUpsert(true))
	s.Token, s.TokenExpires = token, &until
	return c, s, nil
}

// noteEDIError writes down what the operator said, in their words.
//
// ⚠️ **Kept for the screen rather than only for the log.** "Ulanmadi" sends an
// owner to us; "422 User not registered" sends them to their own accountant,
// which is where the fix is.
func (h *Handler) noteEDIError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	_, _ = h.Store.EDISettings.UpdateOne(ctx, bson.M{},
		bson.M{"$set": bson.M{"lastError": err.Error(), "lastErrorAt": time.Now()}},
		options.Update().SetUpsert(true))
}

// AdminGetEDI returns the settings without the secrets — only whether each one
// is stored. Same rule as the payment and SMS keys.
func (h *Handler) AdminGetEDI(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	s := h.ediSettings(r.Context())
	// How much post is waiting, from our own mirror: the screen has to answer
	// "is there anything to do" before anybody dials the operator.
	waiting, _ := h.Store.EDIDocuments.CountDocuments(r.Context(), bson.M{
		"direction":  models.EDIIn,
		"purchaseId": bson.M{"$exists": false},
	})
	httpx.JSON(w, http.StatusOK, map[string]any{
		"provider":      models.EDIDidox,
		"enabled":       s.Enabled,
		"sandbox":       s.Sandbox,
		"tin":           s.TIN,
		"seller":        s.Seller,
		"hasPartnerKey": s.PartnerToken != "",
		"hasPassword":   s.Password != "",
		"lastSyncAt":    s.LastSyncAt,
		"lastError":     s.LastError,
		"notImported":   waiting,
		// ⚠️ Said on the screen rather than assumed: the one question an
		// accountant asks about any EDI integration is who signs.
		"signsHere": false,
	})
}

type ediSettingsRequest struct {
	Enabled      bool            `json:"enabled"`
	Sandbox      bool            `json:"sandbox"`
	TIN          string          `json:"tin"`
	PartnerToken string          `json:"partnerToken"`
	Password     string          `json:"password"`
	Seller       models.EDIParty `json:"seller"`
}

// AdminUpdateEDI saves the connection.
func (h *Handler) AdminUpdateEDI(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req ediSettingsRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	tin := tinDigits(req.TIN)
	if req.Enabled && tin == "" {
		httpx.Error(w, http.StatusBadRequest, "STIR kerak")
		return
	}
	set := bson.M{
		"provider":  models.EDIDidox,
		"enabled":   req.Enabled,
		"sandbox":   req.Sandbox,
		"tin":       tin,
		"updatedAt": time.Now(),
		"seller": bson.M{
			"name":         clampText(req.Seller.Name, 200),
			"vatRegCode":   clampText(req.Seller.VatRegCode, 40),
			"vatRegStatus": req.Seller.VatRegStatus,
			"account":      clampText(req.Seller.Account, 40),
			"bankId":       clampText(req.Seller.BankID, 10),
			"address":      clampText(req.Seller.Address, 300),
			"director":     clampText(req.Seller.Director, 200),
			"accountant":   clampText(req.Seller.Accountant, 200),
		},
	}
	// ⚠️ **An empty secret keeps the stored one**, the rule every settings page
	// in this codebase follows: an owner correcting an address must not
	// silently disconnect the integration because the password field rendered
	// blank.
	if v := strings.TrimSpace(req.PartnerToken); v != "" {
		set["partnerToken"] = v
	}
	if v := strings.TrimSpace(req.Password); v != "" {
		set["password"] = v
		// ⚠️ **A new password invalidates the cached session.** Keeping the old
		// token would leave the integration working until it expired and then
		// fail hours later, with nothing on screen connecting the two.
		set["token"] = ""
		set["tokenExpires"] = nil
	}
	if _, err := h.Store.EDISettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": set}, options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, "edi.settings", "edi", "", models.EDIDidox, "")
	h.AdminGetEDI(w, r)
}

// AdminEDISync pulls the post from the operator into our mirror.
//
// ⚠️ **Upserted on the operator's document id.** The pull runs by hand and on a
// timer, and two runs that overlap must not produce two copies of one invoice —
// a storekeeper looking at a pair of identical deliveries cannot tell which is
// the copy, and importing the wrong one is a delivery counted twice.
func (h *Handler) AdminEDISync(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	c, s, err := h.ediClient(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// ⚠️ **A window rather than "everything", and it starts before the last
	// pull.** A document can be signed days after it was written and the list
	// is filtered by the document's own date, so a window that began at the
	// last sync would step over exactly the invoices that arrived late.
	days := 30
	if v, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && v > 0 && v <= 366 {
		days = v
	}
	to := time.Now().In(time.Local)
	from := to.AddDate(0, 0, -days)

	saved, err := h.ediPull(r.Context(), c, false, from, to)
	if err != nil {
		h.noteEDIError(r.Context(), err)
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	out, err := h.ediPull(r.Context(), c, true, from, to)
	if err != nil {
		h.noteEDIError(r.Context(), err)
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	now := time.Now()
	_, _ = h.Store.EDISettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": bson.M{"lastSyncAt": now, "lastError": ""}},
		options.Update().SetUpsert(true))
	s.LastSyncAt = &now
	httpx.JSON(w, http.StatusOK, map[string]any{
		"incoming": saved, "outgoing": out, "at": now,
	})
}

// ediPull walks one direction of the operator's list into our mirror.
func (h *Handler) ediPull(
	ctx context.Context, c *didox.Client, outgoing bool, from, to time.Time,
) (int, error) {
	dir := models.EDIIn
	if outgoing {
		dir = models.EDIOut
	}
	saved := 0
	// ⚠️ Paged with a hard stop. The operator answers `total`, but a filter
	// that matches more than we expect must not turn one button press into a
	// hundred requests: twenty pages of fifty is a month of post for anybody.
	for page := 1; page <= 20; page++ {
		rows, _, err := c.List(ctx, didox.ListFilter{
			Outgoing: outgoing,
			From:     from.Format("2006-01-02"),
			To:       to.Format("2006-01-02"),
			Page:     page,
			Limit:    50,
		})
		if err != nil {
			return saved, err
		}
		for _, row := range rows {
			if row.DocID == "" {
				continue
			}
			set := bson.M{
				"direction":    dir,
				"type":         row.DocType,
				"number":       row.Number,
				"date":         row.Date,
				"status":       row.Status,
				"partnerTin":   row.PartnerTIN,
				"partnerName":  row.Partner,
				"contractNo":   row.Contract,
				"contractDate": row.ContractAt,
				"total":        didox.Num(row.Delivery),
				"vatTotal":     didox.Num(row.VatTotal),
				"totalWithVat": didox.Num(row.TotalWith),
				"hasVat":       row.HasVat,
				"hasMarks":     row.HasMarks == 1,
				"syncedAt":     time.Now(),
			}
			res, err := h.Store.EDIDocuments.UpdateOne(ctx,
				bson.M{"docId": row.DocID},
				bson.M{"$set": set, "$setOnInsert": bson.M{
					"docId": row.DocID, "createdAt": time.Now(),
				}},
				options.Update().SetUpsert(true))
			if err != nil {
				return saved, err
			}
			if res.UpsertedCount > 0 {
				saved++
			}
		}
		if len(rows) < 50 {
			break
		}
	}
	return saved, nil
}

// AdminEDIDocuments lists our mirror.
func (h *Handler) AdminEDIDocuments(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	q := r.URL.Query()
	filter := bson.M{"direction": models.EDIIn}
	if q.Get("direction") == models.EDIOut {
		filter["direction"] = models.EDIOut
	}
	switch q.Get("imported") {
	case "1":
		filter["purchaseId"] = bson.M{"$exists": true}
	case "0":
		filter["purchaseId"] = bson.M{"$exists": false}
	}
	if tin := tinDigits(q.Get("partner")); tin != "" {
		filter["partnerTin"] = tin
	}
	cur, err := h.Store.EDIDocuments.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "date", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ Empty slice, never nil: this goes straight into a list on the panel,
	// and `null.map` is a blank screen (CLAUDE.md §10).
	rows := []models.EDIDocument{}
	_ = cur.All(r.Context(), &rows)
	s := h.ediSettings(r.Context())
	httpx.JSON(w, http.StatusOK, map[string]any{
		"documents":  rows,
		"lastSyncAt": s.LastSyncAt,
		"enabled":    s.Enabled,
	})
}

// AdminEDIDocument opens one document, fetching its lines from the operator the
// first time somebody looks at it.
//
// ⚠️ **The lines are fetched once and kept.** The list endpoint does not carry
// them; asking the operator on every open would make a screen an accountant
// scrolls through into a screen that dials Didox forty times.
func (h *Handler) AdminEDIDocument(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	doc, err := h.ediDoc(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	if len(doc.Lines) == 0 {
		if err := h.ediFetchLines(r.Context(), doc); err != nil {
			// ⚠️ The document is still shown. What we have — number, date,
			// partner, total — is what the list gave us, and a screen that
			// refuses to open because the operator is slow is worse than one
			// that says the lines could not be read.
			httpx.JSON(w, http.StatusOK, map[string]any{
				"document": doc, "linesError": err.Error(),
			})
			return
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"document": doc,
		// The panel's best guess for each line, so a storekeeper confirms
		// rather than types. See ediGuess.
		"guess": h.ediGuess(r, doc),
	})
}

func (h *Handler) ediDoc(ctx context.Context, raw string) (*models.EDIDocument, error) {
	id, err := objectID(raw)
	filter := bson.M{"docId": raw}
	if err == nil {
		filter = bson.M{"_id": id}
	}
	var doc models.EDIDocument
	if err := h.Store.EDIDocuments.FindOne(ctx, filter).Decode(&doc); err != nil {
		return nil, errors.New("hujjat topilmadi")
	}
	return &doc, nil
}

// ediFetchLines reads the document body and stores its rows.
func (h *Handler) ediFetchLines(ctx context.Context, doc *models.EDIDocument) error {
	c, _, err := h.ediClient(ctx)
	if err != nil {
		return err
	}
	full, err := c.Get(ctx, doc.DocID, doc.Direction == models.EDIOut)
	if err != nil {
		return err
	}
	f, err := didox.ParseFactura(full.JSON)
	if err != nil {
		return err
	}
	lines := make([]models.EDILine, 0, len(f.ProductList.Products))
	for _, p := range f.ProductList.Products {
		lines = append(lines, models.EDILine{
			No:          p.OrdNo,
			Name:        p.Name,
			CatalogCode: p.CatalogCode,
			Barcode:     p.Barcode,
			PackageName: p.PackageName,
			PackageCode: p.PackageCode,
			Qty:         didox.Num(p.Count),
			Price:       didox.ParseMoney(p.Summa),
			Sum:         didox.ParseMoney(p.DeliverySum),
			VatRate:     didox.ParseMoney(p.VatRate),
			VatSum:      didox.ParseMoney(p.VatSum),
			SumWithVat:  didox.ParseMoney(p.DeliverySumWithVat),
		})
	}
	doc.Lines = lines
	if f.FacturaDoc.FacturaNo != "" {
		doc.Number = f.FacturaDoc.FacturaNo
	}
	_, err = h.Store.EDIDocuments.UpdateOne(ctx, bson.M{"docId": doc.DocID},
		bson.M{"$set": bson.M{"lines": lines, "number": doc.Number}})
	return err
}

// ediGuess proposes an ingredient for each line of an incoming invoice.
//
// ⚠️ **A proposal, never a decision.** The match is made on names typed by two
// different companies for the same thing, which is exactly the comparison that
// is right nine times and wrong once — and the wrong one puts a delivery of
// beef onto the shelf of butter, silently, in the one part of the product where
// a wrong number is invisible until a stocktake. So the server guesses, the
// storekeeper confirms, and the import writes only what came back.
func (h *Handler) ediGuess(r *http.Request, doc *models.EDIDocument) map[string]string {
	out := map[string]string{}
	sc, err := h.adminScope(r)
	if err != nil {
		return out
	}
	cur, err := h.Store.Ingredients.Find(r.Context(), sc.brandFilter(bson.M{}))
	if err != nil {
		return out
	}
	var ings []models.Ingredient
	_ = cur.All(r.Context(), &ings)
	byName := map[string]primitive.ObjectID{}
	for _, in := range ings {
		byName[normalizeEDIName(in.Name)] = in.ID
	}
	for _, l := range doc.Lines {
		if id, ok := byName[normalizeEDIName(l.Name)]; ok {
			out[strconv.Itoa(l.No)] = id.Hex()
		}
	}
	return out
}

// normalizeEDIName folds the ways two companies write one product.
//
// ⚠️ **Case, spaces and punctuation only.** No stemming, no synonyms, no
// "starts with": every cleverer rule matches something it should not, and the
// cost of a false match here is stock arithmetic that is wrong and looks right.
func normalizeEDIName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'а' && r <= 'я', r == 'ё':
			b.WriteRune(r)
		case r == '\'' || r == '`' || r == '’' || r == 'ʻ':
			// The apostrophes of o‘ and g‘ are written four ways on one
			// invoice; folding them is the difference between "Go'sht" and
			// "Go‘sht" being one product.
		}
	}
	return b.String()
}

type ediImportLine struct {
	No           int     `json:"no"`
	IngredientID string  `json:"ingredientId"`
	Qty          float64 `json:"qty"`
	Price        float64 `json:"price"`
}

type ediImportRequest struct {
	Lines []ediImportLine `json:"lines"`
	// Whether to create the supplier from the document when no row matches its
	// tax number.
	CreateSupplier bool `json:"createSupplier"`
}

// AdminEDIImport turns an incoming invoice into a delivery.
//
// ⚠️ **Once per document, and the document's own id is what enforces it.** The
// second import is a second delivery on the same shelf, paid for twice in every
// report — and nothing downstream could tell the pair apart, because they would
// be identical in every field that exists.
func (h *Handler) AdminEDIImport(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	// ⚠️ The delivery lands in **one** branch's store, which is why this goes
	// through the same gate every other stock screen does (CLAUDE.md §5): a
	// company-wide delivery is not a thing anybody can count.
	sc, branch, brand, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = sc
	doc, err := h.ediDoc(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	if !doc.Incoming() {
		httpx.Error(w, http.StatusBadRequest, "faqat kiruvchi hujjat kirim bo'ladi")
		return
	}
	if doc.Imported() {
		httpx.Error(w, http.StatusConflict, "bu hujjat allaqachon kirim qilingan")
		return
	}
	var req ediImportRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	byNo := map[int]models.EDILine{}
	for _, l := range doc.Lines {
		byNo[l.No] = l
	}
	lines := make([]models.PurchaseLine, 0, len(req.Lines))
	total := 0
	for _, in := range req.Lines {
		id, err := objectID(in.IngredientID)
		if err != nil || id.IsZero() {
			// A line nobody matched is skipped rather than guessed at — see
			// ediGuess. The document keeps saying what it said.
			continue
		}
		qty, price := in.Qty, in.Price
		if src, ok := byNo[in.No]; ok {
			if qty <= 0 {
				qty = src.Qty
			}
			if price <= 0 {
				price = src.Price
			}
		}
		if qty <= 0 {
			continue
		}
		line := models.PurchaseLine{IngredientID: id, Qty: qty, Price: int(price + 0.5)}
		lines = append(lines, line)
		total += line.Sum()
	}
	if len(lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, "birorta ham qator masalliqqa bog'lanmadi")
		return
	}

	supplierID, supplier := h.ediSupplier(r, brand, doc, req.CreateSupplier)

	// ⚠️ **The invoice's own date, not today's.** A delivery is dated by the
	// paper (models/purchase.go), and an electronic invoice has that date
	// printed on it — using today's would file Friday's beef under Monday and
	// move the price history with it.
	at := ediDate(doc.Date)
	p := models.Purchase{
		BranchID:   branch,
		At:         at,
		SupplierID: supplierID,
		Supplier:   supplier,
		Lines:      lines,
		// ⚠️ **The document's own total, VAT included, rather than the sum of
		// our rounded lines.** They differ, and when they do the invoice is
		// right about the money — the same rule `Purchase.Total` already
		// carries for a hand-typed delivery.
		Total:     ediTotal(doc, total),
		Note:      "Didox: " + doc.Number,
		CreatedAt: time.Now(),
		CreatedBy: h.adminName(r),
	}
	res, err := h.Store.Purchases.InsertOne(r.Context(), p)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	pid := oidOf(res.InsertedID)
	now := time.Now()
	if _, err := h.Store.EDIDocuments.UpdateOne(r.Context(),
		bson.M{"docId": doc.DocID, "purchaseId": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{
			"purchaseId": pid, "importedAt": now, "importedBy": h.adminName(r),
			"lines": ediLinesWithMatches(doc.Lines, req.Lines),
		}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, "edi.import", "purchase", pid.Hex(), doc.Number, "")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"purchaseId": pid, "lines": len(lines), "total": p.Total,
	})
}

// ediLinesWithMatches records what each line was matched to, on the document.
//
// ⚠️ Written onto our copy rather than onto the delivery: the delivery is the
// stock fact, and this answers "which row of the invoice became which row of
// the delivery" — the question somebody asks a year later with the paper in
// their hand.
func ediLinesWithMatches(lines []models.EDILine, matched []ediImportLine) []models.EDILine {
	by := map[int]primitive.ObjectID{}
	for _, m := range matched {
		if id, err := primitive.ObjectIDFromHex(m.IngredientID); err == nil {
			by[m.No] = id
		}
	}
	out := make([]models.EDILine, 0, len(lines))
	for _, l := range lines {
		l.IngredientID = by[l.No]
		out = append(out, l)
	}
	return out
}

// ediSupplier finds the supplier this invoice came from, by tax number.
//
// ⚠️ **By the number, never by the name.** The document names its sender with
// nine digits; our list holds whatever somebody typed. Matching on text would
// file one company's deliveries under three rows, which is the fault the
// supplier list was created to end (models/supplier.go).
func (h *Handler) ediSupplier(
	r *http.Request, brand primitive.ObjectID, doc *models.EDIDocument, create bool,
) (primitive.ObjectID, string) {
	name := doc.PartnerName
	tin := tinDigits(doc.PartnerTIN)
	if tin == "" {
		return primitive.NilObjectID, name
	}
	var found models.Supplier
	err := h.Store.Suppliers.FindOne(r.Context(), bson.M{"tin": tin}).Decode(&found)
	if err == nil {
		return found.ID, found.Name
	}
	if !errors.Is(err, mongo.ErrNoDocuments) || !create || name == "" {
		return primitive.NilObjectID, name
	}
	now := time.Now()
	res, err := h.Store.Suppliers.InsertOne(r.Context(), models.Supplier{
		BrandID: brand, Name: clampText(name, 120), TIN: tin,
		IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return primitive.NilObjectID, name
	}
	return oidOf(res.InsertedID), name
}

// ediDate reads the operator's "yyyy-mm-dd" as a local day.
//
// ⚠️ **Local, not UTC.** A delivery belongs to the day the wall calendar says,
// and reading it as UTC would file every morning's invoice against the previous
// evening in Tashkent — the trap CLAUDE.md describes twice.
func ediDate(s string) time.Time {
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t
	}
	return time.Now()
}

// ediTotal is what the delivery cost: the document's figure when it has one.
func ediTotal(doc *models.EDIDocument, fromLines int) int {
	if doc.TotalWithVat > 0 {
		return int(doc.TotalWithVat + 0.5)
	}
	if doc.Total > 0 {
		return int(doc.Total + 0.5)
	}
	return fromLines
}

// AdminEDIPrint proxies the operator's own printable form.
func (h *Handler) AdminEDIPrint(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	doc, err := h.ediDoc(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	c, _, err := h.ediClient(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	html, err := c.HTML(r.Context(), doc.DocID, httpx.LangOf(w))
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// ⚠️ Not framed into our own page: it is the operator's document, styles
	// and all, and re-wrapping it would be us redrawing a legal form.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(html)
}

type ediOutgoingLine struct {
	Name        string  `json:"name"`
	CatalogCode string  `json:"catalogCode"`
	CatalogName string  `json:"catalogName"`
	PackageCode string  `json:"packageCode"`
	PackageName string  `json:"packageName"`
	Barcode     string  `json:"barcode"`
	Qty         float64 `json:"qty"`
	Price       float64 `json:"price"`
	VatRate     float64 `json:"vatRate"`
}

type ediOutgoingRequest struct {
	Number       string            `json:"number"`
	Date         string            `json:"date"`
	ContractNo   string            `json:"contractNo"`
	ContractDate string            `json:"contractDate"`
	BuyerTIN     string            `json:"buyerTin"`
	Buyer        models.EDIParty   `json:"buyer"`
	HasVat       bool              `json:"hasVat"`
	Lines        []ediOutgoingLine `json:"lines"`
}

// AdminEDIOutgoing drafts an outgoing invoice at the operator.
//
// ⚠️ **A draft and nothing more, and the response says so in the same breath.**
// Nothing is filed until somebody signs it with their key at didox.uz. A panel
// that reported "yuborildi" here would be describing a document the buyer never
// receives and the tax committee never sees.
func (h *Handler) AdminEDIOutgoing(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req ediOutgoingRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	c, s, err := h.ediClient(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	buyerTIN := tinDigits(req.BuyerTIN)
	if buyerTIN == "" || len(req.Lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, "xaridorning STIRi va kamida bitta qator kerak")
		return
	}
	date := strings.TrimSpace(req.Date)
	if date == "" {
		date = time.Now().In(time.Local).Format("2006-01-02")
	}
	body, err := buildFactura(s, req, date)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := c.Create(r.Context(), models.EDITypeFacturaNoAct, body)
	if err != nil {
		h.noteEDIError(r.Context(), err)
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	h.logAction(r, "edi.draft", "edi", id, req.Number, "")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"docId": id,
		// ⚠️ The one field the screen must read out loud.
		"signedHere": false,
	})
}

// buildFactura turns the panel's request into the operator's own structure.
//
// ⚠️ **Every number is formatted once, here, as a string.** Go writes ten
// million as `1e+07`, which is valid JSON and an invalid invoice — refused
// whole by the receiving side with a message about a field that looks perfectly
// normal on our screen. See internal/didox/factura.go.
func buildFactura(s *models.EDISettings, req ediOutgoingRequest, date string) (*didox.Factura, error) {
	if s.Seller.Name == "" || s.TIN == "" {
		return nil, errors.New("sotuvchi ma'lumotlari to'liq emas — sozlamalarni to'ldiring")
	}
	products := make([]didox.Product, 0, len(req.Lines))
	for i, l := range req.Lines {
		if l.Qty <= 0 || strings.TrimSpace(l.Name) == "" {
			return nil, errors.New("qator to'liq emas: nomi va miqdori kerak")
		}
		sum := l.Qty * l.Price
		vat := 0.0
		if req.HasVat && l.VatRate > 0 {
			vat = sum * l.VatRate / 100
		}
		products = append(products, didox.Product{
			OrdNo:              i + 1,
			LgotaID:            nil,
			Name:               l.Name,
			CatalogCode:        l.CatalogCode,
			CatalogName:        l.CatalogName,
			Barcode:            l.Barcode,
			PackageCode:        l.PackageCode,
			PackageName:        l.PackageName,
			Count:              didox.Count(l.Qty),
			Summa:              didox.Money(l.Price),
			DeliverySum:        didox.Money(sum),
			VatRate:            strconv.FormatFloat(l.VatRate, 'f', -1, 64),
			VatSum:             didox.Money(vat),
			DeliverySumWithVat: didox.Money(sum + vat),
			WithoutVat:         req.HasVat && l.VatRate == 0,
			Origin:             3,
		})
	}
	buyer := didox.Party{
		Name:         req.Buyer.Name,
		VatRegCode:   req.Buyer.VatRegCode,
		VatRegStatus: req.Buyer.VatRegStatus,
		Account:      req.Buyer.Account,
		BankID:       req.Buyer.BankID,
		Address:      req.Buyer.Address,
		Director:     req.Buyer.Director,
		Accountant:   req.Buyer.Accountant,
	}
	return &didox.Factura{
		Version:         1,
		WaybillLocalIds: []string{},
		FacturaType:     0,
		ProductList: didox.ProductList{
			Tin:      s.TIN,
			HasVat:   req.HasVat,
			Products: products,
		},
		FacturaDoc:  didox.FacturaDoc{FacturaNo: req.Number, FacturaDate: date},
		ContractDoc: didox.ContractDoc{ContractNo: req.ContractNo, ContractDate: req.ContractDate},
		LotID:       "",
		SellerTin:   s.TIN,
		Seller: didox.Party{
			Name:         s.Seller.Name,
			VatRegCode:   s.Seller.VatRegCode,
			VatRegStatus: s.Seller.VatRegStatus,
			Account:      s.Seller.Account,
			BankID:       s.Seller.BankID,
			Address:      s.Seller.Address,
			Director:     s.Seller.Director,
			Accountant:   s.Seller.Accountant,
		},
		BuyerTin: tinDigits(req.BuyerTIN),
		Buyer:    &buyer,
	}, nil
}

// compile-time reminder that our copy carries the operator's raw body shape.
var _ = json.RawMessage{}
