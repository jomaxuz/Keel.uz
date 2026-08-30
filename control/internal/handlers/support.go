package handlers

// ---- Support: the queue, from both ends ----
//
// ⚠️ **Two audiences, one store, and deliberately two handler sets.** A
// restaurant may only ever see its own thread and may never see who else is
// waiting; an operator sees every thread and the customer record behind it.
// Those are different enough that sharing a handler and branching on the caller
// is how the branch eventually gets it wrong — so the tenant-facing half below
// scopes every query by slug from the credential, never from the request.
//
// ⚠️ **The restaurant's browser never reaches this service.** The panel talks
// to its own server, which forwards with the per-tenant token it already holds
// for the briefing and the domain link. Same boundary as everything else here:
// a page served from a customer's domain does not carry a platform credential.

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ---- Waking the other side ----
//
// ⚠️ **A long poll with a nudge, not a poll loop and not a second socket.**
// The panel holds a WebSocket to its *own* server — same origin, no CORS, no
// platform credential in a browser. That server holds one request open here
// until something happens. The alternative, a WebSocket proxied through two
// services and a reverse proxy, buys nothing over this and adds a connection
// that has to be kept alive through both.
//
// The nudge is what makes it push rather than polling: a reply wakes every
// waiter on that thread's slug immediately, and the timeout below is only the
// ceiling on how long an idle connection sits there.
const supportWait = 25 * time.Second

type supportHub struct {
	mu      sync.Mutex
	waiting map[string][]chan struct{}
}

var hub = supportHub{waiting: map[string][]chan struct{}{}}

// listen returns a channel closed the next time this slug has news.
func (s *supportHub) listen(slug string) (<-chan struct{}, func()) {
	ch := make(chan struct{})
	s.mu.Lock()
	s.waiting[slug] = append(s.waiting[slug], ch)
	s.mu.Unlock()
	return ch, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		rest := s.waiting[slug][:0]
		for _, c := range s.waiting[slug] {
			if c != ch {
				rest = append(rest, c)
			}
		}
		if len(rest) == 0 {
			delete(s.waiting, slug)
		} else {
			s.waiting[slug] = rest
		}
	}
}

// wake releases everyone waiting on a slug.
//
// ⚠️ Closing the channel rather than sending on it: a send needs a reader that
// is still there, and the one case that matters is the reader whose request was
// cancelled a moment ago. A close reaches all of them and cannot block.
func (s *supportHub) wake(slug string) {
	s.mu.Lock()
	chans := s.waiting[slug]
	delete(s.waiting, slug)
	s.mu.Unlock()
	for _, c := range chans {
		close(c)
	}
}

// ---- The restaurant's side ----

type supportAskRequest struct {
	// Empty starts a new thread. ⚠️ The panel decides, not this handler: an
	// owner asking a second, unrelated question wants a second thread, and only
	// their screen knows whether they pressed "new question" or typed into an
	// open one.
	ThreadID string `json:"threadId"`
	Text     string `json:"text"`
	AskedBy  string `json:"askedBy"`
	Role     string `json:"role"`
}

