package handlers

// ---- Scanning marked goods when they arrive ----
//
// ⚠️ **The whole feature is about *where* a refusal lands.** A marked bottle
// that was bought outside the system, or whose code has already been withdrawn,
// is refused by the tax register at the moment of sale — in front of a customer,
// with the cashier's only remedy being to scan it again, which cannot help.
// Scanned at goods-in, the same fact turns up in the store room with the box
// still open and the supplier's number still on the invoice.
//
// ⚠️ **Opt-in per branch** (`Branch.MarkingInbound`). Every restaurant running
// today scans only at the till, and a check that began refusing codes nobody had
// ever received would refuse every sale in the product on the day it shipped.
//
// ⚠️ **This is not an Asl Belgisi API and does not pretend to be.** Whether a
// code was ever issued is the national system's answer, given through the
// fiscal receipt (docs/markirovka.md). What is held here is a fact about this
// shop: these are the bottles we took in.

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/marking"
	"restaurant-backend/internal/models"
)

// maxCodesPerScan caps one save from the receiving screen.
//
// ⚠️ **A pallet is a real delivery**, and a request carrying ten thousand codes
// is one somebody will send by holding the trigger down. The refusal names the
// number so the box can be finished in two saves.
const maxCodesPerScan = 500

type receiveMarksRequest struct {
	MenuItemID string   `json:"menuItemId"`
	PurchaseID string   `json:"purchaseId"`
	Codes      []string `json:"codes"`
}

// AdminReceiveMarks records the codes somebody scanned off a delivery.
//
// ⚠️ **Every code is answered separately.** A box of forty with one unreadable
// sticker must not fail as a box of forty: the thirty-nine are in the store room
// either way, and an all-or-nothing save is one that gets abandoned halfway
// through unpacking.
func (h *Handler) AdminReceiveMarks(w http.ResponseWriter, r *http.Request) {
	_, branchID, _, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req receiveMarksRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Codes) == 0 {
		httpx.Error(w, http.StatusBadRequest, "hech narsa skanerlanmadi")
		return
	}
	if len(req.Codes) > maxCodesPerScan {
		httpx.Error(w, http.StatusBadRequest, "bir marta 500 tagacha kod saqlanadi")
		return
	}
	itemID, _ := objectID(req.MenuItemID)
	purchaseID, _ := objectID(req.PurchaseID)

	added, dup, bad, err := h.saveMarks(
		r.Context(), branchID, itemID, purchaseID, req.Codes, h.adminName(r))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.logAction(r, "marking.receive", "menu", req.MenuItemID, "", "")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"added": added,
		// Already in the store: a second scan of the same bottle, or a code
		// somebody else received. Worth looking at, never worth failing over.
		"duplicates": dup,
		// Not a marking code at all — a barcode read instead of a DataMatrix, a
		// half-read scan, a scanner set to the wrong symbology.
		"bad": bad,
	})
}

// saveMarks files the codes somebody scanned, whichever screen they used.
//
// ⚠️ **One implementation for the panel and the phone**, the rule the stocktake
// already follows: the shape check, the duplicate answer and the "one code, one
// row" guarantee have to be one piece of code, or a box scanned on a phone
// files differently from the same box scanned at the counter — and the
// disagreement surfaces weeks later as a sale the till refuses.
//
// ⚠️ **Every code is answered separately.** A box of forty with one unreadable
// sticker must not fail as a box of forty: the thirty-nine are in the store room
// either way, and an all-or-nothing save is one that gets abandoned halfway
// through unpacking.
func (h *Handler) saveMarks(
	ctx context.Context,
	branchID, itemID, purchaseID primitive.ObjectID,
	codes []string, who string,
) (added int, dup, bad []string, err error) {
	now := time.Now()
	// ⚠️ Reported by code rather than counted: the person unpacking has the
	// bottle in their hand, and "one duplicate" without saying which is a box
	// they have to scan again from the start.
	dup = []string{}
	bad = []string{}

	for _, raw := range codes {
		code := marking.Normalize(raw)
		if err := marking.Check(code); err != nil {
			bad = append(bad, raw)
			continue
		}
		_, insErr := h.Store.MarkedUnits.InsertOne(ctx, models.MarkedUnit{
			BranchID:   branchID,
			Code:       code,
			MenuItemID: itemID,
			PurchaseID: purchaseID,
			ReceivedAt: now,
			ReceivedBy: who,
		})
		switch {
		case insErr == nil:
			added++
		case mongo.IsDuplicateKeyError(insErr):
			// ⚠️ **The unique index is the check, not a lookup before it.** Two
			// people unpacking two boxes at two tills would both find nothing
			// and both insert; the index is the only thing that is true at the
			// moment of writing.
			dup = append(dup, code)
		default:
			return added, dup, bad, insErr
		}
	}
	return added, dup, bad, nil
}

