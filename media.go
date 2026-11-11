package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
)

type tagInfo struct {
	Title, Artist, Album, AlbumArtist, Genre string
	Compilation                              bool
	Track, Disc, Year                        int
	Duration                                 float64
	Bitrate, SampleRate, BitDepth, Channels  int
	Codec                                    string
	HasCover, HasLyrics                      bool
	// loudness from ReplayGain / R128 tags (LUFS), nil when absent
	LUFS, AlbumLUFS *float64
}

// ReplayGain 2.0 reference level; R128 tags are relative to -23 LUFS.
const rgReference = -18.0

var reGain = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)

// gainToLUFS converts a ReplayGain gain string ("-7.23 dB") to the track loudness in LUFS.
func gainToLUFS(s string) *float64 {
	m := reGain.FindString(s)
	if m == "" {
		return nil
	}
	g := atof(m)
	if g < -60 || g > 60 {
		return nil
	}
	v := rgReference - g
	return &v
}

// r128ToLUFS converts an R128_*_GAIN tag (Q7.8 fixed point, relative to -23 LUFS).
func r128ToLUFS(s string) *float64 {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return nil
	}
	v := -23.0 - float64(n)/256
	return &v
}

type probeOut struct {
	Streams []struct {
		CodecType        string            `json:"codec_type"`
		CodecName        string            `json:"codec_name"`
		SampleRate       string            `json:"sample_rate"`
		Channels         int               `json:"channels"`
		BitsPerRawSample string            `json:"bits_per_raw_sample"`
		BitsPerSample    int               `json:"bits_per_sample"`
		SampleFmt        string            `json:"sample_fmt"`
		BitRate          string            `json:"bit_rate"`
		Duration         string            `json:"duration"`
		Tags             map[string]string `json:"tags"`
		Disposition      map[string]int    `json:"disposition"`
	} `json:"streams"`
	Format struct {
		Duration string            `json:"duration"`
		BitRate  string            `json:"bit_rate"`
		Tags     map[string]string `json:"tags"`
	} `json:"format"`
}

func (a *App) ffprobe(path string) (*probeOut, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 2*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, a.cfg.FFprobe, "-v", "error", "-print_format", "json=string_validation=ignore", "-show_format", "-show_streams", path).Output()
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(out) {
		out = repairSJIS(out)
	}
	var p probeOut
	if err := json.Unmarshal(out, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// repairSJIS converts raw (non-UTF-8) Shift_JIS byte runs in ffprobe output to UTF-8.
// ffmpeg copies ID3v1 tags byte-for-byte, so old Japanese MP3s produce invalid UTF-8.
func repairSJIS(b []byte) []byte {
	out := make([]byte, 0, len(b)+len(b)/2)
	dec := japanese.ShiftJIS.NewDecoder()
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if r != utf8.RuneError || size != 1 {
			out = append(out, b[i:i+size]...)
			i += size
			continue
		}
		j := i
		for j < len(b) && b[j] >= 0x80 {
			c := b[j]
			if (c >= 0x81 && c <= 0x9F) || (c >= 0xE0 && c <= 0xFC) {
				j += 2 // lead + trail byte
			} else {
				j++
			}
		}
		if j > len(b) {
			j = len(b)
		}
		if d, err := dec.Bytes(b[i:j]); err == nil {
			out = append(out, d...)
		} else {
			out = append(out, []byte(string([]rune{utf8.RuneError}))...)
		}
		i = j
	}
	return out
}

// fixText repairs Shift_JIS / double-encoded UTF-8 tags that ffmpeg read as Latin-1.
func fixText(s string) string {
	s = strings.TrimSpace(strings.Trim(s, "\x00"))
	if s == "" {
		return s
	}
	high := false
	for _, r := range s {
		if r > 0xFF {
			return s
		}
		if r >= 0x80 {
			high = true
		}
	}
	if !high {
		return s
	}
	b := make([]byte, 0, len(s))
	for _, r := range s {
		b = append(b, byte(r))
	}
	if utf8.Valid(b) {
		return string(b)
	}
	if dec, err := japanese.ShiftJIS.NewDecoder().Bytes(b); err == nil {
		ds := string(dec)
		if !strings.ContainsRune(ds, utf8.RuneError) && utf8.ValidString(ds) {
			return ds
		}
	}
	return s
}