// SupportAsk takes one line from a restaurant.
func (h *Handler) SupportAsk(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req supportAskRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	text := clampSupport(req.Text)
	if text == "" {
		httpx.Error(w, http.StatusBadRequest, "text required")
		return
	}
	ctx := r.Context()
	now := time.Now()

	var thread models.SupportThread
	if id, err := primitive.ObjectIDFromHex(req.ThreadID); err == nil {
		// ⚠️ Scoped by slug as well as by id. A thread id is a guessable-ish
		// handle arriving from a customer's server, and without the slug one
		// restaurant could post into another's conversation.
		err = h.Store.SupportThreads.FindOne(ctx,
			bson.M{"_id": id, "slug": t.Slug}).Decode(&thread)
		if err != nil {
			httpx.Error(w, http.StatusNotFound, "not found")
			return
		}
	}

	if thread.ID.IsZero() {
		thread = models.SupportThread{
			ID: primitive.NewObjectID(), Slug: t.Slug,
			Restaurant: t.Name,
			// ⚠️ The subject is the first line, cut. Asking for one separately
			// is a field people leave empty or fill with "problem".
			Subject:   summarise(text),
			Status:    models.SupportWaiting,
			AskedBy:   clampName(req.AskedBy),
			AskedRole: clampName(req.Role),
			CreatedAt: now,
		}
		if _, err := h.Store.SupportThreads.InsertOne(ctx, thread); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	msg := models.SupportMessage{
		ID: primitive.NewObjectID(), ThreadID: thread.ID,
		From: models.FromOwner, Author: clampName(req.AskedBy),
		Text: text, At: now,
	}
	if _, err := h.Store.SupportMessages.InsertOne(ctx, msg); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ **A closed thread reopens when the restaurant writes again.** The
	// alternative — a new thread per follow-up — splits one problem across two
	// records, and the operator who closed it never learns the answer did not
	// work.
	_, _ = h.Store.SupportThreads.UpdateByID(ctx, thread.ID, bson.M{
		"$set": bson.M{
			"status": models.SupportWaiting, "lastText": text,
			"lastFrom": string(models.FromOwner), "lastAt": now,
			"unreadForOwner": 0,
		},
		"$inc":   bson.M{"unreadForUs": 1},
		"$unset": bson.M{"closedAt": ""},
	})
	hub.wake(t.Slug)
	httpx.JSON(w, http.StatusOK, map[string]any{"threadId": thread.ID.Hex(), "message": msg})
}

// SupportThreads is this restaurant's own history.
func (h *Handler) SupportThreads(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	list, err := h.supportList(r.Context(), bson.M{"slug": t.Slug}, 30)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"threads": list})
}

// SupportRead is one thread's lines, for the restaurant that owns it.
func (h *Handler) SupportRead(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := primitive.ObjectIDFromHex(r.URL.Query().Get("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id required")
		return
	}
	var thread models.SupportThread
	if h.Store.SupportThreads.FindOne(r.Context(),
		bson.M{"_id": id, "slug": t.Slug}).Decode(&thread) != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	msgs, err := h.supportMessages(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Reading is what clears the owner's badge — not receiving.
	_, _ = h.Store.SupportThreads.UpdateByID(r.Context(), id,
		bson.M{"$set": bson.M{"unreadForOwner": 0}})
	httpx.JSON(w, http.StatusOK, map[string]any{"thread": thread, "messages": msgs})
}

// SupportWait holds a request open until this restaurant has news.
func (h *Handler) SupportWait(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	since := time.Time{}
	if v := r.URL.Query().Get("since"); v != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, v); err == nil {
			since = parsed
		}
	}
	// ⚠️ **Checked before waiting, always.** A caller that reconnects after a
	// dropped connection has a `since` in the past and news already stored; if
	// the answer only ever came from the nudge, that reply would sit unread
	// until the next one arrived.
	if fresh, err := h.supportSince(r.Context(), t.Slug, since); err == nil && len(fresh) > 0 {
		httpx.JSON(w, http.StatusOK, map[string]any{"messages": fresh, "now": time.Now()})
		return
	}
	ch, stop := hub.listen(t.Slug)
	defer stop()
	select {
	case <-ch:
		fresh, _ := h.supportSince(r.Context(), t.Slug, since)
		httpx.JSON(w, http.StatusOK, map[string]any{"messages": fresh, "now": time.Now()})
	case <-time.After(supportWait):
		// ⚠️ An empty answer rather than a 204: the caller reconnects either
		// way, and a body carrying the server's clock is what keeps `since`
		// from drifting against a machine whose own clock is wrong.
		httpx.JSON(w, http.StatusOK, map[string]any{"messages": []any{}, "now": time.Now()})
	case <-r.Context().Done():
	}
}

