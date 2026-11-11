// Package secure holds the pure (I/O-free) security helpers: TOTP, recovery codes and
// a login rate limiter. Kept separate so it can be unit-tested on any platform.
package secure

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	totpPeriod = 30
	totpDigits = 6
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewTOTPSecret returns a random 160-bit secret, base32 encoded (RFC 4226 recommendation).
func NewTOTPSecret() string {
	b := make([]byte, 20)
	rand.Read(b)
	return b32.EncodeToString(b)
}

// TOTPURI builds the otpauth:// URI understood by authenticator apps.
func TOTPURI(issuer, account, secret string) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(totpDigits))
	q.Set("period", fmt.Sprint(totpPeriod))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// hotp implements RFC 4226 with SHA-1 and 6 digits.
func hotp(key []byte, counter uint64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	m := hmac.New(sha1.New, key)
	m.Write(msg[:])
	sum := m.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	code := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", totpDigits, code%1000000)
}

// TOTPCode returns the code for the given time (exported for tests).
func TOTPCode(secret string, t time.Time) (string, error) {
	key, err := b32.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", err
	}
	return hotp(key, uint64(t.Unix())/totpPeriod), nil
}

// VerifyTOTP checks code against the secret allowing ±1 time step of clock drift.
// lastStep is the last step that was accepted for this user; codes for that step or
// earlier are rejected so a code cannot be replayed. Returns the matched step.
func VerifyTOTP(secret, code string, now time.Time, lastStep int64) (int64, bool) {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(code) != totpDigits {
		return 0, false
	}
	key, err := b32.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil || len(key) == 0 {
		return 0, false
	}
	cur := now.Unix() / totpPeriod
	for _, d := range []int64{0, -1, 1} {
		step := cur + d
		if step <= lastStep {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(hotp(key, uint64(step))), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}

// ---------- recovery codes ----------

const recoveryAlphabet = "abcdefghjkmnpqrstuvwxyz23456789" // no look-alike characters

// NewRecoveryCodes returns n human-friendly one-time codes like "k7m2-x9qp".
func NewRecoveryCodes(n int) []string {
	out := make([]string, n)
	for i := range out {
		b := make([]byte, 8)
		rand.Read(b)
		var s strings.Builder
		for j, c := range b {
			if j == 4 {
				s.WriteByte('-')
			}
			s.WriteByte(recoveryAlphabet[int(c)%len(recoveryAlphabet)])
		}
		out[i] = s.String()
	}
	return out
}

// NormalizeRecovery lower-cases and strips separators/spaces so users can type loosely.
func NormalizeRecovery(code string) string {
	code = strings.ToLower(code)
	var s strings.Builder
	for _, r := range code {
		if strings.ContainsRune(recoveryAlphabet, r) {
			s.WriteRune(r)
		}
	}
	return s.String()
}

// HashToken returns the hex SHA-256 of a secret value (session tokens, recovery codes).
func HashToken(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
