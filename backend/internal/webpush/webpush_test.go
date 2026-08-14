package webpush

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// This file is the browser.
//
// Every failure mode in RFC 8291 encryption is silent from the server's side:
// the body encrypts, the push service accepts it with a 201, and the
// notification simply never appears on the phone. There is no error to log and
// nothing to observe — which makes "it worked when I tried it" the only
// verification most implementations ever get, and a wrong constant can survive
// that for months.
//
// So the test decrypts. It derives the same keys from the client's side of the
// exchange and reads the message back, which is exactly what a browser does and
// the only check that can actually fail when something is wrong.

// browser is a subscription with the private half kept, so the test can decrypt.
type browser struct {
	sub  Subscription
	priv *ecdh.PrivateKey
	auth []byte
}

func newBrowser(t *testing.T) browser {
	t.Helper()
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		t.Fatal(err)
	}
	return browser{
		sub: Subscription{
			Endpoint: "https://push.example/abc",
			P256dh:   base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes()),
			Auth:     base64.RawURLEncoding.EncodeToString(auth),
		},
		priv: priv,
		auth: auth,
	}
}

// decrypt is the browser's half of RFC 8291.
func (b browser) decrypt(t *testing.T, body []byte) []byte {
	t.Helper()
	if len(body) < 21 {
		t.Fatal("body too short to contain a header")
	}
	salt := body[:16]
	idLen := int(body[20])
	serverPub := body[21 : 21+idLen]
	ciphertext := body[21+idLen:]

	remote, err := ecdh.P256().NewPublicKey(serverPub)
	if err != nil {
		t.Fatalf("server public key is not a valid point: %v", err)
	}
	shared, err := b.priv.ECDH(remote)
	if err != nil {
		t.Fatal(err)
	}

	// The client's own key comes first in key_info — the same order the sender
	// must use. This is the assertion that matters most in the whole file.
	keyInfo := append([]byte("WebPush: info\x00"), b.priv.PublicKey().Bytes()...)
	keyInfo = append(keyInfo, serverPub...)
	ikm, err := hkdf.Key(sha256.New, shared, b.auth, string(keyInfo), 32)
	if err != nil {
		t.Fatal(err)
	}
	cek, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		t.Fatal(err)
	}
	nonce, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		t.Fatal(err)
	}

	block, err := aes.NewCipher(cek)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		t.Fatalf("the browser could not decrypt this message: %v", err)
	}
	if len(plain) == 0 || plain[len(plain)-1] != 0x02 {
		t.Fatal("missing the last-record delimiter; the browser waits for a continuation that never comes")
	}
	return plain[:len(plain)-1]
}

// The whole point: a real browser can read what this package writes.
func TestBrowserCanDecryptTheMessage(t *testing.T) {
	b := newBrowser(t)
	want := []byte(`{"title":"Lag'mon 20% chegirma","body":"Bugun kechgacha"}`)

	body, err := encrypt(b.sub, want)
	if err != nil {
		t.Fatal(err)
	}
	if got := b.decrypt(t, body); string(got) != string(want) {
		t.Fatalf("decrypted %q, want %q", got, want)
	}
}

