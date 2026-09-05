package handlers

// ---- One model, many things on the shelf ----
//
// ⚠️ **A clothes shop does not sell "a shirt", it sells M/black.** Each size
// and colour has its own barcode, its own count and its own delivery, which is
// to say each is a product in every sense the stockroom already understands. So
// a variant *is* a menu item, tied to a model row by `variantOf` — and nothing
// in the stockroom, the scanner or the receipt had to learn a new idea.
//
// ⚠️ **What this screen removes is the typing, not the rows.** A hundred models
// in five sizes and three colours is fifteen hundred products either way; the
// question is whether a shop enters them by hand over a fortnight or presses a
// button. Every till sold here has some version of this, and its absence is the
// single thing that would send a clothes shop to BILLZ on the first demo.
//
// ⚠️ **Generating never deletes.** A variant that has gone from the axes may
// still be on a shelf, in a delivery and on last month's receipts. The sweep
// that would "tidy" it is the sweep that silently drops a stock row and its
// history — so a variant no longer named is simply left alone, and the shop
// removes it deliberately if it means to.

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// errHTTP is a message meant for the person at the screen. ⚠️ Uzbek, because
// the server translates on the way out — see internal/i18n.
func errHTTP(msg string) error { return errors.New(msg) }

type variantAxis struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type variantRequest struct {
	Axes []variantAxis `json:"axes"`
}

// AdminGenerateVariants fills out a model's matrix.
func (h *Handler) AdminGenerateVariants(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req variantRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	axes, combos, err := expandAxes(req.Axes)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	var model models.MenuItem
	if err := h.Store.Menu.FindOne(r.Context(), bson.M{"_id": id}).Decode(&model); err != nil {
		httpx.Error(w, http.StatusNotFound, "taom topilmadi")
		return
	}
	// ⚠️ **A variant of a variant is a shape nothing downstream can draw.** The
	// till groups one level, the panel lists one level, and the second level
	// would be invisible on both while quietly holding stock.
	if !model.VariantOf.IsZero() {
		httpx.Error(w, http.StatusBadRequest, "variantning varianti bo'lmaydi")
		return
	}

	// ⚠️ **The model stops being a thing on the shelf the moment it has one.**
	// Its own barcode is cleared: a scan has to land on a size, and a code left
	// on the model would ring up a shirt with no size on the receipt — and
	// would take the barcode away from the variant that should have carried it.
	set := bson.M{
		"variantAxes": axes,
		"barcode":     "",
		"updatedAt":   time.Now(),
	}
	if _, err := h.Store.Menu.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// What is already there, so pressing the button twice is not a way to end
	// up with two of everything.
	cur, err := h.Store.Menu.Find(r.Context(), bson.M{"variantOf": id})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var have []models.MenuItem
	if err := cur.All(r.Context(), &have); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	seen := map[string]bool{}
	for _, v := range have {
		seen[strings.Join(v.Variant, "\x00")] = true
	}

	made := 0
	for _, combo := range combos {
		if seen[strings.Join(combo, "\x00")] {
			continue
		}
		child := model
		child.ID = primitiveNil
		child.VariantOf = id
		child.Variant = combo
		child.VariantAxes = nil
		// ⚠️ Cleared, never copied: a barcode is unique within a brand, and a
		// copy would be refused by the index — after the first variant, with
		// the rest of the matrix half written.
		child.Barcode = ""
		child.StockID = primitiveNil
		// ⚠️ **Every variant sells itself.** It is the object that was bought:
		// the shop orders twelve M/black and twelve of them arrive. Without
		// this the server would not keep a stock row behind it and the shelf
		// count would stay at zero however many deliveries were entered.
		child.SellsItself = true
		if err := h.syncProductStock(r.Context(), &child); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := h.Store.Menu.InsertOne(r.Context(), child); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				continue
			}
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		made++
	}

	h.logAction(r, ActMenuUpdate, "menu", id.Hex(), model.Name, "")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"created": made, "total": len(combos),
	})
}

// expandAxes turns the axes into every combination, in axis order.
//
// ⚠️ **Bounded, because this is a form that multiplies.** Four axes of ten
// values is ten thousand products from one press — typed in by somebody trying
// things out, and undone one row at a time. The limit is far past any real
// shop's matrix and far short of an accident.
func expandAxes(in []variantAxis) (names []string, combos [][]string, err error) {
	const maxCombos = 500
	for _, a := range in {
		name := strings.TrimSpace(a.Name)
		vals := []string{}
		for _, v := range a.Values {
			v = strings.TrimSpace(v)
			if v != "" {
				vals = append(vals, v)
			}
		}
		// ⚠️ A half-filled axis is skipped rather than refused only when it is
		// *entirely* empty — the row a "add axis" button just made. An axis
		// with values and no name is a question with answers and no question,
		// and saving it silently is the defect `optionProblems` exists for.
		if name == "" && len(vals) == 0 {
			continue
		}
		if name == "" {
			return nil, nil, errHTTP("o'lcham yoki rang — nomsiz variant bo'lmaydi")
		}
		if len(vals) == 0 {
			return nil, nil, errHTTP(name + ": qiymatlar kiritilmagan")
		}
		names = append(names, name)
		if len(combos) == 0 {
			for _, v := range vals {
				combos = append(combos, []string{v})
			}
			continue
		}
		next := make([][]string, 0, len(combos)*len(vals))
		for _, c := range combos {
			for _, v := range vals {
				row := make([]string, len(c), len(c)+1)
				copy(row, c)
				next = append(next, append(row, v))
			}
		}
		combos = next
		if len(combos) > maxCombos {
			return nil, nil, errHTTP("variantlar juda ko'p — kamroq qiymat tanlang")
		}
	}
	if len(names) == 0 {
		return nil, nil, errHTTP("variant o'lchovi kiritilmagan")
	}
	return names, combos, nil
}
