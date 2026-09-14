package handlers

// ---- Raising an alert, and getting it to the owner ----
//
// The rules are on models.Alert. This is the machinery: threshold, ceiling,
// record, deliver.
//
// ⚠️ **Recorded first, delivered second, and never the other way round.** A
// Telegram outage, an unlinked chat, a reached ceiling — none of them may make
// the event disappear. The stored list is what the panel reads and what
// survives somebody muting the bot; the message is a convenience on top of it.
//
// ⚠️ **Nothing here may fail the thing that triggered it.** An alert is raised
// as a side effect of closing a check or saving a count. A cashier must never
// be unable to take money because a notification could not be sent — the same
// rule the print queue follows, and for the same reason.

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"

	"restaurant-backend/internal/models"
	"restaurant-backend/internal/telegram"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// alertSettingsOf reads what this branch calls unusual.
func (h *Handler) alertSettingsOf(
	ctx context.Context, branchID primitive.ObjectID,
) models.AlertSettings {
	var s models.AlertSettings
	if !branchID.IsZero() {
		if err := h.Store.AlertSettings.FindOne(ctx,
			bson.M{"branchId": branchID}).Decode(&s); err == nil {
			return s.WithDefaults()
		}
	}
	// ⚠️ **A zero branch means "the company", and the company's answer is
	// whichever branch has this switched on.**
	//
	// This was silently wrong and it switched the whole feature off for the
	// events that matter most. Settings are saved against a real branch, but
	// two kinds of event have no branch to be saved against: a recipe belongs
	// to the brand, and a panel action belongs to whoever is logged in — and an
	// owner of the whole company is pinned to no branch at all, so their id is
	// zero. Reading zero found nothing, `WithDefaults` returned `Enabled:
	// false`, and every recipe edit and every data export was dropped before it
	// was even recorded. The restaurant had turned the feature on and nothing
	// arrived.
	//
	// ⚠️ Falling back to *enabled* settings rather than to any settings: a
	// company with three branches where one has switched this on has switched
	// it on, and the thresholds of a branch that never opened the page are
	// defaults nobody chose.
	if err := h.Store.AlertSettings.FindOne(ctx,
		bson.M{"enabled": true}).Decode(&s); err == nil {
		return s.WithDefaults()
	}
	return models.AlertSettings{}.WithDefaults()
}

// raiseAlert records something worth the owner's attention and tries to send it.
//
// ⚠️ **Runs in its own goroutine with its own context.** The request that
// triggered it is finishing — a cashier is watching a spinner — and delivering
// a Telegram message takes a network round trip to another country. The context
// is deliberately not the request's: cancelling the response must not cancel
// the record.
func (h *Handler) raiseAlert(a models.LossAlert) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		h.raiseAlertSync(ctx, a)
	}()
}

func (h *Handler) raiseAlertSync(ctx context.Context, a models.LossAlert) {
	h.deliverAlert(ctx, a, false)
}

