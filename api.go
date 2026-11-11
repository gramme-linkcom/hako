package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/setup", a.handleSetupStatus)
	mux.HandleFunc("POST /api/setup", a.handleSetup)
	mux.HandleFunc("POST /api/login", a.handleLogin)
	mux.HandleFunc("POST /api/login/mfa", a.handleLoginMFA)
	mux.HandleFunc("POST /api/logout", a.handleLogout)
	mux.HandleFunc("GET /api/me", a.auth(a.handleMe))
	mux.HandleFunc("POST /api/me/password", a.auth(a.handleMyPassword))
	mux.HandleFunc("GET /api/me/security", a.auth(a.handleSecurity))
	mux.HandleFunc("POST /api/me/totp/setup", a.auth(a.handleTOTPSetup))
	mux.HandleFunc("POST /api/me/totp/enable", a.auth(a.handleTOTPEnable))
	mux.HandleFunc("POST /api/me/totp/disable", a.auth(a.handleTOTPDisable))
	mux.HandleFunc("POST /api/me/recovery", a.auth(a.handleRecoveryRegenerate))
	mux.HandleFunc("DELETE /api/me/sessions/{id}", a.auth(a.handleRevokeSession))
	mux.HandleFunc("POST /api/me/sessions/revoke-others", a.auth(a.handleRevokeOtherSessions))
	mux.HandleFunc("POST /api/users/{id}/totp/reset", a.admin(a.handleResetUserTOTP))

	mux.HandleFunc("GET /api/users", a.admin(a.handleUsers))
	mux.HandleFunc("POST /api/users", a.admin(a.handleCreateUser))
	mux.HandleFunc("DELETE /api/users/{id}", a.admin(a.handleDeleteUser))
	mux.HandleFunc("POST /api/users/{id}/password", a.admin(a.handleSetUserPassword))

	mux.HandleFunc("GET /api/info", a.admin(a.handleInfo))
	mux.HandleFunc("GET /api/libraries", a.admin(a.handleLibraries))
	mux.HandleFunc("POST /api/libraries", a.admin(a.handleCreateLibrary))
	mux.HandleFunc("DELETE /api/libraries/{id}", a.admin(a.handleDeleteLibrary))
	mux.HandleFunc("POST /api/libraries/{id}/scan", a.admin(a.handleScanLibrary))
	mux.HandleFunc("POST /api/scan", a.admin(a.handleScanAll))
	mux.HandleFunc("GET /api/scan", a.auth(a.handleScanStatus))
	mux.HandleFunc("GET /api/fs", a.admin(a.handleFS))

	mux.HandleFunc("GET /api/home", a.auth(a.handleHome))
	mux.HandleFunc("GET /api/search", a.auth(a.handleSearch))
	mux.HandleFunc("GET /api/albums", a.auth(a.handleAlbums))
	mux.HandleFunc("GET /api/albums/{id}", a.auth(a.handleAlbum))
	mux.HandleFunc("GET /api/albums/{id}/cover", a.auth(a.handleCover))
	mux.HandleFunc("GET /api/artists", a.auth(a.handleArtists))
	mux.HandleFunc("GET /api/artists/{id}", a.auth(a.handleArtist))
	mux.HandleFunc("GET /api/tracks", a.auth(a.handleTracks))
	mux.HandleFunc("GET /api/tracks/batch", a.auth(a.handleTrackBatch))
	mux.HandleFunc("GET /api/tracks/{id}", a.auth(a.handleTrack))
	mux.HandleFunc("GET /api/tracks/{id}/lyrics", a.auth(a.handleLyrics))
	mux.HandleFunc("GET /api/tracks/{id}/stream", a.auth(a.handleStream))
	mux.HandleFunc("POST /api/tracks/{id}/scrobble", a.auth(a.handleScrobble))
	mux.HandleFunc("GET /api/favorites", a.auth(a.handleFavorites))
	mux.HandleFunc("POST /api/favorites", a.auth(a.handleSetFavorite))
	mux.HandleFunc("GET /api/playlists", a.auth(a.handlePlaylists))
	mux.HandleFunc("POST /api/playlists", a.auth(a.handleCreatePlaylist))
	mux.HandleFunc("GET /api/playlists/{id}", a.auth(a.handlePlaylist))
	mux.HandleFunc("PUT /api/playlists/{id}", a.auth(a.handleUpdatePlaylist))
	mux.HandleFunc("POST /api/playlists/{id}/tracks", a.auth(a.handleAppendPlaylist))
	mux.HandleFunc("DELETE /api/playlists/{id}", a.auth(a.handleDeletePlaylist))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { httpError(w, 404, "not found") })
	mux.Handle("/", a.static())

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob:; connect-src 'self'; worker-src 'self'; manifest-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
		if isSecure(r) {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		if !sameOrigin(r) {
			httpError(w, 403, "不正なリクエストです（別のサイトからの操作は拒否されます）")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

// sameOrigin blocks cross-site state-changing requests (CSRF). Browsers send
// Sec-Fetch-Site on every request; older ones are checked via the Origin header.
func sameOrigin(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	if s := r.Header.Get("Sec-Fetch-Site"); s != "" {
		return s == "same-origin" || s == "none"
	}
	if o := r.Header.Get("Origin"); o != "" && o != "null" {
		u, err := url.Parse(o)
		if err != nil {
			return false
		}
		host := r.Host
		if trustProxy && r.Header.Get("X-Forwarded-Host") != "" {
			host = r.Header.Get("X-Forwarded-Host")
		}
		return strings.EqualFold(u.Host, host)
	}
	// no browser headers at all (curl, scripts): allowed; cookies are still required
	return true
}

func (a *App) static() http.Handler {
	sub, _ := fs.Sub(webFS, "web")
	fsrv := http.FileServer(http.FS(sub))
	index, _ := fs.ReadFile(sub, "index.html")
	mime.AddExtensionType(".webmanifest", "application/manifest+json")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		st, err := fs.Stat(sub, p)
		if p == "" || p == "index.html" || err != nil || st.IsDir() {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.Write(index)
			return
		}
		if strings.HasPrefix(p, "vendor/") || strings.HasPrefix(p, "icons/") {
			w.Header().Set("Cache-Control", "public, max-age=604800")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		fsrv.ServeHTTP(w, r)
	})
}

func qInt(r *http.Request, k string, def int) int {
	if n, err := strconv.Atoi(r.URL.Query().Get(k)); err == nil {
		return n
	}
	return def
}

func likePattern(q string) string {
	return "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q) + "%"
}

// ---------- admin ----------

func (a *App) handleInfo(w http.ResponseWriter, r *http.Request) {
	var tracks, albums, artists int
	a.db.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&tracks)
	a.db.QueryRow("SELECT COUNT(*) FROM albums").Scan(&albums)
	a.db.QueryRow("SELECT COUNT(*) FROM artists").Scan(&artists)
	var lufsDone, lufsFailed int
	a.db.QueryRow("SELECT COUNT(*) FROM tracks WHERE lufs IS NOT NULL").Scan(&lufsDone)
	a.db.QueryRow("SELECT COUNT(*) FROM tracks WHERE lufs_state=2").Scan(&lufsFailed)
	writeJSON(w, 200, map[string]any{
		"loudness_done": lufsDone, "loudness_failed": lufsFailed,
		"version": version, "ffmpeg": a.ffmpegVersion(), "data_dir": a.cfg.DataDir,
		"scan_interval": int(a.cfg.ScanInterval.Minutes()),
		"tracks":        tracks, "albums": albums, "artists": artists,
	})
}

