package main

import (
	"database/sql"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS users(
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE COLLATE NOCASE,
  pass TEXT NOT NULL,
  admin INTEGER NOT NULL DEFAULT 0,
  created INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions(
  token TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS libraries(
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  path TEXT NOT NULL,
  created INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS artists(
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE COLLATE NOCASE,
  sort_name TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS albums(
  id INTEGER PRIMARY KEY,
  artist_id INTEGER NOT NULL REFERENCES artists(id),
  title TEXT NOT NULL COLLATE NOCASE,
  sort_title TEXT NOT NULL,
  year INTEGER NOT NULL DEFAULT 0,
  genre TEXT NOT NULL DEFAULT '',
  dir TEXT NOT NULL DEFAULT '',
  cover TEXT NOT NULL DEFAULT '',
  cover_track INTEGER NOT NULL DEFAULT 0,
  added INTEGER NOT NULL,
  updated INTEGER NOT NULL DEFAULT 0,
  UNIQUE(artist_id, title)
);
CREATE TABLE IF NOT EXISTS tracks(
  id INTEGER PRIMARY KEY,
  library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
  album_id INTEGER REFERENCES albums(id),
  artist_id INTEGER REFERENCES artists(id),
  path TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  sort_title TEXT NOT NULL,
  artist TEXT NOT NULL DEFAULT '',
  album TEXT NOT NULL DEFAULT '',
  album_artist TEXT NOT NULL DEFAULT '',
  compilation INTEGER NOT NULL DEFAULT 0,
  track_no INTEGER NOT NULL DEFAULT 0,
  disc_no INTEGER NOT NULL DEFAULT 0,
  year INTEGER NOT NULL DEFAULT 0,
  genre TEXT NOT NULL DEFAULT '',
  duration REAL NOT NULL DEFAULT 0,
  bitrate INTEGER NOT NULL DEFAULT 0,
  codec TEXT NOT NULL DEFAULT '',
  sample_rate INTEGER NOT NULL DEFAULT 0,
  bit_depth INTEGER NOT NULL DEFAULT 0,
  channels INTEGER NOT NULL DEFAULT 0,
  has_cover INTEGER NOT NULL DEFAULT 0,
  has_lyrics INTEGER NOT NULL DEFAULT 0,
  size INTEGER NOT NULL DEFAULT 0,
  mtime INTEGER NOT NULL DEFAULT 0,
  added INTEGER NOT NULL,
  seen_scan INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_tracks_album ON tracks(album_id);
CREATE INDEX IF NOT EXISTS idx_tracks_artist ON tracks(artist_id);
CREATE INDEX IF NOT EXISTS idx_tracks_lib ON tracks(library_id);
CREATE INDEX IF NOT EXISTS idx_albums_artist ON albums(artist_id);
CREATE TABLE IF NOT EXISTS favorites(
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  item_id INTEGER NOT NULL,
  created INTEGER NOT NULL,
  PRIMARY KEY(user_id, kind, item_id)
);
CREATE TABLE IF NOT EXISTS plays(
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  track_id INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
  count INTEGER NOT NULL DEFAULT 0,
  last_played INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY(user_id, track_id)
);
CREATE INDEX IF NOT EXISTS idx_plays_last ON plays(user_id, last_played);
CREATE TABLE IF NOT EXISTS playlists(
  id INTEGER PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  created INTEGER NOT NULL,
  updated INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS playlist_tracks(
  playlist_id INTEGER NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
  pos INTEGER NOT NULL,
  track_id INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
  PRIMARY KEY(playlist_id, pos)
);
`

func openDB(path string) (*sql.DB, error) {
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(15000)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	// additive migrations (ignored when the column already exists)
	for _, m := range []string{
		// loudness in LUFS: from ReplayGain/R128 tags or measured with ffmpeg ebur128
		"ALTER TABLE tracks ADD COLUMN lufs REAL",
		"ALTER TABLE tracks ADD COLUMN album_lufs_tag REAL",
		// 0 = not analysed yet, 1 = done, 2 = analysis failed
		"ALTER TABLE tracks ADD COLUMN lufs_state INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE albums ADD COLUMN lufs REAL",
		// security: two-step verification and session metadata
		"ALTER TABLE users ADD COLUMN totp_secret TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE users ADD COLUMN totp_pending TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE users ADD COLUMN totp_last INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE users ADD COLUMN recovery TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE sessions ADD COLUMN last_seen INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE sessions ADD COLUMN expires INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE sessions ADD COLUMN ip TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE sessions ADD COLUMN user_agent TEXT NOT NULL DEFAULT ''",
	} {
		if _, err := db.Exec(m); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return nil, err
		}
	}
	return db, nil
}

func (a *App) getSetting(key, def string) string {
	var v string
	if err := a.db.QueryRow("SELECT value FROM settings WHERE key=?", key).Scan(&v); err != nil || v == "" {
		return def
	}
	return v
}

func (a *App) setSetting(key, value string) error {
	_, err := a.db.Exec("INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, value)
	return err
}

func idList(ids []int64) (string, []any) {
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i] = "?"
		args[i] = id
	}
	return strings.Join(ph, ","), args
}

func sortKey(s string) string {
	t := strings.ToLower(strings.TrimSpace(s))
	for _, p := range []string{"the ", "a ", "an "} {
		if strings.HasPrefix(t, p) && len(t) > len(p) {
			return t[len(p):]
		}
	}
	return t
}

// ---------- models ----------

type Track struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Artist      string  `json:"artist"`
	ArtistID    int64   `json:"artist_id"`
	Album       string  `json:"album"`
	AlbumID     int64   `json:"album_id"`
	AlbumArtist string  `json:"album_artist"`
	TrackNo     int     `json:"track_no,omitempty"`
	DiscNo      int     `json:"disc_no,omitempty"`
	Year        int     `json:"year,omitempty"`
	Genre       string  `json:"genre,omitempty"`
	Duration    float64 `json:"duration"`
	Bitrate     int     `json:"bitrate"`
	Codec       string  `json:"codec"`
	SampleRate  int     `json:"sample_rate,omitempty"`
	BitDepth    int     `json:"bit_depth,omitempty"`
	Channels    int     `json:"channels,omitempty"`
	Size        int64   `json:"size"`
	Ext         string  `json:"ext"`
	HasLyrics   bool    `json:"has_lyrics,omitempty"`
	CoverV      int64   `json:"cv"`
	Favorite    bool    `json:"fav,omitempty"`
	PlayCount   int     `json:"plays,omitempty"`
	Mime        string  `json:"mime"`
	Quality     string  `json:"q,omitempty"` // hires | lossless | ""
	// loudness (LUFS) for volume normalisation; nil until known
	LUFS      *float64 `json:"lufs,omitempty"`
	AlbumLUFS *float64 `json:"album_lufs,omitempty"`
	Path      string   `json:"-"`
}

// trackSelect selects tracks joined with album info and the user's favorites / play counts.
// The first query argument must be the user id (used twice).
const trackSelect = `SELECT t.id, t.title, t.artist, COALESCE(t.artist_id,0), COALESCE(al.title,''), COALESCE(t.album_id,0),
  COALESCE(aa.name,''), t.track_no, t.disc_no, t.year, t.genre, t.duration, t.bitrate, t.codec, t.sample_rate, t.bit_depth,
  t.channels, t.size, t.path, t.has_lyrics, COALESCE(al.updated,0),
  EXISTS(SELECT 1 FROM favorites f WHERE f.user_id=?1 AND f.kind='track' AND f.item_id=t.id),
  COALESCE((SELECT count FROM plays p WHERE p.user_id=?1 AND p.track_id=t.id),0),
  t.lufs, al.lufs
FROM tracks t LEFT JOIN albums al ON al.id=t.album_id LEFT JOIN artists aa ON aa.id=al.artist_id `

func scanTrack(sc interface{ Scan(...any) error }) (*Track, error) {
	t := &Track{}
	var lyr, fav int
	err := sc.Scan(&t.ID, &t.Title, &t.Artist, &t.ArtistID, &t.Album, &t.AlbumID, &t.AlbumArtist, &t.TrackNo, &t.DiscNo,
		&t.Year, &t.Genre, &t.Duration, &t.Bitrate, &t.Codec, &t.SampleRate, &t.BitDepth, &t.Channels, &t.Size, &t.Path,
		&lyr, &t.CoverV, &fav, &t.PlayCount, &t.LUFS, &t.AlbumLUFS)
	if err != nil {
		return nil, err
	}
	t.HasLyrics, t.Favorite = lyr == 1, fav == 1
	t.Ext = strings.TrimPrefix(strings.ToLower(filepath.Ext(t.Path)), ".")
	t.Mime = mimeFor(t.Ext, t.Codec)
	t.Quality = qualityOf(t.Codec, t.BitDepth, t.SampleRate)
	return t, nil
}

// mimeFor returns a MIME type (with codecs) the browser can test with canPlayType, or "" if
// the file always needs conversion.
func mimeFor(ext, codec string) string {
	switch {
	case codec == "mp3":
		return "audio/mpeg"
	case codec == "aac" && (ext == "m4a" || ext == "m4b" || ext == "mp4"):
		return `audio/mp4; codecs="mp4a.40.2"`
	case codec == "aac" && ext == "aac":
		return "audio/aac"
	case codec == "alac":
		return `audio/mp4; codecs="alac"`
	case codec == "flac" && ext == "flac":
		return "audio/flac"
	case codec == "flac" && (ext == "ogg" || ext == "oga"):
		return `audio/ogg; codecs="flac"`
	case codec == "vorbis" && (ext == "ogg" || ext == "oga"):
		return `audio/ogg; codecs="vorbis"`
	case codec == "opus" && (ext == "ogg" || ext == "opus" || ext == "oga"):
		return `audio/ogg; codecs="opus"`
	case codec == "opus" && ext == "webm":
		return `audio/webm; codecs="opus"`
	case ext == "wav" && (codec == "pcm_s16le" || codec == "pcm_u8" || codec == "pcm_s24le" || codec == "pcm_f32le"):
		return `audio/wav; codecs="1"`
	}
	return ""
}

// queryTracks runs trackSelect + rest. args must not include the user id.
func (a *App) queryTracks(uid int64, rest string, args ...any) ([]*Track, error) {
	rows, err := a.db.Query(trackSelect+rest, append([]any{uid}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Track{}
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (a *App) getTrack(uid, id int64) (*Track, error) {
	ts, err := a.queryTracks(uid, "WHERE t.id=?", id)
	if err != nil {
		return nil, err
	}
	if len(ts) == 0 {
		return nil, sql.ErrNoRows
	}
	return ts[0], nil
}

type Album struct {
	ID         int64   `json:"id"`
	Title      string  `json:"title"`
	ArtistID   int64   `json:"artist_id"`
	Artist     string  `json:"artist"`
	Year       int     `json:"year,omitempty"`
	Genre      string  `json:"genre,omitempty"`
	TrackCount int     `json:"track_count"`
	Duration   float64 `json:"duration"`
	Added      int64   `json:"added"`
	CoverV     int64   `json:"cv"`
	Favorite   bool    `json:"fav,omitempty"`
	Quality    string  `json:"q,omitempty"` // hires | lossless | "" (all tracks must be lossless)
	MaxBits    int     `json:"max_bits,omitempty"`
	MaxRate    int     `json:"max_rate,omitempty"`
}

// albumSelect: first arg is the user id.
const albumSelect = `SELECT al.id, al.title, al.artist_id, ar.name, al.year, al.genre,
  (SELECT COUNT(*) FROM tracks t WHERE t.album_id=al.id), COALESCE((SELECT SUM(duration) FROM tracks t WHERE t.album_id=al.id),0),
  al.added, al.updated,
  EXISTS(SELECT 1 FROM favorites f WHERE f.user_id=?1 AND f.kind='album' AND f.item_id=al.id),
  (SELECT COUNT(*) FROM tracks t WHERE t.album_id=al.id AND NOT ` + sqlLossless + `),
  (SELECT COUNT(*) FROM tracks t WHERE t.album_id=al.id AND ` + sqlHiRes + `),
  COALESCE((SELECT MAX(bit_depth) FROM tracks t WHERE t.album_id=al.id),0), COALESCE((SELECT MAX(sample_rate) FROM tracks t WHERE t.album_id=al.id),0)
FROM albums al JOIN artists ar ON ar.id=al.artist_id `

func (a *App) queryAlbums(uid int64, rest string, args ...any) ([]*Album, error) {
	rows, err := a.db.Query(albumSelect+rest, append([]any{uid}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Album{}
	for rows.Next() {
		al := &Album{}
		var fav, lossy, hires int
		if err := rows.Scan(&al.ID, &al.Title, &al.ArtistID, &al.Artist, &al.Year, &al.Genre, &al.TrackCount, &al.Duration, &al.Added, &al.CoverV, &fav, &lossy, &hires, &al.MaxBits, &al.MaxRate); err != nil {
			return nil, err
		}
		al.Favorite = fav == 1
		if al.TrackCount > 0 && lossy == 0 {
			al.Quality = "lossless"
			if hires > 0 {
				al.Quality = "hires"
			}
		}
		out = append(out, al)
	}
	return out, rows.Err()
}

type Artist struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	AlbumCount int    `json:"album_count"`
	TrackCount int    `json:"track_count"`
	CoverAlbum int64  `json:"cover_album,omitempty"`
	CoverV     int64  `json:"cv"`
	Favorite   bool   `json:"fav,omitempty"`
}

// artistSelect: first arg is the user id.
const artistSelect = `SELECT ar.id, ar.name,
  (SELECT COUNT(*) FROM albums al WHERE al.artist_id=ar.id),
  (SELECT COUNT(*) FROM tracks t WHERE t.artist_id=ar.id),
  COALESCE((SELECT al.id FROM albums al WHERE al.artist_id=ar.id ORDER BY al.year DESC, al.id DESC LIMIT 1),
           (SELECT t.album_id FROM tracks t WHERE t.artist_id=ar.id AND t.album_id IS NOT NULL LIMIT 1), 0),
  EXISTS(SELECT 1 FROM favorites f WHERE f.user_id=?1 AND f.kind='artist' AND f.item_id=ar.id)
FROM artists ar `

func (a *App) queryArtists(uid int64, rest string, args ...any) ([]*Artist, error) {
	rows, err := a.db.Query(artistSelect+rest, append([]any{uid}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Artist{}
	for rows.Next() {
		ar := &Artist{}
		var fav int
		if err := rows.Scan(&ar.ID, &ar.Name, &ar.AlbumCount, &ar.TrackCount, &ar.CoverAlbum, &fav); err != nil {
			return nil, err
		}
		ar.Favorite = fav == 1
		out = append(out, ar)
	}
	return out, rows.Err()
}

type Library struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	Count int    `json:"count"`
}

func (a *App) listLibraries() ([]Library, error) {
	rows, err := a.db.Query("SELECT l.id, l.name, l.path, (SELECT COUNT(*) FROM tracks t WHERE t.library_id=l.id) FROM libraries l ORDER BY l.id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	libs := []Library{}
	for rows.Next() {
		var l Library
		if err := rows.Scan(&l.ID, &l.Name, &l.Path, &l.Count); err != nil {
			return nil, err
		}
		libs = append(libs, l)
	}
	return libs, rows.Err()
}
