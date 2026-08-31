// Command menucheck answers "why did import fail for this link?" in one run.
//
// ⚠️ **This exists because the question always arrives as a link.** An owner
// reports that import does not work and pastes an address; the panel can only
// say which reader answered, and when none did, the interesting part — what the
// page actually contained — is on the server and gone. Reproducing that by hand
// means a fetch, four readers and a text extraction, which is twenty minutes
// and was done twice before this was written.
//
// It reads. It never writes, and it never touches the database — so it is safe
// to run against production while somebody is on the phone.
//
//	go run ./cmd/menucheck 'https://eats.yandex.com/en-uz/tashkent/r/x?placeSlug=y'
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"restaurant-backend/internal/menuimport"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "menucheck <havola>")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	page, final, err := menuimport.Fetch(ctx, os.Args[1])
	if err != nil {
		// The fetcher's own words: they already name what happened.
		fmt.Printf("olinmadi: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("olindi: %s (%d bayt)\n", final, len(page))

	// ⚠️ Checked before the readers, exactly as the handler does: a challenge
	// page reads as "this menu has no dishes" in every other line of output.
	if menuimport.BotWall(page, final) {
		fmt.Println("⚠️  robot tekshiruvi (captcha) — server bu saytni o'qiy olmaydi")
	}
	if name := menuimport.AggregatorName(final); name != "" {
		fmt.Printf("agregator: %s\n", name)
	}

	for _, r := range menuimport.Readers() {
		found := r.Read(ctx, page, final)
		fmt.Printf("%-11s %-52s %d ta\n", r.ID, r.Label, len(found))
		for i, d := range found {
			if i == 3 {
				fmt.Printf("            … yana %d ta\n", len(found)-3)
				break
			}
			fmt.Printf("            %-40s %8d  %s\n", d.Name, d.Price, d.Category)
		}
	}

	// What the assistant would be given. Zero characters is the answer to
	// "why did the model say there was no menu" — it was handed nothing.
	text := menuimport.PageText(page)
	fmt.Printf("AI ga boradigan matn: %d belgi\n", len(text))
	if len(text) > 300 {
		text = text[:300] + "…"
	}
	if text != "" {
		fmt.Println(text)
	}
}