func (a *App) handleLibraries(w http.ResponseWriter, r *http.Request) {
	libs, err := a.listLibraries()
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, libs)
}

func (a *App) handleCreateLibrary(w http.ResponseWriter, r *http.Request) {
	var l Library
	if err := readJSON(r, &l); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	l.Name = strings.TrimSpace(l.Name)
	l.Path = filepath.Clean(strings.TrimSpace(l.Path))
	if l.Name == "" {
		l.Name = filepath.Base(l.Path)
	}
	if !dirExists(l.Path) {
		httpError(w, 400, "フォルダが見つかりません: "+l.Path)
		return
	}
	res, err := a.db.Exec("INSERT INTO libraries(name,path,created) VALUES(?,?,?)", l.Name, l.Path, time.Now().Unix())
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	l.ID, _ = res.LastInsertId()
	a.scanner.Request(l.ID)
	writeJSON(w, 200, l)
}

func (a *App) handleDeleteLibrary(w http.ResponseWriter, r *http.Request) {
	if _, err := a.db.Exec("DELETE FROM libraries WHERE id=?", pathID(r, "id")); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	if err := a.organize(); err != nil {
		log.Printf("organize: %v", err)
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleScanLibrary(w http.ResponseWriter, r *http.Request) {
	a.scanner.Request(pathID(r, "id"))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleScanAll(w http.ResponseWriter, r *http.Request) {
	a.scanner.Request(0)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleScanStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, a.scanner.Status())
}

func (a *App) handleFS(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	if p == "" {
		p = "/music"
		if !dirExists(p) {
			p = "/"
		}
	}
	p = filepath.Clean(p)
	entries, err := os.ReadDir(p)
	if err != nil {
		httpError(w, 400, "フォルダを開けません: "+err.Error())
		return
	}
	dirs := []string{}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.IsDir() || (e.Type()&os.ModeSymlink != 0 && dirExists(filepath.Join(p, e.Name()))) {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	parent := filepath.Dir(p)
	if parent == p {
		parent = ""
	}
	writeJSON(w, 200, map[string]any{"path": p, "parent": parent, "dirs": dirs})
}

// ---------- browsing ----------

func (a *App) handleHome(w http.ResponseWriter, r *http.Request) {
	uid := currentUser(r).ID
	recent, err := a.queryAlbums(uid, "ORDER BY al.added DESC, al.id DESC LIMIT 24")
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	played, _ := a.queryAlbums(uid, `JOIN (SELECT t.album_id aid, MAX(p.last_played) lp FROM plays p JOIN tracks t ON t.id=p.track_id
		WHERE p.user_id=?1 GROUP BY t.album_id) x ON x.aid=al.id ORDER BY x.lp DESC LIMIT 24`)
	random, _ := a.queryAlbums(uid, "ORDER BY RANDOM() LIMIT 24")
	top, _ := a.queryTracks(uid, "JOIN plays p ON p.track_id=t.id AND p.user_id=?1 WHERE p.count>0 ORDER BY p.count DESC, p.last_played DESC LIMIT 20")
	writeJSON(w, 200, map[string]any{"recent": recent, "played": played, "random": random, "top": top})
}

func (a *App) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	uid := currentUser(r).ID
	if q == "" {
		writeJSON(w, 200, map[string]any{"artists": []any{}, "albums": []any{}, "tracks": []any{}})
		return
	}
	like := likePattern(q)
	artists, _ := a.queryArtists(uid, `WHERE ar.name LIKE ? ESCAPE '\' ORDER BY ar.sort_name LIMIT 20`, like)
	albums, _ := a.queryAlbums(uid, `WHERE al.title LIKE ? ESCAPE '\' OR ar.name LIKE ? ESCAPE '\' ORDER BY al.sort_title LIMIT 40`, like, like)
	tracks, _ := a.queryTracks(uid, `WHERE t.title LIKE ? ESCAPE '\' OR t.artist LIKE ? ESCAPE '\' OR t.album LIKE ? ESCAPE '\'
		ORDER BY CASE WHEN t.title LIKE ? ESCAPE '\' THEN 0 ELSE 1 END, t.sort_title LIMIT 100`, like, like, like, like)
	writeJSON(w, 200, map[string]any{"artists": artists, "albums": albums, "tracks": tracks})
}

func (a *App) handleAlbums(w http.ResponseWriter, r *http.Request) {
	uid := currentUser(r).ID
	q := r.URL.Query()
	where := []string{}
	args := []any{}
	if s := strings.TrimSpace(q.Get("q")); s != "" {
		where = append(where, `(al.title LIKE ? ESCAPE '\' OR ar.name LIKE ? ESCAPE '\')`)
		args = append(args, likePattern(s), likePattern(s))
	}
	if q.Get("fav") == "1" {
		where = append(where, "EXISTS(SELECT 1 FROM favorites f WHERE f.user_id=?1 AND f.kind='album' AND f.item_id=al.id)")
	}
	if g := q.Get("genre"); g != "" {
		where = append(where, "al.genre=?")
		args = append(args, g)
	}
	switch q.Get("quality") {
	case "lossless": // includes hi-res
		where = append(where, "EXISTS(SELECT 1 FROM tracks t WHERE t.album_id=al.id) AND NOT EXISTS(SELECT 1 FROM tracks t WHERE t.album_id=al.id AND NOT "+sqlLossless+")")
	case "hires":
		where = append(where, "EXISTS(SELECT 1 FROM tracks t WHERE t.album_id=al.id AND "+sqlHiRes+") AND NOT EXISTS(SELECT 1 FROM tracks t WHERE t.album_id=al.id AND NOT "+sqlLossless+")")
	}
	order := map[string]string{
		"added":  "al.added DESC, al.id DESC",
		"title":  "al.sort_title",
		"artist": "ar.sort_name, al.year, al.sort_title",
		"year":   "al.year DESC, al.sort_title",
		"random": "RANDOM()",
	}[q.Get("sort")]
	if order == "" {
		order = "al.added DESC, al.id DESC"
	}
	rest := ""
	if len(where) > 0 {
		rest = "WHERE " + strings.Join(where, " AND ") + " "
	}
	rest += fmt.Sprintf("ORDER BY %s LIMIT %d OFFSET %d", order, min(max(qInt(r, "limit", 5000), 1), 5000), max(qInt(r, "offset", 0), 0))
	albums, err := a.queryAlbums(uid, rest, args...)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, albums)
}

func (a *App) handleAlbum(w http.ResponseWriter, r *http.Request) {
	uid := currentUser(r).ID
	albums, err := a.queryAlbums(uid, "WHERE al.id=?", pathID(r, "id"))
	if err != nil || len(albums) == 0 {
		httpError(w, 404, "アルバムが見つかりません")
		return
	}
	tracks, _ := a.queryTracks(uid, "WHERE t.album_id=? ORDER BY t.disc_no, t.track_no, t.path", albums[0].ID)
	writeJSON(w, 200, map[string]any{"album": albums[0], "tracks": tracks})
}

func (a *App) handleCover(w http.ResponseWriter, r *http.Request) {
	src := a.albumCoverSource(pathID(r, "id"))
	if src == "" {
		w.Header().Set("Cache-Control", "private, max-age=600")
		httpError(w, 404, "no cover")
		return
	}
	if width := qInt(r, "w", 0); width > 0 {
		if out := a.resizedCover(src, width); out != "" {
			src = out
		}
	}
	w.Header().Set("Cache-Control", "private, max-age=2592000")
	http.ServeFile(w, r, src)
}

func (a *App) handleArtists(w http.ResponseWriter, r *http.Request) {
	uid := currentUser(r).ID
	where := "WHERE EXISTS(SELECT 1 FROM albums al WHERE al.artist_id=ar.id)"
	if r.URL.Query().Get("all") == "1" {
		where = "WHERE 1=1"
	}
	args := []any{}
	if s := strings.TrimSpace(r.URL.Query().Get("q")); s != "" {
		where += ` AND ar.name LIKE ? ESCAPE '\'`
		args = append(args, likePattern(s))
	}
	if r.URL.Query().Get("fav") == "1" {
		where += " AND EXISTS(SELECT 1 FROM favorites f WHERE f.user_id=?1 AND f.kind='artist' AND f.item_id=ar.id)"
	}
	artists, err := a.queryArtists(uid, where+" ORDER BY ar.sort_name", args...)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, artists)
}

