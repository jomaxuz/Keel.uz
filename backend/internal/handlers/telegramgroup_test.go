package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **A group is not a guest, and the bot was treating it as one.**
//
// Every guest flow — choose a language, browse the menu, leave a rating — is
// written for one person. Run in a group they are nonsense and they are loud:
// somebody types anything and the bot answers with a menu, or with "thank you
// for your feedback". In the alerts group, whose entire value is that it stays
// quiet until something matters, that is the fastest way to teach everybody in
// it to stop reading.
func TestTheBotIgnoresGroupsAndChannels(t *testing.T) {
	src := readLossSource(t, "telegrambot.go")
	if !strings.Contains(src, `msg.Chat.Type != "" && msg.Chat.Type != "private"`) {
		t.Fatal("group and channel messages are being handled as guest chats again")
	}
	// ⚠️ A button under a message the bot posted in a group is still a group.
	if !strings.Contains(src, `t := cb.Message.Chat.Type; t != "" && t != "private"`) {
		t.Fatal("a button press in a group is still handled")
	}
}

// ⚠️ **An empty type is treated as private**, because that is what every update
// this bot has ever handled looks like when the field is absent — and reading a
// missing field as "not private" would make the bot stop answering guests
// entirely. Failing towards the behaviour that already worked.
func TestAMissingChatTypeIsStillAGuest(t *testing.T) {
	src := readLossSource(t, "telegrambot.go")
	// The guard requires the type to be *both* non-empty and non-private.
	if strings.Contains(src, `msg.Chat.Type != "private" {`) &&
		!strings.Contains(src, `msg.Chat.Type != "" &&`) {
		t.Fatal("an update with no chat type would be dropped — every guest goes silent")
	}
}

// ⚠️ **A group becoming a supergroup is not a mistake anybody made.** Adding a
// bot, or giving somebody administrator rights, upgrades a small group on its
// own — so a restaurant that set this up correctly on Monday finds it silent on
// Tuesday having changed nothing. Telegram hands back the new id in the body of
// the response that refuses the message, and that is the only place it ever
// appears: the owner never sees it and the old id is dead permanently.
func TestASupergroupUpgradeIsFollowed(t *testing.T) {
	src := readLossSource(t, "telegram.go")
	if !strings.Contains(src, "telegram.MigratedError") {
		t.Fatal("the new chat id is being thrown away with the error")
	}
	// ⚠️ Stored before the retry: if the second attempt also fails, the
	// restaurant is still better off holding an id that exists.
	store := strings.Index(src, "h.Store.TelegramSettings.UpdateOne(ctx, bson.M{},")
	retry := strings.LastIndex(src, "telegram.SendMessage(ctx, token, moved.NewChatID")
	if store < 0 || retry < 0 || store > retry {
		t.Fatal("the new id is only kept when the retry succeeds")
	}
}

// The two ids are updated independently: a send to the feedback group must not
// rewrite the alerts group's id.
func TestEachGroupUpdatesItsOwnId(t *testing.T) {
	if chatField("feedback") != "feedbackChatId" {
		t.Fatal("a feedback send would rewrite the wrong field")
	}
	if chatField("alerts") != "alertChatId" || chatField("") != "alertChatId" {
		t.Fatal("an alert send would rewrite the wrong field")
	}
}
