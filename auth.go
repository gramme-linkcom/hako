package main

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"hako/internal/secure"

	qrcode "github.com/skip2/go-qrcode"
)

const (
	cookieName      = "hako_session"
	pbkdfIter       = 600000 // OWASP 2023 recommendation for PBKDF2-HMAC-SHA256
	minPasswordLen  = 8
	sessionLifetime = 90 * 24 * time.Hour
	mfaTicketTTL    = 5 * time.Minute
	recoveryCount   = 10
)

var (
	// 5 wrong attempts for one account within 15 min -> locked for 15 min
	userLimiter = secure.NewLimiter(5, 15*time.Minute, 15*time.Minute)
	// 20 wrong attempts from one IP within 15 min -> locked for 30 min
	ipLimiter = secure.NewLimiter(20, 15*time.Minute, 30*time.Minute)
	// used to make "unknown user" take as long as "wrong password"
	dummyHash, _ = hashPassword("hako-dummy-password")
	trustProxy   = os.Getenv("HAKO_TRUST_PROXY") == "1" || strings.EqualFold(os.Getenv("HAKO_TRUST_PROXY"), "true")
)

type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Admin bool   `json:"admin"`
	TOTP  bool   `json:"totp"`
}

func randToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ---------- passwords ----------

func hashPassword(pw string) (string, error) {
	salt := make([]byte, 16)
	rand.Read(salt)
	k, err := pbkdf2.Key(sha256.New, pw, salt, pbkdfIter, 32)
	if err != nil {
		return "", err
	}
	enc := base64.RawStdEncoding
	return fmt.Sprintf("pbkdf2$%d$%s$%s", pbkdfIter, enc.EncodeToString(salt), enc.EncodeToString(k)), nil
}

// checkPassword verifies pw; needsRehash is true when the stored hash uses fewer iterations.
func checkPassword(stored, pw string) (ok, needsRehash bool) {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2" {
		return false, false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 1000 {
		return false, false
	}
	enc := base64.RawStdEncoding
	salt, err1 := enc.DecodeString(parts[2])
	want, err2 := enc.DecodeString(parts[3])
	if err1 != nil || err2 != nil {
		return false, false
	}
	got, err := pbkdf2.Key(sha256.New, pw, salt, iter, len(want))
	if err != nil {
		return false, false
	}
	ok = subtle.ConstantTimeCompare(got, want) == 1
	return ok, ok && iter < pbkdfIter
}

func validPassword(pw string) string {
	if len([]rune(pw)) < minPasswordLen {
		return fmt.Sprintf("パスワードは%d文字以上にしてください", minPasswordLen)
	}
	if len(pw) > 256 {
		return "パスワードが長すぎます"
	}
	return ""
}

func (a *App) setPassword(uid int64, pw string) error {
	hash, err := hashPassword(pw)
	if err != nil {
		return err
	}
	_, err = a.db.Exec("UPDATE users SET pass=? WHERE id=?", hash, uid)
	return err
}

func (a *App) verifyUserPassword(uid int64, pw string) bool {
	var hash string
	if a.db.QueryRow("SELECT pass FROM users WHERE id=?", uid).Scan(&hash) != nil {
		return false
	}
	ok, _ := checkPassword(hash, pw)
	return ok
}

// ---------- request helpers ----------

type ctxKey int

const (
	userKey ctxKey = iota
	sessionKey
)

func currentUser(r *http.Request) *User {
	u, _ := r.Context().Value(userKey).(*User)
	return u
}

func currentSession(r *http.Request) int64 {
	id, _ := r.Context().Value(sessionKey).(int64)
	return id
}

