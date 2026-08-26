package handlers

// ---- What the notification groups are told, and in which language ----
//
// ⚠️ **A third language setting, and it is not one too many.** The panel has
// one, the receipts have one, and this is a third — because each has a
// different reader. The panel's is whoever logged in. The receipt's is the
// guest at the table. This one belongs to whoever the owner added to a Telegram
// group, and that is frequently somebody who will never log in at all: an
// accountant, a partner, a head-office manager in another city. Borrowing any
// of the other two would be right by accident and wrong on purpose.
//
// ⚠️ **The messages are a struct rather than a map.** A map lets a language
// silently miss a line, and the line it misses is the one added last — which is
// how a Russian group ends up with one Uzbek word in the middle of every
// message. With a struct the compiler names the gap.

import (
	"context"
	"strings"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

// notifyWords is every phrase the two groups use.
type notifyWords struct {
	// Headings, one per alert kind.
	VoidAfterPrecheck string
	BigDiscount       string
	CashShort         string
	StockShort        string
	RecipeUp          string
	PanelAction       string
	Unknown           string

	// Labels inside an alert.
	Who      string
	Approved string // "(X tasdiqladi)" — the word inside the brackets
	Reason   string
	Currency string

	// ⚠️ The sentence that keeps this channel from becoming an accusation.
	// Sent once, when the group is linked, because it is read by everybody who
	// is ever added afterwards.
	NotAnAccusation string

	// Guest feedback.
	FeedbackFrom string
	Order        string
}

// notifyWordsFor picks the language.
//
// ⚠️ **An unknown or empty value is Uzbek**, which is what every message sent
// before this setting existed — so a restaurant that never opens the dropdown
// sees no change at all.
func notifyWordsFor(lang string) notifyWords {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "ru":
		return notifyWords{
			VoidAfterPrecheck: "После счёта удалено блюдо",
			BigDiscount:       "Крупная скидка",
			CashShort:         "Недостача в кассе",
			StockShort:        "Недостача на складе",
			RecipeUp:          "В техкарте увеличен расход",
			PanelAction:       "Действие в панели",
			Unknown:           "Внимание",
			Who:               "Кто",
			Approved:          "подтвердил",
			Reason:            "Причина",
			Currency:          "сум",
			NotAnAccusation: "Готово. Сюда будут приходить необычные случаи.\n\n" +
				"Это не обвинение, а вопрос. У каждого случая может быть " +
				"обычная причина: гость пожаловался, постоянному клиенту дали " +
				"скидку, из кассы заплатили курьеру. Сначала спросите.",
			FeedbackFrom: "Отзыв гостя",
			Order:        "Заказ",
		}
	case "en":
		return notifyWords{
			VoidAfterPrecheck: "A dish was removed after the bill",
			BigDiscount:       "Large discount",
			CashShort:         "Till shortfall",
			StockShort:        "Stock shortfall",
			RecipeUp:          "Recipe norm increased",
			PanelAction:       "Action in the panel",
			Unknown:           "Notice",
			Who:               "Who",
			Approved:          "approved",
			Reason:            "Reason",
			Currency:          "so'm",
			NotAnAccusation: "Done. Unusual events will arrive here.\n\n" +
				"These are questions, not accusations. Each has an ordinary " +
				"explanation: a guest who complained, a regular given something " +
				"off, a courier paid out of the till. Ask first.",
			FeedbackFrom: "Guest feedback",
			Order:        "Order",
		}
	}
	return notifyWords{
		VoidAfterPrecheck: "Hisob chiqarilgandan keyin taom olib tashlandi",
		BigDiscount:       "Katta chegirma",
		CashShort:         "Kassada kamomad",
		StockShort:        "Omborda kamomad",
		RecipeUp:          "Texkartada sarf oshirildi",
		PanelAction:       "Panelda amal",
		Unknown:           "Diqqat",
		Who:               "Kim",
		Approved:          "tasdiqladi",
		Reason:            "Sabab",
		Currency:          "so'm",
		NotAnAccusation: "Tayyor. Shubhali holatlar shu yerga keladi.\n\n" +
			"Bu xabarlar ayblov emas — savol. Har birining oddiy sababi " +
			"bo'lishi mumkin: mehmon shikoyat qildi, doimiy mijozga chegirma " +
			"berildi, kassadan kuryerga pul berildi. Avval so'rang.",
		FeedbackFrom: "Mehmon fikri",
		Order:        "Buyurtma",
	}
}

// NotifyLangs is what the panel offers.
//
// ⚠️ The same three the rest of the product speaks. A fourth here would be a
// language nothing else in the restaurant is written in, so the messages would
// arrive translated with the dish names still in Uzbek — the exact half-built
// result the receipts had.
var NotifyLangs = []string{"uz", "ru", "en"}

// notifyLang is the language this restaurant's groups are written in.
//
// ⚠️ Read per message rather than cached: it is one small read against a
// singleton, on a path that already makes a network call to Telegram, and a
// cache here would mean an owner changing the language sees no effect until
// something restarts — which reads as the setting not working.
func (h *Handler) notifyLang(ctx context.Context) string {
	var s models.TelegramSettings
	if err := h.Store.TelegramSettings.FindOne(ctx, bson.M{}).Decode(&s); err != nil {
		return ""
	}
	return s.NotifyLang
}

// cleanNotifyLang keeps only a language the messages are actually written in.
//
// ⚠️ An unrecognised value falls back to Uzbek when a message is built, so
// storing one would leave the dropdown showing a language the groups are not
// written in — a setting that displays one thing and does another.
func cleanNotifyLang(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	for _, l := range NotifyLangs {
		if l == v {
			return v
		}
	}
	return ""
}
