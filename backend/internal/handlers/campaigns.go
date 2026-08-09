package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/telegram"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Sending one message to a segment.
//
// The segments already existed and led nowhere: the panel could tell an owner
// that eleven of their VIPs had gone quiet and then offer nothing to do about
// it. This is the other half — and it is the half that spends real money and
// reaches real people's phones, so almost everything here is a guard.
//
// The rules, in the order they matter:
//
//   - **Nobody is messaged who asked not to be.** `user.noMarketing` is a hard
//     exclusion, not a filter on a screen. A guest who opts out and then gets
//     another advert does not complain to us — they stop being the restaurant's
//     customer, which is the exact opposite of what this feature is for.
//   - **The count is shown before the send, and it is the real count.** A
//     campaign is spent money: the preview runs the same audience query the
//     send does, so the number in the confirmation is the number of messages
//     that will be paid for.
//   - **One campaign at a time.** Two overlapping sends to overlapping segments
//     is how a customer gets the same advert twice, and it cannot be undone.
//   - **Owner only.** The customer base is the company's, a manager runs one
//     kitchen, and this is the button that can annoy every guest at once.
//   - **Everything is recorded.** What was sent, to which segment, by whom, how
//     many arrived and how many failed. "Did we message them?" has to be
//     answerable months later, usually because somebody is annoyed.
const (
	// Hard ceiling on one campaign. Not a technical limit — a circuit breaker:
	// the failure mode worth preventing is a typo in a segment name reaching
	// the entire base, and no restaurant in this product legitimately messages
	// more than this in one go.
	campaignMaxRecipients = 5000
	// Text length. Two SMS parts of Cyrillic is already a long advert; past
	// this it is somebody pasting a document into the box.
	campaignMaxText = 480
	// How fast messages go out. Gateways here rate-limit and some silently drop
	// bursts, and a campaign that half-arrives is worse than a slow one.
	campaignSendDelay = 120 * time.Millisecond
)

// audienceMember is one recipient. Deliberately tiny: the send loop holds the
// whole audience in memory, and it has no use for the rest of the customer.
type audienceMember struct {
	UserID primitive.ObjectID
	Name   string
	Phone  string
	// Set when this guest has opened the bot. The Telegram channel messages these
	// and nobody else; SMS ignores it.
	TelegramID int64
	// Which language to write to them in: what they chose, then Telegram's guess,
	// then Uzbek. Carried on the member so the send loop does not read the user
	// again per message — and so the buttons are labelled in it.
	Lang string
}

// audience returns who is in a segment right now, and what was excluded.
//
// ⚠️ **Computed, never stored** — the same rule as the segments themselves. A
// saved audience list is wrong by the next order: somebody who ordered
// yesterday is no longer "asleep", and messaging them a "we miss you" discount
// is how a working feature reads as a broken one.
// ⚠️ `channel` decides both who is reachable and how duplicates are counted, and
// the two differ in a way that matters: a guest who signed in through Telegram has
// no phone number at all, so the SMS audience correctly excludes them while the
// Telegram audience must not. Deduplication follows the same logic — one message per
// phone for SMS, one per Telegram account for the bot.
func (h *Handler) audience(ctx context.Context, segment, channel string) (list []audienceMember, optedOut, missing int, err error) {
	cur, err := h.Store.Users.Find(ctx, bson.M{})
	if err != nil {
		return nil, 0, 0, err
	}
	var users []models.User
	if err := cur.All(ctx, &users); err != nil {
		return nil, 0, 0, err
	}

	facts, floor := h.customerFactsByUser(ctx)
	now := time.Now()
	unhappy := h.unhappyUserSet(ctx)
	seen := map[string]bool{}

	for _, u := range users {
		f := facts[u.ID.Hex()]
		f.Birthday = u.Birthday
		f.Unhappy = unhappy[u.ID.Hex()]
		inSegment := false
		for _, s := range segmentsFor(f, floor, now) {
			if s == segment {
				inSegment = true
				break
			}
		}
		if !inSegment {
			continue
		}
		// Counted separately from "not in the segment" so the panel can say
		// *why* an audience is smaller than the segment badge suggests. A
		// number that quietly disagrees with the one on the customers page is a
		// number nobody trusts.
		if u.NoMarketing {
			optedOut++
			continue
		}
		phone := strings.TrimSpace(u.Phone)

		if channel == models.CampaignTelegram {
			if u.TelegramID == 0 {
				// Never opened the bot. Counted, not silently dropped: "invite them
				// to the bot" and "collect a number" are different jobs.
				missing++
				continue
			}
			key := strconv.FormatInt(u.TelegramID, 10)
			if seen[key] {
				continue
			}
			seen[key] = true
			list = append(list, audienceMember{
				UserID: u.ID, Name: u.FirstName, Phone: phone,
				TelegramID: u.TelegramID, Lang: notifyLang(&u),
			})
			continue
		}

		if phone == "" {
			missing++
			continue
		}
		// One message per phone, not per account: a family sharing a number, or
		// a customer with two accounts, must not be paid for twice — and must
		// not get the same advert twice.
		if seen[phone] {
			continue
		}
		seen[phone] = true
		list = append(list, audienceMember{UserID: u.ID, Name: u.FirstName, Phone: phone})
	}
	return list, optedOut, missing, nil
}