// deliverAlert records an event and, unless the owner muted its kind, sends it.
//
// ⚠️ `force` is the settings page's test button and nothing else: the test is
// sent as a large discount, and an owner who muted discounts and pressed
// "test" would hear nothing and conclude the whole channel is broken.
func (h *Handler) deliverAlert(ctx context.Context, a models.LossAlert, force bool) {
	set := h.alertSettingsOf(ctx, a.BranchID)
	if !set.Enabled {
		return
	}
	if a.At.IsZero() {
		a.At = time.Now()
	}
	res, err := h.Store.LossAlerts.InsertOne(ctx, a)
	if err != nil {
		return
	}
	a.ID = oidOf(res.InsertedID)

	// ⚠️ **After the record, before the buzzing.** A muted kind still lands in
	// the list the panel and the owner app read; only the phone and the chat
	// stay quiet. Checked before the ceiling too, so a muted kind never uses up
	// a message another kind would have needed.
	if !force && set.IsMuted(a.Kind) {
		return
	}

	// ⚠️ **The ceiling is counted over what was *sent*, not what was
	// recorded.** A bad night keeps producing records — that is the point of
	// them — and only the buzzing stops. Counting records instead would make
	// the panel's list stop growing too, which is the one thing that must not
	// happen on the night it matters.
	sent, err := h.Store.LossAlerts.CountDocuments(ctx, bson.M{
		"branchId": a.BranchID,
		"sentAt":   bson.M{"$gte": time.Now().Add(-24 * time.Hour)},
	})
	if err != nil || sent >= int64(set.DailyMax) {
		return
	}

	name := h.restaurantName(ctx)
	// ⚠️ **Two channels, and they must not be chained.** The phone buzzes
	// tonight; the group chat keeps the history, survives the owner changing
	// their number and can hold an accountant. Sending the push *before* the
	// Telegram attempt, and never inside its error path, is deliberate — this
	// codebase has already paid once for two independent deliveries joined by
	// an `else if` (the Caddy note in CLAUDE.md), and the failure there was
	// silent in exactly this shape.
	//
	// ⚠️ Owners only, the same rule `sendToOwners` applies: a manager is one of
	// the people these messages are about.
	h.notifyAdmins(a.BranchID, true, func(lang string) (string, string) {
		return alertTitle(a, lang), alertText(a, name, lang)
	}, map[string]any{"type": "alert", "kind": string(a.Kind)})

	text := alertText(a, name, h.notifyLang(ctx))
	if err := h.sendToOwners(ctx, a.BranchID, text); err != nil {
		_, _ = h.Store.LossAlerts.UpdateByID(ctx, a.ID,
			bson.M{"$set": bson.M{"sendErr": err.Error()}})
		return
	}
	now := time.Now()
	_, _ = h.Store.LossAlerts.UpdateByID(ctx, a.ID,
		bson.M{"$set": bson.M{"sentAt": now}})
}

// sendToOwners delivers to every owner who has linked a chat.
//
// ⚠️ **Owners only, and that is an access rule rather than a preference.** A
// manager is one of the people these messages are about. Sending "Dilnoza took
// 400 000 off table 6" to Dilnoza is not a leak of anything she does not know —
// it is a warning that she has been noticed, which is precisely the opposite of
// what the channel is for.
func (h *Handler) sendToOwners(
	ctx context.Context, branchID primitive.ObjectID, text string,
) error {
	var tg models.TelegramSettings
	if err := h.Store.TelegramSettings.FindOne(ctx, bson.M{}).Decode(&tg); err != nil {
		return err
	}
	if !tg.Enabled || tg.BotToken == "" {
		return errNoAlertChannel
	}
	// ⚠️ **The group first, and it is the channel that actually survives.** An
	// owner's own chat is one person, one phone and one holiday away from
	// nobody seeing any of this; a group keeps a searchable history, survives
	// the owner changing their number, and lets them add an accountant without
	// asking us. Personal chats stay as well — somebody who linked one before
	// there were groups must not silently stop being told.
	delivered := 0
	// ⚠️ **The group's error is kept, not swallowed.** It used to be dropped
	// here — and with nothing delivered the function then returned "no owner
	// has linked Telegram", which is a different fault with a different fix.
	// The real reason is almost always one sentence from Telegram ("bot is not
	// a member of the chat", "chat not found"), and that sentence is the whole
	// difference between a five-minute fix and an evening of guessing. Exactly
	// the failure the print queue was built wrong around once already.
	last := error(nil)
	if tg.AlertChatID != 0 {
		if err := h.sendNotify(ctx, tg.BotToken, tg.AlertChatID, "alertChatId", text); err != nil {
			last = err
		} else {
			delivered++
		}
	}

	cur, err := h.Store.Admins.Find(ctx, bson.M{
		"role":        "owner",
		"alertChatId": bson.M{"$gt": 0},
	})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var a models.AdminUser
		if cur.Decode(&a) != nil || a.AlertChatID == 0 {
			continue
		}
		// ⚠️ An owner pinned to one branch hears about that branch. An owner
		// pinned to none owns the company and hears about all of them — the
		// scoping rule every admin screen already follows.
		if !a.BranchID.IsZero() && a.BranchID != branchID {
			continue
		}
		if err := telegram.SendMessage(ctx, tg.BotToken, a.AlertChatID, text); err != nil {
			last = err
			continue
		}
		delivered++
	}
	if delivered == 0 {
		if last != nil {
			return last
		}
		return errNoAlertChannel
	}
	return nil
}

