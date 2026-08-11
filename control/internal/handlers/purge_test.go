package handlers

import (
	"testing"

	"keel-control/internal/models"
)

// ⚠️ The guards, not the deletion.
//
// What is worth sealing here is every way the button must **refuse**: the
// deletion itself is three library calls, while the refusals are the whole
// reason the file exists and each of them is one edit away from being dropped
// as "an extra check nobody needs".
//
// They are asserted as one table because they are one rule with four halves —
// a purge is allowed only when the customer is already switched off, the slug
// was typed, a reason was given, and a backup exists to answer for it.
func TestPurgeIsRefusedUntilEveryGuardPasses(t *testing.T) {
	const slug = "osh"

	cases := []struct {
		name    string
		status  string
		confirm string
		reason  string
		staleBk bool
		allow   bool
	}{
		{
			// The shape of an accidental deletion: the right buttons pressed on
			// the wrong row, while the customer is still serving lunch.
			name: "jonli mijoz", status: models.StatusActive,
			confirm: slug, reason: "mijoz so'radi", allow: false,
		},
		{
			name: "sinov muddatidagi mijoz", status: models.StatusTrial,
			confirm: slug, reason: "mijoz so'radi", allow: false,
		},
		{
			// Typing the name of the restaurant instead of the slug: two
			// customers can share a name, and none share a slug.
			name: "slug noto'g'ri yozilgan", status: models.StatusSuspended,
			confirm: "Osh Markazi", reason: "mijoz so'radi", allow: false,
		},
		{
			name: "sabab yo'q", status: models.StatusSuspended,
			confirm: slug, reason: "   ", allow: false,
		},
		{
			// The only undo there is. Stale means last night produced nothing.
			name: "zaxira eskirgan", status: models.StatusDeleted,
			confirm: slug, reason: "shartnoma tugadi", staleBk: true, allow: false,
		},
		{
			name: "hammasi joyida", status: models.StatusSuspended,
			confirm: slug, reason: "mijoz so'radi", allow: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, msg := purgeRefusal(c.status, slug, c.confirm, c.reason, c.staleBk)
			if allowed := code == 0; allowed != c.allow {
				t.Errorf("allowed = %v, want %v (%d %q)", allowed, c.allow, code, msg)
			}
			if code != 0 && msg == "" {
				t.Error("refused without saying why")
			}
		})
	}
}