// oneCustomer is the audience for a message to a single named guest.
//
// ⚠️ `noMarketing` is honoured here too, deliberately, even though the operator
// picked this person on purpose. The guest asked not to be advertised to, and a
// direct send is not a different kind of advert — it is the same one with more
// intent. Order updates and login codes are unaffected: those are the service they
// asked for.
func (h *Handler) oneCustomer(ctx context.Context, id primitive.ObjectID,
	channel string) (list []audienceMember, blocked string, err error) {
	var u models.User
	if err := h.Store.Users.FindOne(ctx, bson.M{"_id": id}).Decode(&u); err != nil {
		return nil, "mijoz topilmadi", nil
	}
	if u.NoMarketing {
		return nil, "bu mijoz reklama xabarlarini olishni rad etgan", nil
	}
	if channel == models.CampaignTelegram {
		if u.TelegramID == 0 {
			return nil, "bu mijozning hisobi Telegram botga bog'lanmagan", nil
		}
		return []audienceMember{{
			UserID: u.ID, Name: u.FirstName, Phone: u.Phone,
			TelegramID: u.TelegramID, Lang: notifyLang(&u),
		}}, "", nil
	}
	phone := strings.TrimSpace(u.Phone)
	if phone == "" {
		return nil, "bu mijozning telefon raqami yo'q", nil
	}
	return []audienceMember{{UserID: u.ID, Name: u.FirstName, Phone: phone}}, "", nil
}

// customerFactsByUser is the order history behind every segment decision, in one
// aggregation rather than a query per customer, plus the VIP cut computed from
// the whole base.
//
// Shared with the customers list on purpose: two pieces of code deciding who is
// a VIP is two different answers on two screens, and the owner would be right
// not to believe either.
func (h *Handler) customerFactsByUser(ctx context.Context) (map[string]customerFacts, int) {
	type agg struct {
		ID    any       `bson:"_id"`
		Count int       `bson:"count"`
		Total int       `bson:"total"`
		First time.Time `bson:"first"`
		Last  time.Time `bson:"last"`
	}
	out := map[string]customerFacts{}
	totals := []int{}

	pipeline := []bson.M{
		{"$match": bson.M{"userId": bson.M{"$exists": true}}},
		{"$group": bson.M{
			"_id":   "$userId",
			"count": bson.M{"$sum": 1},
			"total": bson.M{"$sum": bson.M{"$cond": []any{
				bson.M{"$eq": []any{"$status", string(models.StatusCancelled)}},
				0, "$total",
			}}},
			"first": bson.M{"$min": "$createdAt"},
			"last":  bson.M{"$max": "$createdAt"},
		}},
	}
	cur, err := h.Store.Orders.Aggregate(ctx, pipeline)
	if err != nil {
		return out, 0
	}
	var rows []agg
	if err := cur.All(ctx, &rows); err != nil {
		return out, 0
	}
	for _, row := range rows {
		f := customerFacts{OrdersCount: row.Count, OrdersTotal: row.Total}
		if !row.First.IsZero() {
			first := row.First
			f.FirstOrder = &first
		}
		if !row.Last.IsZero() {
			last := row.Last
			f.LastOrder = &last
		}
		out[toHex(row.ID)] = f
		totals = append(totals, row.Total)
	}
	return out, vipFloor(totals)
}

