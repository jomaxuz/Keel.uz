package handlers

import "testing"

// Whether a one-time code may be handed back in the API response.
//
// ⚠️ **This is the hole that shipped, and it is worth stating plainly.** With
// no gateway configured the server falls back to the demo sender, whose entire
// purpose is to return the code so a developer can finish the flow without a
// paid account. On a hosted tenant that meant a freshly created restaurant's
// site let anybody sign in as anybody: ask for a code for a stranger's number,
// read it out of the JSON, and you are them — orders, addresses, history.
//
// Nothing ever errored. The request succeeded, the response was well-formed,
// and only somebody reading the JSON would have seen it. Which is exactly why
// the rule is pinned here rather than trusted to a review.
func TestExposeDemoCode(t *testing.T) {
	cases := []struct {
		name          string
		demo, allowed bool
		wantExpose    bool
		wantErr       bool
	}{
		{
			// The default on every hosted tenant, and the one that matters.
			name: "no gateway, flag off — refuse",
			demo: true, allowed: false,
			wantExpose: false, wantErr: true,
		},
		{
			// A developer's own machine, switched on deliberately.
			name: "no gateway, flag on — hand it back",
			demo: true, allowed: true,
			wantExpose: true, wantErr: false,
		},
		{
			// A working gateway texts the code; the response never carries it.
			name: "real gateway, flag off",
			demo: false, allowed: false,
			wantExpose: false, wantErr: false,
		},
		{
			// And the flag must not loosen a real gateway either: it exists
			// only for the demo path.
			name: "real gateway, flag on — still never exposed",
			demo: false, allowed: true,
			wantExpose: false, wantErr: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expose, err := exposeDemoCode(c.demo, c.allowed)
			if expose != c.wantExpose {
				t.Errorf("expose = %v, want %v", expose, c.wantExpose)
			}
			if (err != nil) != c.wantErr {
				t.Errorf("err = %v, want error: %v", err, c.wantErr)
			}
		})
	}
}