func (a *App) handleArtist(w http.ResponseWriter, r *http.Request) {
	uid := currentUser(r).ID
	id := pathID(r, "id")
	artists, err := a.queryArtists(uid, "WHERE ar.id=?", id)
	if err != nil || len(artists) == 0 {
		httpError(w, 404, "アーティストが見つかりません")
		return
	}
	albums, _ := a.queryAlbums(uid, "WHERE al.artist_id=? ORDER BY al.year DESC, al.sort_title", id)
	appears, _ := a.queryAlbums(uid, "WHERE al.artist_id<>?2 AND EXISTS(SELECT 1 FROM tracks t WHERE t.album_id=al.id AND t.artist_id=?2) ORDER BY al.year DESC", id)
	top, _ := a.queryTracks(uid, `LEFT JOIN plays p ON p.track_id=t.id AND p.user_id=?1 WHERE t.artist_id=?2 OR al.artist_id=?2
		ORDER BY COALESCE(p.count,0) DESC, t.year DESC, t.disc_no, t.track_no LIMIT 10`, id)
	all, _ := a.queryTracks(uid, "WHERE t.artist_id=?2 OR al.artist_id=?2 ORDER BY al.year, al.sort_title, t.disc_no, t.track_no", id)
	ids := make([]int64, len(all))
	for i, t := range all {
		ids[i] = t.ID
	}
	writeJSON(w, 200, map[string]any{"artist": artists[0], "albums": albums, "appears_on": appears, "top": top, "track_ids": ids})
}