// supportSince is everything said to this restaurant after a moment.
func (h *Handler) supportSince(
	ctx context.Context, slug string, since time.Time,
) ([]models.SupportMessage, error) {
	ids, err := h.Store.SupportThreads.Distinct(ctx, "_id", bson.M{"slug": slug})
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	filter := bson.M{"threadId": bson.M{"$in": ids}}
	if !since.IsZero() {
		filter["at"] = bson.M{"$gt": since}
	}
	cur, err := h.Store.SupportMessages.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: 1}}).SetLimit(200))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []models.SupportMessage{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ---- Shared reads ----

func (h *Handler) supportList(
	ctx context.Context, filter bson.M, limit int64,
) ([]models.SupportThread, error) {
	cur, err := h.Store.SupportThreads.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "lastAt", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	// ⚠️ Never nil: Go marshals a nil slice as `null` and the console maps over
	// this. The trap this codebase has paid for twice.
	out := []models.SupportThread{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (h *Handler) supportMessages(
	ctx context.Context, id primitive.ObjectID,
) ([]models.SupportMessage, error) {
	cur, err := h.Store.SupportMessages.Find(ctx, bson.M{"threadId": id},
		options.Find().SetSort(bson.D{{Key: "at", Value: 1}}).SetLimit(500))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []models.SupportMessage{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ---- Text ----

func clampSupport(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > models.SupportMaxText {
		s = s[:models.SupportMaxText]
	}
	return s
}

func clampName(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

// summarise is the thread's title, taken from its first line.
//
// ⚠️ **Cut on a word and on the first sentence end, not at 60 characters.** A
// queue of titles chopped mid-word is a queue an operator reads twice, and the
// first sentence of a support message is almost always the question.
func summarise(text string) string {
	line := strings.TrimSpace(strings.SplitN(text, "\n", 2)[0])
	for _, end := range []string{". ", "? ", "! "} {
		if i := strings.Index(line, end); i > 12 {
			return line[:i+1]
		}
	}
	if len([]rune(line)) <= 70 {
		return line
	}
	runes := []rune(line)[:70]
	if i := strings.LastIndex(string(runes), " "); i > 30 {
		return string(runes[:i]) + "…"
	}
	return string(runes) + "…"
}

// ---- The operator's side ----

// ConsoleSupportList is the queue, with the filters an operator works it by.
func (h *Handler) ConsoleSupportList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := bson.M{}
	// ⚠️ **"waiting" is the default and not "everything".** A support screen
	// that opens on every thread ever received opens on history; the queue is
	// the job.
	switch status := q.Get("status"); status {
	case "", "waiting":
		filter["status"] = models.SupportWaiting
	case "all":
	default:
		filter["status"] = status
	}
	if slug := strings.TrimSpace(q.Get("slug")); slug != "" {
		filter["slug"] = slug
	}
	if term := strings.TrimSpace(q.Get("q")); term != "" {
		filter["$or"] = supportSearch(term)
	}
	list, err := h.supportList(r.Context(), filter, 200)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	waiting, _ := h.Store.SupportThreads.CountDocuments(r.Context(),
		bson.M{"status": models.SupportWaiting})
	httpx.JSON(w, http.StatusOK, map[string]any{"threads": list, "waiting": waiting})
}

// supportSearch is the operator's search box.
//
// ⚠️ **A prefix regex, not the text index — and the text index was the bug.**
// Mongo's text search matches whole tokens against a stemmer that has no Uzbek
// in it. Uzbek is agglutinative: the operator types "printer" and the message
// says "printerdan", "printerga", "printerni". Those are four different tokens
// and none of them matches, so the search box returns nothing for a thread that
// is sitting in the list two rows below — the worst kind of failure, because it
// looks like the thread does not exist rather than like a broken search.
//
// A regex anchored at a word start finds all four. It is a collection scan, and
// that is an acceptable trade here in a way it would not be on orders: support
// threads are counted in thousands for the whole platform, the status filter is
// almost always applied alongside, and a search that is fast and wrong is not a
// search.
//
// ⚠️ The term is quoted before it becomes a pattern. An operator pasting a
// customer's error message containing `(` or `*` would otherwise get an invalid
// regex — or, worse, a valid one that means something else.
func supportSearch(term string) []bson.M {
	pattern := `(^|\W)` + regexp.QuoteMeta(term)
	rx := primitive.Regex{Pattern: pattern, Options: "i"}
	return []bson.M{
		{"restaurant": rx}, {"slug": rx}, {"subject": rx}, {"lastText": rx},
	}
}

// ConsoleSupportThread is one conversation, with the customer behind it.
//
// ⚠️ **The restaurant's record comes with it, in the same response.** An
// operator who has to open a second screen to find out whether the customer
// pays, what plan they are on and whether their container is up will answer
// without looking — and the answer to half of all support questions is on that
// record.
func (h *Handler) ConsoleSupportThread(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad id")
		return
	}
	ctx := r.Context()
	var thread models.SupportThread
	if h.Store.SupportThreads.FindOne(ctx, bson.M{"_id": id}).Decode(&thread) != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	msgs, err := h.supportMessages(ctx, id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = h.Store.SupportThreads.UpdateByID(ctx, id,
		bson.M{"$set": bson.M{"unreadForUs": 0}})

	var tenant models.Tenant
	_ = h.Store.Tenants.FindOne(ctx, bson.M{"slug": thread.Slug}).Decode(&tenant)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"thread": thread, "messages": msgs, "tenant": supportTenantCard(&tenant),
	})
}

// supportTenantCard is what an operator needs to know about who is asking.
//
// ⚠️ A named subset, not the tenant document. That record carries provisioning
// details and internal notes; this screen is for answering a question.
func supportTenantCard(t *models.Tenant) map[string]any {
	if t == nil || t.Slug == "" {
		return nil
	}
	return map[string]any{
		"slug": t.Slug, "name": t.Name, "kind": t.Kind,
		"owner": t.OwnerName, "phone": t.OwnerPhone,
		"domains": t.Domains, "status": t.Status,
		// ⚠️ Whether the container is actually up, because "the panel is
		// blank" and "the panel is down" are the same sentence from a customer
		// and completely different answers from us.
		"container": t.ContainerStatus,
		"free":      t.Free, "till": t.Till.Plan,
		"createdAt": t.CreatedAt, "subscribedAt": t.SubscribedAt,
	}
}

type supportReplyRequest struct {
	Text string `json:"text"`
	// Closing with the reply, which is what an operator actually does: the
	// answer and "that's sorted" are one action, and a separate button means
	// half the queue stays open forever.
	Close bool `json:"close"`
}

// ConsoleSupportReply is an operator answering.
func (h *Handler) ConsoleSupportReply(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad id")
		return
	}
	var req supportReplyRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	text := clampSupport(req.Text)
	if text == "" && !req.Close {
		httpx.Error(w, http.StatusBadRequest, "text required")
		return
	}
	ctx := r.Context()
	var thread models.SupportThread
	if h.Store.SupportThreads.FindOne(ctx, bson.M{"_id": id}).Decode(&thread) != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	// ⚠️ Read from the database rather than the token, the way every other
	// console handler does: a deactivated operator must stop being able to
	// answer customers on the next request, not when their token expires.
	user, _ := h.actor(r)
	now := time.Now()

	set := bson.M{"status": models.SupportOpen, "unreadForUs": 0}
	if user != nil {
		set["operatorId"] = user.ID
		set["operatorName"] = user.Name
	}
	if text != "" {
		msg := models.SupportMessage{
			ID: primitive.NewObjectID(), ThreadID: id,
			From: models.FromOperator, Author: operatorName(user),
			Text: text, At: now,
		}
		if _, err := h.Store.SupportMessages.InsertOne(ctx, msg); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		set["lastText"] = text
		set["lastFrom"] = string(models.FromOperator)
		set["lastAt"] = now
	}
	update := bson.M{"$set": set}
	if text != "" {
		update["$inc"] = bson.M{"unreadForOwner": 1}
	}
	if req.Close {
		set["status"] = models.SupportClosed
		set["closedAt"] = now
	}
	if _, err := h.Store.SupportThreads.UpdateByID(ctx, id, update); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// The restaurant's panel is holding a request open on this slug.
	hub.wake(thread.Slug)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func operatorName(u *models.User) string {
	if u == nil || strings.TrimSpace(u.Name) == "" {
		return "Keel"
	}
	return u.Name
}

var _ = mongo.ErrNoDocuments
