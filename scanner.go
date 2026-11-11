package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var audioExts = map[string]bool{
	".mp3": true, ".flac": true, ".m4a": true, ".aac": true, ".ogg": true, ".oga": true, ".opus": true,
	".wav": true, ".wma": true, ".ape": true, ".alac": true, ".aiff": true, ".aif": true, ".wv": true,
	".mpc": true, ".dsf": true, ".dff": true, ".mka": true, ".tta": true, ".tak": true, ".m4b": true, ".webm": true,
}

var skipDirs = map[string]bool{"@eadir": true, "#recycle": true, "$recycle.bin": true, "system volume information": true, "lost+found": true, "#snapshot": true}

const unknownArtist = "不明なアーティスト"
const variousArtists = "Various Artists"

type ScanStatus struct {
	Running      bool   `json:"running"`
	Library      string `json:"library"`
	Phase        string `json:"phase"`
	Done         int    `json:"done"`
	Total        int    `json:"total"`
	LastFinished int64  `json:"last_finished"`
	LastError    string `json:"last_error,omitempty"`
}

type Scanner struct {
	a     *App
	mu    sync.Mutex
	st    ScanStatus
	queue chan int64
}

func newScanner(a *App) *Scanner { return &Scanner{a: a, queue: make(chan int64, 32)} }

func (s *Scanner) Request(libID int64) {
	select {
	case s.queue <- libID:
	default:
	}
}

func (s *Scanner) Status() ScanStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st
}

func (s *Scanner) phase(p string, done, total int) {
	s.mu.Lock()
	s.st.Phase, s.st.Done, s.st.Total = p, done, total
	s.mu.Unlock()
}

func (s *Scanner) loop() {
	s.Request(0)
	var tick <-chan time.Time
	if s.a.cfg.ScanInterval > 0 {
		t := time.NewTicker(s.a.cfg.ScanInterval)
		defer t.Stop()
		tick = t.C
	}
	for {
		select {
		case <-s.a.ctx.Done():
			return
		case id := <-s.queue:
			ids := map[int64]bool{id: true}
		drain:
			for {
				select {
				case x := <-s.queue:
					ids[x] = true
				default:
					break drain
				}
			}
			if ids[0] {
				s.run(0)
			} else {
				for x := range ids {
					s.run(x)
				}
			}
		case <-tick:
			s.run(0)
		}
	}
}

func (s *Scanner) run(libID int64) {
	libs, err := s.a.listLibraries()
	if err != nil {
		log.Printf("scan: %v", err)
		return
	}
	s.mu.Lock()
	s.st.Running, s.st.LastError = true, ""
	s.mu.Unlock()
	var errs []string
	for _, l := range libs {
		if (libID != 0 && l.ID != libID) || s.a.ctx.Err() != nil {
			continue
		}
		s.mu.Lock()
		s.st.Library = l.Name
		s.mu.Unlock()
		start := time.Now()
		if err := s.scanLibrary(l); err != nil {
			log.Printf("scan %q: %v", l.Name, err)
			errs = append(errs, l.Name+": "+err.Error())
		} else {
			log.Printf("scan %q done in %s", l.Name, time.Since(start).Round(time.Millisecond))
		}
	}
	s.phase("アルバムを整理中", 0, 0)
	if err := s.a.organize(); err != nil {
		log.Printf("organize: %v", err)
		errs = append(errs, err.Error())
	}
	if err := s.analyzeLoudness(); err != nil {
		log.Printf("loudness: %v", err)
	}
	s.mu.Lock()
	s.st = ScanStatus{LastFinished: time.Now().Unix(), LastError: strings.Join(errs, " / ")}
	s.mu.Unlock()
}

type existing struct {
	id          int64
	size, mtime int64
	codec       string
}

