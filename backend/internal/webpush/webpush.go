// Package webpush sends a browser push notification.
//
// Three channels reach a customer, and they are not interchangeable:
//
//   - **SMS** reaches everybody and costs real money on every message.
//   - **Telegram** is free and reaches only the guests who opened the bot.
//   - **Web push** is free and reaches the guests who allowed notifications on
//     the site — which is a different set of people again, and the only one
//     that works on a desktop browser with no phone involved.
//
// ⚠️ **Written against the RFC rather than pulled from a library, on purpose.**
// Everything needed is in the standard library as of Go 1.24 (`crypto/ecdh`,
// `crypto/hkdf`), and this is about two hundred lines. The same trade the image
// resizer made: a dependency that reaches the network on behalf of every tenant
// is a dependency whose next version has to be trusted, and there is nothing
// here that a library would do differently.
//
// The pieces, and the two specifications they come from:
//
//   - **RFC 8291** — how the payload is encrypted so that the push service
//     (Google, Mozilla, Apple) carries it without being able to read it. This
//     matters more here than it sounds: the message is a restaurant's promotion
//     addressed to a named customer.
//   - **RFC 8292 (VAPID)** — how the server proves to the push service that it
//     is the same server that the subscription was issued to.
package webpush

import (
	"bytes"
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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Subscription is what the browser handed the page when the guest allowed
// notifications. Stored verbatim; none of it is ours to interpret.
type Subscription struct {
	// Where the push service wants the message. The whole URL is the address
	// *and* the identity of this subscription — it is unguessable and unique,
	// which is why it is the natural unique key in the database.
	Endpoint string
	// The browser's public key (uncompressed P-256 point) and the shared
	// authentication secret, both base64url without padding.
	P256dh string
	Auth   string
}

// Keys is the server's VAPID identity.
//
// One pair per restaurant, generated once and then never changed: the push
// services tie every existing subscription to the public key it was created
// with, so rotating these silently invalidates every subscription the
// restaurant has ever collected — and nothing reports it, the messages simply
// stop arriving.
type Keys struct {
	// Base64url, unpadded. The public key is genuinely public — it is handed to
	// every visitor's browser, exactly like the map key (§ "Xarita kaliti sir
	// emas"). The private key is not, and never leaves the server.
	Public  string
	Private string
	// "mailto:..." or an https URL, so the push service has somebody to contact
	// about a misbehaving sender. Required by RFC 8292.
	Subject string
}

// ErrGone means the subscription is dead and should be deleted.
//
// ⚠️ Worth its own error because the correct response is *not* to retry. A push
// service answers 404 or 410 when the guest cleared their site data, revoked
// the permission or uninstalled the browser — the subscription will never work
// again, and keeping it means paying for the attempt on every campaign, for
// ever, while the "sent" count quietly overstates who was reached.
var ErrGone = errors.New("webpush: subscription is gone")

// maxPayload is the largest message body the specification guarantees a push
// service will carry.
//
// ⚠️ 4096 bytes is the *encrypted record* size, and the encryption adds 103
// bytes of overhead (16 salt, 4 length, 1 id length, 65 public key, 16 GCM tag,
// 1 delimiter). A payload that fits by the naive measure is rejected by the
// push service with a 413 — which arrives as a silent failure for that one
// guest.
const maxPayload = 4096 - 103

// NewKeys generates a fresh VAPID identity.
func NewKeys(subject string) (Keys, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return Keys{}, err
	}
	// The uncompressed point and the raw scalar — exactly the two wire formats
	// VAPID names, and the reason these are stored as bytes rather than as PEM:
	// the public half is handed to a browser, which wants precisely this.
	pub, err := priv.PublicKey.Bytes()
	if err != nil {
		return Keys{}, err
	}
	raw, err := priv.Bytes()
	if err != nil {
		return Keys{}, err
	}
	return Keys{
		Public:  b64(pub),
		Private: b64(raw),
		Subject: subject,
	}, nil
}

// Send delivers one encrypted message to one subscription.
//
// `ttl` is how long the push service should hold the message for a browser that
// is currently offline.
func Send(client *http.Client, keys Keys, sub Subscription, payload []byte, ttl time.Duration) error {
	if len(payload) > maxPayload {
		return fmt.Errorf("webpush: payload is %d bytes, the limit is %d", len(payload), maxPayload)
	}

	body, err := encrypt(sub, payload)
	if err != nil {
		return err
	}
	auth, err := vapidHeader(keys, sub.Endpoint)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, sub.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", strconv.Itoa(int(ttl.Seconds())))
	req.Header.Set("Authorization", auth)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Read and discard so the connection can be reused: a campaign sends
	// hundreds of these in a row.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return ErrGone
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	}
	return fmt.Errorf("webpush: push service answered %d", resp.StatusCode)
}