func (a *App) handleTracks(w http.ResponseWriter, r *http.Request) {
	uid := currentUser(r).ID
	q := r.URL.Query()
	where := []string{}
	args := []any{}
	join := ""
	if s := strings.TrimSpace(q.Get("q")); s != "" {
		where = append(where, `(t.title LIKE ? ESCAPE '\' OR t.artist LIKE ? ESCAPE '\' OR t.album LIKE ? ESCAPE '\')`)
		args = append(args, likePattern(s), likePattern(s), likePattern(s))
	}
	if q.Get("fav") == "1" {
		where = append(where, "EXISTS(SELECT 1 FROM favorites f WHERE f.user_id=?1 AND f.kind='track' AND f.item_id=t.id)")
	}
	switch q.Get("quality") {
	case "lossless":
		where = append(where, sqlLossless)
	case "hires":
		where = append(where, sqlHiRes)
	}
	order := map[string]string{
		"title":  "t.sort_title",
		"artist": "t.artist COLLATE NOCASE, al.sort_title, t.disc_no, t.track_no",
		"album":  "al.sort_title, t.disc_no, t.track_no",
		"added":  "t.added DESC, t.id DESC",
		"random": "RANDOM()",
		"plays":  "COALESCE(pl.count,0) DESC, t.sort_title",
		"recent": "COALESCE(pl.last_played,0) DESC",
	}[q.Get("sort")]
	if order == "" {
		order = "t.sort_title"
	}
	if q.Get("sort") == "plays" || q.Get("sort") == "recent" {
		join = "LEFT JOIN plays pl ON pl.track_id=t.id AND pl.user_id=?1 "
		if q.Get("sort") == "recent" {
			where = append(where, "pl.last_played > 0")
		}
	}
	rest := join
	if len(where) > 0 {
		rest += "WHERE " + strings.Join(where, " AND ") + " "
	}
	limit := min(max(qInt(r, "limit", 200), 1), 2000)
	offset := max(qInt(r, "offset", 0), 0)
	rest += fmt.Sprintf("ORDER BY %s LIMIT %d OFFSET %d", order, limit, offset)
	tracks, err := a.queryTracks(uid, rest, args...)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	var total int
	countQ := "SELECT COUNT(*) FROM tracks t LEFT JOIN albums al ON al.id=t.album_id " + strings.ReplaceAll(join, "?1", "?1")
	if len(where) > 0 {
		countQ += "WHERE " + strings.Join(where, " AND ")
	}
	a.db.QueryRow(countQ, append([]any{uid}, args...)...).Scan(&total)
	writeJSON(w, 200, map[string]any{"tracks": tracks, "total": total})
}