// The RFC 8188 header is fixed-width and in a fixed order. A field written in
// the wrong place produces a body the push service still accepts.
func TestHeaderLayout(t *testing.T) {
	b := newBrowser(t)
	body, err := encrypt(b.sub, []byte("hi"))
	if err != nil {
		t.Fatal(err)
	}
	if rs := binary.BigEndian.Uint32(body[16:20]); rs != 4096 {
		t.Fatalf("record size %d, want 4096", rs)
	}
	if body[20] != 65 {
		t.Fatalf("key length byte is %d, want 65 (an uncompressed P-256 point)", body[20])
	}
	// Two messages must never share a salt or an ephemeral key: reusing either
	// would let the push service link two messages to the same recipient, which
	// is the property the encryption exists to deny it.
	other, err := encrypt(b.sub, []byte("hi"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body[:16]) == string(other[:16]) {
		t.Fatal("two messages shared a salt")
	}
	if string(body[21:86]) == string(other[21:86]) {
		t.Fatal("two messages shared an ephemeral key")
	}
}

// ⚠️ The JWS signature is raw r‖s, not ASN.1 DER.
//
// DER is what `ecdsa.SignASN1` produces and what most Go code reaches for. It
// is the same signature and every push service rejects it with a 401 whose body
// says only "invalid JWT" — an error that points at the key, not at the
// encoding.
func TestVapidSignatureIsRawNotDER(t *testing.T) {
	keys, err := NewKeys("mailto:owner@restaurant.uz")
	if err != nil {
		t.Fatal(err)
	}
	header, err := vapidHeader(keys, "https://fcm.googleapis.com/fcm/send/xyz")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(header, "vapid t=") || !strings.Contains(header, ", k=") {
		t.Fatalf("header %q is not the RFC 8292 form", header)
	}

	token := strings.TrimPrefix(strings.SplitN(header, ", k=", 2)[0], "vapid t=")
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3", len(parts))
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 64 {
		t.Fatalf("signature is %d bytes, want exactly 64 (raw r‖s, not DER)", len(sig))
	}

	// And it verifies against the advertised public key — the check the push
	// service itself performs.
	pubRaw, err := base64.RawURLEncoding.DecodeString(strings.SplitN(header, ", k=", 2)[1])
	if err != nil {
		t.Fatal(err)
	}
	pub, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), pubRaw)
	if err != nil {
		t.Fatalf("advertised public key does not parse: %v", err)
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	var r, s big.Int
	r.SetBytes(sig[:32])
	s.SetBytes(sig[32:])
	if !ecdsa.Verify(pub, sum[:], &r, &s) {
		t.Fatal("the push service would reject this signature")
	}

	// The audience is the origin, not the full endpoint: a token scoped to the
	// whole path is more authority than this needs.
	claims, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(claims, &got); err != nil {
		t.Fatal(err)
	}
	if got["aud"] != "https://fcm.googleapis.com" {
		t.Fatalf("aud is %v, want the origin only", got["aud"])
	}
	if got["sub"] != "mailto:owner@restaurant.uz" {
		t.Fatalf("sub is %v", got["sub"])
	}
}

// ⚠️ 404 and 410 are not failures to retry — they are a dead subscription.
//
// The guest cleared their site data or revoked the permission, and it will
// never work again. Treating it as a transient error means paying for the
// attempt on every campaign for ever, while the "sent" count overstates who was
// actually reached.
func TestGoneSubscriptionIsItsOwnError(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusGone} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(code)
		}))
		b := newBrowser(t)
		b.sub.Endpoint = srv.URL

		keys, err := NewKeys("mailto:a@b.uz")
		if err != nil {
			t.Fatal(err)
		}
		err = Send(srv.Client(), keys, b.sub, []byte("hi"), time.Hour)
		if err != ErrGone {
			t.Fatalf("status %d gave %v, want ErrGone", code, err)
		}
		srv.Close()
	}
}

// A message too large for the record is refused here rather than by the push
// service, where it arrives as a 413 for that one guest and nothing else.
func TestOversizedPayloadIsRefusedLocally(t *testing.T) {
	b := newBrowser(t)
	keys, _ := NewKeys("mailto:a@b.uz")
	err := Send(http.DefaultClient, keys, b.sub, make([]byte, maxPayload+1), time.Hour)
	if err == nil {
		t.Fatal("accepted a payload larger than the record can hold")
	}
	if err == ErrGone {
		t.Fatal("an oversized payload is not a dead subscription")
	}
}

// Browsers are not consistent about padding the base64 in a PushSubscription.
// Refusing a padded key would break push on whichever browser the owner happens
// to test with.
func TestPaddedKeysAreAccepted(t *testing.T) {
	b := newBrowser(t)
	padded := b.sub
	padded.P256dh = base64.StdEncoding.EncodeToString(b.priv.PublicKey().Bytes())
	padded.Auth = base64.StdEncoding.EncodeToString(b.auth)
	// Standard encoding uses + and / rather than - and _; the parts that differ
	// are the padding, which is what this is about.
	padded.P256dh = strings.NewReplacer("+", "-", "/", "_").Replace(padded.P256dh)
	padded.Auth = strings.NewReplacer("+", "-", "/", "_").Replace(padded.Auth)

	body, err := encrypt(padded, []byte("hi"))
	if err != nil {
		t.Fatalf("a padded subscription was refused: %v", err)
	}
	if got := b.decrypt(t, body); string(got) != "hi" {
		t.Fatalf("decrypted %q", got)
	}
}
