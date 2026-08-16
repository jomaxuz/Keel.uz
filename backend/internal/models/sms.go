package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SMSSettings is the singleton holding the gateway this restaurant sends its
// login codes through.
//
// Its own collection, for the same reason as PaymentSettings: the `restaurant`
// document is returned in full to every visitor of the site, and a gateway
// password one forgotten `json:"-"` away from a public response is a password
// waiting to leak. A leaked SMS login is not only somebody else's bill — it is
// somebody else sending messages under the restaurant's moderated sender name.
//
// **Per restaurant, not per platform.** Every restaurant signs its own contract
// with its own gateway and types its own credentials here. One shared platform
// account would put every restaurant's codes on one contract, and a single
// restaurant's moderation problem would silence everybody else's login.
type SMSSettings struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	// Which gateway: see sms.Provider* — demo | eskiz | playmobile | getsms |
	// onesignal. Empty means "never configured", which falls back to the
	// environment and then to demo.
	Provider string `bson:"provider" json:"provider"`
	// The sender name the operator moderated ("RESTORAN"). Providers that carry
	// their own originator field override it below.
	From string `bson:"from" json:"from"`

	// Numbers allowed to receive a **demo** code in the API response, so the
	// owner can test signing in before a paid gateway exists.
	//
	// ⚠️ This is the narrow, safe version of the hole that shipped once (see
	// handlers/smsdemo_test.go). Demo mode's whole purpose is to hand the code
	// back in the JSON, which on a live site means *anybody can sign in as
	// anybody*. Restricting it to numbers somebody typed in here keeps the one
	// legitimate use — the owner testing their own login — and removes the
	// attack, because a stranger's number is simply refused.
	//
	// Stored as 998XXXXXXXXX, the same normalisation every phone here uses: a
	// list that only matches when written in one particular format is a list
	// that silently fails to match.
	TestPhones []string `bson:"testPhones,omitempty" json:"testPhones"`

	// The login-code message itself, with `{code}` where the digits go.
	//
	// A setting rather than a constant because the restaurant owns the decision
	// and pays for it: each one signs its own gateway contract and puts its own
	// template through moderation, and a business-lunch place in Tashkent and a
	// family kitchen in Namangan do not have the same guests. Empty means the
	// built-in Uzbek wording — the zero value has to be today's behaviour, the
	// same rule as an empty mapProvider meaning 2GIS.
	//
	// ⚠️ Changing this **invalidates the gateway's moderation**. Eskiz and Play
	// Mobile approve an exact string; an edited template is a new string and is
	// refused until it is approved again. So an edit clears LastTestOk (see
	// AdminUpdateSMS): leaving a green tick over a template the gateway has
	// never seen is how logins stop working with the panel still claiming they
	// were checked.
	CodeTemplate string `bson:"codeTemplate,omitempty" json:"codeTemplate"`

	Eskiz      EskizSMS      `bson:"eskiz" json:"eskiz"`
	PlayMobile PlayMobileSMS `bson:"playmobile" json:"playmobile"`
	GetSMS     GetSMS        `bson:"getsms" json:"getsms"`
	OneSignal  OneSignalSMS  `bson:"onesignal" json:"onesignal"`

	// What the last "Send a test message" attempt did. Kept because a gateway
	// that stopped working says nothing on its own: the restaurant finds out
	// when a guest cannot log in, which is the one moment nobody is looking at
	// this page.
	LastTestAt    time.Time `bson:"lastTestAt" json:"lastTestAt"`
	LastTestOk    bool      `bson:"lastTestOk" json:"lastTestOk"`
	LastTest      string    `bson:"lastTest" json:"lastTest"`
	LastTestPhone string    `bson:"lastTestPhone" json:"lastTestPhone"`

	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// EskizSMS — notify.eskiz.uz. Email + password, exchanged for a bearer token.
type EskizSMS struct {
	Email    string `bson:"email" json:"email"`
	Password string `bson:"password" json:"-"`
	BaseURL  string `bson:"baseUrl" json:"baseUrl"`
}

// PlayMobileSMS — playmobile.uz broker-api, HTTP basic auth.
type PlayMobileSMS struct {
	URL      string `bson:"url" json:"url"`
	Login    string `bson:"login" json:"login"`
	Password string `bson:"password" json:"-"`
}

// GetSMS — getsms.uz. Credentials travel in the request body, and the sender
// name is a "nickname" registered on their side.
type GetSMS struct {
	URL      string `bson:"url" json:"url"`
	Login    string `bson:"login" json:"login"`
	Password string `bson:"password" json:"-"`
	Nickname string `bson:"nickname" json:"nickname"`
}

// OneSignalSMS — the SMS channel of an existing OneSignal app.
type OneSignalSMS struct {
	AppID   string `bson:"appId" json:"appId"`
	APIKey  string `bson:"apiKey" json:"-"`
	From    string `bson:"from" json:"from"`
	BaseURL string `bson:"baseUrl" json:"baseUrl"`
}