func (a *App) handleTrackBatch(w http.ResponseWriter, r *http.Request) {
	uid := currentUser(r).ID
	var ids []int64
	for _, s := range strings.Split(r.URL.Query().Get("ids"), ",") {
		if n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
			ids = append(ids, n)
		}
	}
	if len(ids) > 2000 {
		ids = ids[:2000]
	}
	out := []*Track{}
	if len(ids) > 0 {
		byID := map[int64]*Track{}
		for start := 0; start < len(ids); start += 500 {
			chunk := ids[start:min(start+500, len(ids))]
			ph, args := idList(chunk)
			// shift placeholders so ?1 stays the user id
			ts, err := a.queryTracks(uid, "WHERE t.id IN ("+ph+")", args...)
			if err != nil {
				httpError(w, 500, err.Error())
				return
			}
			for _, t := range ts {
				byID[t.ID] = t
			}
		}
		for _, id := range ids {
			if t := byID[id]; t != nil {
				out = append(out, t)
			}
		}
	}
	writeJSON(w, 200, out)
}

func (a *App) handleTrack(w http.ResponseWriter, r *http.Request) {
	t, err := a.getTrack(currentUser(r).ID, pathID(r, "id"))
	if err != nil {
		httpError(w, 404, "曲が見つかりません")
		return
	}
	writeJSON(w, 200, map[string]any{"track": t, "file": filepath.Base(t.Path)})
}