func clientIP(r *http.Request) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			first, _, _ := strings.Cut(xff, ",")
			if ip := strings.TrimSpace(first); net.ParseIP(ip) != nil {
				return ip
			}
		}
		if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(ip) != nil {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isSecure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// ---------- sessions ----------

// Session tokens are random 256-bit values; only their SHA-256 is stored, so a leaked
// database cannot be used to hijack sessions.
func (a *App) userFromRequest(r *http.Request) (*User, int64) {
	c, err := r.Cookie(cookieName)
	if err != nil || len(c.Value) != 64 {
		return nil, 0
	}
	h := secure.HashToken(c.Value)
	u := &User{}
	var admin int
	var sid, expires, lastSeen int64
	var totp string
	err = a.db.QueryRow(`SELECT s.rowid, s.expires, s.last_seen, u.id, u.name, u.admin, u.totp_secret
		FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token=?`, h).Scan(&sid, &expires, &lastSeen, &u.ID, &u.Name, &admin, &totp)
	if err != nil {
		return nil, 0
	}
	now := time.Now()
	if expires < now.Unix() {
		a.db.Exec("DELETE FROM sessions WHERE rowid=?", sid)
		return nil, 0
	}
	// sliding expiry, updated at most every 5 minutes
	if now.Unix()-lastSeen > 300 {
		a.db.Exec("UPDATE sessions SET last_seen=?, expires=?, ip=?, user_agent=? WHERE rowid=?",
			now.Unix(), now.Add(sessionLifetime).Unix(), clientIP(r), truncate(r.UserAgent(), 300), sid)
	}
	u.Admin, u.TOTP = admin == 1, totp != ""
	return u, sid
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func (a *App) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, sid := a.userFromRequest(r)
		if u == nil {
			httpError(w, 401, "ログインが必要です")
			return
		}
		ctx := context.WithValue(context.WithValue(r.Context(), userKey, u), sessionKey, sid)
		h(w, r.WithContext(ctx))
	}
}

func (a *App) admin(h http.HandlerFunc) http.HandlerFunc {
	return a.auth(func(w http.ResponseWriter, r *http.Request) {
		if !currentUser(r).Admin {
			httpError(w, 403, "管理者権限が必要です")
			return
		}
		h(w, r)
	})
}

func (a *App) startSession(w http.ResponseWriter, r *http.Request, uid int64) error {
	tok := randToken()
	now := time.Now()
	_, err := a.db.Exec("INSERT INTO sessions(token,user_id,created,last_seen,expires,ip,user_agent) VALUES(?,?,?,?,?,?,?)",
		secure.HashToken(tok), uid, now.Unix(), now.Unix(), now.Add(sessionLifetime).Unix(), clientIP(r), truncate(r.UserAgent(), 300))
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: tok, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: isSecure(r), MaxAge: int(sessionLifetime.Seconds()),
	})
	return nil
}

// revokeSessions deletes all sessions of uid except keep (0 = delete all).
func (a *App) revokeSessions(uid, keep int64) {
	a.db.Exec("DELETE FROM sessions WHERE user_id=? AND rowid<>?", uid, keep)
}

func (a *App) securityLoop() {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-t.C:
			userLimiter.Cleanup()
			ipLimiter.Cleanup()
			tickets.cleanup()
			a.db.Exec("DELETE FROM sessions WHERE expires < ?", time.Now().Unix())
		}
	}
}

// ---------- two-step login tickets ----------

type mfaTicket struct {
	uid     int64
	name    string
	expires time.Time
	tries   int
}

type ticketStore struct {
	mu sync.Mutex
	m  map[string]*mfaTicket
}

var tickets = &ticketStore{m: map[string]*mfaTicket{}}

func (s *ticketStore) create(uid int64, name string) string {
	tok := randToken()
	s.mu.Lock()
	s.m[tok] = &mfaTicket{uid: uid, name: name, expires: time.Now().Add(mfaTicketTTL)}
	s.mu.Unlock()
	return tok
}

func (s *ticketStore) get(tok string) *mfaTicket {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.m[tok]
	if t == nil || time.Now().After(t.expires) {
		delete(s.m, tok)
		return nil
	}
	return t
}

// fail counts a wrong code for the ticket and returns the number of failures so far.
func (s *ticketStore) fail(tok string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t := s.m[tok]; t != nil {
		t.tries++
		return t.tries
	}
	return 99
}

func (s *ticketStore) remove(tok string) {
	s.mu.Lock()
	delete(s.m, tok)
	s.mu.Unlock()
}

func (s *ticketStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, t := range s.m {
		if now.After(t.expires) {
			delete(s.m, k)
		}
	}
}

// ---------- rate limiting ----------

func lockedMessage(d time.Duration) string {
	m := int(d.Minutes() + 0.999)
	if m < 1 {
		m = 1
	}
	return fmt.Sprintf("ログインの失敗が続いたため一時的にロックしています。%d分後にもう一度お試しください", m)
}

func checkLimits(w http.ResponseWriter, ip, name string) bool {
	if b, d := ipLimiter.Blocked(ip); b {
		w.Header().Set("Retry-After", strconv.Itoa(int(d.Seconds())+1))
		httpError(w, 429, lockedMessage(d))
		return false
	}
	if name != "" {
		if b, d := userLimiter.Blocked(strings.ToLower(name)); b {
			w.Header().Set("Retry-After", strconv.Itoa(int(d.Seconds())+1))
			httpError(w, 429, lockedMessage(d))
			return false
		}
	}
	return true
}

