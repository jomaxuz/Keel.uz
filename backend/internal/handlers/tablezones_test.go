package handlers

import (
	"testing"

	"restaurant-backend/internal/models"
)

func zonedBooking() models.BookingSettings {
	return models.BookingSettings{
		Zones: []models.TableZone{
			{ID: "z1", Name: "Zal", Bookable: true},
			{ID: "z2", Name: "Saboy", Bookable: false},
		},
		Tables: []models.FloorTable{
			{ID: "t1", Number: "7", ZoneID: "z1", IsActive: true},
			{ID: "t2", Number: "112", ZoneID: "z2", IsActive: true},
			// Drawn before zones existed.
			{ID: "t3", Number: "3", IsActive: true},
			{ID: "t4", Number: "9", ZoneID: "z1", IsActive: false},
			// Points at a zone somebody deleted.
			{ID: "t5", Number: "5", ZoneID: "gone", IsActive: true},
		},
	}
}

// ⚠️ **A takeaway number is not a table a guest can reserve**, and it is still a
// table the till needs — a takeaway order has to be opened against something.
// One list would either put those numbers on the booking page or keep them off
// the till.
func TestTakeawayZoneIsNotBookable(t *testing.T) {
	b := zonedBooking()
	if !b.Bookable(b.Tables[0]) {
		t.Fatal("a hall table is not bookable")
	}
	if b.Bookable(b.Tables[1]) {
		t.Fatal("a takeaway number can be reserved by a guest")
	}
}

// ⚠️ **A table with no zone is bookable.** Every table drawn before zones
// existed has no id here, and reading that as "no zone" would empty every
// restaurant's booking page on the deploy that added them. The zero-value rule,
// in the only direction that cannot break a live restaurant.
func TestTablesWithNoZoneStayBookable(t *testing.T) {
	b := zonedBooking()
	if !b.Bookable(b.Tables[2]) {
		t.Fatal("a table drawn before zones existed lost its booking")
	}
}

// ⚠️ A table pointing at a deleted zone stays bookable rather than vanishing: a
// deleted zone is an editing accident, and silently removing tables from the
// booking page is the kind of failure nobody notices until a guest cannot
// reserve anything.
func TestTableInADeletedZoneStaysBookable(t *testing.T) {
	b := zonedBooking()
	if !b.Bookable(b.Tables[4]) {
		t.Fatal("deleting a zone silently removed its tables from booking")
	}
}

// An inactive table is never bookable, whatever its zone says — that check
// predates zones and must survive them.
func TestInactiveTableIsNeverBookable(t *testing.T) {
	b := zonedBooking()
	if b.Bookable(b.Tables[3]) {
		t.Fatal("a table taken out of service can be reserved")
	}
}