func firstNum(s string) int {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "/ "); i >= 0 {
		s = s[:i]
	}
	n, _ := strconv.Atoi(s)
	return n
}

var reYear = regexp.MustCompile(`(1[89]\d{2}|20\d{2})`)

func (a *App) readTags(path string) (*tagInfo, error) {
	p, err := a.ffprobe(path)
	if err != nil {
		return nil, err
	}
	tags := map[string]string{}
	add := func(m map[string]string) {
		for k, v := range m {
			k = strings.ToLower(k)
			if _, ok := tags[k]; !ok && strings.TrimSpace(v) != "" {
				tags[k] = v
			}
		}
	}
	add(p.Format.Tags)
	m := &tagInfo{}
	for _, s := range p.Streams {
		switch {
		case s.CodecType == "audio" && m.Codec == "":
			add(s.Tags)
			m.Codec = s.CodecName
			m.SampleRate = atoi(s.SampleRate)
			m.Channels = s.Channels
			m.BitDepth = atoi(s.BitsPerRawSample)
			if m.BitDepth == 0 {
				m.BitDepth = s.BitsPerSample
			}
			if m.BitDepth == 0 && isLossless(s.CodecName) {
				switch {
				case strings.HasPrefix(s.SampleFmt, "s16"):
					m.BitDepth = 16
				case strings.HasPrefix(s.SampleFmt, "s32"), strings.HasPrefix(s.SampleFmt, "flt"):
					m.BitDepth = 24
				}
			}
			if m.Bitrate == 0 {
				m.Bitrate = atoi(s.BitRate) / 1000
			}
			if d := atof(s.Duration); d > 0 {
				m.Duration = d
			}
		case s.CodecType == "video" && (s.Disposition["attached_pic"] == 1 || s.CodecName == "mjpeg" || s.CodecName == "png"):
			m.HasCover = true
		}
	}
	if m.Codec == "" {
		return nil, fmt.Errorf("音声ストリームがありません")
	}
	if d := atof(p.Format.Duration); d > 0 {
		m.Duration = d
	}
	if br := atoi(p.Format.BitRate) / 1000; br > 0 {
		m.Bitrate = br
	}
	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := tags[k]; ok {
				return fixText(v)
			}
		}
		return ""
	}
	m.Title = get("title")
	m.Artist = get("artist", "performer", "artists")
	m.Album = get("album")
	m.AlbumArtist = get("album_artist", "albumartist", "album artist", "band")
	m.Genre = get("genre")
	m.Track = firstNum(get("track", "tracknumber", "trck"))
	m.Disc = firstNum(get("disc", "discnumber", "tpos"))
	if y := reYear.FindString(get("date", "year", "originaldate", "tdrc", "tyer", "release_date")); y != "" {
		m.Year = atoi(y)
	}
	if v := get("replaygain_track_gain"); v != "" {
		m.LUFS = gainToLUFS(v)
	} else if v := get("r128_track_gain"); v != "" {
		m.LUFS = r128ToLUFS(v)
	}
	if v := get("replaygain_album_gain"); v != "" {
		m.AlbumLUFS = gainToLUFS(v)
	} else if v := get("r128_album_gain"); v != "" {
		m.AlbumLUFS = r128ToLUFS(v)
	}
	c := strings.ToLower(get("compilation", "cpil", "tcmp"))
	m.Compilation = c == "1" || c == "true" || c == "yes"
	for k := range tags {
		if strings.HasPrefix(k, "lyrics") || k == "unsyncedlyrics" || k == "uslt" {
			m.HasLyrics = true
		}
	}
	if !m.HasLyrics && lrcFile(path) != "" {
		m.HasLyrics = true
	}
	if m.Title == "" {
		m.Title = titleFromFile(path)
	}
	if m.Track == 0 {
		d, t := trackFromFile(path)
		m.Track = t
		if m.Disc == 0 {
			m.Disc = d
		}
	}
	return m, nil
}

