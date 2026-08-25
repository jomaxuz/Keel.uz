package insight

import "testing"

func facts() []Fact {
	return []Fact{
		{Key: "lapsed_regulars", Area: Guests, Weight: 3_500_000, Action: NewCampaign},
		{Key: "sales_down", Area: Money, Weight: 9_000_000, Action: OpenReports},
		{Key: "dead_dishes", Area: Menu, Weight: 400_000, Action: OpenMenu},
		{Key: "quiet_week", Area: Money, Weight: 1_000_000},
	}
}

// ⚠️ **The one rule that makes the whole feature trustworthy.** A model that
// answers with a key it was never given has invented the subject, and no amount
// of well-written prose about an invented subject is worth printing.
func TestACardAboutNothingIsDropped(t *testing.T) {
	f := facts()
	got := Keep([]Card{
		{Key: "sales_down", Title: "Savdo", Body: "..."},
		{Key: "guests_are_unhappy", Title: "Mijozlar", Body: "78% norozi"},
	}, f)

	if len(got) != 1 {
		t.Fatalf("got %d cards, want 1: %+v", len(got), got)
	}
	if got[0].Key != "sales_down" {
		t.Fatalf("the invented card survived: %+v", got)
	}
}

// ⚠️ The model chooses what to say; it does not choose what a button does. An
// answer that relabelled a card's action would send an owner to the wrong
// screen with a sentence that justified it.
func TestTheActionComesFromTheFactNotTheAnswer(t *testing.T) {
	got := Keep([]Card{{
		Key: "lapsed_regulars", Title: "x", Body: "y",
		Action: Stocktake, Area: Stock,
	}}, facts())

	if len(got) != 1 {
		t.Fatal("the card was dropped")
	}
	if got[0].Action != NewCampaign || got[0].Area != Guests {
		t.Fatalf("the answer overrode the fact: %+v", got[0])
	}
}

// ⚠️ The most expensive problem stays at the top however the model listed them
// — it is judging relevance, not arithmetic, and the arithmetic is ours.
func TestRankedOrderSurvivesTheModelsOrdering(t *testing.T) {
	f := Rank(facts())
	got := Keep([]Card{
		{Key: "dead_dishes", Title: "a", Body: "b"},
		{Key: "sales_down", Title: "c", Body: "d"},
	}, f)

	if len(got) != 2 || got[0].Key != "sales_down" {
		t.Fatalf("the expensive fact was not first: %+v", got)
	}
}

// ⚠️ A slow week is a money fact, a guest fact and a menu fact at once — all
// true, all the same problem. Three sentences about it read as three problems
// and as padding.
func TestOneCardPerArea(t *testing.T) {
	ranked := Rank(facts())
	for i := range ranked {
		for j := range ranked {
			if i != j && ranked[i].Area == ranked[j].Area {
				t.Fatalf("two facts from %s survived: %+v", ranked[i].Area, ranked)
			}
		}
	}
	// The heavier of the two money facts is the one that stayed.
	for _, f := range ranked {
		if f.Area == Money && f.Key != "sales_down" {
			t.Fatalf("the cheaper money fact won: %s", f.Key)
		}
	}
}

// ⚠️ A briefing of eleven items is a report, and a report is read once. The
// limit is the feature.
func TestTheBriefingIsShortEnoughToRead(t *testing.T) {
	var many []Fact
	var cards []Card
	areas := []Area{Guests, Menu, Stock, Team, Money}
	for i, a := range areas {
		k := string(a)
		many = append(many, Fact{Key: k, Area: a, Weight: int64(100 - i)})
		cards = append(cards, Card{Key: k, Title: k, Body: k})
	}
	if got := Keep(cards, Rank(many)); len(got) > MaxCards {
		t.Fatalf("%d cards reached the screen", len(got))
	}
}

// A model that lists one fact twice gets one card, not two identical ones.
func TestARepeatedFactIsOneCard(t *testing.T) {
	got := Keep([]Card{
		{Key: "sales_down", Title: "birinchi", Body: "a"},
		{Key: "sales_down", Title: "ikkinchi", Body: "b"},
	}, facts())
	if len(got) != 1 || got[0].Title != "birinchi" {
		t.Fatalf("%+v", got)
	}
}

// ⚠️ Two identical mornings must produce an identically ordered briefing —
// otherwise an owner reads movement into map iteration order.
func TestEqualWeightsOrderStably(t *testing.T) {
	same := []Fact{
		{Key: "b_fact", Area: Menu, Weight: 5},
		{Key: "a_fact", Area: Stock, Weight: 5},
	}
	for i := 0; i < 20; i++ {
		if Rank(same)[0].Key != "a_fact" {
			t.Fatal("the order moved between two identical runs")
		}
	}
}

// An empty answer is an empty briefing, not a crash and not a placeholder card.
func TestNothingToSayIsNoCards(t *testing.T) {
	if got := Keep(nil, facts()); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
	if got := Keep([]Card{{Key: "sales_down"}}, facts()); len(got) != 0 {
		t.Fatal("a card with no words was rendered")
	}
}