type alertErr string

func (e alertErr) Error() string { return string(e) }

const errNoAlertChannel = alertErr("hech bir ega Telegram ulamagan")

// alertText is what arrives on the phone.
//
// ⚠️ **What happened, who, how much — and no verdict.** Every kind here has an
// ordinary explanation that happens weekly in a busy restaurant: a guest who
// complained after seeing the bill, a regular given something off, a till short
// because somebody paid a courier out of it. A message that concluded anything
// would be wrong often enough to be resented, and resented notifications get
// muted rather than argued with.
// alertTitle is the heading a phone shows above the sentence.
//
// ⚠️ **The kind, and nothing else.** A notification title is one line on a lock
// screen: "Katta chegirma" tells an owner whether to open the phone now, and
// the amount, the person and the table are in the body where there is room for
// them. Telegram has no titles at all, which is why this is new here.
func alertTitle(a models.LossAlert, lang string) string {
	w := notifyWordsFor(lang)
	if head, ok := map[models.AlertKind]string{
		models.AlertVoidAfterPrecheck: w.VoidAfterPrecheck,
		models.AlertBigDiscount:       w.BigDiscount,
		models.AlertCashShort:         w.CashShort,
		models.AlertStockShort:        w.StockShort,
		models.AlertRecipeUp:          w.RecipeUp,
		models.AlertPanelAction:       w.PanelAction,
		models.AlertCheckCancelled:    w.CheckCancelled,
		models.AlertShiftOverdue:      w.ShiftOverdue,
		models.AlertCheckRefunded:     w.CheckRefunded,
		models.AlertOrderCancelled:    w.OrderCancelled,
		models.AlertCashOut:           w.CashOut,
		models.AlertBigWriteoff:       w.BigWriteoff,
		models.AlertDebtWritten:       w.DebtWritten,
	}[a.Kind]; ok && head != "" {
		return head
	}
	return w.Unknown
}

func alertText(a models.LossAlert, restaurant, lang string) string {
	w := notifyWordsFor(lang)
	head := map[models.AlertKind]string{
		models.AlertVoidAfterPrecheck: w.VoidAfterPrecheck,
		models.AlertBigDiscount:       w.BigDiscount,
		models.AlertCashShort:         w.CashShort,
		models.AlertStockShort:        w.StockShort,
		models.AlertRecipeUp:          w.RecipeUp,
		models.AlertPanelAction:       w.PanelAction,
		models.AlertCheckCancelled:    w.CheckCancelled,
		models.AlertShiftOverdue:      w.ShiftOverdue,
		models.AlertCheckRefunded:     w.CheckRefunded,
		models.AlertOrderCancelled:    w.OrderCancelled,
		models.AlertCashOut:           w.CashOut,
		models.AlertBigWriteoff:       w.BigWriteoff,
		models.AlertDebtWritten:       w.DebtWritten,
	}[a.Kind]
	if head == "" {
		head = w.Unknown
	}
	out := "⚠️ " + head
	if restaurant != "" {
		out += " · " + restaurant
	}
	// ⚠️ A zero amount prints nothing rather than "0 so'm". Some kinds have no
	// meaningful figure yet — a recipe change costs whatever gets sold — and a
	// zero on the phone reads as a bug in the alert, not as an absence.
	if a.Amount != 0 {
		// ⚠️ **One kind counts hours, not money.** A shift left open has lost
		// nobody anything yet — what it has lost is the ability to check — and
		// printing a sum of so'm beside it would read as an accusation about a
		// figure that does not exist.
		unit := w.Currency
		if a.Kind == models.AlertShiftOverdue {
			unit = w.Hours
		}
		out += "\n" + formatSom(a.Amount) + " " + unit
	}
	// ⚠️ **The table and the check number are rendered here, in the language
	// the group reads, not stored as a sentence.** They used to be glued into
	// `Subject` as "6-stol" — an Uzbek word arriving in a Russian message, and
	// a table number with no check number, which names a table that has had
	// nine checks today.
	where := ""
	if a.Table != "" {
		where = w.Table + " " + a.Table
	}
	if a.Number != "" {
		if where != "" {
			where += " · "
		}
		where += "#" + a.Number
	}
	if where != "" {
		out += "\n" + where
	}
	if a.Subject != "" {
		out += "\n" + a.Subject
	}
	if a.AfterPrecheck {
		out += "\n⚠️ " + w.AfterPrecheck
	}
	if a.By != "" {
		out += "\n" + w.Who + ": " + a.By
		if a.AuthBy != "" {
			out += " (" + a.AuthBy + " " + w.Approved + ")"
		}
	}
	if a.Reason != "" {
		out += "\n" + w.Reason + ": " + a.Reason
	}
	out += "\n" + a.At.In(time.Local).Format("02.01 15:04")
	return out
}

