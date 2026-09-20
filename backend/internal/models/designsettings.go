package models

import (
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Typed settings on a section, and blocks inside it — the shape Shopify's theme
// editor is built on, and the reason its editor stays consistent as sections are
// added.
//
// ⚠️ **The point is that the editor is generated, not written.** Until now every
// element type needed its own hand-written panel in the console, so adding one
// meant editing the editor — which is why ours drifted into a form that describes
// a page rather than a tool for building one. A section declares what it can be
// asked (`design_schema.json`, served to the console) and the console draws the
// controls from that declaration.
//
// The guard here is deliberately about **shape**, not about each key:
//
//   - The renderer only ever reads keys it knows, so an unrecognised key is inert
//     — it cannot reach a page, and dropping it would break a design saved by a
//     newer console against an older tenant.
//   - What *can* hurt is a value of the wrong kind. Only strings, numbers, bools,
//     three-language text and **lists of ids** survive; everything else is
//     dropped. See idList for why an array is never anything but ids.
//   - Values that become URLs or image sources go through the same allowlists the
//     canvas elements use. ⚠️ Chosen by key **suffix** (`…Image`, `…Link`), which
//     is a convention — so it is stated here and mirrored in the schema, and the
//     test covers both.

// DesignBlock is one repeatable item inside a section: a slide, a perk, a link.
type DesignBlock struct {
	Type     string         `bson:"type" json:"type"`
	Settings map[string]any `bson:"settings,omitempty" json:"settings,omitempty"`
	Hidden   bool           `bson:"hidden,omitempty" json:"hidden,omitempty"`
}

const (
	maxSettingKeys   = 40
	maxSettingString = 2000
	maxBlocksPerBand = 24
)

// sanitizeSettings keeps only values a section can be asked for.
func sanitizeSettings(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, raw := range in {
		if !settingKeyOK(key) || len(out) >= maxSettingKeys {
			continue
		}
		switch v := raw.(type) {
		case bool:
			out[key] = v
		case string:
			out[key] = sanitizeSettingString(key, v)
		case float64:
			out[key] = clampSettingNumber(v)
		case int32:
			out[key] = clampSettingNumber(float64(v))
		case int64:
			out[key] = clampSettingNumber(float64(v))
		case int:
			out[key] = clampSettingNumber(float64(v))
		case map[string]any:
			// Three-language text, and only that: a nested object of anything else
			// is a document nobody's editor produced.
			if t, ok := localizedFrom(v); ok {
				out[key] = t
			}
		case LocalizedText:
			out[key] = v
		case []any:
			out[key] = idList(v)
		case []string:
			anys := make([]any, 0, len(v))
			for _, s := range v {
				anys = append(anys, s)
			}
			out[key] = idList(anys)
		case primitive.A:
			out[key] = idList(v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func settingKeyOK(key string) bool {
	if key == "" || len(key) > 32 {
		return false
	}
	for i, c := range key {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9' && i > 0:
		case c == '_' && i > 0:
		default:
			return false
		}
	}
	return true
}

// sanitizeSettingString applies the allowlists a value of that kind needs.
func sanitizeSettingString(key, v string) string {
	switch {
	case strings.HasSuffix(key, "Image"), key == "image":
		return sanitizeImagePath(v)
	case strings.HasSuffix(key, "Link"), key == "link":
		// ⚠️ **The scheme, not the destination** — the same change the canvas
		// buttons and the navigation bar made, for the same reason. A hero's
		// button on a shop's site goes to the section that shop is divided into
		// ("/menu?cat=ayollar") or to its Telegram channel, and no list we write
		// can anticipate those. `elementLinks` is still the console's picker;
		// it stopped being the wall.
		return sanitizeHref(v)
	case strings.HasSuffix(key, "Tone"), key == "tone", key == "background":
		if designTones[v] {
			return v
		}
		return ""
	case strings.HasSuffix(key, "Color"), key == "color":
		if elementColors[v] {
			return v
		}
		return ""
	}
	if len(v) > maxSettingString {
		return v[:maxSettingString]
	}
	return v
}

// idList keeps a setting's array of document ids.
//
// ⚠️ **An array setting is an id list and nothing else**, which is narrower than
// it looks and deliberately so. The only thing a band ever needs a list of is
// *which of the restaurant's own records to show* — which categories a menu grid
// draws from — and every id is a 24-character hex string. Anything else in the
// array is dropped rather than passed through: a free-form list of strings would
// be the one place in this file where arbitrary text reaches a page in bulk, and
// no schema declares one.
//
// A value that is not an id is dropped on its own rather than voiding the list.
// Half a selection is a visible, fixable mistake; an emptied selection reads as
// "all categories", which is a different page and looks intentional.
func idList(in []any) []string {
	out := make([]string, 0, len(in))
	for _, raw := range in {
		s, ok := raw.(string)
		if !ok || !isHexID(s) {
			continue
		}
		out = append(out, s)
		// More than this is not a selection — a restaurant with 24 categories has
		// a menu nobody scrolls, and "all" is already the empty list.
		if len(out) >= 24 {
			break
		}
	}
	return out
}

func isHexID(s string) bool {
	if len(s) != 24 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

func clampSettingNumber(v float64) int {
	// Every numeric setting in the schema is a small integer — a count, a percent,
	// a step. Nothing needs a float, and a float is how a range control ends up
	// writing 33.333333 into a document a person has to read.
	n := int(v)
	if n < -1000 {
		return -1000
	}
	if n > 1000 {
		return 1000
	}
	return n
}

func localizedFrom(v map[string]any) (LocalizedText, bool) {
	t := LocalizedText{}
	found := false
	for _, lang := range []string{"uz", "ru", "en"} {
		s, ok := v[lang].(string)
		if !ok {
			continue
		}
		found = true
		if len(s) > maxSettingString {
			s = s[:maxSettingString]
		}
		switch lang {
		case "uz":
			t.Uz = s
		case "ru":
			t.Ru = s
		case "en":
			t.En = s
		}
	}
	return t, found
}

// sanitizeBlocks cleans the repeatable items inside a section.
func sanitizeBlocks(in []DesignBlock) []DesignBlock {
	if len(in) == 0 {
		return nil
	}
	out := make([]DesignBlock, 0, len(in))
	for _, b := range in {
		if !settingKeyOK(b.Type) {
			continue
		}
		b.Settings = sanitizeSettings(b.Settings)
		out = append(out, b)
		if len(out) >= maxBlocksPerBand {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