func (a *App) handleLyrics(w http.ResponseWriter, r *http.Request) {
	t, err := a.getTrack(currentUser(r).ID, pathID(r, "id"))
	if err != nil {
		httpError(w, 404, "曲が見つかりません")
		return
	}
	l := a.lyricsFor(t)
	if l == nil {
		l = &Lyrics{Lines: []LyricLine{}}
	}
	writeJSON(w, 200, l)
}

func (a *App) handleScrobble(w http.ResponseWriter, r *http.Request) {
	uid := currentUser(r).ID
	id := pathID(r, "id")
	_, err := a.db.Exec(`INSERT INTO plays(user_id,track_id,count,last_played) VALUES(?,?,1,?)
		ON CONFLICT(user_id,track_id) DO UPDATE SET count=count+1, last_played=excluded.last_played`, uid, id, time.Now().UnixMilli())
	if err != nil {
		httpError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------- favorites ----------

func (a *App) handleFavorites(w http.ResponseWriter, r *http.Request) {
	uid := currentUser(r).ID
	tracks, _ := a.queryTracks(uid, "JOIN favorites f ON f.user_id=?1 AND f.kind='track' AND f.item_id=t.id ORDER BY f.created DESC")
	albums, _ := a.queryAlbums(uid, "JOIN favorites f ON f.user_id=?1 AND f.kind='album' AND f.item_id=al.id ORDER BY f.created DESC")
	artists, _ := a.queryArtists(uid, "JOIN favorites f ON f.user_id=?1 AND f.kind='artist' AND f.item_id=ar.id ORDER BY f.created DESC")
	writeJSON(w, 200, map[string]any{"tracks": tracks, "albums": albums, "artists": artists})
}

func (a *App) handleSetFavorite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind string `json:"kind"`
		ID   int64  `json:"id"`
		Fav  bool   `json:"fav"`
	}
	if err := readJSON(r, &body); err != nil || (body.Kind != "track" && body.Kind != "album" && body.Kind != "artist") {
		httpError(w, 400, "invalid request")
		return
	}
	uid := currentUser(r).ID
	var err error
	if body.Fav {
		_, err = a.db.Exec("INSERT OR IGNORE INTO favorites(user_id,kind,item_id,created) VALUES(?,?,?,?)", uid, body.Kind, body.ID, time.Now().UnixMilli())
	} else {
		_, err = a.db.Exec("DELETE FROM favorites WHERE user_id=? AND kind=? AND item_id=?", uid, body.Kind, body.ID)
	}
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"fav": body.Fav})
}

// ---------- playlists ----------

type Playlist struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	TrackCount int     `json:"track_count"`
	Duration   float64 `json:"duration"`
	Updated    int64   `json:"updated"`
	Covers     []int64 `json:"covers"`
}