// AdminMarkStock is what this branch is holding, by product.
func (h *Handler) AdminMarkStock(w http.ResponseWriter, r *http.Request) {
	_, branchID, _, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{"branchId": branchID, "soldAt": nil}
	if id, err := objectID(r.URL.Query().Get("menuItemId")); err == nil && !id.IsZero() {
		filter["menuItemId"] = id
	}
	held, err := h.Store.MarkedUnits.CountDocuments(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"held": held})
}

// markInboundRefusal is the till's half: has this branch actually got this
// bottle?
//
// ⚠️ **Silent when the branch does not scan at goods-in**, which is every
// install today. Turning it on is a decision a shop makes once it has started
// unpacking with a scanner in hand — and until then a check that asked this
// question would refuse every sale.
func (h *Handler) markInboundRefusal(
	ctx context.Context, branchID primitive.ObjectID, items []models.OrderItem,
) (string, error) {
	branch, err := h.branchByIDCtx(ctx, branchID)
	if err != nil || !branch.MarkingInbound {
		return "", nil
	}
	for _, it := range items {
		if it.Void != nil || it.MarkCode == "" {
			continue
		}
		code := marking.Normalize(it.MarkCode)
		var unit models.MarkedUnit
		err := h.Store.MarkedUnits.FindOne(ctx, bson.M{"code": code}).Decode(&unit)
		if errors.Is(err, mongo.ErrNoDocuments) {
			// ⚠️ Named as what it is rather than as a failure: the bottle is in
			// the room and the paperwork is not, and the person who can fix that
			// is the one who unpacked the box.
			return it.Name + ": " + errMarkNotReceived, nil
		}
		if err != nil {
			return "", err
		}
		if unit.SoldAt != nil {
			return it.Name + ": " + errMarkAlreadySold, nil
		}
	}
	return "", nil
}

const (
	errMarkNotReceived = "bu kod kirimda qabul qilinmagan"
	errMarkAlreadySold = "bu kod allaqachon sotilgan"
)

// markUnitsSold records which bottles left on this receipt.
//
// ⚠️ **Marked rather than deleted.** "This bottle was sold on that receipt" is
// the answer to the only question anybody asks afterwards, and a row that
// disappeared would leave a re-scanned code looking exactly like one that never
// arrived.
//
// ⚠️ **Best effort, after the sale is already filed.** The receipt is the legal
// record and the money is taken; a store row that failed to update must not
// unwind either. It is logged by its absence — the code reads as unsold, which
// the next scan refuses, and that is the safe direction.
func (h *Handler) markUnitsSold(
	ctx context.Context, orderID primitive.ObjectID, items []models.OrderItem,
) {
	now := time.Now()
	for _, it := range items {
		if it.Void != nil || it.MarkCode == "" {
			continue
		}
		_, _ = h.Store.MarkedUnits.UpdateOne(ctx,
			bson.M{"code": marking.Normalize(it.MarkCode), "soldAt": nil},
			bson.M{"$set": bson.M{"soldAt": now, "orderId": orderID}})
	}
}