func (s *Scanner) scanLibrary(lib Library) error {
	a := s.a
	if !dirExists(lib.Path) {
		return fmt.Errorf("フォルダにアクセスできません: %s", lib.Path)
	}
	scanID := time.Now().UnixNano()
	s.phase("ファイルを検索中", 0, 0)

	var files []string
	filepath.WalkDir(lib.Path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if p != lib.Path && (strings.HasPrefix(name, ".") || skipDirs[strings.ToLower(name)]) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasPrefix(name, ".") && audioExts[strings.ToLower(filepath.Ext(name))] {
			files = append(files, p)
		}
		return nil
	})
	sort.Strings(files)

	known := map[string]existing{}
	rows, err := a.db.Query("SELECT id, path, size, mtime, codec FROM tracks WHERE library_id=?", lib.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var e existing
		var p string
		rows.Scan(&e.id, &p, &e.size, &e.mtime, &e.codec)
		known[p] = e
	}
	rows.Close()

	if len(files) == 0 && len(known) > 0 {
		return errors.New("音楽ファイルが1つも見つかりません（マウントが外れている可能性があるため削除をスキップしました）")
	}

	type job struct {
		path        string
		size, mtime int64
		id          int64
	}
	var todo []job
	var seen []int64
	for _, f := range files {
		st, err := os.Stat(f)
		if err != nil {
			continue
		}
		e, ok := known[f]
		if ok && e.size == st.Size() && e.mtime == st.ModTime().Unix() && e.codec != "" {
			seen = append(seen, e.id)
			continue
		}
		todo = append(todo, job{f, st.Size(), st.ModTime().Unix(), e.id})
	}

	// mark unchanged tracks as seen (chunked)
	for len(seen) > 0 {
		chunk := seen[:min(500, len(seen))]
		seen = seen[len(chunk):]
		ph, args := idList(chunk)
		if _, err := a.db.Exec("UPDATE tracks SET seen_scan=? WHERE id IN ("+ph+")", append([]any{scanID}, args...)...); err != nil {
			return err
		}
	}

	if len(todo) > 0 {
		s.phase("タグを読み込み中", 0, len(todo))
		jobs := make(chan job)
		var wg sync.WaitGroup
		var mu sync.Mutex
		done := 0
		now := time.Now().Unix()
		for w := 0; w < 6; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range jobs {
					m, err := a.readTags(j.path)
					if err != nil {
						log.Printf("probe %s: %v", j.path, err)
						m = &tagInfo{Title: titleFromFile(j.path), Codec: "unknown"}
					}
					// loudness from tags is final; otherwise it is measured later (state 0)
					lufsState := 0
					if m.LUFS != nil {
						lufsState = 1
					}
					var dbErr error
					if j.id > 0 {
						_, dbErr = a.db.Exec(`UPDATE tracks SET title=?, sort_title=?, artist=?, album=?, album_artist=?, compilation=?, track_no=?, disc_no=?,
							year=?, genre=?, duration=?, bitrate=?, codec=?, sample_rate=?, bit_depth=?, channels=?, has_cover=?, has_lyrics=?, size=?, mtime=?, seen_scan=?,
							lufs=?, album_lufs_tag=?, lufs_state=? WHERE id=?`,
							m.Title, sortKey(m.Title), m.Artist, m.Album, m.AlbumArtist, b2i(m.Compilation), m.Track, m.Disc, m.Year, m.Genre, m.Duration, m.Bitrate,
							m.Codec, m.SampleRate, m.BitDepth, m.Channels, b2i(m.HasCover), b2i(m.HasLyrics), j.size, j.mtime, scanID,
							m.LUFS, m.AlbumLUFS, lufsState, j.id)
					} else {
						_, dbErr = a.db.Exec(`INSERT INTO tracks(library_id, path, title, sort_title, artist, album, album_artist, compilation, track_no, disc_no, year, genre,
							duration, bitrate, codec, sample_rate, bit_depth, channels, has_cover, has_lyrics, size, mtime, added, seen_scan, lufs, album_lufs_tag, lufs_state)
							VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
							lib.ID, j.path, m.Title, sortKey(m.Title), m.Artist, m.Album, m.AlbumArtist, b2i(m.Compilation), m.Track, m.Disc, m.Year, m.Genre,
							m.Duration, m.Bitrate, m.Codec, m.SampleRate, m.BitDepth, m.Channels, b2i(m.HasCover), b2i(m.HasLyrics), j.size, j.mtime, now, scanID,
							m.LUFS, m.AlbumLUFS, lufsState)
					}
					if dbErr != nil {
						log.Printf("db %s: %v", j.path, dbErr)
					}
					mu.Lock()
					done++
					if done%20 == 0 || done == len(todo) {
						s.phase("タグを読み込み中", done, len(todo))
					}
					mu.Unlock()
				}
			}()
		}
		for _, j := range todo {
			if a.ctx.Err() != nil {
				break
			}
			jobs <- j
		}
		close(jobs)
		wg.Wait()
	}
	if a.ctx.Err() != nil {
		return a.ctx.Err()
	}
	_, err = a.db.Exec("DELETE FROM tracks WHERE library_id=? AND seen_scan<>?", lib.ID, scanID)
	return err
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

var reLeadTrack = regexp.MustCompile(`^\s*(?:(\d{1,2})[-.])?(\d{1,3})(?:\s*[-._)]\s*|\s+)`)

func titleFromFile(p string) string {
	base := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	if m := reLeadTrack.FindStringIndex(base); m != nil && m[1] < len(base) {
		base = base[m[1]:]
	}
	return strings.TrimSpace(strings.ReplaceAll(base, "_", " "))
}

func trackFromFile(p string) (disc, track int) {
	base := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	if m := reLeadTrack.FindStringSubmatch(base); m != nil {
		return atoi(m[1]), atoi(m[2])
	}
	return 0, 0
}

// ---------- organize: build artists / albums from track tags ----------

type orgTrack struct {
	id, albumID, artistID                   int64
	path, artist, album, albumArtist, genre string
	compilation                             bool
	year                                    int
	added                                   int64
	hasCover                                bool
}

var reDiscDir = regexp.MustCompile(`(?i)^(?:cd|disc|disk|ディスク)\s*\d+$`)

// albumDir returns the album folder, treating "CD1"/"Disc 2" sub-folders as part of the parent.
func albumDir(p string) string {
	d := filepath.Dir(p)
	if reDiscDir.MatchString(strings.TrimSpace(filepath.Base(d))) {
		return filepath.Dir(d)
	}
	return d
}

func (a *App) organize() error {
	rows, err := a.db.Query("SELECT id, COALESCE(album_id,0), COALESCE(artist_id,0), path, artist, album, album_artist, genre, compilation, year, added, has_cover FROM tracks")
	if err != nil {
		return err
	}
	var ts []*orgTrack
	for rows.Next() {
		t := &orgTrack{}
		var comp, cov int
		if err := rows.Scan(&t.id, &t.albumID, &t.artistID, &t.path, &t.artist, &t.album, &t.albumArtist, &t.genre, &comp, &t.year, &t.added, &cov); err != nil {
			rows.Close()
			return err
		}
		t.compilation, t.hasCover = comp == 1, cov == 1
		ts = append(ts, t)
	}
	rows.Close()

	// group by folder + album tag to detect compilations without album-artist tags
	type gkey struct{ dir, album string }
	groupArtists := map[gkey]map[string]bool{}
	for _, t := range ts {
		k := gkey{albumDir(t.path), strings.ToLower(albumTitleOf(t))}
		if groupArtists[k] == nil {
			groupArtists[k] = map[string]bool{}
		}
		groupArtists[k][strings.ToLower(trackArtistOf(t))] = true
	}

	artistIDs := map[string]int64{}
	r2, err := a.db.Query("SELECT id, name FROM artists")
	if err != nil {
		return err
	}
	for r2.Next() {
		var id int64
		var n string
		r2.Scan(&id, &n)
		artistIDs[strings.ToLower(n)] = id
	}
	r2.Close()
	albumIDs := map[string]int64{}
	r3, err := a.db.Query("SELECT id, artist_id, title FROM albums")
	if err != nil {
		return err
	}
	for r3.Next() {
		var id, aid int64
		var t string
		r3.Scan(&id, &aid, &t)
		albumIDs[fmt.Sprintf("%d\x00%s", aid, strings.ToLower(t))] = id
	}
	r3.Close()

	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	artistID := func(name string) (int64, error) {
		if id, ok := artistIDs[strings.ToLower(name)]; ok {
			return id, nil
		}
		res, err := tx.Exec("INSERT INTO artists(name, sort_name) VALUES(?,?)", name, sortKey(name))
		if err != nil {
			return 0, err
		}
		id, _ := res.LastInsertId()
		artistIDs[strings.ToLower(name)] = id
		return id, nil
	}
	now := time.Now().Unix()
	touched := map[int64]bool{}
	for _, t := range ts {
		ta := trackArtistOf(t)
		title := albumTitleOf(t)
		aa := strings.TrimSpace(t.albumArtist)
		if aa == "" {
			k := gkey{albumDir(t.path), strings.ToLower(title)}
			if t.compilation || len(groupArtists[k]) > 1 {
				aa = variousArtists
			} else {
				aa = ta
			}
		}
		taID, err := artistID(ta)
		if err != nil {
			return err
		}
		aaID, err := artistID(aa)
		if err != nil {
			return err
		}
		key := fmt.Sprintf("%d\x00%s", aaID, strings.ToLower(title))
		alID, ok := albumIDs[key]
		if !ok {
			res, err := tx.Exec("INSERT INTO albums(artist_id, title, sort_title, year, genre, dir, added, updated) VALUES(?,?,?,?,?,?,?,?)",
				aaID, title, sortKey(title), t.year, t.genre, albumDir(t.path), t.added, now)
			if err != nil {
				return err
			}
			alID, _ = res.LastInsertId()
			albumIDs[key] = alID
		}
		touched[alID] = true
		if t.albumID != alID || t.artistID != taID {
			if _, err := tx.Exec("UPDATE tracks SET album_id=?, artist_id=? WHERE id=?", alID, taID, t.id); err != nil {
				return err
			}
		}
	}
	// refresh album aggregate fields
	if _, err := tx.Exec(`UPDATE albums SET
		year = COALESCE((SELECT MAX(year) FROM tracks t WHERE t.album_id=albums.id),0),
		genre = COALESCE((SELECT genre FROM tracks t WHERE t.album_id=albums.id AND genre<>'' GROUP BY genre ORDER BY COUNT(*) DESC LIMIT 1),''),
		added = COALESCE((SELECT MIN(added) FROM tracks t WHERE t.album_id=albums.id), added),
		cover_track = COALESCE((SELECT id FROM tracks t WHERE t.album_id=albums.id AND has_cover=1 ORDER BY disc_no, track_no LIMIT 1),0)`); err != nil {
		return err
	}
	// drop empty albums/artists and dangling favorites
	for _, q := range []string{
		"DELETE FROM albums WHERE NOT EXISTS(SELECT 1 FROM tracks t WHERE t.album_id=albums.id)",
		"DELETE FROM artists WHERE NOT EXISTS(SELECT 1 FROM albums al WHERE al.artist_id=artists.id) AND NOT EXISTS(SELECT 1 FROM tracks t WHERE t.artist_id=artists.id)",
		"DELETE FROM favorites WHERE kind='track' AND item_id NOT IN (SELECT id FROM tracks)",
		"DELETE FROM favorites WHERE kind='album' AND item_id NOT IN (SELECT id FROM albums)",
		"DELETE FROM favorites WHERE kind='artist' AND item_id NOT IN (SELECT id FROM artists)",
	} {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return a.findCovers()
}

func trackArtistOf(t *orgTrack) string {
	if s := strings.TrimSpace(t.artist); s != "" {
		return s
	}
	if s := strings.TrimSpace(t.albumArtist); s != "" {
		return s
	}
	return unknownArtist
}

func albumTitleOf(t *orgTrack) string {
	if s := strings.TrimSpace(t.album); s != "" {
		return s
	}
	base := filepath.Base(albumDir(t.path))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "不明なアルバム"
	}
	return base
}

var coverNames = []string{"cover", "folder", "front", "album", "albumart", "albumartlarge", "thumb", "jacket", "artwork"}

// findCovers sets album.cover to an image file in the album folder when one exists.
func (a *App) findCovers() error {
	rows, err := a.db.Query("SELECT id, dir, cover FROM albums")
	if err != nil {
		return err
	}
	type row struct {
		id         int64
		dir, cover string
	}
	var list []row
	for rows.Next() {
		var r row
		rows.Scan(&r.id, &r.dir, &r.cover)
		list = append(list, r)
	}
	rows.Close()
	dirCache := map[string]string{}
	now := time.Now().Unix()
	for _, r := range list {
		c, ok := dirCache[r.dir]
		if !ok {
			c = coverInDir(r.dir)
			dirCache[r.dir] = c
		}
		if c != r.cover {
			a.db.Exec("UPDATE albums SET cover=?, updated=? WHERE id=?", c, now, r.id)
		}
	}
	return nil
}

func coverInDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	byName := map[string]string{}
	var images []string
	for _, e := range entries {
		if e.IsDir() {
			// look one level down for scans/artwork folders
			continue
		}
		n := strings.ToLower(e.Name())
		ext := filepath.Ext(n)
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			continue
		}
		byName[strings.TrimSuffix(n, ext)] = filepath.Join(dir, e.Name())
		images = append(images, filepath.Join(dir, e.Name()))
	}
	for _, n := range coverNames {
		if p, ok := byName[n]; ok {
			return p
		}
	}
	for n, p := range byName {
		if strings.Contains(n, "cover") || strings.Contains(n, "front") || strings.Contains(n, "folder") {
			return p
		}
	}
	if len(images) == 1 {
		return images[0]
	}
	return ""
}