// ---- Segment sizes ----

// AdminSegments lists every segment with how many people are in it and how many
// of those can actually be messaged.
//
// Both numbers, always. "VIP: 24" next to a send that goes to 19 people is a
// screen an owner stops believing, and the gap (opted out, no phone) is exactly
// what they need to know before writing the message.
func (h *Handler) AdminSegments(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	ctx := r.Context()
	type row struct {
		Segment   string `json:"segment"`
		Total     int    `json:"total"`
		Reachable int    `json:"reachable"`
		OptedOut  int    `json:"optedOut"`
		NoPhone   int    `json:"noPhone"`
	}
	out := []row{}
	for _, seg := range []string{
		SegNew, SegRegular, SegVIP, SegSleeping, SegLost, SegBirthday,
		SegUnhappy, SegNoOrders,
	} {
		// The segments screen counts the SMS audience, which is the one that costs
		// money and the one the numbers on the customers page are about.
		list, optedOut, noPhone, err := h.audience(ctx, seg, models.CampaignSMS)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, row{
			Segment:   seg,
			Total:     len(list) + optedOut + noPhone,
			Reachable: len(list),
			OptedOut:  optedOut,
			NoPhone:   noPhone,
		})
	}
	httpx.JSON(w, http.StatusOK, out)
}

// ---- Preview and send ----

type campaignRequest struct {
	Segment string `json:"segment"`
	Text    string `json:"text"`
	// "sms" (the default, and what every existing client sends) or "telegram".
	Channel string `json:"channel"`
	// Telegram only: a photograph sent with the message.
	Image string `json:"image"`
	// ⚠️ One named customer instead of a segment.
	//
	// A segment answers "who is like this"; sometimes the operator knows exactly who
	// they want — the guest who complained this morning, the regular whose birthday
	// it is. Sent through the same pipeline rather than a second one: the opt-out
	// rule, the channel rules and the record in the campaign log are the same, and a
	// separate path would eventually forget one of them.
	UserID string `json:"userId"`
}

// campaignChannel narrows what the panel asked for. Anything unrecognised is SMS —
// the channel every existing client sends nothing for.
func campaignChannel(v string) string {
	if strings.TrimSpace(v) == models.CampaignTelegram {
		return models.CampaignTelegram
	}
	return models.CampaignSMS
}

// smsParts is how many messages the gateway will bill for one text.
//
// Shown because this is per-part pricing and the boundary is invisible: an
// owner adding a polite closing sentence can double the cost of a campaign
// without anything on screen changing. Uzbek written properly (oʻ, gʻ) and
// anything Cyrillic falls outside GSM-7, so it is 70 characters per part, not
// 160 — which surprises everybody the first time.
func smsParts(text string) int {
	n := utf8.RuneCountInString(text)
	if n == 0 {
		return 0
	}
	single, multi := 160, 153
	if !isGSM7(text) {
		single, multi = 70, 67
	}
	if n <= single {
		return 1
	}
	return (n + multi - 1) / multi
}

// isGSM7 reports whether every character fits the basic SMS alphabet. Kept
// deliberately conservative: guessing "probably fits" understates the cost, and
// the number this feeds is the one an owner budgets from.
func isGSM7(s string) bool {
	const gsm = "@£$¥èéùìòÇØøÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?" +
		"¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà\n\r"
	const ext = "^{}\\[~]|€"
	for _, r := range s {
		if !strings.ContainsRune(gsm, r) && !strings.ContainsRune(ext, r) {
			return false
		}
	}
	return true
}

