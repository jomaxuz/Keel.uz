package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- The blog ----
//
// ⚠️ **Three languages side by side, not one with translations bolted on.** A
// post written in Uzbek and machine-shaped into Russian reads like a machine to
// the person deciding whether to buy a till from us, which is the one page
// where that matters. So each language is its own title, its own summary and
// its own body — and a language left empty is a language the post simply does
// not appear in, rather than a page of somebody else's words.
//
// ⚠️ **The body is text, and the renderer decides what it means.** Storing HTML
// would make every post a thing that can break the site's layout — or worse,
// the site's safety — and would tie what we can write today to what a template
// happened to allow. See the renderer on the site: a line that is a YouTube
// link becomes a player, `![](…)` becomes a picture, and everything else is
// paragraphs.

// BlogText is one language of one post.
type BlogText struct {
	Title string `bson:"title" json:"title"`
	// One or two sentences for the card and for the search result. ⚠️ Written,
	// never cut from the body: the first paragraph of an article is an
	// introduction, and an introduction truncated at 160 characters is the
	// worst possible summary of anything.
	Excerpt string `bson:"excerpt" json:"excerpt"`
	Body    string `bson:"body" json:"body"`
}

// Filled reports whether this language has enough to publish.
//
// ⚠️ **A title is not enough.** A card with a heading and no words behind it is
// a link the reader follows once, and the post that taught them that is the
// last one of ours they open.
func (t BlogText) Filled() bool { return t.Title != "" && t.Body != "" }

// BlogPost is one article.
type BlogPost struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	// The address it lives at. ⚠️ **Stable once published**, because it is what
	// somebody shared: an edited slug is a broken link in every message that
	// ever carried it, and nothing on our side would notice.
	Slug string `bson:"slug" json:"slug"`
	// The picture at the top and on the card.
	Cover string `bson:"cover,omitempty" json:"cover,omitempty"`

	UZ BlogText `bson:"uz" json:"uz"`
	RU BlogText `bson:"ru" json:"ru"`
	EN BlogText `bson:"en" json:"en"`

	// ⚠️ **Draft is the default, and that is not caution for its own sake.** A
	// post is written over several sittings; one that went live on the first
	// save would put a half-written paragraph on the front of the company.
	Published bool `bson:"published" json:"published"`
	// When it went out. ⚠️ Set the first time it is published and kept
	// afterwards: a post edited in March is not a post from March, and a date
	// that moves on every typo makes the whole list untrustworthy.
	PublishedAt *time.Time `bson:"publishedAt,omitempty" json:"publishedAt,omitempty"`

	// How many times a post's page was opened.
	//
	// ⚠️ **Page opens, not readers**, and the difference is worth keeping in
	// mind before anybody quotes it: one person refreshing four times is four.
	// It is a relative figure — which post interested people — and never a
	// number to put in front of an advertiser.
	Views int `bson:"views" json:"views"`

	Author    string    `bson:"author,omitempty" json:"author,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// Text returns the language asked for, falling back the way the rest of the
// platform does.
//
// ⚠️ **Uzbek is the fallback and English is not.** Every post is written in
// Uzbek first because that is who reads keel.uz; a Russian reader shown an
// English page has been handed a third language nobody chose for them.
func (p BlogPost) Text(lang string) BlogText {
	switch lang {
	case "ru":
		if p.RU.Filled() {
			return p.RU
		}
	case "en":
		if p.EN.Filled() {
			return p.EN
		}
	}
	return p.UZ
}

// Has reports whether this post exists in a language at all.
func (p BlogPost) Has(lang string) bool {
	switch lang {
	case "ru":
		return p.RU.Filled()
	case "en":
		return p.EN.Filled()
	}
	return p.UZ.Filled()
}

// BlogImage is a picture pasted into a post.
//
// ⚠️ **Kept in the database rather than on a disk.** The control plane has no
// upload volume and no CDN, and inventing one for the blog would mean a mount
// that has to survive every redeploy and a backup that has to know about it.
// An article's pictures are a few hundred kilobytes; Mongo holds them, they are
// backed up with everything else, and there is nothing to forget.
type BlogImage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type      string             `bson:"type" json:"type"`
	Data      []byte             `bson:"data" json:"-"`
	Size      int                `bson:"size" json:"size"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}
