package main

import (
	"log"
	"math"
	"runtime"
	"sync"
)

// analyzeLoudness measures tracks that have no ReplayGain/R128 tags (lufs_state=0),
// then recomputes album loudness. Runs after each scan; already-measured tracks are skipped.
func (s *Scanner) analyzeLoudness() error {
	a := s.a
	rows, err := a.db.Query("SELECT id, path FROM tracks WHERE lufs_state=0 ORDER BY added DESC")
	if err != nil {
		return err
	}
	type job struct {
		id   int64
		path string
	}
	var jobs []job
	for rows.Next() {
		var j job
		rows.Scan(&j.id, &j.path)
		jobs = append(jobs, j)
	}
	rows.Close()

	if len(jobs) > 0 {
		s.phase("音量を解析中", 0, len(jobs))
		workers := max(1, min(4, runtime.NumCPU()/2))
		ch := make(chan job)
		var wg sync.WaitGroup
		var mu sync.Mutex
		done := 0
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range ch {
					v, err := a.measureLoudness(j.path)
					if err != nil {
						if a.ctx.Err() == nil {
							log.Printf("loudness %s: %v", j.path, err)
							a.db.Exec("UPDATE tracks SET lufs_state=2 WHERE id=?", j.id)
						}
					} else {
						a.db.Exec("UPDATE tracks SET lufs=?, lufs_state=1 WHERE id=?", v, j.id)
					}
					mu.Lock()
					done++
					if done%5 == 0 || done == len(jobs) {
						s.phase("音量を解析中", done, len(jobs))
					}
					// refresh album values now and then so normalisation works before the pass ends
					if done%200 == 0 {
						a.updateAlbumLoudness()
					}
					mu.Unlock()
				}
			}()
		}
		for _, j := range jobs {
			if a.ctx.Err() != nil {
				break
			}
			ch <- j
		}
		close(ch)
		wg.Wait()
	}
	return a.updateAlbumLoudness()
}

// updateAlbumLoudness sets albums.lufs. If every track carries a ReplayGain album tag that
// value is used; otherwise it is the duration-weighted energy average of the track loudness.
func (a *App) updateAlbumLoudness() error {
	rows, err := a.db.Query("SELECT album_id, duration, lufs, album_lufs_tag FROM tracks WHERE album_id IS NOT NULL")
	if err != nil {
		return err
	}
	type acc struct {
		n, nTag, nLufs int
		tagSum         float64
		energy, dur    float64
	}
	albums := map[int64]*acc{}
	for rows.Next() {
		var id int64
		var d float64
		var l, tag *float64
		if err := rows.Scan(&id, &d, &l, &tag); err != nil {
			rows.Close()
			return err
		}
		x := albums[id]
		if x == nil {
			x = &acc{}
			albums[id] = x
		}
		x.n++
		if tag != nil {
			x.nTag++
			x.tagSum += *tag
		}
		if l != nil {
			if d <= 0 {
				d = 1
			}
			x.nLufs++
			x.energy += d * math.Pow(10, *l/10)
			x.dur += d
		}
	}
	rows.Close()
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for id, x := range albums {
		var v any
		switch {
		case x.nTag == x.n && x.n > 0:
			v = x.tagSum / float64(x.nTag)
		case x.nLufs > 0 && x.energy > 0:
			v = 10 * math.Log10(x.energy/x.dur)
		}
		if _, err := tx.Exec("UPDATE albums SET lufs=? WHERE id=?", v, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
