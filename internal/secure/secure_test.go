package secure

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"
)

// RFC 6238 Appendix B test vectors (SHA-1, secret "12345678901234567890"), last 6 digits.
func TestTOTPRFC6238(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))
	cases := map[int64]string{
		59:          "287082",
		1111111109:  "081804",
		1111111111:  "050471",
		1234567890:  "005924",
		2000000000:  "279037",
		20000000000: "353130",
	}
	for ts, want := range cases {
		got, err := TOTPCode(secret, time.Unix(ts, 0))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("t=%d: got %s want %s", ts, got, want)
		}
	}
}

func TestVerifyTOTPWindowAndReplay(t *testing.T) {
	s := NewTOTPSecret()
	now := time.Unix(1_800_000_000, 0)
	code, _ := TOTPCode(s, now)
	step, ok := VerifyTOTP(s, code, now, 0)
	if !ok {
		t.Fatal("current code rejected")
	}
	if _, ok := VerifyTOTP(s, code, now, step); ok {
		t.Fatal("replayed code accepted")
	}
	prev, _ := TOTPCode(s, now.Add(-30*time.Second))
	if _, ok := VerifyTOTP(s, prev, now, 0); !ok {
		t.Fatal("previous step (clock drift) rejected")
	}
	old, _ := TOTPCode(s, now.Add(-90*time.Second))
	if _, ok := VerifyTOTP(s, old, now, 0); ok {
		t.Fatal("code from 3 steps ago accepted")
	}
	if _, ok := VerifyTOTP(s, "12345", now, 0); ok {
		t.Fatal("short code accepted")
	}
	if _, ok := VerifyTOTP(s, " "+code[:3]+" "+code[3:]+" ", now.Add(time.Second), 0); !ok {
		t.Fatal("code with spaces rejected")
	}
}

func TestTOTPURI(t *testing.T) {
	u := TOTPURI("Hako", "yu go", "ABC")
	if !strings.HasPrefix(u, "otpauth://totp/Hako:yu%20go?") || !strings.Contains(u, "secret=ABC") || !strings.Contains(u, "issuer=Hako") {
		t.Fatal(u)
	}
}

func TestRecoveryCodes(t *testing.T) {
	codes := NewRecoveryCodes(10)
	seen := map[string]bool{}
	for _, c := range codes {
		if len(c) != 9 || c[4] != '-' {
			t.Fatalf("bad format %q", c)
		}
		if seen[c] {
			t.Fatal("duplicate code")
		}
		seen[c] = true
		if NormalizeRecovery(strings.ToUpper(" "+c[:4]+" "+c[5:])) != NormalizeRecovery(c) {
			t.Fatal("normalize mismatch")
		}
	}
	if HashToken("a") == HashToken("b") || len(HashToken("a")) != 64 {
		t.Fatal("hash")
	}
}

func TestLimiter(t *testing.T) {
	now := time.Unix(1000, 0)
	l := NewLimiter(3, time.Minute, 5*time.Minute)
	l.now = func() time.Time { return now }
	l.Fail("u")
	l.Fail("u")
	if b, _ := l.Blocked("u"); b {
		t.Fatal("blocked too early")
	}
	if !l.Fail("u") {
		t.Fatal("should lock on 3rd failure")
	}
	if b, d := l.Blocked("u"); !b || d != 5*time.Minute {
		t.Fatal("not blocked", d)
	}
	now = now.Add(6 * time.Minute)
	if b, _ := l.Blocked("u"); b {
		t.Fatal("still blocked after lockout")
	}
	// failures outside the window do not accumulate
	l.Fail("v")
	now = now.Add(2 * time.Minute)
	l.Fail("v")
	now = now.Add(2 * time.Minute)
	if l.Fail("v") {
		t.Fatal("old failures counted")
	}
	l.Reset("v")
	l.Cleanup()
}