func (a *App) listPlaylists(uid int64, where string, args ...any) ([]*Playlist, error) {
	rows, err := a.db.Query(`SELECT p.id, p.name, p.updated,
		(SELECT COUNT(*) FROM playlist_tracks pt WHERE pt.playlist_id=p.id),
		COALESCE((SELECT SUM(t.duration) FROM playlist_tracks pt JOIN tracks t ON t.id=pt.track_id WHERE pt.playlist_id=p.id),0)
		FROM playlists p WHERE p.user_id=? `+where+` ORDER BY p.updated DESC`, append([]any{uid}, args...)...)
	if err != nil {
		return nil, err
	}
	out := []*Playlist{}
	for rows.Next() {
		p := &Playlist{Covers: []int64{}}
		if err := rows.Scan(&p.ID, &p.Name, &p.Updated, &p.TrackCount, &p.Duration); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, p)
	}
	rows.Close()
	for _, p := range out {
		cr, err := a.db.Query(`SELECT DISTINCT t.album_id FROM playlist_tracks pt JOIN tracks t ON t.id=pt.track_id
			WHERE pt.playlist_id=? AND t.album_id IS NOT NULL ORDER BY pt.pos LIMIT 4`, p.ID)
		if err == nil {
			for cr.Next() {
				var id int64
				cr.Scan(&id)
				p.Covers = append(p.Covers, id)
			}
			cr.Close()
		}
	}
	return out, nil
}

func (a *App) handlePlaylists(w http.ResponseWriter, r *http.Request) {
	pls, err := a.listPlaylists(currentUser(r).ID, "")
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, pls)
}

func (a *App) ownsPlaylist(uid, id int64) bool {
	var n int
	a.db.QueryRow("SELECT COUNT(*) FROM playlists WHERE id=? AND user_id=?", id, uid).Scan(&n)
	return n == 1
}

func (a *App) setPlaylistTracks(id int64, ids []int64, appendOnly bool) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	start := 0
	if appendOnly {
		tx.QueryRow("SELECT COALESCE(MAX(pos)+1,0) FROM playlist_tracks WHERE playlist_id=?", id).Scan(&start)
	} else if _, err := tx.Exec("DELETE FROM playlist_tracks WHERE playlist_id=?", id); err != nil {
		return err
	}
	for i, tid := range ids {
		if _, err := tx.Exec("INSERT INTO playlist_tracks(playlist_id,pos,track_id) SELECT ?,?,id FROM tracks WHERE id=?", id, start+i, tid); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("UPDATE playlists SET updated=? WHERE id=?", time.Now().UnixMilli(), id); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *App) handleCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string  `json:"name"`
		TrackIDs []int64 `json:"track_ids"`
	}
	if err := readJSON(r, &body); err != nil || strings.TrimSpace(body.Name) == "" {
		httpError(w, 400, "名前を入力してください")
		return
	}
	now := time.Now().UnixMilli()
	res, err := a.db.Exec("INSERT INTO playlists(user_id,name,created,updated) VALUES(?,?,?,?)", currentUser(r).ID, strings.TrimSpace(body.Name), now, now)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	if len(body.TrackIDs) > 0 {
		if err := a.setPlaylistTracks(id, body.TrackIDs, true); err != nil {
			httpError(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, 200, map[string]int64{"id": id})
}

func (a *App) handlePlaylist(w http.ResponseWriter, r *http.Request) {
	uid := currentUser(r).ID
	id := pathID(r, "id")
	pls, err := a.listPlaylists(uid, "AND p.id=?", id)
	if err != nil || len(pls) == 0 {
		httpError(w, 404, "プレイリストが見つかりません")
		return
	}
	tracks, err := a.queryTracks(uid, "JOIN playlist_tracks pt ON pt.track_id=t.id WHERE pt.playlist_id=? ORDER BY pt.pos", id)
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"playlist": pls[0], "tracks": tracks})
}