func recordFailure(ip, name string) {
	ipLimiter.Fail(ip)
	if name != "" {
		if userLimiter.Fail(strings.ToLower(name)) {
			log.Printf("security: account %q locked after repeated failed logins (last from %s)", name, ip)
		}
	}
}

// ---------- setup / login / logout ----------

func (a *App) userCount() int {
	var n int
	a.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&n)
	return n
}

func (a *App) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"needs_setup": a.userCount() == 0, "version": version, "min_password": minPasswordLen})
}

type credentials struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Admin    bool   `json:"admin"`
}

func (c *credentials) validate() string {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" || len(c.Name) > 64 {
		return "ユーザー名を入力してください"
	}
	return validPassword(c.Password)
}

func (a *App) createUser(c credentials) (int64, error) {
	hash, err := hashPassword(c.Password)
	if err != nil {
		return 0, err
	}
	res, err := a.db.Exec("INSERT INTO users(name,pass,admin,created) VALUES(?,?,?,?)", c.Name, hash, b2i(c.Admin), time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

var setupMu sync.Mutex

func (a *App) handleSetup(w http.ResponseWriter, r *http.Request) {
	setupMu.Lock()
	defer setupMu.Unlock()
	if a.userCount() > 0 {
		httpError(w, 403, "セットアップは完了しています")
		return
	}
	var c credentials
	if err := readJSON(r, &c); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	if msg := c.validate(); msg != "" {
		httpError(w, 400, msg)
		return
	}
	c.Admin = true
	id, err := a.createUser(c)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	if err := a.startSession(w, r, id); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, User{ID: id, Name: c.Name, Admin: true})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := readJSON(r, &c); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	name := strings.TrimSpace(c.Name)
	ip := clientIP(r)
	if !checkLimits(w, ip, name) {
		return
	}
	var u User
	var hash, totp string
	var admin int
	err := a.db.QueryRow("SELECT id,name,pass,admin,totp_secret FROM users WHERE name=?", name).Scan(&u.ID, &u.Name, &hash, &admin, &totp)
	ok, rehash := false, false
	if err == nil {
		ok, rehash = checkPassword(hash, c.Password)
	} else {
		checkPassword(dummyHash, c.Password) // equalise timing for unknown users
	}
	if !ok {
		recordFailure(ip, name)
		log.Printf("security: failed login for %q from %s", name, ip)
		time.Sleep(300 * time.Millisecond)
		httpError(w, 401, "ユーザー名またはパスワードが違います")
		return
	}
	if rehash {
		a.setPassword(u.ID, c.Password)
	}
	u.Admin, u.TOTP = admin == 1, totp != ""
	if u.TOTP {
		writeJSON(w, 200, map[string]any{"mfa_required": true, "ticket": tickets.create(u.ID, u.Name)})
		return
	}
	userLimiter.Reset(strings.ToLower(name))
	if err := a.startSession(w, r, u.ID); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, u)
}

// verifySecondFactor accepts a 6-digit TOTP code or a one-time recovery code.
// It updates the replay guard / consumes the recovery code on success.
func (a *App) verifySecondFactor(uid int64, code string) (ok bool, usedRecovery bool, left int) {
	var secret, recovery string
	var last int64
	if a.db.QueryRow("SELECT totp_secret, totp_last, recovery FROM users WHERE id=?", uid).Scan(&secret, &last, &recovery) != nil || secret == "" {
		return false, false, 0
	}
	digits := strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(digits) == 6 && strings.Trim(digits, "0123456789") == "" {
		if step, ok := secure.VerifyTOTP(secret, digits, time.Now(), last); ok {
			a.db.Exec("UPDATE users SET totp_last=? WHERE id=?", step, uid)
			return true, false, 0
		}
		return false, false, 0
	}
	norm := secure.NormalizeRecovery(code)
	if len(norm) != 8 {
		return false, false, 0
	}
	var hashes []string
	json.Unmarshal([]byte(recovery), &hashes)
	h := secure.HashToken(norm)
	for i, x := range hashes {
		if subtle.ConstantTimeCompare([]byte(x), []byte(h)) == 1 {
			hashes = append(hashes[:i], hashes[i+1:]...)
			b, _ := json.Marshal(hashes)
			a.db.Exec("UPDATE users SET recovery=? WHERE id=?", string(b), uid)
			return true, true, len(hashes)
		}
	}
	return false, false, 0
}