// qualityOf classifies a track: "hires" (lossless and >=24bit or >48kHz, or DSD),
// "lossless", or "" for lossy formats. Follows the JAS Hi-Res Audio definition.
func qualityOf(codec string, bitDepth, sampleRate int) string {
	if !isLossless(codec) {
		return ""
	}
	if strings.HasPrefix(codec, "dsd_") || bitDepth >= 24 || sampleRate > 48000 {
		return "hires"
	}
	return "lossless"
}

// SQL equivalents of isLossless / qualityOf (column prefix t.)
const sqlLossless = `(t.codec IN ('flac','alac','ape','wavpack','tta','tak','mlp','truehd') OR t.codec LIKE 'pcm\_%' ESCAPE '\' OR t.codec LIKE 'dsd\_%' ESCAPE '\')`
const sqlHiRes = `(` + sqlLossless + ` AND (t.bit_depth>=24 OR t.sample_rate>48000 OR t.codec LIKE 'dsd\_%' ESCAPE '\'))`

var reLUFS = regexp.MustCompile(`I:\s+(-?\d+(?:\.\d+)?) LUFS`)

// measureLoudness returns the integrated loudness (EBU R128) of the first audio stream.
func (a *App) measureLoudness(path string) (float64, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, a.cfg.FFmpeg, "-hide_banner", "-nostats", "-nostdin", "-i", path,
		"-map", "0:a:0", "-af", "ebur128=framelog=quiet", "-f", "null", "-").CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("%v: %s", err, lastLine(string(out)))
	}
	ms := reLUFS.FindAllSubmatch(out, -1)
	if len(ms) == 0 {
		return 0, fmt.Errorf("loudness not found in ffmpeg output")
	}
	return atof(string(ms[len(ms)-1][1])), nil
}

func lastLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "\n"); i >= 0 {
		return s[i+1:]
	}
	return s
}

func isLossless(codec string) bool {
	switch codec {
	case "flac", "alac", "ape", "wavpack", "tta", "tak", "mlp", "truehd":
		return true
	}
	return strings.HasPrefix(codec, "pcm_") || strings.HasPrefix(codec, "dsd_")
}

func lrcFile(audio string) string {
	base := strings.TrimSuffix(audio, filepath.Ext(audio))
	for _, ext := range []string{".lrc", ".LRC", ".Lrc"} {
		if fileExists(base + ext) {
			return base + ext
		}
	}
	return ""
}

// ---------- lyrics ----------

type LyricLine struct {
	Time float64 `json:"t"`
	Text string  `json:"text"`
}

type Lyrics struct {
	Synced bool        `json:"synced"`
	Lines  []LyricLine `json:"lines"`
}

var reLrcTime = regexp.MustCompile(`\[(\d{1,3}):(\d{1,2}(?:[.:]\d{1,3})?)\]`)

func decodeText(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}):
		return string(b[3:])
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}):
		s, _ := unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewDecoder().Bytes(b)
		return string(s)
	case bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		s, _ := unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM).NewDecoder().Bytes(b)
		return string(s)
	}
	if utf8.Valid(b) {
		return string(b)
	}
	if s, err := japanese.ShiftJIS.NewDecoder().Bytes(b); err == nil {
		return string(s)
	}
	return string(b)
}

func parseLyrics(text string) *Lyrics {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	l := &Lyrics{Lines: []LyricLine{}}
	var synced []LyricLine
	var plain []LyricLine
	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		ms := reLrcTime.FindAllStringSubmatchIndex(line, -1)
		if len(ms) == 0 {
			t := strings.TrimSpace(line)
			// skip LRC metadata tags like [ar:...]
			if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") && strings.Contains(t, ":") {
				continue
			}
			plain = append(plain, LyricLine{Text: t})
			continue
		}
		rest := strings.TrimSpace(line[ms[len(ms)-1][1]:])
		for _, m := range ms {
			min := atof(line[m[2]:m[3]])
			sec := atof(strings.Replace(line[m[4]:m[5]], ":", ".", 1))
			synced = append(synced, LyricLine{Time: min*60 + sec, Text: rest})
		}
	}
	if len(synced) > 0 {
		sort.SliceStable(synced, func(i, j int) bool { return synced[i].Time < synced[j].Time })
		l.Synced, l.Lines = true, synced
		return l
	}
	// trim leading/trailing blank lines
	for len(plain) > 0 && plain[0].Text == "" {
		plain = plain[1:]
	}
	for len(plain) > 0 && plain[len(plain)-1].Text == "" {
		plain = plain[:len(plain)-1]
	}
	l.Lines = plain
	return l
}

