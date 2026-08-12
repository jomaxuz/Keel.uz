package config

import "testing"

// The signing key is the whole authentication boundary, so the weak defaults
// that must never sign a real token are sealed here — including the exact
// strings the dev fallback and the compose files ship.
func TestValidateRefusesWeakJWTSecret(t *testing.T) {
	for _, weak := range []string{"", "change-me", "change-me-in-production", "secret", "short"} {
		c := &Config{JWTSecret: weak}
		if err := c.Validate(); err == nil {
			t.Errorf("Validate accepted weak secret %q", weak)
		}
	}
	// A real 32-byte hex secret passes.
	c := &Config{JWTSecret: "9f8b1c2d3e4f5a6b7c8d9e0f1a2b3c4d"}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate rejected a strong secret: %v", err)
	}
}
