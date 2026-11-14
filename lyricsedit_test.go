package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLyricsPriority(t *testing.T) {
	dir := t.TempDir()
	a := &App{cfg: Config{DataDir: dir}}
	song := filepath.Join(dir, "music", "01 song.flac")
	os.MkdirAll(filepath.Dir(song), 0o755)
	os.WriteFile(song, []byte("x"), 0o644)
	tr := &Track{Path: song}

	// .lrc next to the file, Shift_JIS encoded
	sjis := []byte{'[', '0', '0', ':', '0', '1', '.', '0', '0', ']', 0x82, 0xa0} // "あ"
	os.WriteFile(filepath.Join(dir, "music", "01 song.lrc"), sjis, 0o644)
	text, src := a.rawLyrics(tr)
	if src != "file" || parseLyrics(text).Lines[0].Text != "あ" {
		t.Fatalf(".lrc: got %q %q", src, text)
	}

	// editor lyrics win over the .lrc
	p := a.editedLyricsPath(song)
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte("[00:02.50]edited\n"), 0o644)
	text, src = a.rawLyrics(tr)
	l := parseLyrics(text)
	if src != "hako" || !l.Synced || l.Lines[0].Text != "edited" || l.Lines[0].Time != 2.5 {
		t.Fatalf("edited: got %q %+v", src, l)
	}
	if a.lyricsFor(tr).Lines[0].Text != "edited" {
		t.Fatal("lyricsFor should use edited lyrics")
	}

	// a different file gets a different storage path
	if a.editedLyricsPath(song) == a.editedLyricsPath(song+"x") || filepath.Dir(p) != filepath.Join(dir, "lyrics") {
		t.Fatal("bad storage path")
	}

	// removing the edited file falls back to the .lrc
	os.Remove(p)
	if _, src = a.rawLyrics(tr); src != "file" {
		t.Fatalf("fallback: got %q", src)
	}
}
