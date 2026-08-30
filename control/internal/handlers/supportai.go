package handlers

// ---- The assistant's answer, before a person's ----
//
// ⚠️ **It answers only from the help articles it was given, and says so when
// they do not cover the question.** This is the same seam the briefing is built
// on and for the same reason: a model that invents an answer about how the
// product behaves is worse than no answer at all. A restaurant that follows one
// wrong instruction — reprints a fiscal receipt, clears a stocktake, switches a
// printer's code page — has a real problem caused by us, and after that nothing
// this widget says will be believed again.
//
// So the articles travel with the question, the prompt forbids going beyond
// them, and a refusal is a first-class answer that leaves the thread in the
// operator's queue.
//
// ⚠️ **A machine answer never takes a human out of the loop.** The thread stays
// `waiting` and the operator still sees it, marked as already answered. The
// alternative — closing it, or moving it out of the queue — means a wrong
// answer is one nobody ever looks at, which is exactly the case that has to be
// caught.
//
// ⚠️ **The articles arrive from the panel, and that is deliberate.** They are
// our own bundled text and the browser already searched them, so duplicating
// the base in Go would give two copies that drift. Somebody editing the request
// could feed the model text of their own — and receive it back, in their own
// chat, having told themselves something. The blast radius is one person's own
// screen, which is why this is acceptable where a posted `plan: "enterprise"`
// would not be.

import (
	"encoding/json"
	"net/http"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ Byte-identical for every restaurant and every question — the prompt-caching
// rule the briefing already documents. Everything specific is in the user
// message.
const assistSystem = `You are the support assistant for Keel, a restaurant management
system used in Uzbekistan. A restaurant owner or their staff has asked a question in
the panel's help widget.

You are given the question and a small set of HELP ARTICLES from the product's own
documentation.

Rules, in order of importance:

1. ANSWER ONLY FROM THE ARTICLES. Everything you say about how Keel behaves must come
   from the articles you were given. You have no other knowledge of this product, and
   what you might guess about "restaurant software in general" is not it.

2. IF THE ARTICLES DO NOT ANSWER THE QUESTION, SAY SO. Set answered to false and leave
   the text empty. A human operator is already going to read this conversation; a
   plausible wrong answer is worse than nothing, because the person acts on it and
   nobody finds out until something is broken.

   This includes questions about their own data ("how much did I sell yesterday",
   "why is table 6 still open"), about money owed, about their contract, and about
   anything that would change a setting you cannot see. Those are for the operator.

3. Never invent a screen, a button, a menu path or a setting name. If the article does
   not name it, do not name it.

4. Write two to four sentences to a person standing in a working restaurant. Give the
   steps in the order they are done. No greeting, no sign-off, no "as an AI".

5. Do not promise anything on the company's behalf — no refunds, no timescales, no "we
   will fix it today". You do not know, and the operator does.

6. If the question suggests something is broken and the articles explain how it should
   work, say what it should do and what to check, then say an operator will look.

Write in the language named in the request. Uzbek means Latin-script Uzbek as spoken in
Tashkent, not Turkish and not Cyrillic.`

var assistSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"answered": map[string]any{"type": "boolean"},
		"text":     map[string]any{"type": "string"},
	},
	"required":             []string{"answered", "text"},
	"additionalProperties": false,
}

type assistRequest struct {
	ThreadID string `json:"threadId"`
	Question string `json:"question"`
	Lang     string `json:"lang"`
	Articles []struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	} `json:"articles"`
}

// How many articles the model is given.
//
// ⚠️ Capped rather than "whatever the panel sent". Every article is tokens paid
// for on every question, and past the first few the search's own ranking has
// stopped meaning anything — the fifth candidate for a question is rarely the
// answer to it.
const assistMaxArticles = 4

// SupportAssist writes the first answer, when it can.
func (h *Handler) SupportAssist(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !h.aiConfigured() {
		// Not an error: a platform with no key simply has no assistant, and the
		// operator answers exactly as before.
		httpx.JSON(w, http.StatusOK, map[string]any{"answered": false, "off": true})
		return
	}
	var req assistRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	question := clampSupport(req.Question)
	id, idErr := primitive.ObjectIDFromHex(req.ThreadID)
	if question == "" || idErr != nil {
		httpx.Error(w, http.StatusBadRequest, "question and threadId required")
		return
	}
	ctx := r.Context()

	// ⚠️ Scoped by slug: the thread id arrives from a customer's server, and
	// without this one restaurant could have the assistant write into another's
	// conversation.
	var thread models.SupportThread
	if h.Store.SupportThreads.FindOne(ctx,
		bson.M{"_id": id, "slug": t.Slug}).Decode(&thread) != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	// ⚠️ **Nothing is written once an operator is in the conversation.** A
	// machine sentence appearing under a person's reply reads as the operator
	// contradicting themselves, and the owner cannot tell which of the two to
	// follow.
	if thread.OperatorName != "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"answered": false})
		return
	}

	articles := req.Articles
	if len(articles) > assistMaxArticles {
		articles = articles[:assistMaxArticles]
	}
	if len(articles) == 0 {
		// Nothing to answer from, and rule 1 says that is a refusal. Asking
		// anyway would be paying a model to say "I do not know".
		httpx.JSON(w, http.StatusOK, map[string]any{"answered": false})
		return
	}

	blob, _ := json.Marshal(map[string]any{
		"language": languageName(req.Lang),
		"question": question,
		"articles": articles,
	})
	text, usage, err := h.engine().JSON(ctx, assistSystem, string(blob), assistSchema, "low")
	if err != nil {
		// ⚠️ Silent to the owner. The operator is coming either way, and "the
		// assistant is over quota" is our problem described in their chat.
		httpx.JSON(w, http.StatusOK, map[string]any{"answered": false})
		return
	}
	h.recordBriefing(ctx, t.Slug, usage)

	var parsed struct {
		Answered bool   `json:"answered"`
		Text     string `json:"text"`
	}
	if json.Unmarshal([]byte(text), &parsed) != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"answered": false})
		return
	}
	answer := clampSupport(parsed.Text)
	if !parsed.Answered || answer == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"answered": false})
		return
	}

	now := time.Now()
	msg := models.SupportMessage{
		ID: primitive.NewObjectID(), ThreadID: id,
		From: models.FromAssistant, Author: "Keel",
		Text: answer, At: now,
	}
	if _, err := h.Store.SupportMessages.InsertOne(ctx, msg); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ **The status is not touched.** The thread stays in the operator's
	// queue and `unreadForUs` is left alone: an assistant reply is a first
	// response, not a resolution, and the one case that must never happen is a
	// wrong answer nobody reads. Only `lastText` moves, so the queue row shows
	// what the restaurant last saw.
	_, _ = h.Store.SupportThreads.UpdateByID(ctx, id, bson.M{
		"$set": bson.M{
			"lastText": answer, "lastFrom": string(models.FromAssistant), "lastAt": now,
		},
		"$inc": bson.M{"unreadForOwner": 1},
	})
	hub.wake(t.Slug)
	httpx.JSON(w, http.StatusOK, map[string]any{"answered": true, "message": msg})
}
