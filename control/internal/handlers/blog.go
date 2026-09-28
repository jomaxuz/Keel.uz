package handlers

// ---- Writing, and being read ----
//
// ⚠️ **Three languages side by side, never one translated.** A post shaped into
// Russian by a machine reads like a machine to the person deciding whether to
// buy a till from us, and that is the one page where it matters. A language
// left empty is a language the post does not appear in — a card that opens onto
// somebody else's language is worse than a shorter list.
//
// ⚠️ **The body is stored as text and rendered by the site.** Storing HTML
// would let one article break the layout of every page around it, or the safety
// of the whole site, and would tie what we can write to what some template
// happened to allow. The renderer turns a lone YouTube link into a player and
// `![](…)` into a picture; everything else is paragraphs.

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"keel-control/internal/httpx"
	"keel-control/internal/models"
)

// ---- What the site reads ----

// blogCard is one post as a list shows it.
type blogCard struct {
	Slug        string     `json:"slug"`
	Cover       string     `json:"cover,omitempty"`
	Title       string     `json:"title"`
	Excerpt     string     `json:"excerpt"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	Views       int        `json:"views"`
}

// BlogList is the published posts in one language, newest first.
func (h *Handler) BlogList(w http.ResponseWriter, r *http.Request) {
	lang := blogLang(r.URL.Query().Get("lang"))
	cur, err := h.Store.Blog.Find(r.Context(),
		bson.M{"published": true},
		options.Find().SetSort(bson.D{{Key: "publishedAt", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var posts []models.BlogPost
	if err := cur.All(r.Context(), &posts); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := []blogCard{}
	for _, p := range posts {
		// ⚠️ A post with nothing written in this language is left out, not
		// shown in another one: a reader who follows a Russian card onto an
		// Uzbek page has been handed a language nobody offered them.
		if !p.Has(lang) {
			continue
		}
		t := p.Text(lang)
		out = append(out, blogCard{
			Slug: p.Slug, Cover: p.Cover, Title: t.Title, Excerpt: t.Excerpt,
			PublishedAt: p.PublishedAt, Views: p.Views,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"posts": out})
}

// blogBody is one post with its words, for the site's `/llms-full.txt`.
type blogBody struct {
	blogCard
	Body string `json:"body"`
}

// BlogFullList is every published post in one language, bodies included.
//
// ⚠️ **One answer for the whole list**, because the file it feeds holds every
// post: built from BlogRead it would be a request per post on a route crawlers
// fetch. Same rule as the list — a post with nothing in this language is left
// out rather than shown in another.
func (h *Handler) BlogFullList(w http.ResponseWriter, r *http.Request) {
	lang := blogLang(r.URL.Query().Get("lang"))
	cur, err := h.Store.Blog.Find(r.Context(),
		bson.M{"published": true},
		options.Find().SetSort(bson.D{{Key: "publishedAt", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var posts []models.BlogPost
	if err := cur.All(r.Context(), &posts); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := []blogBody{}
	for _, p := range posts {
		if !p.Has(lang) {
			continue
		}
		t := p.Text(lang)
		out = append(out, blogBody{
			blogCard: blogCard{
				Slug: p.Slug, Cover: p.Cover, Title: t.Title, Excerpt: t.Excerpt,
				PublishedAt: p.PublishedAt, Views: p.Views,
			},
			Body: t.Body,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"posts": out})
}

// blogFull is one post as its own page shows it.
type blogFull struct {
	blogCard
	Body string `json:"body"`
	// Which languages this post exists in, so the page can offer the switch
	// honestly rather than sending a reader to an empty page.
	Langs []string `json:"langs"`
}

// BlogPost is one published post, and counts the read.
func (h *Handler) BlogRead(w http.ResponseWriter, r *http.Request) {
	lang := blogLang(r.URL.Query().Get("lang"))
	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	var p models.BlogPost
	if err := h.Store.Blog.FindOne(r.Context(),
		bson.M{"slug": slug, "published": true}).Decode(&p); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	t := p.Text(lang)
	langs := []string{}
	for _, l := range []string{"uz", "ru", "en"} {
		if p.Has(l) {
			langs = append(langs, l)
		}
	}

	httpx.JSON(w, http.StatusOK, blogFull{
		blogCard: blogCard{
			Slug: p.Slug, Cover: p.Cover, Title: t.Title, Excerpt: t.Excerpt,
			PublishedAt: p.PublishedAt, Views: p.Views,
		},
		Body:  t.Body,
		Langs: langs,
	})
}

// BlogCountView records one reading.
//
// ⚠️ **Its own call, because reading the post is not reading the post.** The
// counter used to sit inside the read — and the page fetches itself twice on
// every render (once to build the title and description, once to draw the
// words), so a single visit counted two, and switching language counted three.
// The number was wrong from the first day and wrong in the flattering
// direction, which is the kind nobody questions.
//
// ⚠️ **Fired by the reader's browser, once per page.** That is also what keeps
// a crawler, a link preview and a sitemap fetch out of it: they take the words
// and never run the script. It costs the reader with JavaScript turned off,
// who is not counted at all — an honest undercount against a flattering
// overcount, and this figure only ever compares posts with each other.
func (h *Handler) BlogCountView(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	if slug == "" {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	// ⚠️ Published only. An unpublished draft is opened by whoever is writing
	// it, repeatedly, and a counter that learned to include that would report
	// the author's own afternoon as an audience.
	_, _ = h.Store.Blog.UpdateOne(r.Context(),
		bson.M{"slug": slug, "published": true},
		bson.M{"$inc": bson.M{"views": 1}})
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// BlogImage serves a picture out of the database.
func (h *Handler) BlogImage(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var img models.BlogImage
	if err := h.Store.BlogImages.FindOne(r.Context(), bson.M{"_id": id}).Decode(&img); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", img.Type)
	// ⚠️ Immutable: the id is the content. A picture never changes under its
	// own address, so a reader's browser never has to ask about it twice.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(img.Data)
}

// ---- What the console writes ----

// ConsoleBlogList is every post, draft included.
func (h *Handler) ConsoleBlogList(w http.ResponseWriter, r *http.Request) {
	cur, err := h.Store.Blog.Find(r.Context(), bson.M{},
		options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	posts := []models.BlogPost{}
	if err := cur.All(r.Context(), &posts); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"posts": posts})
}

// ConsoleBlogSave creates or updates one post.
func (h *Handler) ConsoleBlogSave(w http.ResponseWriter, r *http.Request) {
	var req models.BlogPost
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Slug = slugify(req.Slug, req.UZ.Title)
	if req.Slug == "" {
		httpx.Error(w, http.StatusBadRequest, "manzil (slug) bo'sh")
		return
	}
	// ⚠️ **Uzbek has to be there.** It is the language of the site and the
	// fallback every other one falls back to; a post without it is a post that
	// disappears for most of the people it was written for.
	if !req.UZ.Filled() {
		httpx.Error(w, http.StatusBadRequest, "o'zbekcha sarlavha va matn kerak")
		return
	}

	now := time.Now()
	req.UpdatedAt = now

	if req.ID.IsZero() {
		req.CreatedAt = now
		if req.Published {
			req.PublishedAt = &now
		}
		res, err := h.Store.Blog.InsertOne(r.Context(), req)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				httpx.Error(w, http.StatusConflict, "bu manzil band")
				return
			}
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		req.ID, _ = res.InsertedID.(primitive.ObjectID)
		// The AI files are rebuilt from the blog; look at them once the site
		// has picked the change up, and ping what moved (llms.go).
		h.NudgeLLMs()
		httpx.JSON(w, http.StatusOK, req)
		return
	}

	var old models.BlogPost
	if err := h.Store.Blog.FindOne(r.Context(), bson.M{"_id": req.ID}).Decode(&old); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	req.CreatedAt = old.CreatedAt
	// ⚠️ **The counter is never written from the form.** It is the one field
	// here that belongs to the readers rather than to the author, and a save
	// that carried a stale copy of it would quietly undo every read since the
	// editor was opened.
	req.Views = old.Views
	// ⚠️ **The date is set once.** A post edited in March is not a post from
	// March; a date that moved on every typo would make the whole list
	// untrustworthy, and the list is ordered by it.
	switch {
	case old.PublishedAt != nil:
		req.PublishedAt = old.PublishedAt
	case req.Published:
		req.PublishedAt = &now
	}
	if _, err := h.Store.Blog.ReplaceOne(r.Context(), bson.M{"_id": req.ID}, req); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			httpx.Error(w, http.StatusConflict, "bu manzil band")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.NudgeLLMs()
	httpx.JSON(w, http.StatusOK, req)
}

// ConsoleBlogDelete removes a post.
func (h *Handler) ConsoleBlogDelete(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.Store.Blog.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.NudgeLLMs()
	httpx.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// maxBlogImage is the largest picture the editor accepts.
//
// ⚠️ **Four megabytes, and the limit exists twice over.** A Mongo document
// cannot hold sixteen, and a photograph straight off a phone is easily eight —
// so without this the failure is a save that works for the author and a page
// that never loads for anybody else.
const maxBlogImage = 4 << 20

var blogImageTypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true,
}

// ConsoleBlogUpload stores one picture and answers with its address.
func (h *Handler) ConsoleBlogUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBlogImage+1024)
	if err := r.ParseMultipartForm(maxBlogImage); err != nil {
		httpx.Error(w, http.StatusBadRequest, "rasm juda katta")
		return
	}
	file, head, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "rasm tanlanmagan")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxBlogImage+1))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(data) > maxBlogImage {
		httpx.Error(w, http.StatusBadRequest, "rasm juda katta")
		return
	}
	// ⚠️ **The bytes decide the type, not the file name.** A browser sends
	// whatever the operating system guessed, and a `.jpg` that is really
	// something else would be served back with a content type we invented.
	kind := http.DetectContentType(data)
	if i := strings.IndexByte(kind, ';'); i > 0 {
		kind = kind[:i]
	}
	if !blogImageTypes[kind] {
		httpx.Error(w, http.StatusBadRequest, "faqat rasm yuklash mumkin")
		return
	}
	_ = head

	res, err := h.Store.BlogImages.InsertOne(r.Context(), models.BlogImage{
		Type: kind, Data: data, Size: len(data), CreatedAt: time.Now(),
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.InsertedID.(primitive.ObjectID)
	// ⚠️ **The address the site serves, not the one this handler answers on.**
	// `/internal` is reachable inside the docker network and nowhere else, so a
	// post whose pictures pointed here rendered every one of them as a broken
	// box for every reader — while looking perfectly correct in the editor. The
	// site proxies this route at /blog-image/{id}.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"url":  "/blog-image/" + id.Hex(),
		"size": len(data),
	})
}

// blogLang normalises the language asked for. Anything unknown is Uzbek, which
// is what the site is written in.
func blogLang(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ru":
		return "ru"
	case "en":
		return "en"
	}
	return "uz"
}

var slugDrop = regexp.MustCompile(`[^a-z0-9]+`)

// slugify turns a title into an address.
//
// ⚠️ **Latin letters and digits only, and the Uzbek title is only a starting
// point.** A slug carrying an apostrophe or a Cyrillic letter is a link that
// survives being pasted into some places and not others — and the place it
// breaks is a messaging app, which is where every one of these is shared.
func slugify(slug, fallback string) string {
	s := strings.ToLower(strings.TrimSpace(slug))
	if s == "" {
		s = strings.ToLower(strings.TrimSpace(fallback))
	}
	s = strings.NewReplacer("ʻ", "", "'", "", "’", "").Replace(s)
	s = slugDrop.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 80 {
		s = strings.Trim(s[:80], "-")
	}
	return s
}
