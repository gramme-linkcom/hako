package main

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Lyrics made with the built-in editor are stored in <data>/lyrics/<hash of the audio path>.lrc,
// so the music folder can stay read-only. They take priority over a .lrc next to the file and
// over lyrics embedded in tags.

const maxLyricsBytes = 256 << 10

func (a *App) editedLyricsPath(audioPath string) string {
	h := sha256.Sum256([]byte(audioPath))
	return filepath.Join(a.cfg.DataDir, "lyrics", hex.EncodeToString(h[:16])+".lrc")
}

func (a *App) editedLyricsFile(audioPath string) string {
	if p := a.editedLyricsPath(audioPath); fileExists(p) {
		return p
	}
	return ""
}

// rawLyrics returns the lyrics text of a track and where it came from: "hako" (editor),
// "file" (.lrc next to the audio file), "tag" (embedded) or "" (none).
func (a *App) rawLyrics(t *Track) (string, string) {
	if f := a.editedLyricsFile(t.Path); f != "" {
		if b, err := os.ReadFile(f); err == nil && strings.TrimSpace(string(b)) != "" {
			return decodeText(b), "hako"
		}
	}
	if f := lrcFile(t.Path); f != "" {
		if b, err := os.ReadFile(f); err == nil && strings.TrimSpace(decodeText(b)) != "" {
			return decodeText(b), "file"
		}
	}
	p, err := a.ffprobe(t.Path)
	if err != nil {
		return "", ""
	}
	tags := map[string]string{}
	for k, v := range p.Format.Tags {
		tags[strings.ToLower(k)] = v
	}
	for _, s := range p.Streams {
		if s.CodecType == "audio" {
			for k, v := range s.Tags {
				tags[strings.ToLower(k)] = v
			}
		}
	}
	keys := make([]string, 0)
	for k := range tags {
		if strings.HasPrefix(k, "lyrics") || k == "unsyncedlyrics" || k == "uslt" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		if v := fixText(tags[k]); strings.TrimSpace(v) != "" {
			return v, "tag"
		}
	}
	return "", ""
}

// GET /api/tracks/{id}/lyrics/source — the unparsed text, for the editor
func (a *App) handleLyricsSource(w http.ResponseWriter, r *http.Request) {
	t, err := a.getTrack(currentUser(r).ID, pathID(r, "id"))
	if err != nil {
		httpError(w, 404, "曲が見つかりません")
		return
	}
	text, src := a.rawLyrics(t)
	l := parseLyrics(text)
	writeJSON(w, 200, map[string]any{"text": text, "source": src, "synced": l.Synced})
}

// PUT /api/tracks/{id}/lyrics {"text": "..."} — save editor lyrics (empty text = delete them)
func (a *App) handleSaveLyrics(w http.ResponseWriter, r *http.Request) {
	t, err := a.getTrack(currentUser(r).ID, pathID(r, "id"))
	if err != nil {
		httpError(w, 404, "曲が見つかりません")
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "invalid request")
		return
	}
	if len(body.Text) > maxLyricsBytes {
		httpError(w, 400, "歌詞が長すぎます")
		return
	}
	text := strings.ReplaceAll(strings.ReplaceAll(body.Text, "\r\n", "\n"), "\r", "\n")
	p := a.editedLyricsPath(t.Path)
	if strings.TrimSpace(text) == "" {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			httpError(w, 500, err.Error())
			return
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		tmp := p + ".tmp"
		if err := os.WriteFile(tmp, []byte(text), 0o644); err != nil {
			httpError(w, 500, err.Error())
			return
		}
		if err := os.Rename(tmp, p); err != nil {
			os.Remove(tmp)
			httpError(w, 500, err.Error())
			return
		}
	}
	// keep the has_lyrics flag in sync (other sources may still provide lyrics after a delete)
	raw, _ := a.rawLyrics(t)
	has := len(parseLyrics(raw).Lines) > 0
	a.db.Exec("UPDATE tracks SET has_lyrics=? WHERE id=?", b2i(has), t.ID)
	log.Printf("lyrics: %s saved lyrics for track %d (%d bytes)", currentUser(r).Name, t.ID, len(text))
	l := parseLyrics(raw)
	writeJSON(w, 200, l)
}
