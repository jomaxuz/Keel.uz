package i18n

import "testing"

// ⚠️ **The notifications are the only text the phone cannot translate itself**,
// so they go through this catalogue like an error does — and unlike an error,
// nothing in the app would show that a translation is missing: an Uzbek
// sentence simply arrives on a Russian phone, once, and is swiped away.
func TestCourierPushMessagesAreTranslated(t *testing.T) {
	// The bodies as the server builds them, with values already in place.
	cases := []struct{ msg, wantRU string }{
		{"Yangi buyurtma", "Новый заказ"},
		{"#K7QX buyurtma sizga berildi. Manzil: Chilonzor 5", "Заказ #K7QX передан вам. Адрес: Chilonzor 5"},
		{"#K7QX buyurtma sizdan olindi", "Заказ #K7QX забрали у вас"},
		{"#K7QX buyurtma bekor qilindi", "Заказ #K7QX отменён"},
		{"#K7QX buyurtma bekor qilindi: mijoz javob bermadi", "Заказ #K7QX отменён: mijoz javob bermadi"},
		{"#K7QX buyurtma tayyor — olib chiqing", "Заказ #K7QX готов — забирайте"},
		{"#K7QX buyurtmaning manzili o'zgardi: Amir Temur 12", "Адрес заказа #K7QX изменился: Amir Temur 12"},
		{"120000 so'm naqd pul qabul qilindi", "Принято 120000 сум наличными"},
	}
	for _, c := range cases {
		if got := Localize(RU, c.msg); got != c.wantRU {
			t.Errorf("Localize(ru, %q)\n got: %q\nwant: %q", c.msg, got, c.wantRU)
		}
		// English is not asserted word for word — it moves with the wording —
		// but it must not come back as the Uzbek sentence.
		if got := Localize(EN, c.msg); got == c.msg {
			t.Errorf("inglizchasi tarjima qilinmadi: %q", c.msg)
		}
	}
}
