package handlers

import (
	"encoding/json"
	"testing"
)

// The team app writes prices through JsonPrimitive(Double), i.e. "30000.0".
// A plain int refused that and every priced tick at the market failed.
func TestWholeSomAcceptsKotlinDoubles(t *testing.T) {
	for in, want := range map[string]wholeSom{`30000`: 30000, `30000.0`: 30000, `2999.6`: 3000} {
		var got struct {
			Price wholeSom `json:"price"`
		}
		if err := json.Unmarshal([]byte(`{"price":`+in+`}`), &got); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if got.Price != want {
			t.Fatalf("%s: got %d, want %d", in, got.Price, want)
		}
	}
}