func (a *App) handleUpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	uid := currentUser(r).ID
	id := pathID(r, "id")
	if !a.ownsPlaylist(uid, id) {
		httpError(w, 404, "プレイリストが見つかりません")
		return
	}
	var body struct {
		Name     *string  `json:"name"`
		TrackIDs *[]int64 `json:"track_ids"`
	}
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	if body.Name != nil && strings.TrimSpace(*body.Name) != "" {
		a.db.Exec("UPDATE playlists SET name=?, updated=? WHERE id=?", strings.TrimSpace(*body.Name), time.Now().UnixMilli(), id)
	}
	if body.TrackIDs != nil {
		if err := a.setPlaylistTracks(id, *body.TrackIDs, false); err != nil {
			httpError(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleAppendPlaylist(w http.ResponseWriter, r *http.Request) {
	uid := currentUser(r).ID
	id := pathID(r, "id")
	if !a.ownsPlaylist(uid, id) {
		httpError(w, 404, "プレイリストが見つかりません")
		return
	}
	var body struct {
		TrackIDs []int64 `json:"track_ids"`
	}
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	if err := a.setPlaylistTracks(id, body.TrackIDs, true); err != nil {
		httpError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *App) handleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	a.db.Exec("DELETE FROM playlists WHERE id=? AND user_id=?", pathID(r, "id"), currentUser(r).ID)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------- streaming ----------

var audioMime = map[string]string{
	".mp3": "audio/mpeg", ".m4a": "audio/mp4", ".m4b": "audio/mp4", ".aac": "audio/aac", ".flac": "audio/flac",
	".ogg": "audio/ogg", ".oga": "audio/ogg", ".opus": "audio/ogg", ".wav": "audio/wav", ".webm": "audio/webm",
}

// handleStream serves a track. mode=direct serves the original file (with Range support);
// otherwise ffmpeg converts it to MP3 and the result is sent progressively as it is produced.
// Seeking in converted streams is done by the client re-requesting with ?start=<seconds>.
func (a *App) handleStream(w http.ResponseWriter, r *http.Request) {
	t, err := a.getTrack(currentUser(r).ID, pathID(r, "id"))
	if err != nil {
		httpError(w, 404, "曲が見つかりません")
		return
	}
	q := r.URL.Query()
	if q.Get("mode") == "direct" || q.Get("dl") == "1" {
		f, err := os.Open(t.Path)
		if err != nil {
			httpError(w, 404, "ファイルを開けません")
			return
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			httpError(w, 500, err.Error())
			return
		}
		ct := audioMime[strings.ToLower(filepath.Ext(t.Path))]
		if ct == "" {
			ct = "application/octet-stream"
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "private, max-age=86400")
		if q.Get("dl") == "1" {
			w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filepath.Base(t.Path)))
		}
		http.ServeContent(w, r, "", st.ModTime(), f)
		return
	}

	br := qInt(r, "br", 320)
	switch {
	case br <= 96:
		br = 96
	case br <= 128:
		br = 128
	case br <= 192:
		br = 192
	case br <= 256:
		br = 256
	default:
		br = 320
	}
	start := atof(q.Get("start"))
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin"}
	if start > 0 {
		args = append(args, "-ss", strconv.FormatFloat(start, 'f', 3, 64))
	}
	args = append(args, "-i", t.Path, "-map", "0:a:0", "-vn", "-sn", "-map_metadata", "-1",
		"-c:a", "libmp3lame", "-b:a", fmt.Sprintf("%dk", br), "-ar", "44100", "-ac", "2", "-write_xing", "0", "-id3v2_version", "0",
		"-f", "mp3", "pipe:1")
	cmd := exec.CommandContext(r.Context(), a.cfg.FFmpeg, args...)
	stderr := &limitedBuffer{n: 2048}
	cmd.Stderr = stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		httpError(w, 500, err.Error())
		return
	}
	if err := cmd.Start(); err != nil {
		httpError(w, 500, "ffmpegを起動できません: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Accept-Ranges", "none")
	w.Header().Set("X-Accel-Buffering", "no")
	rc := http.NewResponseController(w)
	rc.SetWriteDeadline(time.Time{})
	w.WriteHeader(200)
	buf := make([]byte, 32*1024)
	var sent int64
	for {
		n, rerr := out.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				break
			}
			sent += int64(n)
			rc.Flush()
		}
		if rerr != nil {
			if rerr != io.EOF {
				log.Printf("stream read: %v", rerr)
			}
			break
		}
	}
	cmd.Process.Kill()
	if err := cmd.Wait(); err != nil && r.Context().Err() == nil && sent == 0 {
		log.Printf("ffmpeg stream failed (%s): %v\n%s", t.Path, err, stderr.String())
	}
}