// formatSom groups thousands the way every other figure in this product does.
func formatSom(n int) string {
	if n < 0 {
		return "-" + formatSom(-n)
	}
	s := fmt.Sprintf("%d", n)
	out := ""
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += " "
		}
		out += string(r)
	}
	return out
}

// ---- The triggers ----

// alertOnVoidsAfterPrecheck raises one per line removed after the guest saw the
// bill.
//
// ⚠️ **The order of the two events is the entire signal.** Removing a line
// before the precheck is ordinary work — a wrong order, a changed mind, a
// kitchen that ran out — and is not raised at all. After it, the guest has been
// shown a total and the total went down.
func (h *Handler) alertOnVoidsAfterPrecheck(o *models.Order, set models.AlertSettings) {
	if o == nil || o.Check == nil {
		return
	}
	for _, it := range o.Items {
		v := it.Void
		// ⚠️ **Cooked food, whether or not a bill was printed first — widened
		// after a restaurant voided a main course and heard nothing.**
		//
		// This used to require a precheck, on the reasoning that removing a
		// line before the guest sees the bill is ordinary work. Half right:
		// removing a line *nobody has cooked* is ordinary work, and that is
		// what `FiredAt` tests. Removing one the kitchen has already made is
		// food the restaurant paid for — the sentence written on
		// `CheckLineVoid` itself, which calls it the single event a till exists
		// to record. The precheck makes it worse; it was never what made it
		// worth knowing.
		if v == nil || it.FiredAt == nil {
			continue
		}
		value := it.Price * it.Qty
		if value < set.VoidFrom {
			continue
		}
		// The stronger fact, in the words rather than in the filter: the guest
		// had been shown a total and then the total went down.
		subject := it.Name
		afterBill := o.Check.PrecheckAt != nil && v.At.After(*o.Check.PrecheckAt)
		h.raiseAlert(models.LossAlert{
			BranchID: o.BranchID,
			Kind:     models.AlertVoidAfterPrecheck,
			At:       v.At,
			ByID:     v.ByID, By: v.By, AuthBy: v.AuthBy,
			Amount: value,
			Reason: v.Reason,
			// The dish, which is the only part of this that is not a
			// language-neutral fact and is the restaurant's own word anyway.
			Subject:       subject,
			AfterPrecheck: afterBill,
			Number:        o.Number,
			Table:         o.TableNumber,
			RefID:         o.ID,
		})
	}
}

// tableLabel is gone: the table and the check number travel as fields and are
// worded at send time. ⚠️ Left as a note rather than silently deleted, because
// the reason it went is the bug it caused — an Uzbek word baked into stored
// data and read out in a Russian message.

// alertOnDiscount raises one when a person took a large amount off a bill.
func (h *Handler) alertOnDiscount(o *models.Order, set models.AlertSettings) {
	if o == nil {
		return
	}
	for _, d := range o.Discounts {
		// ⚠️ A discount with no name on it was decided by a rule — a promotion
		// that matched, points spent. Nobody chose it, so there is nobody to
		// tell the owner about.
		if d.ByID.IsZero() && d.By == "" {
			continue
		}
		if d.Amount < set.DiscountFrom {
			continue
		}
		h.raiseAlert(models.LossAlert{
			BranchID: o.BranchID,
			Kind:     models.AlertBigDiscount,
			ByID:     d.ByID, By: d.By, AuthBy: d.AuthBy,
			Amount: d.Amount,
			Reason: d.Reason,
			Number: o.Number,
			Table:  o.TableNumber,
			RefID:  o.ID,
		})
	}
}

