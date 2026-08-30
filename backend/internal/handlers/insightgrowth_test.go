package handlers

import "testing"

// ⚠️ **These are the judgements, and they are the whole value of the facts.**
// The queries around them are plumbing; what decides whether a card is worth an
// owner's morning is which rows count as comparable and how big a difference
// has to be. Left inside the aggregation they would only ever run against a
// live database, which means nobody ever checks them.

func TestADishThatSoldNothingLastWeekIsNotAMover(t *testing.T) {
	// A dish added on Tuesday is up infinitely. Without the guard it wins every
	// week and crowds out every real movement.
	this := []dishSale{
		{ID: "new", Name: "Yangi taom", Qty: 10, Money: 400_000},
		{ID: "osh", Name: "Osh", Qty: 60, Money: 2_700_000},
	}
	prev := []dishSale{
		{ID: "osh", Name: "Osh", Qty: 40, Money: 1_800_000},
	}
	now, _, ok := pickDishMovement(this, prev)
	if !ok {
		t.Fatal("a real mover was dropped")
	}
	if now.ID != "osh" {
		t.Fatalf("picked %q; a dish with no previous week is not a mover", now.ID)
	}
}

func TestADishThatDriftedWithTheWeekIsNotNews(t *testing.T) {
	this := []dishSale{{ID: "osh", Name: "Osh", Money: 1_900_000}}
	prev := []dishSale{{ID: "osh", Name: "Osh", Money: 1_800_000}}
	if _, _, ok := pickDishMovement(this, prev); ok {
		t.Fatal("a 5% drift was reported as a movement")
	}
}

// Both directions, one fact: a collapse matters as much as a jump.
func TestADishThatCollapsedIsAMovement(t *testing.T) {
	this := []dishSale{{ID: "osh", Name: "Osh", Money: 400_000}}
	prev := []dishSale{{ID: "osh", Name: "Osh", Money: 1_800_000}}
	now, was, ok := pickDishMovement(this, prev)
	if !ok || now.Money >= was.Money {
		t.Fatal("a dish that fell by three quarters was not reported")
	}
}

// ⚠️ Two waiters always have a "weaker" one. That is a ranking, not a finding,
// and putting a name on it in front of an owner at 8am is how this feature
// causes harm.
func TestTwoWaitersAreNotAComparison(t *testing.T) {
	rows := []serverWeek{
		{Name: "Aziz", Checks: 40, Money: 8_000_000},
		{Name: "Dilnoza", Checks: 40, Money: 4_000_000},
	}
	if _, ok := pickServerGap(rows); ok {
		t.Fatal("a floor of two produced a comparison")
	}
}

// ⚠️ Below ten checks an "average check" is one large table.
func TestAWaiterWithAlmostNoChecksIsNotRanked(t *testing.T) {
	rows := []serverWeek{
		{Name: "Aziz", Checks: 40, Money: 12_000_000},   // 300 000
		{Name: "Dilnoza", Checks: 38, Money: 5_700_000}, // 150 000
		{Name: "Sardor", Checks: 42, Money: 8_400_000},  // 200 000
		// One enormous table on a trial shift: 1 500 000 a check, and top of
		// the floor on two covers.
		{Name: "Yangi", Checks: 2, Money: 3_000_000},
	}
	gap, ok := pickServerGap(rows)
	if !ok {
		t.Fatal("a real floor produced nothing")
	}
	if gap.best.Name == "Yangi" || gap.worst.Name == "Yangi" {
		t.Fatalf("a two-check shift was ranked: best=%s weakest=%s",
			gap.best.Name, gap.worst.Name)
	}
}

func TestATeamWorkingTheSameWayIsNotACard(t *testing.T) {
	rows := []serverWeek{
		{Name: "Aziz", Checks: 40, Money: 8_000_000},
		{Name: "Dilnoza", Checks: 40, Money: 7_900_000},
		{Name: "Sardor", Checks: 40, Money: 8_100_000},
	}
	if _, ok := pickServerGap(rows); ok {
		t.Fatal("a 2% spread was reported as a gap")
	}
}

func TestTheFloorGapNamesBothEnds(t *testing.T) {
	rows := []serverWeek{
		{Name: "Aziz", Checks: 40, Money: 12_000_000},   // 300 000
		{Name: "Dilnoza", Checks: 40, Money: 8_000_000}, // 200 000
		{Name: "Sardor", Checks: 40, Money: 6_000_000},  // 150 000
	}
	gap, ok := pickServerGap(rows)
	if !ok {
		t.Fatal("a two-to-one spread was not reported")
	}
	if gap.best.Name != "Aziz" || gap.worst.Name != "Sardor" {
		t.Fatalf("best=%s weakest=%s", gap.best.Name, gap.worst.Name)
	}
	if gap.houseAvg <= gap.worstAvg || gap.houseAvg >= gap.bestAvg {
		t.Fatalf("the house average %d is not between %d and %d",
			gap.houseAvg, gap.worstAvg, gap.bestAvg)
	}
}

// ⚠️ **The quietest hour of any day is the one it opens in.** A card saying
// "you are quiet at 10am" is a card about the timetable, and the owner set the
// timetable.
func TestTheOpeningHourIsNotTheQuietHour(t *testing.T) {
	rows := []hourRow{
		{Hour: 19, Orders: 60, Money: 9_000_000},
		{Hour: 13, Orders: 50, Money: 7_000_000},
		{Hour: 20, Orders: 40, Money: 6_000_000},
		{Hour: 14, Orders: 30, Money: 4_000_000},
		{Hour: 18, Orders: 20, Money: 2_000_000},
		{Hour: 16, Orders: 8, Money: 900_000},
		// Opening, hours away from the peak.
		{Hour: 9, Orders: 2, Money: 120_000},
	}
	peak, quiet, ok := pickQuietHour(rows)
	if !ok {
		t.Fatal("a day with a clear dead hour reported nothing")
	}
	if quiet.Hour == 9 {
		t.Fatal("the opening hour was reported as the dead hour")
	}
	if peak.Hour != 19 || quiet.Hour != 16 {
		t.Fatalf("peak=%d quiet=%d, want 19 and 16", peak.Hour, quiet.Hour)
	}
}

func TestAnEvenDayHasNoDeadHour(t *testing.T) {
	rows := []hourRow{
		{Hour: 19, Money: 9_000_000},
		{Hour: 13, Money: 8_600_000},
		{Hour: 20, Money: 8_400_000},
		{Hour: 14, Money: 8_200_000},
		{Hour: 18, Money: 8_000_000},
		{Hour: 15, Money: 7_800_000},
	}
	if _, _, ok := pickQuietHour(rows); ok {
		t.Fatal("an evenly busy day produced a dead-hour card")
	}
}

func TestADayWithNoShapeIsNotReported(t *testing.T) {
	rows := []hourRow{
		{Hour: 19, Money: 9_000_000},
		{Hour: 13, Money: 1_000_000},
	}
	if _, _, ok := pickQuietHour(rows); ok {
		t.Fatal("two trading hours produced a dead-hour card")
	}
}