func (a *App) lyricsFor(t *Track) *Lyrics {
	if f := lrcFile(t.Path); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			if l := parseLyrics(decodeText(b)); len(l.Lines) > 0 {
				return l
			}
		}
	}
	p, err := a.ffprobe(t.Path)
	if err != nil {
		return nil
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
		if l := parseLyrics(fixText(tags[k])); len(l.Lines) > 0 {
			return l
		}
	}
	return nil
}

// ---------- cover art ----------

func (a *App) runFF(timeout time.Duration, args ...string) error {
	a.imgSem <- struct{}{}
	defer func() { <-a.imgSem }()
	ctx, cancel := context.WithTimeout(a.ctx, timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, a.cfg.FFmpeg, append([]string{"-hide_banner", "-loglevel", "error", "-nostdin", "-y"}, args...)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// albumCoverSource returns the path of a full-size cover image for an album (extracting embedded art if needed).
func (a *App) albumCoverSource(albumID int64) string {
	var cover string
	var coverTrack int64
	if err := a.db.QueryRow("SELECT cover, cover_track FROM albums WHERE id=?", albumID).Scan(&cover, &coverTrack); err != nil {
		return ""
	}
	if fileExists(cover) {
		return cover
	}
	if coverTrack == 0 {
		return ""
	}
	var path string
	var mtime int64
	if err := a.db.QueryRow("SELECT path, mtime FROM tracks WHERE id=?", coverTrack).Scan(&path, &mtime); err != nil {
		return ""
	}
	out := filepath.Join(a.cfg.DataDir, "cache", "covers", fmt.Sprintf("emb-%d-%d.jpg", coverTrack, mtime))
	if fileExists(out) {
		return out
	}
	if fileExists(out + ".fail") {
		return ""
	}
	unlock := a.locks.lock(out)
	defer unlock()
	if fileExists(out) {
		return out
	}
	tmp := strings.TrimSuffix(out, ".jpg") + ".tmp.jpg"
	if err := a.runFF(60*time.Second, "-i", path, "-map", "0:v:0", "-frames:v", "1", "-q:v", "2", tmp); err != nil || !fileExists(tmp) {
		os.WriteFile(out+".fail", []byte(fmt.Sprint(err)), 0o644)
		return ""
	}
	if os.Rename(tmp, out) != nil {
		return ""
	}
	return out
}

var widthBuckets = []int{96, 200, 300, 500, 800, 1200}

func (a *App) resizedCover(src string, w int) string {
	bw := widthBuckets[len(widthBuckets)-1]
	for _, b := range widthBuckets {
		if w <= b {
			bw = b
			break
		}
	}
	st, err := os.Stat(src)
	if err != nil {
		return ""
	}
	h := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d", src, st.ModTime().UnixNano(), st.Size())))
	out := filepath.Join(a.cfg.DataDir, "cache", "covers", fmt.Sprintf("r-%s-%d.jpg", hex.EncodeToString(h[:8]), bw))
	if fileExists(out) {
		return out
	}
	unlock := a.locks.lock(out)
	defer unlock()
	if fileExists(out) {
		return out
	}
	tmp := strings.TrimSuffix(out, ".jpg") + ".tmp.jpg"
	if err := a.runFF(60*time.Second, "-i", src, "-frames:v", "1", "-vf", fmt.Sprintf("scale=w=min(%d\\,iw):h=-2", bw), "-q:v", "3", tmp); err != nil {
		return ""
	}
	if os.Rename(tmp, out) != nil {
		return ""
	}
	return out
}
