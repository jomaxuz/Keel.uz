package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **Both of these produced the same complaint and have different causes**, so
// both are sealed: "the scanner does not find this product". One is a stray
// newline from the reader, the other is a lookup that crossed into another
// brand's catalogue and found the wrong price.

func TestAScannedCodeIsTrimmed(t *testing.T) {
	// Most readers send the code followed by Enter, and a keyboard wedge
	// delivers that as part of the value. Untrimmed, every scan misses.
	body := between(t, readSrc(t, "tillbarcode.go"), "func (h *Handler) StaffScanBarcode", "\n}\n")
	if !strings.Contains(body, "strings.TrimSpace(r.URL.Query().Get(\"code\"))") {
		t.Fatal("the scanned code is used untrimmed")
	}
}

func TestTheLookupCannotCrossBrands(t *testing.T) {
	// Two brands under one owner may stock the same EAN. A counter that reached
	// across them would ring up the other shop's price — correct on the receipt
	// and wrong in the drawer.
	body := between(t, readSrc(t, "tillbarcode.go"), "func (h *Handler) StaffScanBarcode", "\n}\n")
	if !strings.Contains(body, "filter[\"brandId\"]") {
		t.Fatal("the barcode lookup is not scoped to the till's brand")
	}
	find := strings.Index(body, "h.Store.Menu.FindOne")
	scope := strings.Index(body, "filter[\"brandId\"]")
	if find < 0 || scope < 0 || scope > find {
		t.Fatal("the brand is applied after the lookup has already run")
	}
}

// ⚠️ **An unknown code is a state, not an error.** A 404 puts a red message in
// front of somebody holding a packet with a queue behind them, and sends them
// looking for a fault instead of adding the product.
func TestAnUnknownCodeIsNotAnError(t *testing.T) {
	body := between(t, readSrc(t, "tillbarcode.go"), "func (h *Handler) StaffScanBarcode", "\n}\n")
	if strings.Contains(body, "http.StatusNotFound") {
		t.Fatal("an unknown barcode is answered as an error")
	}
	if !strings.Contains(body, "\"found\":  false") && !strings.Contains(body, "\"found\": false") {
		t.Fatal("an unknown barcode has no state of its own")
	}
}