// AdminLossAlerts is the list, newest first.
//
// ⚠️ Named apart from `AdminAlerts`, which already exists and means something
// completely different: the panel's few-second poll for "has an order
// arrived". One word, two features, and the collision would be discovered by
// whoever next went looking for one of them.
//
// ⚠️ **Owner only, like the report.** A manager is one of the names in it.
func (h *Handler) AdminLossAlerts(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	cur, err := h.Store.LossAlerts.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(100))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cur.Close(r.Context())
	// Never nil — the JSON trap this codebase has been bitten by twice, and
	// this list is empty in every restaurant that has had a quiet month.
	out := []models.LossAlert{}
	for cur.Next(r.Context()) {
		var a models.LossAlert
		if cur.Decode(&a) == nil {
			out = append(out, a)
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"alerts": out})
}

// AdminAlertSettings is what this branch calls unusual, and how to switch it on.
func (h *Handler) AdminAlertSettings(w http.ResponseWriter, r *http.Request) {
	admin, err := h.adminUser(r)
	if err != nil || admin.Role != "owner" {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	_, branch, _, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := h.alertSettingsOf(r.Context(), branch)
	// ⚠️ **Whether there is anywhere for these to go, said on the page that
	// switches them on.** Two settings govern one feature — a channel over on
	// the Telegram page, and this switch — and a restaurant that configures the
	// channel, tests it successfully and hears nothing has done everything that
	// looked like the job. The screen has to close that gap itself rather than
	// leaving somebody to guess which of two pages is the one still undone.
	var tg models.TelegramSettings
	_ = h.Store.TelegramSettings.FindOne(r.Context(), bson.M{}).Decode(&tg)
	hasChannel := tg.Enabled && tg.BotToken != "" &&
		(tg.AlertChatID != 0 || admin.AlertChatID != 0)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"settings":   set,
		"hasChannel": hasChannel,
		// ⚠️ Whether *this* owner has linked a chat, and the link to do it.
		// The chat id itself never leaves the server: it is enough to message
		// somebody with.
		"linked": admin.AlertChatID != 0,
		"link":   h.alertLinkFor(r.Context(), admin.ID.Hex()),
	})
}

// alertLinkFor is the deep link the owner taps.
//
// ⚠️ Empty when no bot is configured, so the panel can say "set up Telegram
// first" instead of drawing a link to nowhere — which is how a working feature
// gets reported as broken.
func (h *Handler) alertLinkFor(ctx context.Context, adminID string) string {
	var tg models.TelegramSettings
	if err := h.Store.TelegramSettings.FindOne(ctx, bson.M{}).Decode(&tg); err != nil {
		return ""
	}
	if !tg.Enabled || tg.BotUsername == "" {
		return ""
	}
	return "https://t.me/" + tg.BotUsername + "?start=" +
		alertLinkPrefix + alertLinkToken(h.Cfg.JWTSecret, adminID)
}

// AdminSaveAlertSettings stores the thresholds.
func (h *Handler) AdminSaveAlertSettings(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	_, branch, _, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var in models.AlertSettings
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	in.BranchID = branch
	in.UpdatedAt = time.Now()
	if _, err := h.Store.AlertSettings.UpdateOne(r.Context(),
		bson.M{"branchId": branch},
		bson.M{"$set": bson.M{
			"branchId": branch, "enabled": in.Enabled,
			"discountFrom": in.DiscountFrom, "cashShortFrom": in.CashShortFrom,
			"stockShortFrom": in.StockShortFrom, "voidFrom": in.VoidFrom,
			"dailyMax": in.DailyMax, "muted": mutedKinds(in.Muted),
			"refundFrom": in.RefundFrom, "cashOutFrom": in.CashOutFrom,
			"writeoffFrom": in.WriteoffFrom, "debtFrom": in.DebtFrom,
			"updatedAt": in.UpdatedAt,
		}},
		options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActSettingsUpdate, "alerts", branch.Hex(), "Nazorat sozlamalari", "")
	httpx.JSON(w, http.StatusOK, h.alertSettingsOf(r.Context(), branch).WithDefaults())
}