// AdminCampaignPreview answers "who would get this, and what would it cost".
func (h *Handler) AdminCampaignPreview(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req campaignRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	channel := campaignChannel(req.Channel)
	var (
		list     []audienceMember
		optedOut int
		missing  int
		err      error
	)
	if id, e := objectID(strings.TrimSpace(req.UserID)); e == nil {
		var blocked string
		list, blocked, err = h.oneCustomer(r.Context(), id, channel)
		if blocked != "" {
			// Answered as a preview rather than as an error: the operator is still
			// composing, and the reason is the useful part.
			httpx.JSON(w, http.StatusOK, map[string]any{
				"recipients": 0, "channel": channel, "blocked": blocked,
			})
			return
		}
	} else {
		list, optedOut, missing, err = h.audience(r.Context(), strings.TrimSpace(req.Segment), channel)
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	parts := smsParts(strings.TrimSpace(req.Text))
	sender := h.sender(r.Context())
	if channel == models.CampaignTelegram {
		tg := h.telegramSettings(r.Context())
		// ⚠️ No parts and no cost, and both are stated rather than left blank: the
		// panel's cost line is the one an owner reads before pressing send, and an
		// empty figure reads as "unknown" rather than "free".
		httpx.JSON(w, http.StatusOK, map[string]any{
			"recipients": len(list),
			"optedOut":   optedOut,
			"noTelegram": missing,
			"channel":    channel,
			"free":       true,
			"ready":      tg.Usable(),
		})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"recipients": len(list),
		"optedOut":   optedOut,
		"noPhone":    missing,
		"channel":    channel,
		"parts":      parts,
		// The number that actually gets billed. Spelled out rather than left
		// for the owner to multiply: per-part pricing is the part people get
		// wrong, and they get it wrong after sending.
		"messages": parts * len(list),
		// Said before the send, not discovered after: with no gateway the
		// campaign would "succeed" and reach nobody.
		"demo":     sender.Demo(),
		"provider": sender.Name(),
	})
}