func (a *App) handleLoginMFA(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Ticket string `json:"ticket"`
		Code   string `json:"code"`
	}
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	ip := clientIP(r)
	t := tickets.get(body.Ticket)
	if t == nil {
		httpError(w, 401, "時間切れです。もう一度ログインしてください")
		return
	}
	if !checkLimits(w, ip, t.name) {
		tickets.remove(body.Ticket)
		return
	}
	ok, usedRecovery, left := a.verifySecondFactor(t.uid, body.Code)
	if !ok {
		recordFailure(ip, t.name)
		if tickets.fail(body.Ticket) >= 5 {
			tickets.remove(body.Ticket)
			httpError(w, 401, "確認コードの入力に何度も失敗しました。もう一度ログインしてください")
			return
		}
		log.Printf("security: wrong 2FA code for %q from %s", t.name, ip)
		httpError(w, 400, "確認コードが違います")
		return
	}
	tickets.remove(body.Ticket)
	userLimiter.Reset(strings.ToLower(t.name))
	if err := a.startSession(w, r, t.uid); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	u, _ := a.getUser(t.uid)
	resp := map[string]any{"id": u.ID, "name": u.Name, "admin": u.Admin, "totp": true}
	if usedRecovery {
		resp["recovery_left"] = left
		log.Printf("security: %q signed in with a recovery code (%d left)", u.Name, left)
	}
	writeJSON(w, 200, resp)
}

func (a *App) getUser(id int64) (*User, error) {
	u := &User{}
	var admin int
	var totp string
	err := a.db.QueryRow("SELECT id,name,admin,totp_secret FROM users WHERE id=?", id).Scan(&u.ID, &u.Name, &admin, &totp)
	if err != nil {
		return nil, err
	}
	u.Admin, u.TOTP = admin == 1, totp != ""
	return u, nil
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		a.db.Exec("DELETE FROM sessions WHERE token=?", secure.HashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isSecure(r)})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, currentUser(r))
}

func (a *App) handleMyPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	u := currentUser(r)
	ip := clientIP(r)
	if !checkLimits(w, ip, u.Name) {
		return
	}
	if !a.verifyUserPassword(u.ID, body.Current) {
		recordFailure(ip, u.Name)
		httpError(w, 400, "現在のパスワードが違います")
		return
	}
	if msg := validPassword(body.New); msg != "" {
		httpError(w, 400, msg)
		return
	}
	if err := a.setPassword(u.ID, body.New); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	a.revokeSessions(u.ID, currentSession(r))
	log.Printf("security: %q changed their password; other sessions revoked", u.Name)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------- security settings (2FA, sessions) ----------