// mutedKinds keeps the kinds this server knows, once each. ⚠️ An unknown name
// is dropped rather than stored: a typo saved here would look like a muted kind
// in the database and mute nothing.
func mutedKinds(in []models.AlertKind) []models.AlertKind {
	out := []models.AlertKind{}
	seen := map[models.AlertKind]bool{}
	for _, k := range in {
		if models.IsAlertKind(k) && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

// AdminUnlinkAlerts stops this owner's chat receiving them.
//
// ⚠️ **Present because the channel names colleagues.** Somebody who wants it to
// stop must be able to stop it from the same screen that started it, without
// asking us — a channel you cannot leave is one people block the bot to escape,
// and blocking the bot takes the guest-facing menu with it.
func (h *Handler) AdminUnlinkAlerts(w http.ResponseWriter, r *http.Request) {
	admin, err := h.adminUser(r)
	if err != nil || admin.Role != "owner" {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	_, _ = h.Store.Admins.UpdateByID(r.Context(), admin.ID,
		bson.M{"$unset": bson.M{"alertChatId": ""}})
	httpx.JSON(w, http.StatusOK, map[string]any{"linked": false})
}

// AdminTestAlert runs one alert down the real path and reports what happened.
//
// ⚠️ **Built after a restaurant configured everything correctly and heard
// nothing, twice.** The channel tested fine, the switch was on, and no alert
// arrived — and there was no way to tell which of six gates it had stopped at:
// the settings switch, the branch the settings are filed under, the daily
// ceiling, the bot token, the group id, or simply no event having crossed a
// threshold. Six silent failures behind one silence.
//
// The same button the printers page has, for the same reason, pressed by the
// same person: somebody who can look at the result immediately.
//
// ⚠️ **It uses the real machinery — `raiseAlertSync` — not a shortcut to
// Telegram.** A test that skipped the settings lookup and the ceiling would
// pass on a restaurant where the real thing cannot work, which is worse than
// having no test at all.
func (h *Handler) AdminTestAlert(w http.ResponseWriter, r *http.Request) {
	admin, err := h.adminUser(r)
	if err != nil || admin.Role != "owner" {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	_, branch, _, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := h.alertSettingsOf(r.Context(), branch)
	if !set.Enabled {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok":     false,
			"reason": "Bu bo'limdagi \"Telegramga xabar yuborish\" yoqilmagan.",
		})
		return
	}

	a := models.LossAlert{
		BranchID: branch,
		Kind:     models.AlertBigDiscount,
		At:       time.Now(),
		By:       admin.Name,
		Amount:   set.DiscountFrom,
		Subject:  "Sinov",
		Reason:   "sinov xabari",
	}
	// ⚠️ Run synchronously, unlike every real trigger: the person is watching a
	// spinner and the whole point is the answer. A real alert is raised in a
	// goroutine because a cashier must never wait for Telegram.
	h.deliverAlert(r.Context(), a, true)

	// ⚠️ **The verdict is read back off the stored record**, not assumed from
	// the fact that nothing panicked. `sentAt` and `sendErr` are what the real
	// path writes, so the test reports exactly what a real alert would have
	// reported — including the ceiling, which leaves both fields empty.
	var saved models.LossAlert
	_ = h.Store.LossAlerts.FindOne(r.Context(),
		bson.M{"branchId": branch, "subject": "Sinov"},
		options.FindOne().SetSort(bson.D{{Key: "at", Value: -1}}),
	).Decode(&saved)

	switch {
	case saved.SentAt != nil:
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
	case saved.SendErr != "":
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": false, "reason": saved.SendErr,
		})
	case saved.ID.IsZero():
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": false,
			"reason": "Yozib bo'lmadi — sozlama boshqa filialga saqlangan " +
				"bo'lishi mumkin.",
		})
	default:
		// Recorded and not sent, and no error: the only remaining path is the
		// daily ceiling.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": false,
			"reason": "Bugungi limit tugagan (" +
				formatSom(set.DailyMax) + " ta). Ertaga yana ishlaydi.",
		})
	}
}