// AdminSendCampaign sends one message to one segment.
func (h *Handler) AdminSendCampaign(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req campaignRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	segment := strings.TrimSpace(req.Segment)
	text := strings.TrimSpace(req.Text)
	if text == "" {
		httpx.Error(w, http.StatusBadRequest, "xabar matni bo'sh")
		return
	}
	if utf8.RuneCountInString(text) > campaignMaxText {
		httpx.Error(w, http.StatusBadRequest, "xabar juda uzun")
		return
	}

	ctx := r.Context()
	channel := campaignChannel(req.Channel)
	image := sanitizeCampaignImage(req.Image)
	sender := h.sender(ctx)
	if channel == models.CampaignTelegram {
		// ⚠️ The same refusal the SMS path makes, for the same reason: a campaign
		// that reports "sent to 240" and reached nobody is the worst outcome here,
		// because the owner waits for orders nobody was asked for and concludes the
		// customers stopped caring.
		if s := h.telegramSettings(ctx); !s.Usable() {
			httpx.Error(w, http.StatusServiceUnavailable,
				"Telegram bot ulanmagan — sozlamalardan tokenni kiriting va ulanishni tekshiring")
			return
		}
	} else if sender.Demo() {
		// Refused rather than accepted quietly. A campaign that reports "sent
		// to 240 people" and reached nobody is the worst possible outcome here:
		// the owner acts on it, waits for orders that were never asked for, and
		// concludes the customers no longer care.
		httpx.Error(w, http.StatusServiceUnavailable,
			"SMS shlyuzi sozlanmagan — sozlamalardan provayderni ulang va sinov SMS yuboring")
		return
	}

	// One at a time. Two overlapping campaigns to overlapping segments means a
	// customer gets the same advert twice, and there is no recalling it.
	if n, err := h.Store.DB.Collection("campaign").CountDocuments(ctx,
		bson.M{"status": models.CampaignSending}); err == nil && n > 0 {
		httpx.Error(w, http.StatusConflict,
			"hozir boshqa kampaniya yuborilmoqda — tugashini kuting")
		return
	}

	var (
		list     []audienceMember
		optedOut int
		missing  int
		err      error
	)
	if id, e := objectID(strings.TrimSpace(req.UserID)); e == nil {
		var blocked string
		list, blocked, err = h.oneCustomer(ctx, id, channel)
		if blocked != "" {
			httpx.Error(w, http.StatusBadRequest, blocked)
			return
		}
		if segment == "" {
			// The log has to say what this was. "direct" rather than an empty cell,
			// because a campaign row with no audience named is a row nobody can
			// interpret six months later.
			segment = "direct"
		}
	} else {
		list, optedOut, missing, err = h.audience(ctx, segment, channel)
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(list) == 0 {
		httpx.Error(w, http.StatusBadRequest, "bu segmentda xabar yuboradigan odam yo'q")
		return
	}
	if len(list) > campaignMaxRecipients {
		httpx.Error(w, http.StatusBadRequest, "juda ko'p qabul qiluvchi — segmentni toraytiring")
		return
	}

	admin, err := h.adminUser(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	now := time.Now()
	c := models.Campaign{
		Segment:   segment,
		Text:      text,
		Status:    models.CampaignSending,
		Channel:   channel,
		Image:     image,
		Total:     len(list),
		OptedOut:  optedOut,
		Parts:     smsParts(text),
		Provider:  sender.Name(),
		CreatedBy: admin.Username,
		CreatedAt: now,
		StartedAt: &now,
	}
	if channel == models.CampaignTelegram {
		c.NoTelegram = missing
		c.Parts = 0 // nothing is billed per part here
		c.Provider = "telegram"
	} else {
		c.NoPhone = missing
	}
	res, err := h.Store.DB.Collection("campaign").InsertOne(ctx, c)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.InsertedID.(primitive.ObjectID)
	c.ID = id

	h.logAction(r, ActCampaignSend, "campaign", id.Hex(), segment,
		fmt.Sprintf("%d ta qabul qiluvchi, %d SMS", len(list), c.Parts*len(list)))

	// Sent in the background, with the request's values copied out.
	//
	// `context.WithoutCancel` because the request ends the moment the panel gets
	// its answer, and a campaign that stops halfway through because the operator
	// closed the tab has messaged an arbitrary half of the segment — the one
	// outcome with no way back.
	go h.runCampaign(context.WithoutCancel(ctx), id, campaignSend{
		Text: text, Channel: channel, Image: image,
	}, list)

	httpx.JSON(w, http.StatusAccepted, map[string]any{
		"id":         id.Hex(),
		"recipients": len(list),
		"status":     models.CampaignSending,
	})
}

// runCampaign walks the audience, slowly, recording progress as it goes.
// campaignSend is what one run is sending: the same message, one of two ways.
type campaignSend struct {
	Text    string
	Channel string
	Image   string
}

func (h *Handler) runCampaign(ctx context.Context, id primitive.ObjectID,
	job campaignSend, list []audienceMember) {
	coll := h.Store.DB.Collection("campaign")
	sender := h.sender(ctx)
	tg := h.telegramSettings(ctx)
	// ⚠️ Built **per guest**, not once: the labels are words, and a Russian speaker
	// reading "Fikr bildirish" is a guest who does not press it. The mini app
	// address is per-language too, because the site carries language in the path.
	name := h.restaurantName(ctx)
	_ = name
	photo := ""
	if job.Image != "" {
		// ⚠️ An absolute URL, because Telegram fetches it themselves. A `/uploads/…`
		// path means nothing to their servers, and the message is rejected with an
		// error about the photo rather than about the path.
		photo = strings.TrimRight(h.Cfg.PublicBaseURL, "/") + job.Image
	}
	sent, failed := 0, 0
	firstError := ""

	for _, m := range list {
		sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		var err error
		if job.Channel == models.CampaignTelegram {
			err = telegram.SendCampaign(sendCtx, tg.BotToken, m.TelegramID,
				job.Text, photo, h.campaignButtons(id, m.Lang))
		} else {
			err = sender.Send(sendCtx, m.Phone, job.Text)
		}
		cancel()
		if err != nil {
			failed++
			if firstError == "" {
				// The first reason, kept and shown. A campaign that reports "84
				// failed" with no reason leaves the owner to guess between "no
				// credit", "sender name not approved" and "our bug" — three
				// completely different next steps.
				firstError = err.Error()
			}
		} else {
			sent++
		}

		// Progress is written as it happens, not at the end: this is the one
		// screen somebody watches while it runs, and a campaign that shows
		// nothing for four minutes looks stuck.
		if (sent+failed)%20 == 0 {
			_, _ = coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
				"sent": sent, "failed": failed, "error": firstError,
			}})
		}
		time.Sleep(campaignSendDelay)
	}

	done := time.Now()
	status := models.CampaignDone
	if sent == 0 {
		status = models.CampaignFailed
	}
	if _, err := coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"status": status, "sent": sent, "failed": failed,
		"error": firstError, "finishedAt": done,
	}}); err != nil {
		log.Printf("campaign %s: %v", id.Hex(), err)
	}
}

