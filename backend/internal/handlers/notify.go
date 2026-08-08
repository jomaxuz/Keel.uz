package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"restaurant-backend/internal/models"
	"restaurant-backend/internal/telegram"

	"go.mongodb.org/mongo-driver/bson"
)

// Telling a guest what happened to their order, through the bot.
//
// **This is the band that saves money.** Every one of these messages would
// otherwise be an SMS the restaurant pays for, one per status change per order —
// and the restaurants that need order updates most are the ones sending the most
// of them. Through Telegram it costs nothing.
//
// Three rules, and the first two are about not becoming noise:
//
//   - **Only the changes a guest acts on.** Confirmed ("your order is happening"),
//     on the way ("be at the door"), delivered ("this is closed"), cancelled
//     ("and here is why"). `preparing` is deliberately silent: it is the
//     kitchen's own step, and a guest who gets five pings for one order stops
//     reading the one that matters.
//   - **In their language**, taken from Telegram's own UI setting rather than
//     from the site's cookie: a guest who opened the mini app from a Russian
//     Telegram is reading Russian, whatever the last visitor picked on the site.
//   - **It can never fail the status change.** A bot that is blocked, a guest who
//     never pressed Start, Telegram being down — none of that is the order's
//     problem. The status moves; the message is attempted and logged.

// The statuses worth a message. See the note above on why `preparing` is absent.
func notifiableStatus(s models.OrderStatus) bool {
	switch s {
	case models.StatusConfirmed, models.StatusOnTheWay,
		models.StatusDelivered, models.StatusCancelled:
		return true
	}
	return false
}

// orderStatusMessage is what the guest reads.
//
// Kept as a pure function so the wording can be tested without a bot, a database
// or an order — and so the three languages sit side by side where a missing one
// is obvious.
//
// Plain text on purpose: a dish name can contain any character, and an unescaped
// underscore in Markdown makes Telegram reject the **whole** message. A
// notification that silently does not arrive is worse than one without bold text.
func orderStatusMessage(lang, restaurant, number string,
	status models.OrderStatus, cancelReason string) string {
	switch lang {
	case "ru":
		switch status {
		case models.StatusConfirmed:
			return fmt.Sprintf("%s: заказ №%s принят. Мы начали готовить.", restaurant, number)
		case models.StatusOnTheWay:
			return fmt.Sprintf("%s: заказ №%s в пути.", restaurant, number)
		case models.StatusDelivered:
			return fmt.Sprintf("%s: заказ №%s доставлен. Приятного аппетита!", restaurant, number)
		case models.StatusCancelled:
			msg := fmt.Sprintf("%s: заказ №%s отменён.", restaurant, number)
			if cancelReason != "" {
				msg += " Причина: " + cancelReason
			}
			return msg
		}
	case "en":
		switch status {
		case models.StatusConfirmed:
			return fmt.Sprintf("%s: order #%s accepted. We have started cooking.", restaurant, number)
		case models.StatusOnTheWay:
			return fmt.Sprintf("%s: order #%s is on the way.", restaurant, number)
		case models.StatusDelivered:
			return fmt.Sprintf("%s: order #%s delivered. Enjoy!", restaurant, number)
		case models.StatusCancelled:
			msg := fmt.Sprintf("%s: order #%s was cancelled.", restaurant, number)
			if cancelReason != "" {
				msg += " Reason: " + cancelReason
			}
			return msg
		}
	default:
		switch status {
		case models.StatusConfirmed:
			return fmt.Sprintf("%s: №%s buyurtma qabul qilindi. Tayyorlashni boshladik.", restaurant, number)
		case models.StatusOnTheWay:
			return fmt.Sprintf("%s: №%s buyurtma yo'lda.", restaurant, number)
		case models.StatusDelivered:
			return fmt.Sprintf("%s: №%s buyurtma yetkazildi. Yoqimli iste'mol!", restaurant, number)
		case models.StatusCancelled:
			msg := fmt.Sprintf("%s: №%s buyurtma bekor qilindi.", restaurant, number)
			if cancelReason != "" {
				msg += " Sabab: " + cancelReason
			}
			return msg
		}
	}
	return ""
}

// notifyOrderStatus sends the message, if there is anybody to send it to.
//
// Everything here is a reason **not** to send, and each one is a real case:
// an order placed by a guest who never used Telegram, a restaurant with no bot,
// a status nobody needs told about. Silence in those cases is correct — the SMS
// path is not a fallback here, because the guest who ordered through Telegram is
// reading Telegram.
func (h *Handler) notifyOrderStatus(ctx context.Context, order *models.Order) {
	if order == nil || !notifiableStatus(order.Status) {
		return
	}
	if order.UserID.IsZero() {
		return // a phone order typed in by an operator: nobody to message
	}
	s := h.telegramSettings(ctx)
	if !s.Usable() {
		return
	}

	var user models.User
	if err := h.Store.Users.FindOne(ctx, bson.M{"_id": order.UserID}).Decode(&user); err != nil {
		return
	}
	if user.TelegramID == 0 {
		return // signed in by SMS: they never opened the bot
	}

	var rest models.Restaurant
	_ = h.Store.Restaurant.FindOne(ctx, bson.M{}).Decode(&rest)
	name := strings.TrimSpace(rest.Name)
	if name == "" || name == seedRestaurantName {
		name = "Restoran"
	}

	text := orderStatusMessage(
		normalizeLang(user.TelegramLang), name, order.Number,
		order.Status, order.CancelReason,
	)
	if text == "" {
		return
	}

	// Detached from the request: the operator's status change must not wait on
	// Telegram, and the response they are looking at is about the order, not
	// about a message.
	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
		defer cancel()
		if err := telegram.SendMessage(ctx, s.BotToken, user.TelegramID, text); err != nil {
			// Logged, never surfaced. The commonest cause is a guest who opened
			// the mini app but never pressed Start, which Telegram refuses — and
			// which is not something the restaurant can fix or needs to see.
			log.Printf("telegram notify %s: %v", order.Number, err)
		}
	}()
}

// normalizeLang maps Telegram's language code onto the three the app speaks.
//
// Telegram sends things like "ru-RU", "en-GB" and "uz", and anything unknown is
// Uzbek — the base language everywhere else in this app.
func normalizeLang(code string) string {
	c := strings.ToLower(strings.TrimSpace(code))
	switch {
	case strings.HasPrefix(c, "ru"):
		return "ru"
	case strings.HasPrefix(c, "en"):
		return "en"
	default:
		return "uz"
	}
}
