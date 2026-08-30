package fiscal

import "testing"

// ⚠️ **Every ready provider must be configurable from the panel**, and this is
// the test that says so. The drawer used to decide which boxes to show from the
// transport flag — local means no account — which was true of exactly one
// provider and wrong about the next two: REGOS is local and wants a login and
// password, E-POS is local and wants a token. The symptom was not an error
// anywhere; it was a provider that could be selected, saved and enabled, and
// never authenticated, because the box holding its credential was not on screen.
func TestEveryReadyProviderAsksForWhatItsAdapterNeeds(t *testing.T) {
	full := Creds{
		Login: "kassa", Password: "secret", RegisterID: "1",
		Token: "tok", BaseURL: "http://192.168.1.50:9090",
	}

	for _, p := range Providers() {
		if len(p.Needs) == 0 {
			t.Fatalf("%s asks for nothing at all — even an address is needed", p.ID)
		}
		for _, n := range p.Needs {
			switch n {
			case NeedLogin, NeedPassword, NeedRegisterID, NeedToken, NeedBaseURL:
			default:
				t.Fatalf("%s asks for %q, which the panel cannot draw", p.ID, n)
			}
		}
		if !p.Ready {
			continue
		}

		// Drop each box the provider says it does not need; the adapter must
		// still build. A provider whose adapter needs a credential its drawer
		// never shows is one an owner cannot configure.
		c := full
		has := func(n string) bool {
			for _, v := range p.Needs {
				if v == n {
					return true
				}
			}
			return false
		}
		if !has(NeedLogin) {
			c.Login = ""
		}
		if !has(NeedPassword) {
			c.Password = ""
		}
		if !has(NeedRegisterID) {
			c.RegisterID = ""
		}
		if !has(NeedToken) {
			c.Token = ""
		}
		if _, err := EncoderFor(p.ID, c); err != nil {
			t.Fatalf("%s cannot be built from the boxes its drawer shows (%v): %v",
				p.ID, p.Needs, err)
		}
	}
}
