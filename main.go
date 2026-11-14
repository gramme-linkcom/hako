package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

//go:embed web
var webFS embed.FS

const version = "1.0.0"

type Config struct {
	DataDir      string
	Addr         string
	FFmpeg       string
	FFprobe      string
	ScanInterval time.Duration
}

type App struct {
	cfg     Config
	db      *sql.DB
	ctx     context.Context
	scanner *Scanner
	imgSem  chan struct{}
	locks   *keyLock
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func main() {
	cfg := Config{
		DataDir: env("HAKO_DATA", "./data"),
		Addr:    env("HAKO_ADDR", ":8096"),
		FFmpeg:  env("HAKO_FFMPEG", "ffmpeg"),
		FFprobe: env("HAKO_FFPROBE", "ffprobe"),
	}
	mins, err := strconv.Atoi(env("HAKO_SCAN_INTERVAL_MIN", "60"))
	if err != nil {
		mins = 60
	}
	cfg.ScanInterval = time.Duration(mins) * time.Minute

	for _, d := range []string{"", "cache/covers", "lyrics"} {
		if err := os.MkdirAll(filepath.Join(cfg.DataDir, d), 0o755); err != nil {
			log.Fatal(err)
		}
	}

	db, err := openDB(filepath.Join(cfg.DataDir, "hako.db"))
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a := &App{
		cfg:    cfg,
		db:     db,
		ctx:    ctx,
		imgSem: make(chan struct{}, max(2, runtime.NumCPU())),
		locks:  newKeyLock(),
	}
	if _, err := exec.LookPath(cfg.FFmpeg); err != nil {
		log.Printf("WARNING: ffmpeg not found (%s): streaming conversion and cover extraction will not work", cfg.FFmpeg)
	}
	a.scanner = newScanner(a)
	go a.scanner.loop()
	go a.securityLoop()

	srv := &http.Server{Addr: cfg.Addr, Handler: a.routes(), ReadHeaderTimeout: 20 * time.Second}
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(c)
	}()
	log.Printf("Hako %s listening on %s (data: %s)", version, cfg.Addr, cfg.DataDir)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func (a *App) checkFilter(name string) bool {
	out, err := exec.Command(a.cfg.FFmpeg, "-hide_banner", "-filters").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), " "+name+" ")
}

func (a *App) ffmpegVersion() string {
	out, err := exec.Command(a.cfg.FFmpeg, "-version").Output()
	if err != nil {
		return "not found"
	}
	line, _, _ := strings.Cut(string(out), "\n")
	return strings.TrimSpace(line)
}