type sessionInfo struct {
	ID        int64  `json:"id"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Created   int64  `json:"created"`
	LastSeen  int64  `json:"last_seen"`
	Current   bool   `json:"current"`
}

func (a *App) handleSecurity(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var recovery string
	a.db.QueryRow("SELECT recovery FROM users WHERE id=?", u.ID).Scan(&recovery)
	var hashes []string
	json.Unmarshal([]byte(recovery), &hashes)
	rows, err := a.db.Query("SELECT rowid, ip, user_agent, created, last_seen FROM sessions WHERE user_id=? AND expires>=? ORDER BY last_seen DESC", u.ID, time.Now().Unix())
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	defer rows.Close()
	list := []sessionInfo{}
	cur := currentSession(r)
	for rows.Next() {
		var s sessionInfo
		rows.Scan(&s.ID, &s.IP, &s.UserAgent, &s.Created, &s.LastSeen)
		s.Current = s.ID == cur
		list = append(list, s)
	}
	writeJSON(w, 200, map[string]any{"totp": u.TOTP, "recovery_left": len(hashes), "sessions": list, "min_password": minPasswordLen})
}

func (a *App) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	a.db.Exec("DELETE FROM sessions WHERE rowid=? AND user_id=?", pathID(r, "id"), u.ID)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleRevokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	a.revokeSessions(currentUser(r).ID, currentSession(r))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// requirePassword checks the "password" field for sensitive operations (with rate limiting).
func (a *App) requirePassword(w http.ResponseWriter, r *http.Request, pw string) bool {
	u := currentUser(r)
	ip := clientIP(r)
	if !checkLimits(w, ip, u.Name) {
		return false
	}
	if !a.verifyUserPassword(u.ID, pw) {
		recordFailure(ip, u.Name)
		httpError(w, 400, "パスワードが違います")
		return false
	}
	return true
}

func (a *App) handleTOTPSetup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	if !a.requirePassword(w, r, body.Password) {
		return
	}
	u := currentUser(r)
	secret := secure.NewTOTPSecret()
	if _, err := a.db.Exec("UPDATE users SET totp_pending=? WHERE id=?", secret, u.ID); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	uri := secure.TOTPURI("Hako", u.Name, secret)
	png, err := qrcode.Encode(uri, qrcode.Medium, 512)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{
		"secret": secret,
		"uri":    uri,
		"qr":     "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
	})
}

func (a *App) newRecoveryCodes(uid int64) ([]string, error) {
	codes := secure.NewRecoveryCodes(recoveryCount)
	hashes := make([]string, len(codes))
	for i, c := range codes {
		hashes[i] = secure.HashToken(secure.NormalizeRecovery(c))
	}
	b, _ := json.Marshal(hashes)
	_, err := a.db.Exec("UPDATE users SET recovery=? WHERE id=?", string(b), uid)
	return codes, err
}

func (a *App) handleTOTPEnable(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	u := currentUser(r)
	var pending string
	a.db.QueryRow("SELECT totp_pending FROM users WHERE id=?", u.ID).Scan(&pending)
	if pending == "" {
		httpError(w, 400, "最初からやり直してください")
		return
	}
	step, ok := secure.VerifyTOTP(pending, body.Code, time.Now(), 0)
	if !ok {
		httpError(w, 400, "確認コードが違います。認証アプリに表示されている6桁の数字を入力してください")
		return
	}
	if _, err := a.db.Exec("UPDATE users SET totp_secret=?, totp_pending='', totp_last=? WHERE id=?", pending, step, u.ID); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	codes, err := a.newRecoveryCodes(u.ID)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	a.revokeSessions(u.ID, currentSession(r))
	log.Printf("security: %q enabled two-step verification", u.Name)
	writeJSON(w, 200, map[string]any{"recovery_codes": codes})
}

func (a *App) handleTOTPDisable(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	if !a.requirePassword(w, r, body.Password) {
		return
	}
	u := currentUser(r)
	if ok, _, _ := a.verifySecondFactor(u.ID, body.Code); !ok {
		recordFailure(clientIP(r), u.Name)
		httpError(w, 400, "確認コードが違います")
		return
	}
	a.db.Exec("UPDATE users SET totp_secret='', totp_pending='', totp_last=0, recovery='' WHERE id=?", u.ID)
	log.Printf("security: %q disabled two-step verification", u.Name)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleRecoveryRegenerate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	if !a.requirePassword(w, r, body.Password) {
		return
	}
	u := currentUser(r)
	if !u.TOTP {
		httpError(w, 400, "2段階認証が有効になっていません")
		return
	}
	codes, err := a.newRecoveryCodes(u.ID)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"recovery_codes": codes})
}

// ---------- user admin ----------

func (a *App) handleUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query("SELECT id,name,admin,totp_secret<>'' FROM users ORDER BY id")
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		var u User
		var admin, totp int
		rows.Scan(&u.ID, &u.Name, &admin, &totp)
		u.Admin, u.TOTP = admin == 1, totp == 1
		users = append(users, u)
	}
	writeJSON(w, 200, users)
}

func (a *App) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := readJSON(r, &c); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	if msg := c.validate(); msg != "" {
		httpError(w, 400, msg)
		return
	}
	id, err := a.createUser(c)
	if err != nil {
		httpError(w, 400, "そのユーザー名は既に使われています")
		return
	}
	writeJSON(w, 200, User{ID: id, Name: c.Name, Admin: c.Admin})
}

func (a *App) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	if id == currentUser(r).ID {
		httpError(w, 400, "自分自身は削除できません")
		return
	}
	a.db.Exec("DELETE FROM users WHERE id=?", id)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleSetUserPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	if msg := validPassword(body.Password); msg != "" {
		httpError(w, 400, msg)
		return
	}
	id := pathID(r, "id")
	if err := a.setPassword(id, body.Password); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	keep := int64(0)
	if id == currentUser(r).ID {
		keep = currentSession(r)
	}
	a.revokeSessions(id, keep)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleResetUserTOTP(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "id")
	res, err := a.db.Exec("UPDATE users SET totp_secret='', totp_pending='', totp_last=0, recovery='' WHERE id=?", id)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		httpError(w, 404, "ユーザーが見つかりません")
		return
	}
	keep := int64(0)
	if id == currentUser(r).ID {
		keep = currentSession(r)
	}
	a.revokeSessions(id, keep)
	log.Printf("security: admin %q reset two-step verification of user %d", currentUser(r).Name, id)
	writeJSON(w, 200, map[string]bool{"ok": true})
}