// encrypt builds the aes128gcm body of RFC 8291.
func encrypt(sub Subscription, payload []byte) ([]byte, error) {
	clientPub, err := unb64(sub.P256dh)
	if err != nil {
		return nil, fmt.Errorf("webpush: bad p256dh: %w", err)
	}
	authSecret, err := unb64(sub.Auth)
	if err != nil {
		return nil, fmt.Errorf("webpush: bad auth secret: %w", err)
	}

	curve := ecdh.P256()
	remote, err := curve.NewPublicKey(clientPub)
	if err != nil {
		return nil, fmt.Errorf("webpush: bad client key: %w", err)
	}
	// A fresh key pair per message. Reusing one would let the push service link
	// two messages to the same recipient, which is the property the encryption
	// exists to deny it.
	local, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	shared, err := local.ECDH(remote)
	if err != nil {
		return nil, err
	}
	localPub := local.PublicKey().Bytes()

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// RFC 8291 §3.4. Two HKDF passes, and the order of the two public keys in
	// `key_info` is fixed by the spec: **client first**. Swapping them produces
	// a body that encrypts perfectly and that no browser on earth can decrypt —
	// the send succeeds, the push service accepts it, and the notification
	// never appears.
	keyInfo := append([]byte("WebPush: info\x00"), clientPub...)
	keyInfo = append(keyInfo, localPub...)
	ikm, err := hkdf.Key(sha256.New, shared, authSecret, string(keyInfo), 32)
	if err != nil {
		return nil, err
	}
	cek, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	nonce, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// 0x02 marks the last record. A single record is all that is ever sent
	// here, but the delimiter is not optional — without it the browser waits
	// for a continuation that never comes.
	plaintext := append(append([]byte(nil), payload...), 0x02)
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// The RFC 8188 header, then the body.
	out := make([]byte, 0, 16+4+1+len(localPub)+len(ciphertext))
	out = append(out, salt...)
	out = binary.BigEndian.AppendUint32(out, 4096)
	out = append(out, byte(len(localPub)))
	out = append(out, localPub...)
	return append(out, ciphertext...), nil
}

// vapidHeader builds the Authorization header of RFC 8292.
func vapidHeader(keys Keys, endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	// The audience is the push service's **origin**, not the full endpoint. A
	// token scoped to the whole path would still be accepted today and is
	// simply more authority than this needs.
	aud := u.Scheme + "://" + u.Host

	priv, err := privateKey(keys.Private)
	if err != nil {
		return "", err
	}

	header := b64([]byte(`{"typ":"JWT","alg":"ES256"}`))
	claims, err := json.Marshal(map[string]any{
		"aud": aud,
		// Twelve hours. The spec caps this at 24; a shorter life means a
		// leaked token from a captured request stops working the same day.
		"exp": time.Now().Add(12 * time.Hour).Unix(),
		"sub": keys.Subject,
	})
	if err != nil {
		return "", err
	}
	signing := header + "." + b64(claims)

	sum := sha256.Sum256([]byte(signing))
	r, s, err := ecdsa.Sign(rand.Reader, priv, sum[:])
	if err != nil {
		return "", err
	}
	// ⚠️ JWS wants the raw 64-byte r‖s, **not** the ASN.1 DER encoding that
	// `ecdsa.SignASN1` produces. DER is what most Go code reaches for, it is
	// the same signature, and every push service rejects it with a 401 that
	// says only "invalid JWT".
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])

	return "vapid t=" + signing + "." + b64(sig) + ", k=" + keys.Public, nil
}

// privateKey rebuilds the signing key from its stored scalar.
func privateKey(encoded string) (*ecdsa.PrivateKey, error) {
	raw, err := unb64(encoded)
	if err != nil {
		return nil, err
	}
	// Parsed rather than reassembled from the raw scalar: the parser validates
	// the range and derives the public half itself, which is the part a
	// hand-rolled version gets subtly wrong.
	return ecdsa.ParseRawPrivateKey(elliptic.P256(), raw)
}

// b64 is base64url without padding — the encoding every part of this
// specification uses.
func b64(in []byte) string {
	return base64.RawURLEncoding.EncodeToString(in)
}

// unb64 decodes base64url, tolerating the padding some browsers include.
//
// ⚠️ Not strictness for its own sake: the `p256dh` and `auth` values come
// straight from a browser's `PushSubscription`, and they are not consistent
// about padding across engines. Refusing a padded key would break push on
// whichever browser the restaurant's owner happens to test with.
func unb64(in string) ([]byte, error) {
	in = strings.TrimRight(strings.TrimSpace(in), "=")
	return base64.RawURLEncoding.DecodeString(in)
}