// AdminListCampaigns is what was sent, newest first.
func (h *Handler) AdminListCampaigns(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	cur, err := h.Store.DB.Collection("campaign").Find(r.Context(), bson.M{},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(50))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Campaign{}
	_ = cur.All(r.Context(), &rows)
	httpx.JSON(w, http.StatusOK, rows)
}

// campaignButtons is the pair every Telegram campaign carries.
//
// ⚠️ **They are the reason to send through the bot at all.** An SMS ends in an
// inbox and whatever the guest does next starts from nothing; a bot message can end
// in the menu, one tap away, and can ask what they thought without them typing an
// address anywhere.
//
// The mini app button is a `web_app` when the bot has one configured and an ordinary
// link otherwise — `telegram.SendCampaign` retries without it rather than losing the
// message to a rejected button.
func (h *Handler) campaignButtons(id primitive.ObjectID, lang string) []telegram.MessageButton {
	base := strings.TrimRight(h.Cfg.PublicBaseURL, "/")
	// The site carries language in the path, so a Russian guest's button opens the
	// Russian menu rather than the Uzbek one with a switch to find.
	if lang == "ru" || lang == "en" {
		base += "/" + lang
	}
	menu, feedback := "🍽 Menyu", "💬 Fikr bildirish"
	switch lang {
	case "ru":
		menu, feedback = "🍽 Меню", "💬 Оставить отзыв"
	case "en":
		menu, feedback = "🍽 Menu", "💬 Leave feedback"
	}
	return []telegram.MessageButton{
		{Label: menu, WebApp: base + "/menu"},
		// A callback rather than a link: the answer is a few words, and sending
		// somebody to a form to type them is how feedback stops arriving. The
		// campaign id rides along so a complaint can be traced to what prompted it.
		{Label: feedback, Callback: "fb:" + id.Hex()},
	}
}

// sanitizeCampaignImage keeps only images this site serves.
//
// ⚠️ Telegram fetches the photograph from a public URL, so an arbitrary value here
// would have Telegram's servers pull a stranger's file and attach it to a message
// sent in the restaurant's name.
func sanitizeCampaignImage(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	// ⚠️ The upload control returns an **absolute** URL, and this guard only
	// accepted a path — so every picture an owner attached was silently dropped and
	// the campaign went out as plain text. The upload was fine, the send was fine,
	// and the only sign was a photograph that never arrived.
	//
	// Reduced to a path rather than accepted as a URL: the path is what is stored,
	// and the host is added back at send time from PUBLIC_BASE_URL — which is the
	// domain that will still be right after the customer connects their own.
	if i := strings.Index(v, "/uploads/"); i >= 0 {
		v = v[i:]
	}
	if strings.HasPrefix(v, "/uploads/") && !strings.Contains(v, "..") {
		return v
	}
	return ""
}
