'use strict';

/* ============================================================
   Hako — self-hosted music server (frontend)
   ============================================================ */

const $ = (s, el = document) => el.querySelector(s);
const $$ = (s, el = document) => [...el.querySelectorAll(s)];
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const clamp = (v, a, b) => Math.min(b, Math.max(a, v));
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const P_ = (d) => `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="${d}"/></svg>`;
const I = {
  play: P_('M8 5v14l11-7z'),
  pause: P_('M6 19h4V5H6v14zm8-14v14h4V5h-4z'),
  next: P_('M6 18l8.5-6L6 6v12zM16 6v12h2V6h-2z'),
  prev: P_('M6 6h2v12H6zm3.5 6l8.5 6V6z'),
  shuffle: P_('M10.59 9.17L5.41 4 4 5.41l5.17 5.17 1.42-1.41zM14.5 4l2.04 2.04L4 18.59 5.41 20 17.96 7.46 20 9.5V4h-5.5zm.33 9.41l-1.41 1.41 3.13 3.13L14.5 20H20v-5.5l-2.04 2.04-3.13-3.13z'),
  repeat: P_('M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4z'),
  repeatOne: P_('M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4zm-4-2V9h-1l-2 1v1h1.5v4H13z'),
  heart: P_('M16.5 3c-1.74 0-3.41.81-4.5 2.09C10.91 3.81 9.24 3 7.5 3 4.42 3 2 5.42 2 8.5c0 3.78 3.4 6.86 8.55 11.54L12 21.35l1.45-1.32C18.6 15.36 22 12.28 22 8.5 22 5.42 19.58 3 16.5 3zm-4.4 15.55l-.1.1-.1-.1C7.14 14.24 4 11.39 4 8.5 4 6.5 5.5 5 7.5 5c1.54 0 3.04.99 3.57 2.36h1.87C13.46 5.99 14.96 5 16.5 5c2 0 3.5 1.5 3.5 3.5 0 2.89-3.14 5.74-7.9 10.05z'),
  heartFill: P_('M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z'),
  more: P_('M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z'),
  search: P_('M15.5 14h-.79l-.28-.27A6.47 6.47 0 0016 9.5 6.5 6.5 0 109.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z'),
  home: P_('M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z'),
  album: P_('M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 14.5c-2.49 0-4.5-2.01-4.5-4.5S9.51 7.5 12 7.5s4.5 2.01 4.5 4.5-2.01 4.5-4.5 4.5zm0-5.5c-.55 0-1 .45-1 1s.45 1 1 1 1-.45 1-1-.45-1-1-1z'),
  artist: P_('M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z'),
  playlist: P_('M15 6H3v2h12V6zm0 4H3v2h12v-2zM3 16h8v-2H3v2zM17 6v8.18c-.31-.11-.65-.18-1-.18-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3V8h3V6h-5z'),
  note: P_('M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z'),
  settings: P_('M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58a.49.49 0 00.12-.61l-1.92-3.32a.488.488 0 00-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.484.484 0 00-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.09.63-.09.94s.02.64.07.94l-2.03 1.58a.49.49 0 00-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z'),
  back: P_('M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z'),
  down: P_('M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z'),
  queue: P_('M3 10h11v2H3zm0-4h11v2H3zm0 8h7v2H3zm13-1v8l6-4z'),
  lyrics: P_('M3 5h18v2H3zm0 4h12v2H3zm0 4h18v2H3zm0 4h12v2H3z'),
  plus: P_('M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z'),
  close: P_('M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z'),
  trash: P_('M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z'),
  up: P_('M7.41 15.41L12 10.83l4.59 4.58L18 14l-6-6-6 6z'),
  download: P_('M19 9h-4V3H9v6H5l7 7 7-7zM5 18v2h14v-2H5z'),
  playlistAdd: P_('M14 10H3v2h11v-2zm0-4H3v2h11V6zm4 8v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zM3 16h7v-2H3v2z'),
  playNext: P_('M3 10h11v2H3zm0-4h11v2H3zm0 8h7v2H3zm13-1v8l6-4z'),
  volume: P_('M3 9v6h4l5 5V4L7 9H3zm13.5 3A4.5 4.5 0 0014 7.97v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z'),
  mute: P_('M16.5 12A4.5 4.5 0 0014 7.97v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51A8.796 8.796 0 0021 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06a8.99 8.99 0 003.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z'),
  folder: P_('M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z'),
  edit: P_('M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04a1 1 0 000-1.41l-2.34-2.34a1 1 0 00-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z'),
  history: P_('M13 3a9 9 0 00-9 9H1l3.89 3.89.07.14L9 12H6c0-3.87 3.13-7 7-7s7 3.13 7 7-3.13 7-7 7c-1.93 0-3.68-.79-4.94-2.06l-1.42 1.42A8.954 8.954 0 0013 21a9 9 0 000-18zm-1 5v5l4.28 2.54.72-1.21-3.5-2.08V8H12z'),
  refresh: P_('M17.65 6.35A7.958 7.958 0 0012 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0112 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z'),
  random: P_('M10.59 9.17L5.41 4 4 5.41l5.17 5.17 1.42-1.41zM14.5 4l2.04 2.04L4 18.59 5.41 20 17.96 7.46 20 9.5V4h-5.5zm.33 9.41l-1.41 1.41 3.13 3.13L14.5 20H20v-5.5l-2.04 2.04-3.13-3.13z'),
};

/* ---------------- state & helpers ---------------- */

const state = {
  me: null,
  playlists: [],
  prefs: loadPrefs(),
  lists: new Map(),
  listSeq: 0,
  lyricsCache: new Map(),
};

function loadPrefs() {
  const def = { quality: 'orig', volume: 1, norm: 'auto', target: -14 };
  try { return Object.assign(def, JSON.parse(localStorage.getItem('hako.prefs') || '{}')); } catch { return def; }
}
function savePrefs() { try { localStorage.setItem('hako.prefs', JSON.stringify(state.prefs)); } catch {} }

class HttpError extends Error { constructor(status, msg) { super(msg); this.status = status; } }

async function api(path, { method = 'GET', body, quiet = false } = {}) {
  const opts = { method, credentials: 'same-origin', headers: {} };
  if (body !== undefined) { opts.headers['Content-Type'] = 'application/json'; opts.body = JSON.stringify(body); }
  let r;
  try { r = await fetch('/api' + path, opts); } catch (e) { throw new HttpError(0, 'サーバーに接続できません'); }
  let data = null;
  if ((r.headers.get('content-type') || '').includes('json')) data = await r.json().catch(() => null);
  if (r.status === 401 && !quiet) { state.me = null; go('#/login'); }
  if (!r.ok) throw new HttpError(r.status, (data && data.error) || r.statusText);
  return data;
}

let toastTimer;
function toast(msg, err = false) {
  const t = $('#toast');
  t.textContent = msg; t.className = err ? 'err' : ''; t.hidden = false;
  clearTimeout(toastTimer); toastTimer = setTimeout(() => (t.hidden = true), 2600);
}

const pad = (n) => String(n).padStart(2, '0');
function fmtTime(s) {
  s = Math.max(0, Math.floor(s || 0));
  const h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60), sec = s % 60;
  return h ? `${h}:${pad(m)}:${pad(sec)}` : `${m}:${pad(sec)}`;
}
function fmtLong(s) {
  const m = Math.round((s || 0) / 60);
  return m >= 60 ? `${Math.floor(m / 60)}時間${m % 60}分` : `${m}分`;
}

function hue(str) { let h = 0; for (const c of String(str)) h = (h * 31 + c.codePointAt(0)) >>> 0; return h % 360; }
function grad(str) { const h = hue(str); return `linear-gradient(135deg,hsl(${h} 45% 34%),hsl(${(h + 40) % 360} 50% 22%))`; }
function initial(str) { const s = String(str || '?').trim(); return esc([...s][0] || '?'); }

function coverHTML(albumId, cv, w, label, cls = '', style = '') {
  const img = albumId ? `<img loading="lazy" decoding="async" src="/api/albums/${albumId}/cover?w=${w}&v=${cv || 0}" alt="" data-rm-err>` : '';
  return `<div class="cover ${cls}" style="background:${grad(label)};${style}"><span class="ph">${initial(label)}</span>${img}</div>`;
}

function fmtSpec(bits, rate) {
  if (!rate) return '';
  const k = rate / 1000;
  const r = rate >= 2822400 ? `DSD${Math.round(rate / 44100)}` : `${Number.isInteger(k) ? k : k.toFixed(1)}kHz`;
  return bits > 1 ? `${bits}bit/${r}` : r;
}

// quality badge: q = "hires" | "lossless" | ""
function qBadge(q, { big = false, spec = '' } = {}) {
  if (q === 'hires') return `<i class="qb hires${big ? ' big' : ''}" title="ハイレゾ${spec ? ' ' + esc(spec) : ''}">Hi-Res</i>`;
  if (q === 'lossless') return `<i class="qb lossless${big ? ' big' : ''}" title="ロスレス${spec ? ' ' + esc(spec) : ''}">Lossless</i>`;
  return '';
}

function regList(tracks) {
  const key = 'l' + (++state.listSeq);
  state.lists.set(key, tracks);
  return key;
}

function shuffleArr(a) {
  a = a.slice();
  for (let i = a.length - 1; i > 0; i--) { const j = Math.floor(Math.random() * (i + 1)); [a[i], a[j]] = [a[j], a[i]]; }
  return a;
}

/* ============================================================
   Player engine
   ============================================================ */

const audio = new Audio();
audio.preload = 'auto';

const P = {
  queue: [], orig: null, idx: -1,
  offset: 0, direct: true, needsLoad: false, pendingStart: 0,
  shuffle: false, repeat: 'off',
  listened: 0, lastT: 0, scrobbled: false,
  forceTx: new Set(), dragging: false,
  userVol: clamp(+state.prefs.volume || 1, 0, 1), norm: null,
};
const cur = () => P.queue[P.idx];
const position = () => P.offset + (audio.currentTime || 0);
const duration = () => { const t = cur(); return t ? (t.duration || (isFinite(audio.duration) ? audio.duration : 0)) : 0; };

function qualityBr() { return state.prefs.quality === 'orig' ? 320 : +state.prefs.quality; }

function canDirect(t) {
  if (P.forceTx.has(t.id) || !t.mime) return false;
  if (!audio.canPlayType(t.mime)) return false;
  if (state.prefs.quality !== 'orig' && t.bitrate > +state.prefs.quality + 40) return false;
  return true;
}

function loadTrack(idx, start = 0, play = true) {
  if (idx < 0 || idx >= P.queue.length) return;
  P.idx = idx;
  const t = cur();
  P.direct = canDirect(t);
  P.offset = P.direct ? 0 : start;
  P.listened = 0; P.lastT = 0; P.scrobbled = false; P.needsLoad = false;
  audio.src = P.direct
    ? `/api/tracks/${t.id}/stream?mode=direct`
    : `/api/tracks/${t.id}/stream?br=${qualityBr()}&start=${start.toFixed(2)}`;
  if (P.direct && start > 0) audio.addEventListener('loadedmetadata', () => { audio.currentTime = start; }, { once: true });
  if (play) audio.play().catch((e) => { if (e.name !== 'AbortError') console.warn(e); });
  onTrackChange();
}

function playList(tracks, i = 0, shuffle = false) {
  if (!tracks || !tracks.length) return;
  P.orig = tracks.slice();
  P.shuffle = shuffle;
  if (shuffle) {
    // i < 0 means "no fixed first track"
    const first = i >= 0 && i < tracks.length ? tracks[i] : null;
    const rest = shuffleArr(first ? tracks.filter((_, k) => k !== i) : tracks);
    P.queue = first ? [first, ...rest] : rest;
    i = 0;
  } else {
    P.queue = tracks.slice();
  }
  loadTrack(clamp(i, 0, P.queue.length - 1), 0, true);
}

function togglePlay() {
  if (!cur()) return;
  if (P.needsLoad || !audio.src) { loadTrack(P.idx, P.pendingStart || 0, true); return; }
  if (audio.paused) audio.play().catch(() => {}); else audio.pause();
}

function next(auto = false) {
  if (!P.queue.length) return;
  if (auto && P.repeat === 'one') { loadTrack(P.idx, 0, true); return; }
  if (P.idx + 1 < P.queue.length) loadTrack(P.idx + 1);
  else if (P.repeat === 'all') loadTrack(0);
  else if (auto) { loadTrack(P.idx, 0, false); }
  else toast('キューの最後です');
}

function prev() {
  if (!P.queue.length) return;
  if (position() > 3) { seek(0); if (audio.paused) togglePlay(); return; }
  if (P.idx > 0) loadTrack(P.idx - 1);
  else if (P.repeat === 'all') loadTrack(P.queue.length - 1);
  else seek(0);
}

function seek(t) {
  if (!cur()) return;
  const d = duration();
  t = clamp(t, 0, Math.max(0, d - 0.3));
  if (P.needsLoad) { P.pendingStart = t; updateProgress(); return; }
  if (P.direct) { audio.currentTime = t; updateProgress(); return; }
  const rel = t - P.offset;
  if (rel >= 0) {
    for (let i = 0; i < audio.buffered.length; i++) {
      if (rel >= audio.buffered.start(i) && rel <= audio.buffered.end(i) - 0.5) { audio.currentTime = rel; updateProgress(); return; }
    }
  }
  loadTrack(P.idx, t, !audio.paused);
}

function toggleShuffle() {
  if (!P.queue.length) { P.shuffle = !P.shuffle; return onModeChange(); }
  const c = cur();
  if (!P.shuffle) {
    P.orig = P.queue.slice();
    P.queue = [c, ...shuffleArr(P.queue.filter((_, k) => k !== P.idx))];
    P.idx = 0; P.shuffle = true;
  } else {
    const base = P.orig && P.orig.length ? P.orig : P.queue;
    // keep tracks added while shuffled
    const extra = P.queue.filter((t) => !base.includes(t));
    P.queue = [...base, ...extra];
    P.idx = Math.max(0, P.queue.indexOf(c));
    P.shuffle = false;
  }
  onModeChange(); renderQueuePanel(); saveQueue();
}

function cycleRepeat() {
  P.repeat = { off: 'all', all: 'one', one: 'off' }[P.repeat];
  onModeChange(); saveQueue();
  toast({ off: 'リピート: オフ', all: 'リピート: すべて', one: 'リピート: 1曲' }[P.repeat]);
}

function addNext(tracks) {
  if (!tracks.length) return;
  if (!P.queue.length) return playList(tracks);
  P.queue.splice(P.idx + 1, 0, ...tracks);
  if (P.orig) P.orig.push(...tracks);
  toast(tracks.length > 1 ? `${tracks.length}曲を次に再生します` : '次に再生します');
  renderQueuePanel(); saveQueue();
}

function addToQueue(tracks) {
  if (!tracks.length) return;
  if (!P.queue.length) { P.queue = tracks.slice(); P.orig = tracks.slice(); P.idx = 0; P.needsLoad = true; P.pendingStart = 0; onTrackChange(); }
  else { P.queue.push(...tracks); if (P.orig) P.orig.push(...tracks); }
  toast(`${tracks.length}曲をキューに追加しました`);
  renderQueuePanel(); saveQueue();
}

function removeFromQueue(i) {
  if (i < 0 || i >= P.queue.length) return;
  const wasCur = i === P.idx;
  const [t] = P.queue.splice(i, 1);
  if (P.orig) { const k = P.orig.indexOf(t); if (k >= 0) P.orig.splice(k, 1); }
  if (!P.queue.length) { stopAll(); return; }
  if (i < P.idx) P.idx--;
  if (wasCur) { const playing = !audio.paused; loadTrack(Math.min(P.idx, P.queue.length - 1), 0, playing); }
  renderQueuePanel(); saveQueue();
}

function stopAll() {
  audio.pause(); audio.removeAttribute('src'); audio.load();
  P.queue = []; P.orig = null; P.idx = -1;
  onTrackChange(); saveQueue();
}

/* ---- audio events ---- */

audio.addEventListener('timeupdate', () => {
  const ct = audio.currentTime;
  const d = ct - P.lastT;
  if (d > 0 && d < 2) P.listened += d;
  P.lastT = ct;
  const t = cur();
  if (t && !P.scrobbled && P.listened >= Math.min(240, Math.max(10, (t.duration || 60) * 0.5))) {
    P.scrobbled = true;
    api(`/tracks/${t.id}/scrobble`, { method: 'POST' }).catch(() => {});
    t.plays = (t.plays || 0) + 1;
  }
  updateProgress();
  updateLyrics();
  throttledPosState();
  throttledSave();
});
audio.addEventListener('play', onPlayState);
audio.addEventListener('pause', onPlayState);
audio.addEventListener('ended', () => next(true));
audio.addEventListener('error', () => {
  const t = cur();
  if (!t || !audio.src || audio.error == null) return;
  if (P.direct) {
    P.forceTx.add(t.id);
    const pos = position();
    toast('このファイルは直接再生できないため、変換して再生します');
    loadTrack(P.idx, pos, true);
  } else {
    toast('再生できませんでした', true);
    setTimeout(() => { if (cur() === t && P.idx + 1 < P.queue.length) next(); }, 1500);
  }
});
/* ---- volume & loudness normalisation ----
   audio.volume = user volume × normalisation factor. The factor can only lower the level
   (audio.volume is capped at 1), so the default target (-14 LUFS) is chosen so that most
   modern masters (louder than that) get attenuated rather than needing a boost. */

function normInfo(t) {
  const mode = state.prefs.norm;
  if (!t || mode === 'off') return null;
  let useAlbum = mode === 'album';
  if (mode === 'auto') {
    const p = P.queue[P.idx - 1], n = P.queue[P.idx + 1];
    useAlbum = !!t.album_id && ((p && p.album_id === t.album_id) || (n && n.album_id === t.album_id));
  }
  const l = useAlbum && t.album_lufs != null ? t.album_lufs : t.lufs;
  if (l == null) return null;
  const db = clamp(+state.prefs.target - l, -30, 12);
  return { db, factor: Math.min(1, Math.pow(10, db / 20)), album: useAlbum && t.album_lufs != null };
}

function applyVolume() {
  const n = normInfo(cur());
  P.norm = n;
  audio.volume = clamp(P.userVol * (n ? n.factor : 1), 0, 1);
}

function setUserVolume(v) {
  P.userVol = clamp(v, 0, 1);
  state.prefs.volume = P.userVol; savePrefs();
  applyVolume();
  const s = $('#player .vol'); if (s) { s.value = P.userVol * 100; s.style.setProperty('--p', P.userVol * 100 + '%'); }
}

function throttle(fn, ms) { let last = 0; return (...a) => { const n = Date.now(); if (n - last >= ms) { last = n; fn(...a); } }; }
const throttledSave = throttle(() => saveQueue(), 5000);
const throttledPosState = throttle(() => {
  if (!('mediaSession' in navigator) || !navigator.mediaSession.setPositionState) return;
  const d = duration();
  if (!d) return;
  try { navigator.mediaSession.setPositionState({ duration: d, position: clamp(position(), 0, d), playbackRate: 1 }); } catch {}
}, 1000);

/* ---- media session (lock screen / notification controls) ---- */

function updateMediaSession() {
  if (!('mediaSession' in navigator)) return;
  const t = cur();
  if (!t) { navigator.mediaSession.metadata = null; return; }
  const art = t.album_id ? [96, 256, 512].map((s) => ({ src: `${location.origin}/api/albums/${t.album_id}/cover?w=${s}&v=${t.cv}`, sizes: `${s}x${s}`, type: 'image/jpeg' })) : [{ src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png' }];
  navigator.mediaSession.metadata = new MediaMetadata({ title: t.title, artist: t.artist || t.album_artist, album: t.album, artwork: art });
}
if ('mediaSession' in navigator) {
  const h = (a, f) => { try { navigator.mediaSession.setActionHandler(a, f); } catch {} };
  h('play', () => togglePlay());
  h('pause', () => audio.pause());
  h('stop', () => audio.pause());
  h('previoustrack', () => prev());
  h('nexttrack', () => next());
  h('seekto', (d) => seek(d.seekTime));
  h('seekbackward', (d) => seek(position() - (d.seekOffset || 10)));
  h('seekforward', (d) => seek(position() + (d.seekOffset || 10)));
}

/* ---- persistence of the queue (per device) ---- */

function saveQueue() {
  applyVolume(); // queue neighbours decide album vs track gain in "auto" mode
  try {
    localStorage.setItem('hako.q', JSON.stringify({
      ids: P.queue.map((t) => t.id), idx: P.idx,
      pos: P.needsLoad ? P.pendingStart : position(),
      shuffle: P.shuffle, repeat: P.repeat,
    }));
  } catch {}
}

async function restoreQueue() {
  let s;
  try { s = JSON.parse(localStorage.getItem('hako.q') || 'null'); } catch { s = null; }
  if (!s || !s.ids || !s.ids.length) return;
  try {
    const tracks = await api('/tracks/batch?ids=' + s.ids.slice(0, 2000).join(','));
    if (!tracks.length) return;
    P.queue = tracks; P.orig = tracks.slice();
    const curId = s.ids[s.idx];
    P.idx = Math.max(0, tracks.findIndex((t) => t.id === curId));
    P.shuffle = !!s.shuffle; P.repeat = s.repeat || 'off';
    P.needsLoad = true; P.pendingStart = tracks[P.idx] && tracks[P.idx].id === curId ? (s.pos || 0) : 0;
    onTrackChange();
  } catch {}
}

/* ---- UI sync ---- */

function onTrackChange() {
  applyVolume();
  updateMediaSession();
  renderPlayer();
  renderSheet();
  markPlaying();
  saveQueue();
}

function onPlayState() {
  const playing = !audio.paused;
  $$('[data-pp]').forEach((b) => { b.innerHTML = playing ? I.pause : I.play; b.setAttribute('aria-label', playing ? '一時停止' : '再生'); });
  document.body.classList.toggle('paused', !playing);
  if ('mediaSession' in navigator) navigator.mediaSession.playbackState = playing ? 'playing' : 'paused';
  markPlaying();
}

function onModeChange() {
  $$('[data-shuffle]').forEach((b) => b.classList.toggle('on', P.shuffle));
  $$('[data-repeat]').forEach((b) => { b.classList.toggle('on', P.repeat !== 'off'); b.innerHTML = P.repeat === 'one' ? I.repeatOne : I.repeat; });
}

function updateProgress() {
  const d = duration();
  const pos = P.needsLoad ? P.pendingStart : position();
  const pct = d ? clamp((pos / d) * 100, 0, 100) : 0;
  const bar = $('#player .bar i'); if (bar) bar.style.width = pct + '%';
  if (P.dragging) return;
  $$('.seekbar').forEach((s) => { s.value = d ? Math.round((pos / d) * 1000) : 0; s.style.setProperty('--p', pct + '%'); });
  $$('[data-cur]').forEach((e) => (e.textContent = fmtTime(pos)));
  $$('[data-tot]').forEach((e) => (e.textContent = fmtTime(d)));
}

function markPlaying() {
  const t = cur();
  $$('.trow').forEach((r) => {
    const on = t && +r.dataset.id === t.id && !r.closest('.qlist');
    r.classList.toggle('playing', !!on);
    const num = $('.num', r);
    if (num) num.innerHTML = on ? '<span class="eq"><i></i><i></i><i></i></span>' : esc(num.dataset.n || '');
  });
  $$('.qlist .trow').forEach((r) => r.classList.toggle('playing', +r.dataset.qi === P.idx));
}

function bindSeek(el) {
  el.addEventListener('input', () => {
    P.dragging = true;
    const d = duration();
    const v = (el.value / 1000) * d;
    el.style.setProperty('--p', el.value / 10 + '%');
    $$('[data-cur]').forEach((e) => (e.textContent = fmtTime(v)));
  });
  el.addEventListener('change', () => {
    P.dragging = false;
    seek((el.value / 1000) * duration());
  });
}

function favButton(kind, item, cls = 'icon-btn') {
  return `<button class="${cls}${item.fav ? ' on' : ''}" data-fav="${kind}" data-id="${item.id}" aria-label="お気に入り">${item.fav ? I.heartFill : I.heart}</button>`;
}

async function toggleFav(kind, id, btn) {
  const on = !(btn && btn.classList.contains('on'));
  try {
    await api('/favorites', { method: 'POST', body: { kind, id, fav: on } });
    $$(`[data-fav="${kind}"][data-id="${id}"]`).forEach((b) => { b.classList.toggle('on', on); b.innerHTML = on ? I.heartFill : I.heart; });
    if (kind === 'track') {
      for (const t of P.queue) if (t.id === id) t.fav = on;
      for (const list of state.lists.values()) for (const t of list) if (t.id === id) t.fav = on;
    }
    toast(on ? 'お気に入りに追加しました' : 'お気に入りから削除しました');
  } catch (e) { toast(e.message, true); }
}

function renderPlayer() {
  const el = $('#player');
  const t = cur();
  if (!t) { el.className = 'empty'; el.innerHTML = ''; return; }
  el.className = '';
  const playing = !audio.paused;
  el.innerHTML = `
    <div class="bar"><i></i></div>
    <div data-open style="cursor:pointer">${coverHTML(t.album_id, t.cv, 96, t.album || t.title)}</div>
    <div class="info" data-open>
      <div class="t tline"><span class="ellipsis">${esc(t.title)}</span>${qBadge(t.q)}</div>
      <div class="s ellipsis">${esc(t.artist || t.album_artist)}</div>
    </div>
    <div class="desk">
      <div class="row">
        <button class="icon-btn${P.shuffle ? ' on' : ''}" data-shuffle aria-label="シャッフル">${I.shuffle}</button>
        <button class="icon-btn" data-prev aria-label="前の曲">${I.prev}</button>
        <button class="icon-btn big" data-pp style="color:var(--fg)">${playing ? I.pause : I.play}</button>
        <button class="icon-btn" data-next aria-label="次の曲">${I.next}</button>
        <button class="icon-btn${P.repeat !== 'off' ? ' on' : ''}" data-repeat aria-label="リピート">${P.repeat === 'one' ? I.repeatOne : I.repeat}</button>
      </div>
      <div class="seek"><span data-cur>0:00</span><input type="range" class="seekbar" min="0" max="1000" value="0" aria-label="再生位置"><span data-tot>${fmtTime(duration())}</span></div>
    </div>
    <div class="right">
      ${favButton('track', t)}
      <button class="icon-btn" data-open-tab="lyrics" aria-label="歌詞">${I.lyrics}</button>
      <button class="icon-btn" data-open-tab="queue" aria-label="キュー">${I.queue}</button>
      <button class="icon-btn" data-mute aria-label="ミュート">${audio.muted ? I.mute : I.volume}</button>
      <input type="range" class="vol" min="0" max="100" value="${Math.round(P.userVol * 100)}" style="--p:${P.userVol * 100}%" aria-label="音量">
    </div>
    <div class="ctl mobile">
      <button class="icon-btn big" data-pp aria-label="再生">${playing ? I.pause : I.play}</button>
      <button class="icon-btn" data-next aria-label="次の曲">${I.next}</button>
    </div>`;
  $$('.seekbar', el).forEach(bindSeek);
  const vol = $('.vol', el);
  vol.addEventListener('input', () => { audio.muted = false; setUserVolume(vol.value / 100); vol.style.setProperty('--p', vol.value + '%'); });
  updateProgress();
}

/* ---- now playing sheet ---- */

let sheetTab = 'np';
let sheetOpen = false;

function openSheet(tab = 'np') {
  sheetTab = tab;
  if (!cur()) return;
  if (!sheetOpen) {
    sheetOpen = true;
    $('#sheet').hidden = false;
    pushLayer(() => closeSheetNow());
  }
  renderSheet();
}
function closeSheetNow() { sheetOpen = false; $('#sheet').hidden = true; $('#sheet').innerHTML = ''; }

function renderSheet() {
  if (!sheetOpen) return;
  const t = cur();
  if (!t) { popLayer(); return; }
  const el = $('#sheet');
  const wide = matchMedia('(min-width: 900px)').matches;
  const playing = !audio.paused;
  const normTxt = P.norm ? `音量 ${P.norm.db > 0 ? '+' : ''}${P.norm.db.toFixed(1)}dB${P.norm.album ? '（アルバム）' : ''}${P.norm.db > 0 ? '※上げられないため原音量' : ''}` : '';
  const tech = [t.codec && t.codec.toUpperCase(), fmtSpec(t.bit_depth, t.sample_rate), t.bitrate ? `${t.bitrate}kbps` : '', P.direct ? '' : `→ MP3 ${qualityBr()}kbps に変換中`, normTxt].filter(Boolean).join(' · ');
  const npHTML = `
    <div class="np">
      ${coverHTML(t.album_id, t.cv, 800, t.album || t.title)}
      <div style="width:100%">
        <div class="np-info">
          <div class="txt">
            <div class="t ellipsis">${esc(t.title)}</div>
            <div class="s ellipsis"><a href="#/artist/${t.artist_id}" data-nav>${esc(t.artist || t.album_artist)}</a>${t.album ? ` — <a href="#/album/${t.album_id}" data-nav>${esc(t.album)}</a>` : ''}</div>
          </div>
          ${favButton('track', t)}
          <button class="icon-btn" data-more-track="${t.id}" aria-label="その他">${I.more}</button>
        </div>
        <div class="np-seek">
          <input type="range" class="seekbar" min="0" max="1000" value="0" aria-label="再生位置">
          <div class="times"><span data-cur>0:00</span><span data-tot>${fmtTime(duration())}</span></div>
        </div>
        <div class="np-ctl">
          <button class="icon-btn side${P.shuffle ? ' on' : ''}" data-shuffle aria-label="シャッフル">${I.shuffle}</button>
          <button class="icon-btn" data-prev aria-label="前の曲">${I.prev}</button>
          <button class="icon-btn play" data-pp aria-label="再生">${playing ? I.pause : I.play}</button>
          <button class="icon-btn" data-next aria-label="次の曲">${I.next}</button>
          <button class="icon-btn side${P.repeat !== 'off' ? ' on' : ''}" data-repeat aria-label="リピート">${P.repeat === 'one' ? I.repeatOne : I.repeat}</button>
        </div>
        <div class="tech">${t.q ? qBadge(t.q) + ' ' : ''}${esc(tech)}</div>
        ${t.q && !P.direct ? '<div class="tech" style="margin-top:2px">※ 音質設定によりMP3で再生中（設定で「オリジナル」にするとそのまま再生されます）</div>' : ''}
      </div>
    </div>`;
  const tabs = wide
    ? `<div class="tabs2"><button data-stab="queue" class="${sheetTab !== 'lyrics' ? 'on' : ''}">キュー</button><button data-stab="lyrics" class="${sheetTab === 'lyrics' ? 'on' : ''}">歌詞</button></div>`
    : `<div class="tabs2"><button data-stab="np" class="${sheetTab === 'np' ? 'on' : ''}">再生中</button><button data-stab="queue" class="${sheetTab === 'queue' ? 'on' : ''}">キュー</button><button data-stab="lyrics" class="${sheetTab === 'lyrics' ? 'on' : ''}">歌詞</button></div>`;
  let body;
  if (wide) {
    body = `<div class="sh-body wide">${npHTML}<div class="side-panel">${sheetTab === 'lyrics' ? '<div class="lyrics" id="lyr"></div>' : queuePanelHTML()}</div></div>`;
  } else if (sheetTab === 'queue') {
    body = `<div class="sh-body">${queuePanelHTML()}</div>`;
  } else if (sheetTab === 'lyrics') {
    body = `<div class="sh-body"><div class="np-info" style="margin-top:12px">${coverHTML(t.album_id, t.cv, 96, t.album || t.title, '', 'width:48px;flex:none')}<div class="txt"><div class="t ellipsis" style="font-size:16px">${esc(t.title)}</div><div class="s ellipsis">${esc(t.artist)}</div></div></div><div class="lyrics" id="lyr"></div></div>`;
  } else {
    body = `<div class="sh-body">${npHTML}</div>`;
  }
  el.innerHTML = `<div class="sh-top"><button class="icon-btn" data-close-sheet aria-label="閉じる">${I.down}</button>${tabs}<span style="width:40px"></span></div>${body}`;
  $$('.seekbar', el).forEach(bindSeek);
  updateProgress();
  if ($('#lyr', el)) loadLyrics();
  const q = $('.qlist .playing', el); if (q) q.scrollIntoView({ block: 'center' });
  tintFromCover($('.np .cover img', el));
}

function tintFromCover(img) {
  const sheet = $('#sheet');
  sheet.style.removeProperty('--np-tint');
  if (!img) return;
  const apply = () => {
    try {
      const c = document.createElement('canvas'); c.width = c.height = 8;
      const g = c.getContext('2d'); g.drawImage(img, 0, 0, 8, 8);
      const d = g.getImageData(0, 0, 8, 8).data;
      let r = 0, gg = 0, b = 0;
      for (let i = 0; i < d.length; i += 4) { r += d[i]; gg += d[i + 1]; b += d[i + 2]; }
      const n = d.length / 4;
      sheet.style.setProperty('--np-tint', `rgb(${Math.round(r / n * 0.55)},${Math.round(gg / n * 0.55)},${Math.round(b / n * 0.55)})`);
    } catch {}
  };
  if (img.complete && img.naturalWidth) apply(); else img.addEventListener('load', apply, { once: true });
}

function queuePanelHTML() {
  const rows = P.queue.map((t, i) => `
    <div class="trow${i === P.idx ? ' playing' : ''}" data-qi="${i}" data-id="${t.id}">
      ${coverHTML(t.album_id, t.cv, 96, t.album || t.title)}
      <div class="info"><div class="t tline"><span class="ellipsis">${esc(t.title)}</span>${qBadge(t.q)}</div><div class="s ellipsis">${esc(t.artist)}</div></div>
      <span class="dur">${fmtTime(t.duration)}</span>
      <button class="icon-btn" data-qrm="${i}" aria-label="キューから削除">${I.close}</button>
    </div>`).join('');
  return `<div style="display:flex;align-items:center;justify-content:space-between;margin-top:12px">
      <div class="muted small">${P.queue.length}曲 · ${fmtLong(P.queue.reduce((s, t) => s + (t.duration || 0), 0))}</div>
      <div style="display:flex;gap:6px"><button class="btn small" data-qsave>${I.playlistAdd}保存</button><button class="btn small" data-qclear>クリア</button></div>
    </div><div class="qlist">${rows}</div>`;
}

function renderQueuePanel() {
  if (sheetOpen && (sheetTab === 'queue' || matchMedia('(min-width: 900px)').matches && sheetTab !== 'lyrics')) renderSheet();
}

/* ---- lyrics ---- */

let lyr = null;
let lyrCur = -1;
let lyrUserScroll = 0;

async function loadLyrics() {
  const t = cur();
  const box = $('#lyr');
  if (!t || !box) return;
  box.addEventListener('wheel', () => (lyrUserScroll = Date.now()), { passive: true });
  box.addEventListener('touchmove', () => (lyrUserScroll = Date.now()), { passive: true });
  let l = state.lyricsCache.get(t.id);
  if (!l) {
    box.className = 'lyrics plain';
    box.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    try { l = await api(`/tracks/${t.id}/lyrics`); } catch { l = { synced: false, lines: [] }; }
    state.lyricsCache.set(t.id, l);
    if (cur() !== t || !$('#lyr')) return;
  }
  lyr = l; lyrCur = -1;
  const b = $('#lyr');
  if (!l.lines.length) { b.className = 'lyrics plain'; b.innerHTML = '<div class="empty">歌詞がありません<div class="small dim" style="margin-top:8px">曲と同じ名前の .lrc ファイルを置くか、タグに歌詞を埋め込むと表示されます</div></div>'; return; }
  b.className = l.synced ? 'lyrics' : 'lyrics plain';
  b.innerHTML = l.lines.map((x, i) => `<p data-li="${i}">${esc(x.text) || '&nbsp;'}</p>`).join('');
  updateLyrics(true);
}

function updateLyrics(force = false) {
  if (!lyr || !lyr.synced) return;
  const box = $('#lyr');
  if (!box) return;
  const pos = position() + 0.25;
  let i = -1;
  for (let k = 0; k < lyr.lines.length; k++) { if (lyr.lines[k].t <= pos) i = k; else break; }
  if (i === lyrCur && !force) return;
  lyrCur = i;
  $$('p.cur', box).forEach((p) => p.classList.remove('cur'));
  const p = i >= 0 ? $(`p[data-li="${i}"]`, box) : null;
  if (p) {
    p.classList.add('cur');
    if (Date.now() - lyrUserScroll > 3000) box.scrollTo({ top: p.offsetTop - box.clientHeight / 2 + p.clientHeight / 2, behavior: force ? 'auto' : 'smooth' });
  }
}

/* ============================================================
   Layers (sheet / menus / dialogs) — Android back button closes them
   ============================================================ */

const layers = [];
let pendingNav = null;
function pushLayer(close) { layers.push(close); history.pushState({ layer: layers.length }, ''); }
function popLayer(nav) { if (nav) pendingNav = nav; if (layers.length) history.back(); else if (nav) location.hash = nav; }
window.addEventListener('popstate', () => {
  if (layers.length) { const c = layers.pop(); c(); }
  if (pendingNav) { const n = pendingNav; pendingNav = null; if (!layers.length) location.hash = n; else popLayer(n); }
});

function showOverlay(html, { center = false } = {}) {
  const ov = $('#overlay');
  ov.className = center ? 'center' : '';
  ov.innerHTML = html;
  ov.hidden = false;
  pushLayer(() => { ov.hidden = true; ov.innerHTML = ''; });
  return ov;
}
function closeOverlay(nav) { popLayer(nav); }

$('#overlay').addEventListener('click', (e) => { if (e.target.id === 'overlay') closeOverlay(); });

function menu(headerHTML, items) {
  const ov = showOverlay(`<div class="menu">${headerHTML ? `<div class="mh">${headerHTML}</div>` : ''}${items.map((it, i) => `<button class="mi" data-mi="${i}">${it.icon || ''}<span>${esc(it.label)}</span></button>`).join('')}</div>`);
  $$('[data-mi]', ov).forEach((b) => b.addEventListener('click', () => {
    const it = items[+b.dataset.mi];
    if (it.nav) { closeOverlay(it.nav); return; }
    closeOverlay();
    setTimeout(() => it.run && it.run(), 30);
  }));
}

function trackMenu(t, extra = []) {
  const items = [
    { icon: I.playNext, label: '次に再生', run: () => addNext([t]) },
    { icon: I.queue, label: 'キューに追加', run: () => addToQueue([t]) },
    { icon: I.playlistAdd, label: 'プレイリストに追加', run: () => addToPlaylistDialog([t.id]) },
    { icon: t.fav ? I.heartFill : I.heart, label: t.fav ? 'お気に入りから削除' : 'お気に入りに追加', run: () => toggleFav('track', t.id, { classList: { contains: () => !!t.fav } }) },
    ...extra,
  ];
  if (t.album_id) items.push({ icon: I.album, label: 'アルバムを表示', nav: `#/album/${t.album_id}` });
  if (t.artist_id) items.push({ icon: I.artist, label: 'アーティストを表示', nav: `#/artist/${t.artist_id}` });
  items.push({ icon: I.download, label: 'ダウンロード', run: () => { const a = document.createElement('a'); a.href = `/api/tracks/${t.id}/stream?dl=1`; a.download = ''; document.body.appendChild(a); a.click(); a.remove(); } });
  menu(`${coverHTML(t.album_id, t.cv, 96, t.album || t.title)}<div class="ellipsis"><div class="ellipsis" style="font-weight:600">${esc(t.title)}</div><div class="small muted ellipsis">${esc(t.artist)}${t.album ? ' · ' + esc(t.album) : ''}</div></div>`, items);
}

function dialog(html) {
  return showOverlay(`<div class="dialog">${html}</div>`, { center: true });
}

async function addToPlaylistDialog(trackIds) {
  let pls = [];
  try { pls = await api('/playlists'); } catch (e) { return toast(e.message, true); }
  const ov = dialog(`<h3>プレイリストに追加</h3>
    <div class="form-grid" style="grid-template-columns:1fr auto;margin-bottom:12px"><input class="field" id="npl" placeholder="新しいプレイリスト名" style="margin:0"><button class="btn primary small" id="npl-go">作成</button></div>
    <div class="dir-list">${pls.map((p) => `<button data-pl="${p.id}">${I.playlist}<span class="ellipsis">${esc(p.name)}</span><span class="dim small" style="margin-left:auto">${p.track_count}曲</span></button>`).join('') || '<div class="empty" style="padding:20px">まだプレイリストがありません</div>'}</div>
    <div class="btns"><button class="btn" data-cancel>キャンセル</button></div>`);
  $('[data-cancel]', ov).onclick = () => closeOverlay();
  $$('[data-pl]', ov).forEach((b) => (b.onclick = async () => {
    try { await api(`/playlists/${b.dataset.pl}/tracks`, { method: 'POST', body: { track_ids: trackIds } }); toast('プレイリストに追加しました'); closeOverlay(); refreshSidebarPlaylists(); } catch (e) { toast(e.message, true); }
  }));
  const create = async () => {
    const name = $('#npl', ov).value.trim();
    if (!name) return $('#npl', ov).focus();
    try { await api('/playlists', { method: 'POST', body: { name, track_ids: trackIds } }); toast(`「${name}」を作成しました`); closeOverlay(); refreshSidebarPlaylists(); } catch (e) { toast(e.message, true); }
  };
  $('#npl-go', ov).onclick = create;
  $('#npl', ov).addEventListener('keydown', (e) => { if (e.key === 'Enter') create(); });
}

/* ============================================================
   Rendering helpers
   ============================================================ */

function albumCard(al) {
  return `<a class="card" href="#/album/${al.id}">
    <div class="cover" style="background:${grad(al.title)}"><span class="ph">${initial(al.title)}</span><img loading="lazy" decoding="async" src="/api/albums/${al.id}/cover?w=300&v=${al.cv}" alt="" data-rm-err>${qBadge(al.q, { spec: fmtSpec(al.max_bits, al.max_rate) })}
      <button class="play-fab" data-play-album="${al.id}" aria-label="再生">${I.play}</button></div>
    <div class="t ellipsis">${esc(al.title)}</div>
    <div class="s ellipsis">${esc(al.artist)}${al.year ? ' · ' + al.year : ''}</div></a>`;
}

function artistCard(ar) {
  return `<a class="card artist" href="#/artist/${ar.id}">${coverHTML(ar.cover_album, 0, 300, ar.name, 'round')}<div class="t ellipsis">${esc(ar.name)}</div></a>`;
}

function trackRows(tracks, opts = {}) {
  const key = regList(tracks);
  let lastDisc = null;
  const multiDisc = opts.discs && new Set(tracks.map((t) => t.disc_no || 1)).size > 1;
  const rows = tracks.map((t, i) => {
    let h = '';
    if (multiDisc && (t.disc_no || 1) !== lastDisc) { lastDisc = t.disc_no || 1; h = `<div class="disc-h">ディスク ${lastDisc}</div>`; }
    const sub = opts.album
      ? (t.artist && t.artist !== opts.albumArtist ? esc(t.artist) : '')
      : `${esc(t.artist)}${t.album && !opts.noAlbum ? ' · ' + esc(t.album) : ''}`;
    const lead = opts.num
      ? `<span class="num" data-n="${t.track_no || ''}">${t.track_no || ''}</span>`
      : (opts.rank ? `<span class="num" data-n="${i + 1}">${i + 1}</span>` : '') + coverHTML(t.album_id, t.cv, 96, t.album || t.title);
    const badge = opts.albumQ && t.q === opts.albumQ ? '' : qBadge(t.q, { spec: fmtSpec(t.bit_depth, t.sample_rate) });
    return `${h}<div class="trow" data-list="${key}" data-i="${i}" data-id="${t.id}">
      ${lead}
      <div class="info"><div class="t tline"><span class="ellipsis">${esc(t.title)}</span>${badge}</div>${sub ? `<div class="s ellipsis">${sub}</div>` : ''}</div>
      ${favButton('track', t)}
      <span class="dur">${fmtTime(t.duration)}</span>
      ${opts.edit ? `<button class="icon-btn" data-up="${i}" aria-label="上へ">${I.up}</button><button class="icon-btn" data-rm="${i}" aria-label="削除">${I.close}</button>` : `<button class="icon-btn" data-more="${i}" aria-label="その他">${I.more}</button>`}
    </div>`;
  }).join('');
  return `<div class="tracks" data-listkey="${key}">${rows}</div>`;
}

function section(title, inner, link) {
  return `<section class="section"><div class="section-h"><h2>${esc(title)}</h2>${link ? `<a href="${link}">すべて表示</a>` : ''}</div>${inner}</section>`;
}

function setTop(title, { back = false, actions = '' } = {}) {
  $('#top').innerHTML = `${back ? `<button class="icon-btn" data-back aria-label="戻る">${I.back}</button>` : ''}<h1>${esc(title)}</h1>${actions}
    <a class="icon-btn mobile-only" href="#/settings" aria-label="設定">${I.settings}</a>`;
}

/* ============================================================
   Router & views
   ============================================================ */

const routes = [
  [/^#\/login$/, viewLogin, { auth: true }],
  [/^#\/setup$/, viewSetup, { auth: true }],
  [/^#\/?$/, viewHome, { tab: 'home' }],
  [/^#\/search$/, viewSearch, { tab: 'search' }],
  [/^#\/albums$/, viewAlbums, { tab: 'albums' }],
  [/^#\/album\/(\d+)$/, viewAlbum, { tab: 'albums' }],
  [/^#\/artists$/, viewArtists, { tab: 'artists' }],
  [/^#\/artist\/(\d+)$/, viewArtist, { tab: 'artists' }],
  [/^#\/songs$/, viewSongs, { tab: 'songs' }],
  [/^#\/favorites$/, viewFavorites, { tab: 'favorites' }],
  [/^#\/playlists$/, viewPlaylists, { tab: 'playlists' }],
  [/^#\/playlist\/(\d+)$/, viewPlaylist, { tab: 'playlists' }],
  [/^#\/settings$/, viewSettings, { tab: 'settings' }],
];

let routeCleanup = null;
let routeSeq = 0;
const scrollPos = new Map();

function go(h) { if (location.hash === h) route(); else location.hash = h; }

async function route() {
  const seq = ++routeSeq;
  if (routeCleanup) { try { routeCleanup(); } catch {} routeCleanup = null; }
  const full = location.hash || '#/';
  const [path, qs] = full.split('?');
  const params = new URLSearchParams(qs || '');
  const r = routes.find(([re]) => re.test(path)) || routes[2];
  const m = path.match(r[0]) || [];
  const opt = r[2] || {};
  document.body.classList.toggle('auth-mode', !!opt.auth);
  if (!opt.auth && !state.me) {
    try {
      state.me = await api('/me', { quiet: true });
      renderChrome();
      restoreQueue();
    } catch (e) {
      if (e.status === 401) {
        const s = await api('/setup', { quiet: true }).catch(() => ({ needs_setup: false }));
        go(s.needs_setup ? '#/setup' : '#/login');
      } else {
        $('#view').innerHTML = `<div class="empty">${esc(e.message)}<br><button class="btn" data-reload>再読み込み</button></div>`;
      }
      return;
    }
  }
  if (opt.auth) { $('#side').innerHTML = ''; $('#tabs').innerHTML = ''; $('#top').innerHTML = ''; $('#player').className = 'empty'; }
  else { highlightNav(opt.tab); if ($('#player').className === 'empty' && cur()) renderPlayer(); }
  state.lists.clear();
  $('#view').innerHTML = '<div class="loading"><div class="spinner"></div></div>';
  try {
    await r[1](m, params, () => seq !== routeSeq);
  } catch (e) {
    if (seq === routeSeq && e.status !== 401) $('#view').innerHTML = `<div class="empty">エラー: ${esc(e.message)}</div>`;
  }
  if (seq === routeSeq) {
    markPlaying();
    const y = scrollPos.get(full);
    window.scrollTo(0, y || 0);
  }
}
window.addEventListener('hashchange', route);
window.addEventListener('scroll', throttle(() => scrollPos.set(location.hash || '#/', window.scrollY), 200), { passive: true });

function renderChrome() {
  const tabs = [['home', '#/', I.home, 'ホーム'], ['search', '#/search', I.search, '検索'], ['albums', '#/albums', I.album, 'アルバム'], ['artists', '#/artists', I.artist, 'アーティスト'], ['playlists', '#/playlists', I.playlist, 'プレイリスト']];
  $('#tabs').innerHTML = tabs.map(([k, h, ic, l]) => `<a href="${h}" data-tab="${k}">${ic}<span>${l}</span></a>`).join('');
  const side = [['home', '#/', I.home, 'ホーム'], ['search', '#/search', I.search, '検索'], ['albums', '#/albums', I.album, 'アルバム'], ['artists', '#/artists', I.artist, 'アーティスト'], ['songs', '#/songs', I.note, '曲'], ['favorites', '#/favorites', I.heart, 'お気に入り'], ['playlists', '#/playlists', I.playlist, 'プレイリスト']];
  $('#side').innerHTML = `<div class="brand"><img src="/icons/icon-192.png" alt="">Hako</div>
    ${side.map(([k, h, ic, l]) => `<a class="nav" href="${h}" data-tab="${k}">${ic}<span>${l}</span></a>`).join('')}
    <div id="side-pls"></div>
    <div style="flex:1"></div>
    <a class="nav" href="#/settings" data-tab="settings">${I.settings}<span>設定</span></a>`;
  refreshSidebarPlaylists();
}

async function refreshSidebarPlaylists() {
  try { state.playlists = await api('/playlists'); } catch { return; }
  const box = $('#side-pls');
  if (box) box.innerHTML = state.playlists.length ? `<div class="sep"></div>${state.playlists.slice(0, 30).map((p) => `<a class="pl" href="#/playlist/${p.id}">${esc(p.name)}</a>`).join('')}` : '';
}

function highlightNav(tab) {
  $$('[data-tab]').forEach((a) => a.classList.toggle('active', a.dataset.tab === tab));
}

/* ---------------- auth ---------------- */

function authForm(title, button, onSubmit, extra = '') {
  $('#view').innerHTML = `<div class="auth"><form autocomplete="on">
    <img class="logo" src="/icons/icon-192.png" alt="">
    <h2>${title}</h2>${extra}
    <input class="field" name="name" placeholder="ユーザー名" autocomplete="username" required>
    <input class="field" name="password" type="password" placeholder="パスワード" autocomplete="${button === 'ログイン' ? 'current-password' : 'new-password'}" required>
    <div class="err"></div>
    <button class="btn primary" type="submit">${button}</button></form></div>`;
  const f = $('#view form');
  f.addEventListener('submit', async (e) => {
    e.preventDefault();
    const err = $('.err', f);
    const btn = $('button[type=submit]', f);
    err.textContent = '';
    btn.disabled = true;
    try {
      const r = await onSubmit(f.name.value, f.password.value);
      if (r && r.mfa_required) { mfaForm(r.ticket); return; }
      signedIn(r);
    } catch (ex) { err.textContent = ex.message; } finally { btn.disabled = false; }
  });
  f.name.focus();
}

function signedIn(user) {
  state.me = user;
  renderChrome();
  restoreQueue();
  if (user.recovery_left !== undefined) {
    toast(user.recovery_left > 0 ? `回復コードでログインしました（残り${user.recovery_left}個）` : '回復コードを使い切りました。設定から新しく発行してください', user.recovery_left <= 2);
  }
  go('#/');
}

// second step: authenticator code or recovery code
function mfaForm(ticket) {
  let recovery = false;
  const draw = () => {
    $('#view').innerHTML = `<div class="auth"><form autocomplete="off">
      <img class="logo" src="/icons/icon-192.png" alt="">
      <h2>2段階認証</h2>
      <p class="muted small" style="text-align:center;margin:-4px 0 6px">${recovery ? '保存しておいた回復コード（例: k7m2-x9qp）を入力してください' : '認証アプリに表示されている6桁のコードを入力してください'}</p>
      ${recovery
        ? '<input class="field" name="code" placeholder="xxxx-xxxx" autocomplete="off" autocapitalize="off" spellcheck="false" required>'
        : '<input class="field" name="code" placeholder="123456" inputmode="numeric" pattern="[0-9 ]*" maxlength="7" autocomplete="one-time-code" required style="text-align:center;font-size:22px;letter-spacing:.3em">'}
      <div class="err"></div>
      <button class="btn primary" type="submit">確認</button>
      <button class="btn" type="button" data-switch>${recovery ? '認証アプリのコードを使う' : '回復コードを使う'}</button>
      <button class="btn" type="button" data-cancel-mfa>キャンセル</button></form></div>`;
    const f = $('#view form');
    f.code.focus();
    $('[data-switch]', f).onclick = () => { recovery = !recovery; draw(); };
    $('[data-cancel-mfa]', f).onclick = () => viewLogin();
    if (!recovery) f.code.addEventListener('input', () => { if (/^\d{6}$/.test(f.code.value.replace(/\s/g, ''))) f.requestSubmit(); });
    f.addEventListener('submit', async (e) => {
      e.preventDefault();
      const err = $('.err', f);
      const btn = $('button[type=submit]', f);
      err.textContent = '';
      btn.disabled = true;
      try {
        signedIn(await api('/login/mfa', { method: 'POST', body: { ticket, code: f.code.value }, quiet: true }));
      } catch (ex) {
        err.textContent = ex.message;
        f.code.value = '';
        f.code.focus();
        if (ex.status === 401 || ex.status === 429) setTimeout(() => viewLogin(), 2500);
      } finally { btn.disabled = false; }
    });
  };
  draw();
}

async function viewLogin() {
  authForm('Hako にログイン', 'ログイン', (name, password) => api('/login', { method: 'POST', body: { name, password }, quiet: true }));
}

async function viewSetup() {
  const s = await api('/setup', { quiet: true });
  if (!s.needs_setup) return go('#/login');
  authForm('ようこそ', '管理者アカウントを作成', (name, password) => api('/setup', { method: 'POST', body: { name, password }, quiet: true }),
    `<p class="muted small" style="text-align:center;margin:-4px 0 6px">最初に管理者アカウントを作成してください（パスワードは${s.min_password || 8}文字以上）</p>`);
}

/* ---------------- home ---------------- */

async function viewHome(m, params, stale) {
  setTop('ホーム');
  const [d, scan] = await Promise.all([api('/home'), api('/scan').catch(() => ({}))]);
  if (stale()) return;
  let html = `<div class="chips">
    <a class="chip" href="#/songs">${I.note}曲</a>
    <a class="chip" href="#/favorites">${I.heart}お気に入り</a>
    <a class="chip" href="#/albums?quality=hires"><i class="qb hires">Hi-Res</i>ハイレゾ</a>
    <a class="chip" href="#/albums?quality=lossless"><i class="qb lossless">Lossless</i>ロスレス</a>
    <button class="chip" data-shuffle-all>${I.random}全曲シャッフル</button>
    <a class="chip" href="#/songs?sort=recent">${I.history}最近再生した曲</a></div>`;
  if (scan.running) html += `<div class="banner"><div class="spinner"></div>ライブラリをスキャン中… ${esc(scan.phase || '')} ${scan.total ? `${scan.done}/${scan.total}` : ''}</div>`;
  if (!d.recent.length) {
    html += `<div class="empty">${I.note}<p>まだ曲がありません</p>${state.me.admin ? '<a class="btn primary" href="#/settings">音楽フォルダを追加する</a>' : '<p class="small">管理者に音楽フォルダの追加を依頼してください</p>'}</div>`;
  } else {
    if (d.played.length) html += section('最近聴いたアルバム', `<div class="scroller">${d.played.map(albumCard).join('')}</div>`);
    html += section('最近追加したアルバム', `<div class="scroller">${d.recent.map(albumCard).join('')}</div>`, '#/albums');
    if (d.top.length) html += section('よく聴く曲', trackRows(d.top.slice(0, 10), { rank: true }), '#/songs?sort=plays');
    if (d.random.length > 4) html += section('ランダムピックアップ', `<div class="scroller">${d.random.map(albumCard).join('')}</div>`);
  }
  $('#view').innerHTML = html;
  if (scan.running) {
    const iv = setInterval(async () => {
      const s = await api('/scan').catch(() => null);
      if (s && !s.running) { clearInterval(iv); if (location.hash === '' || location.hash === '#/') route(); }
      else if (s) { const b = $('.banner'); if (b) b.innerHTML = `<div class="spinner"></div>ライブラリをスキャン中… ${esc(s.phase || '')} ${s.total ? `${s.done}/${s.total}` : ''}`; }
    }, 2500);
    routeCleanup = () => clearInterval(iv);
  }
}

/* ---------------- search ---------------- */

async function viewSearch(m, params) {
  setTop('検索');
  const q0 = params.get('q') || '';
  $('#view').innerHTML = `<div class="toolbar"><input class="field grow" id="sq" type="search" placeholder="曲名・アーティスト・アルバム" value="${esc(q0)}" autocomplete="off" enterkeyhint="search"></div><div id="sres"></div>`;
  const inp = $('#sq');
  let t, seq = 0;
  const run = async () => {
    const q = inp.value.trim();
    history.replaceState(history.state, '', q ? `#/search?q=${encodeURIComponent(q)}` : '#/search');
    const my = ++seq;
    const box = $('#sres');
    if (!q) { box.innerHTML = '<div class="empty">キーワードを入力してください</div>'; return; }
    box.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    const r = await api('/search?q=' + encodeURIComponent(q)).catch((e) => ({ error: e.message }));
    if (my !== seq || !$('#sres')) return;
    if (r.error) { box.innerHTML = `<div class="empty">${esc(r.error)}</div>`; return; }
    let html = '';
    if (r.artists.length) html += section('アーティスト', `<div class="scroller">${r.artists.map(artistCard).join('')}</div>`);
    if (r.albums.length) html += section('アルバム', `<div class="scroller">${r.albums.map(albumCard).join('')}</div>`);
    if (r.tracks.length) html += section('曲', trackRows(r.tracks));
    box.innerHTML = html || '<div class="empty">見つかりませんでした</div>';
    markPlaying();
  };
  inp.addEventListener('input', () => { clearTimeout(t); t = setTimeout(run, 250); });
  if (!matchMedia('(pointer: coarse)').matches || !q0) inp.focus();
  run();
}

/* ---------------- albums ---------------- */

async function viewAlbums(m, params, stale) {
  const sort = params.get('sort') || localStorage.getItem('hako.albumSort') || 'added';
  const quality = params.get('quality') || '';
  setTop(quality === 'hires' ? 'ハイレゾのアルバム' : quality === 'lossless' ? 'ロスレスのアルバム' : 'アルバム');
  const albums = await api('/albums?sort=' + encodeURIComponent(sort) + (quality ? '&quality=' + quality : ''));
  if (stale()) return;
  const opts = [['added', '追加日'], ['title', 'タイトル'], ['artist', 'アーティスト'], ['year', '年'], ['random', 'ランダム']];
  const qopts = [['', 'すべての音質'], ['lossless', 'ロスレス以上'], ['hires', 'ハイレゾのみ']];
  $('#view').innerHTML = `<div class="toolbar">
      <input class="field grow" id="af" type="search" placeholder="アルバムを絞り込み" autocomplete="off">
      <select class="field" id="as">${opts.map(([v, l]) => `<option value="${v}"${v === sort ? ' selected' : ''}>${l}</option>`).join('')}</select>
      <select class="field" id="aq">${qopts.map(([v, l]) => `<option value="${v}"${v === quality ? ' selected' : ''}>${l}</option>`).join('')}</select>
    </div><div class="muted small" id="acount" style="margin-bottom:10px"></div><div class="grid" id="ag"></div>`;
  const draw = () => {
    const f = $('#af').value.trim().toLowerCase();
    const list = f ? albums.filter((a) => a.title.toLowerCase().includes(f) || a.artist.toLowerCase().includes(f)) : albums;
    $('#ag').innerHTML = list.map(albumCard).join('') || '<div class="empty" style="grid-column:1/-1">アルバムがありません</div>';
    $('#acount').textContent = `${list.length}枚のアルバム`;
  };
  $('#af').addEventListener('input', draw);
  const nav = () => go('#/albums?sort=' + $('#as').value + ($('#aq').value ? '&quality=' + $('#aq').value : ''));
  $('#as').addEventListener('change', (e) => { try { localStorage.setItem('hako.albumSort', e.target.value); } catch {} nav(); });
  $('#aq').addEventListener('change', nav);
  draw();
}

async function playAlbum(id, shuffle = false) {
  try {
    const d = await api(`/albums/${id}`);
    playList(d.tracks, 0, shuffle);
  } catch (e) { toast(e.message, true); }
}

async function viewAlbum(m, params, stale) {
  const d = await api(`/albums/${m[1]}`);
  if (stale()) return;
  const al = d.album, tracks = d.tracks;
  setTop(al.title, { back: true });
  const meta = [al.year, al.genre, `${tracks.length}曲`, fmtLong(al.duration)].filter(Boolean).join(' · ');
  $('#view').innerHTML = `
    <div class="hero">${coverHTML(al.id, al.cv, 500, al.title)}
      <div class="meta"><div class="kind">アルバム</div><h2>${esc(al.title)}</h2>
        <div class="by"><a href="#/artist/${al.artist_id}"><b>${esc(al.artist)}</b></a></div>
        <div class="muted small">${esc(meta)}</div>
        ${al.q ? `<div class="spec" style="margin-top:8px">${qBadge(al.q, { big: true })}<span class="small muted">${esc(fmtSpec(al.max_bits, al.max_rate))}</span></div>` : ''}</div></div>
    <div class="actions center-m">
      <button class="btn primary" id="pa">${I.play}再生</button>
      <button class="btn" id="ps">${I.shuffle}シャッフル</button>
      ${favButton('album', al)}
      <button class="icon-btn" id="am" aria-label="その他">${I.more}</button>
    </div>
    ${trackRows(tracks, { num: true, album: true, albumArtist: al.artist, discs: true, albumQ: al.q })}`;
  $('#pa').onclick = () => playList(tracks, 0);
  $('#ps').onclick = () => playList(tracks, -1, true);
  $('#am').onclick = () => menu(`${coverHTML(al.id, al.cv, 96, al.title)}<div class="ellipsis"><b>${esc(al.title)}</b><div class="small muted">${esc(al.artist)}</div></div>`, [
    { icon: I.playNext, label: '次に再生', run: () => addNext(tracks) },
    { icon: I.queue, label: 'キューに追加', run: () => addToQueue(tracks) },
    { icon: I.playlistAdd, label: 'プレイリストに追加', run: () => addToPlaylistDialog(tracks.map((t) => t.id)) },
    { icon: I.artist, label: 'アーティストを表示', nav: `#/artist/${al.artist_id}` },
  ]);
}

/* ---------------- artists ---------------- */

async function viewArtists(m, params, stale) {
  setTop('アーティスト');
  const artists = await api('/artists');
  if (stale()) return;
  $('#view').innerHTML = `<div class="toolbar"><input class="field grow" id="arf" type="search" placeholder="アーティストを絞り込み" autocomplete="off"></div><div class="alist" id="arl"></div>`;
  const draw = () => {
    const f = $('#arf').value.trim().toLowerCase();
    const list = f ? artists.filter((a) => a.name.toLowerCase().includes(f)) : artists;
    let last = '';
    $('#arl').innerHTML = list.map((a) => {
      const ch = [...a.name.trim()][0] || '#';
      const L = /[a-z]/i.test(ch) ? ch.toUpperCase() : /[0-9]/.test(ch) ? '#' : /[぀-ヿ]/.test(ch) ? 'かな' : /[一-鿿]/.test(ch) ? '漢字' : '他';
      const h = !f && L !== last ? `<div class="letter">${L}</div>` : '';
      last = L;
      return `${h}<a class="arow" href="#/artist/${a.id}">${coverHTML(a.cover_album, 0, 96, a.name, 'round')}<div class="ellipsis"><div class="ellipsis" style="font-weight:600">${esc(a.name)}</div><div class="small muted">${a.album_count}枚のアルバム</div></div></a>`;
    }).join('') || '<div class="empty">アーティストがいません</div>';
  };
  $('#arf').addEventListener('input', draw);
  draw();
}

async function viewArtist(m, params, stale) {
  const d = await api(`/artists/${m[1]}`);
  if (stale()) return;
  const ar = d.artist;
  setTop(ar.name, { back: true });
  $('#view').innerHTML = `
    <div class="hero">${coverHTML(ar.cover_album, 0, 500, ar.name, 'round')}
      <div class="meta"><div class="kind">アーティスト</div><h2>${esc(ar.name)}</h2>
      <div class="muted small">${d.albums.length}枚のアルバム · ${d.track_ids.length}曲</div></div></div>
    <div class="actions center-m">
      <button class="btn primary" id="pa">${I.play}再生</button>
      <button class="btn" id="ps">${I.shuffle}シャッフル</button>
      ${favButton('artist', ar)}
    </div>
    ${d.top.length ? section('人気の曲', trackRows(d.top.slice(0, 5), { rank: true, noAlbum: false })) : ''}
    ${d.albums.length ? section('アルバム', `<div class="grid">${d.albums.map(albumCard).join('')}</div>`) : ''}
    ${d.appears_on.length ? section('参加作品', `<div class="scroller">${d.appears_on.map(albumCard).join('')}</div>`) : ''}`;
  const getAll = () => api('/tracks/batch?ids=' + d.track_ids.slice(0, 2000).join(','));
  $('#pa').onclick = async () => playList(await getAll(), 0);
  $('#ps').onclick = async () => playList(await getAll(), -1, true);
}

/* ---------------- songs ---------------- */

async function viewSongs(m, params, stale) {
  const sort = params.get('sort') || 'title';
  const fav = params.get('fav') === '1';
  const quality = params.get('quality') || '';
  setTop(sort === 'recent' ? '最近再生した曲' : sort === 'plays' ? 'よく聴く曲' : '曲', { back: sort === 'recent' || sort === 'plays' });
  const opts = [['title', 'タイトル'], ['artist', 'アーティスト'], ['album', 'アルバム'], ['added', '追加日'], ['plays', '再生回数'], ['recent', '最近再生']];
  const qopts = [['', 'すべての音質'], ['lossless', 'ロスレス以上'], ['hires', 'ハイレゾのみ']];
  $('#view').innerHTML = `<div class="toolbar">
      <input class="field grow" id="sf" type="search" placeholder="曲を絞り込み" autocomplete="off">
      <select class="field" id="ss">${opts.map(([v, l]) => `<option value="${v}"${v === sort ? ' selected' : ''}>${l}</option>`).join('')}</select>
      <select class="field" id="sq2">${qopts.map(([v, l]) => `<option value="${v}"${v === quality ? ' selected' : ''}>${l}</option>`).join('')}</select>
    </div>
    <div class="actions" style="margin-top:0"><button class="btn primary small" id="sp">${I.play}すべて再生</button><button class="btn small" id="sh">${I.shuffle}シャッフル</button><span class="muted small" id="scount"></span></div>
    <div id="sl"></div><div id="sentinel" style="height:40px"></div>`;
  let all = [], total = 0, loading = false, q = '';
  const PAGE = 200;
  const key = regList(all);
  const load = async (reset) => {
    if (loading) return;
    if (reset) { all.length = 0; $('#sl').innerHTML = ''; }
    if (!reset && all.length >= total && total) return;
    loading = true;
    try {
      const r = await api(`/tracks?sort=${sort}&limit=${PAGE}&offset=${all.length}${fav ? '&fav=1' : ''}${quality ? '&quality=' + quality : ''}${q ? '&q=' + encodeURIComponent(q) : ''}`);
      if (stale()) return;
      total = r.total;
      const start = all.length;
      all.push(...r.tracks);
      const tmp = document.createElement('div');
      tmp.innerHTML = trackRows(r.tracks);
      // re-key rows so they refer to the full list
      $$('.trow', tmp).forEach((row, k) => { row.dataset.list = key; row.dataset.i = start + k; const mb = $('[data-more]', row); if (mb) mb.dataset.more = start + k; });
      let cont = $('#sl .tracks');
      if (!cont) { $('#sl').innerHTML = `<div class="tracks" data-listkey="${key}"></div>`; cont = $('#sl .tracks'); }
      cont.append(...$$('.trow', tmp));
      $('#scount').textContent = `${total}曲`;
      if (!total) $('#sl').innerHTML = '<div class="empty">曲がありません</div>';
      markPlaying();
    } finally { loading = false; }
  };
  state.lists.set(key, all);
  await load(true);
  const io = new IntersectionObserver((e) => { if (e[0].isIntersecting && all.length < total) load(false); }, { rootMargin: '600px' });
  io.observe($('#sentinel'));
  routeCleanup = () => io.disconnect();
  let ft;
  $('#sf').addEventListener('input', (e) => { clearTimeout(ft); ft = setTimeout(() => { q = e.target.value.trim(); load(true); }, 300); });
  const nav = () => go('#/songs?sort=' + $('#ss').value + ($('#sq2').value ? '&quality=' + $('#sq2').value : ''));
  $('#ss').addEventListener('change', nav);
  $('#sq2').addEventListener('change', nav);
  const fetchAll = async () => {
    if (total <= all.length) return all.slice();
    const r = await api(`/tracks?sort=${sort}&limit=2000&offset=0${quality ? '&quality=' + quality : ''}${q ? '&q=' + encodeURIComponent(q) : ''}`);
    return r.tracks;
  };
  $('#sp').onclick = async () => playList(await fetchAll(), 0);
  $('#sh').onclick = async () => playList(await fetchAll(), -1, true);
}

async function shuffleAll() {
  try {
    const r = await api('/tracks?sort=random&limit=500');
    if (!r.tracks.length) return toast('曲がありません');
    playList(r.tracks, 0);
    P.shuffle = true; onModeChange();
  } catch (e) { toast(e.message, true); }
}

/* ---------------- favorites ---------------- */

async function viewFavorites(m, params, stale) {
  setTop('お気に入り', { back: !matchMedia('(min-width: 900px)').matches });
  const d = await api('/favorites');
  if (stale()) return;
  let html = '';
  if (d.tracks.length) html += `<div class="actions"><button class="btn primary" id="fp">${I.play}再生</button><button class="btn" id="fs">${I.shuffle}シャッフル</button></div>`;
  if (d.artists.length) html += section('アーティスト', `<div class="scroller">${d.artists.map(artistCard).join('')}</div>`);
  if (d.albums.length) html += section('アルバム', `<div class="scroller">${d.albums.map(albumCard).join('')}</div>`);
  if (d.tracks.length) html += section(`曲（${d.tracks.length}）`, trackRows(d.tracks));
  $('#view').innerHTML = html || `<div class="empty">${I.heart}<p>まだお気に入りがありません</p><p class="small">曲やアルバムの ♡ をタップして追加できます</p></div>`;
  if (d.tracks.length) { $('#fp').onclick = () => playList(d.tracks, 0); $('#fs').onclick = () => playList(d.tracks, -1, true); }
}

/* ---------------- playlists ---------------- */

function playlistCover(p) {
  const c = p.covers || [];
  if (c.length >= 4) return `<div class="cover" style="display:grid;grid-template-columns:1fr 1fr">${c.slice(0, 4).map((id) => `<img style="position:static;width:100%;height:100%;aspect-ratio:1" loading="lazy" src="/api/albums/${id}/cover?w=200" alt="" data-hide-err>`).join('')}</div>`;
  return coverHTML(c[0] || 0, 0, 300, p.name);
}

async function viewPlaylists(m, params, stale) {
  setTop('プレイリスト', { actions: `<button class="icon-btn" id="npb" aria-label="新規作成">${I.plus}</button>` });
  const pls = await api('/playlists');
  if (stale()) return;
  state.playlists = pls;
  $('#view').innerHTML = `<div class="grid">${pls.map((p) => `<a class="card" href="#/playlist/${p.id}">${playlistCover(p)}<div class="t ellipsis">${esc(p.name)}</div><div class="s">${p.track_count}曲 · ${fmtLong(p.duration)}</div></a>`).join('')}</div>`
    + (pls.length ? '' : `<div class="empty">${I.playlist}<p>プレイリストはまだありません</p><button class="btn primary" id="npb2">${I.plus}新しいプレイリスト</button></div>`);
  const create = () => {
    const ov = dialog(`<h3>新しいプレイリスト</h3><input class="field" id="pln" placeholder="名前"><div class="btns"><button class="btn" data-cancel>キャンセル</button><button class="btn primary" id="plc">作成</button></div>`);
    $('[data-cancel]', ov).onclick = () => closeOverlay();
    const ok = async () => {
      const name = $('#pln', ov).value.trim();
      if (!name) return;
      try { const r = await api('/playlists', { method: 'POST', body: { name } }); refreshSidebarPlaylists(); closeOverlay(`#/playlist/${r.id}`); } catch (e) { toast(e.message, true); }
    };
    $('#plc', ov).onclick = ok;
    $('#pln', ov).addEventListener('keydown', (e) => { if (e.key === 'Enter') ok(); });
    $('#pln', ov).focus();
  };
  $('#npb').onclick = create;
  if ($('#npb2')) $('#npb2').onclick = create;
}

async function viewPlaylist(m, params, stale) {
  const id = m[1];
  const d = await api(`/playlists/${id}`);
  if (stale()) return;
  const pl = d.playlist;
  let tracks = d.tracks;
  let editing = false;
  setTop(pl.name, { back: true });
  const draw = () => {
    $('#view').innerHTML = `
      <div class="hero">${playlistCover(pl)}<div class="meta"><div class="kind">プレイリスト</div><h2>${esc(pl.name)}</h2><div class="muted small">${tracks.length}曲 · ${fmtLong(tracks.reduce((s, t) => s + t.duration, 0))}</div></div></div>
      <div class="actions center-m">
        <button class="btn primary" id="pa"${tracks.length ? '' : ' disabled'}>${I.play}再生</button>
        <button class="btn" id="ps"${tracks.length ? '' : ' disabled'}>${I.shuffle}シャッフル</button>
        <button class="btn${editing ? ' primary' : ''}" id="pe">${I.edit}${editing ? '完了' : '編集'}</button>
        <button class="icon-btn" id="pm" aria-label="その他">${I.more}</button>
      </div>
      ${tracks.length ? trackRows(tracks, { edit: editing }) : '<div class="empty">曲がありません<div class="small" style="margin-top:6px">曲の「⋯」メニューから追加できます</div></div>'}`;
    $('#pa').onclick = () => playList(tracks, 0);
    $('#ps').onclick = () => playList(tracks, -1, true);
    $('#pe').onclick = () => { editing = !editing; draw(); markPlaying(); };
    $('#pm').onclick = () => menu(`<b>${esc(pl.name)}</b>`, [
      { icon: I.queue, label: 'キューに追加', run: () => addToQueue(tracks) },
      { icon: I.edit, label: '名前を変更', run: rename },
      { icon: I.trash, label: 'プレイリストを削除', run: del },
    ]);
    $$('[data-rm]').forEach((b) => (b.onclick = (e) => { e.stopPropagation(); tracks.splice(+b.dataset.rm, 1); save(); }));
    $$('[data-up]').forEach((b) => (b.onclick = (e) => { e.stopPropagation(); const i = +b.dataset.up; if (i > 0) { [tracks[i - 1], tracks[i]] = [tracks[i], tracks[i - 1]]; save(); } }));
  };
  const save = async () => {
    draw(); markPlaying();
    try { await api(`/playlists/${id}`, { method: 'PUT', body: { track_ids: tracks.map((t) => t.id) } }); } catch (e) { toast(e.message, true); }
  };
  const rename = () => {
    const ov = dialog(`<h3>名前を変更</h3><input class="field" id="rn" value="${esc(pl.name)}"><div class="btns"><button class="btn" data-cancel>キャンセル</button><button class="btn primary" id="rok">保存</button></div>`);
    $('[data-cancel]', ov).onclick = () => closeOverlay();
    $('#rok', ov).onclick = async () => {
      const name = $('#rn', ov).value.trim(); if (!name) return;
      try { await api(`/playlists/${id}`, { method: 'PUT', body: { name } }); pl.name = name; closeOverlay(); setTop(name, { back: true }); draw(); refreshSidebarPlaylists(); } catch (e) { toast(e.message, true); }
    };
  };
  const del = () => {
    const ov = dialog(`<h3>「${esc(pl.name)}」を削除しますか？</h3><p class="muted small">曲のファイルは削除されません。</p><div class="btns"><button class="btn" data-cancel>キャンセル</button><button class="btn danger" id="dok">削除</button></div>`);
    $('[data-cancel]', ov).onclick = () => closeOverlay();
    $('#dok', ov).onclick = async () => {
      try { await api(`/playlists/${id}`, { method: 'DELETE' }); refreshSidebarPlaylists(); closeOverlay('#/playlists'); toast('削除しました'); } catch (e) { toast(e.message, true); }
    };
  };
  draw();
}

/* ---------------- settings & admin ---------------- */

async function viewSettings(m, params, stale) {
  setTop('設定', { back: !matchMedia('(min-width: 900px)').matches });
  const admin = state.me.admin;
  const [sec, libs, users, info, scan] = await Promise.all([
    api('/me/security'),
    ...(admin ? [api('/libraries'), api('/users'), api('/info'), api('/scan')] : [null, null, null, null]),
  ]);
  if (stale()) return;
  const q = state.prefs.quality;
  let html = `
    <div class="panel"><h3>再生</h3>
      <div class="row"><div class="grow">ストリーミング音質<div class="small muted">モバイル通信で聴くときは下げると通信量を節約できます（この端末のみ）</div></div>
        <select class="field" id="qual">${[['orig', 'オリジナル'], ['320', '320kbps'], ['192', '192kbps'], ['128', '128kbps']].map(([v, l]) => `<option value="${v}"${v === q ? ' selected' : ''}>${l}</option>`).join('')}</select></div>
      <div class="row"><div class="grow">音量の均一化<div class="small muted">曲ごとの音量差をなくします。「自動」はアルバムを通しで聴くときはアルバム単位、シャッフル時は曲単位で揃えます</div></div>
        <select class="field" id="norm">${[['auto', '自動'], ['track', '曲ごと'], ['album', 'アルバム単位'], ['off', 'オフ']].map(([v, l]) => `<option value="${v}"${v === state.prefs.norm ? ' selected' : ''}>${l}</option>`).join('')}</select></div>
      <div class="row"><div class="grow">基準の音量<div class="small muted">小さいほど全体の音量が下がり、より多くの曲が揃います</div></div>
        <select class="field" id="target"${state.prefs.norm === 'off' ? ' disabled' : ''}>${[[-12, '大きめ（-12 LUFS）'], [-14, '標準（-14 LUFS）'], [-16, '控えめ（-16 LUFS）'], [-18, '小さめ（-18 LUFS）']].map(([v, l]) => `<option value="${v}"${v === +state.prefs.target ? ' selected' : ''}>${l}</option>`).join('')}</select></div>
    </div>
    <div class="panel"><h3>アカウント</h3>
      <div class="row"><div class="grow">ログイン中: <b>${esc(state.me.name)}</b>${admin ? ' <span class="small muted">（管理者）</span>' : ''}</div><button class="btn small" id="logout">ログアウト</button></div>
      <div class="row"><form id="pwf" class="form-grid" style="width:100%">
        <input class="field" type="password" name="current" placeholder="現在のパスワード" autocomplete="current-password" required>
        <input class="field" type="password" name="new" placeholder="新しいパスワード（${sec.min_password}文字以上）" autocomplete="new-password" required>
        <div><button class="btn small" type="submit">パスワードを変更</button><span class="small muted" style="margin-left:10px">変更すると他の端末はログアウトされます</span></div></form></div>
    </div>
    ${securityPanelHTML(sec)}`;
  if (admin) {
    html += `
    <div class="panel"><h3>音楽フォルダ</h3>
      ${libs.map((l) => `<div class="row"><span style="color:var(--fg2)">${I.folder}</span><div class="grow"><div class="ellipsis"><b>${esc(l.name)}</b></div><div class="small muted ellipsis">${esc(l.path)} · ${l.count}曲</div></div>
        <button class="btn small" data-scan="${l.id}">${I.refresh}スキャン</button><button class="icon-btn" data-dellib="${l.id}" aria-label="削除">${I.trash}</button></div>`).join('') || '<div class="row muted">まだフォルダがありません。下から追加してください。</div>'}
      <div class="row"><form id="libf" class="form-grid" style="width:100%">
        <div style="display:flex;gap:8px"><input class="field" name="path" placeholder="フォルダのパス（例: /music）" style="flex:1" required><button class="btn small" type="button" id="browse" style="height:40px">${I.folder}参照</button></div>
        <input class="field" name="name" placeholder="表示名（省略可）">
        <div><button class="btn primary small" type="submit">${I.plus}フォルダを追加</button></div></form></div>
      <div class="row"><div class="grow small muted" id="scanst">${scanText(scan)}</div><button class="btn small" id="scanall">${I.refresh}すべてスキャン</button></div>
    </div>
    <div class="panel"><h3>ユーザー</h3>
      ${users.map((u) => `<div class="row">${I.artist}<div class="grow">${esc(u.name)}${u.admin ? ' <span class="small muted">管理者</span>' : ''}${u.totp ? ' <span class="small" style="color:var(--accent)">2段階認証</span>' : ''}</div>
        ${u.totp && u.id !== state.me.id ? `<button class="btn small" data-reset2fa="${u.id}" data-name="${esc(u.name)}">2段階認証を解除</button>` : ''}
        <button class="btn small" data-resetpw="${u.id}" data-name="${esc(u.name)}">パスワード再設定</button>${u.id !== state.me.id ? `<button class="icon-btn" data-deluser="${u.id}" data-name="${esc(u.name)}" aria-label="削除">${I.trash}</button>` : ''}</div>`).join('')}
      <div class="row"><form id="uf" class="form-grid" style="width:100%">
        <input class="field" name="name" placeholder="ユーザー名" autocomplete="off" required>
        <input class="field" name="password" type="password" placeholder="パスワード" autocomplete="new-password" required>
        <label class="small"><input type="checkbox" name="admin"> 管理者にする</label>
        <div><button class="btn small" type="submit">${I.plus}ユーザーを追加</button></div></form></div>
    </div>
    <div class="panel"><h3>サーバー情報</h3><dl class="kv">
      <dt>バージョン</dt><dd>Hako ${esc(info.version)}</dd>
      <dt>ライブラリ</dt><dd>${info.artists} アーティスト · ${info.albums} アルバム · ${info.tracks} 曲</dd>
      <dt>音量解析</dt><dd>${info.loudness_done} / ${info.tracks} 曲${info.loudness_failed ? `（失敗 ${info.loudness_failed}）` : ''}</dd>
      <dt>ffmpeg</dt><dd>${esc(info.ffmpeg)}</dd>
      <dt>データ</dt><dd>${esc(info.data_dir)}</dd>
      <dt>自動スキャン</dt><dd>${info.scan_interval > 0 ? `${info.scan_interval}分ごと` : 'オフ'}</dd></dl></div>`;
  }
  $('#view').innerHTML = html;

  $('#qual').onchange = (e) => { state.prefs.quality = e.target.value; savePrefs(); toast('音質を変更しました（次の曲から反映）'); };
  $('#norm').onchange = (e) => { state.prefs.norm = e.target.value; savePrefs(); applyVolume(); $('#target').disabled = e.target.value === 'off'; toast('音量の均一化を変更しました'); };
  $('#target').onchange = (e) => { state.prefs.target = +e.target.value; savePrefs(); applyVolume(); };
  $('#logout').onclick = async () => {
    await api('/logout', { method: 'POST' }).catch(() => {});
    audio.pause(); state.me = null; go('#/login');
  };
  $('#pwf').onsubmit = async (e) => {
    e.preventDefault();
    const f = e.target;
    try { await api('/me/password', { method: 'POST', body: { current: f.current.value, new: f.new.value } }); toast('パスワードを変更しました（他の端末はログアウトしました）'); f.reset(); route(); } catch (ex) { toast(ex.message, true); }
  };
  bindSecurityPanel(sec);
  if (!admin) return;

  $$('[data-reset2fa]').forEach((b) => (b.onclick = () => {
    const ov = dialog(`<h3>「${esc(b.dataset.name)}」の2段階認証を解除しますか？</h3><p class="muted small">認証アプリを紛失した場合などに使います。このユーザーはすべての端末からログアウトされ、パスワードだけでログインできるようになります。</p><div class="btns"><button class="btn" data-cancel>キャンセル</button><button class="btn danger" id="ok">解除</button></div>`);
    $('[data-cancel]', ov).onclick = () => closeOverlay();
    $('#ok', ov).onclick = async () => { try { await api(`/users/${b.dataset.reset2fa}/totp/reset`, { method: 'POST' }); closeOverlay(); toast('2段階認証を解除しました'); route(); } catch (ex) { toast(ex.message, true); } };
  }));

  $('#browse').onclick = () => pickDir($('#libf').path.value.trim(), (p) => { $('#libf').path.value = p; });
  $('#libf').onsubmit = async (e) => {
    e.preventDefault();
    const f = e.target;
    try { await api('/libraries', { method: 'POST', body: { name: f.name.value.trim(), path: f.path.value.trim() } }); toast('フォルダを追加しました。スキャンを開始します'); route(); } catch (ex) { toast(ex.message, true); }
  };
  $$('[data-scan]').forEach((b) => (b.onclick = async () => { await api(`/libraries/${b.dataset.scan}/scan`, { method: 'POST' }); toast('スキャンを開始しました'); pollScan(); }));
  $('#scanall').onclick = async () => { await api('/scan', { method: 'POST' }); toast('スキャンを開始しました'); pollScan(); };
  $$('[data-dellib]').forEach((b) => (b.onclick = () => {
    const ov = dialog(`<h3>このフォルダをライブラリから外しますか？</h3><p class="muted small">ファイル自体は削除されません。再生回数やお気に入りの情報は失われます。</p><div class="btns"><button class="btn" data-cancel>キャンセル</button><button class="btn danger" id="ok">外す</button></div>`);
    $('[data-cancel]', ov).onclick = () => closeOverlay();
    $('#ok', ov).onclick = async () => { try { await api(`/libraries/${b.dataset.dellib}`, { method: 'DELETE' }); closeOverlay(); route(); } catch (ex) { toast(ex.message, true); } };
  }));
  $('#uf').onsubmit = async (e) => {
    e.preventDefault();
    const f = e.target;
    try { await api('/users', { method: 'POST', body: { name: f.name.value.trim(), password: f.password.value, admin: f.admin.checked } }); toast('ユーザーを追加しました'); route(); } catch (ex) { toast(ex.message, true); }
  };
  $$('[data-deluser]').forEach((b) => (b.onclick = () => {
    const ov = dialog(`<h3>ユーザー「${esc(b.dataset.name)}」を削除しますか？</h3><div class="btns"><button class="btn" data-cancel>キャンセル</button><button class="btn danger" id="ok">削除</button></div>`);
    $('[data-cancel]', ov).onclick = () => closeOverlay();
    $('#ok', ov).onclick = async () => { try { await api(`/users/${b.dataset.deluser}`, { method: 'DELETE' }); closeOverlay(); route(); } catch (ex) { toast(ex.message, true); } };
  }));
  $$('[data-resetpw]').forEach((b) => (b.onclick = () => {
    const ov = dialog(`<h3>「${esc(b.dataset.name)}」の新しいパスワード</h3><input class="field" type="password" id="np" autocomplete="new-password"><div class="btns"><button class="btn" data-cancel>キャンセル</button><button class="btn primary" id="ok">設定</button></div>`);
    $('[data-cancel]', ov).onclick = () => closeOverlay();
    $('#ok', ov).onclick = async () => { try { await api(`/users/${b.dataset.resetpw}/password`, { method: 'POST', body: { password: $('#np', ov).value } }); closeOverlay(); toast('パスワードを設定しました'); } catch (ex) { toast(ex.message, true); } };
  }));
  if (scan.running) pollScan();
}

/* ---------------- security settings ---------------- */

function deviceName(ua) {
  ua = ua || '';
  const os = /Android/.test(ua) ? 'Android' : /iPhone|iPad|iPod/.test(ua) ? 'iOS' : /Windows/.test(ua) ? 'Windows' : /Mac OS X|Macintosh/.test(ua) ? 'Mac' : /CrOS/.test(ua) ? 'ChromeOS' : /Linux/.test(ua) ? 'Linux' : '';
  const br = /Edg\//.test(ua) ? 'Edge' : /OPR\//.test(ua) ? 'Opera' : /Firefox\//.test(ua) ? 'Firefox' : /SamsungBrowser/.test(ua) ? 'Samsung Internet' : /Chrome\//.test(ua) ? 'Chrome' : /Safari\//.test(ua) ? 'Safari' : '';
  return [br, os].filter(Boolean).join(' / ') || (ua ? ua.slice(0, 40) : '不明な端末');
}

function fmtDate(sec) {
  return sec ? new Date(sec * 1000).toLocaleString('ja-JP', { year: 'numeric', month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : '—';
}

function securityPanelHTML(sec) {
  const others = sec.sessions.filter((s) => !s.current).length;
  return `
    <div class="panel"><h3>セキュリティ</h3>
      <div class="row"><div class="grow">2段階認証
          <div class="small muted">${sec.totp
            ? `有効です。ログイン時に認証アプリのコードが必要です。回復コード残り <b>${sec.recovery_left}</b> 個${sec.recovery_left <= 2 ? '<span style="color:var(--danger)">（少なくなっています）</span>' : ''}`
            : 'ログイン時にパスワードに加えて、Google Authenticator などの認証アプリのコードを求めます。パスワードが漏れても不正ログインを防げます'}</div></div>
        ${sec.totp
          ? '<button class="btn small" id="rc-regen">回復コードを再発行</button><button class="btn small danger" id="totp-off">無効にする</button>'
          : '<button class="btn primary small" id="totp-on">有効にする</button>'}</div>
      <div class="row" style="display:block"><div style="margin-bottom:6px">ログイン中の端末</div>
        ${sec.sessions.map((s) => `<div class="row" style="padding:8px 0">
          <div class="grow"><div>${esc(deviceName(s.user_agent))}${s.current ? ' <span class="small" style="color:var(--accent)">この端末</span>' : ''}</div>
          <div class="small muted">${esc(s.ip || '')} · 最終アクセス ${fmtDate(s.last_seen)} · ログイン ${fmtDate(s.created)}</div></div>
          ${s.current ? '' : `<button class="btn small" data-revoke="${s.id}">ログアウト</button>`}</div>`).join('')}
        ${others ? `<div style="margin-top:8px"><button class="btn small" id="revoke-others">他の端末をすべてログアウト（${others}）</button></div>` : ''}
      </div>
    </div>`;
}

function askPassword(title, desc, ok) {
  const ov = dialog(`<h3>${title}</h3>${desc ? `<p class="muted small">${desc}</p>` : ''}
    <form id="pwd-f"><input class="field" type="password" name="password" placeholder="パスワード" autocomplete="current-password" required>
    <div class="btns"><button class="btn" type="button" data-cancel>キャンセル</button><button class="btn primary" type="submit">次へ</button></div></form>`);
  $('[data-cancel]', ov).onclick = () => closeOverlay();
  const f = $('#pwd-f', ov);
  f.password.focus();
  f.onsubmit = async (e) => { e.preventDefault(); try { await ok(f.password.value, ov); } catch (ex) { toast(ex.message, true); } };
}

function showRecoveryCodes(codes, ov) {
  const text = `Hako 回復コード（${state.me.name}）\n各コードは1回だけ使えます。\n\n${codes.join('\n')}\n`;
  ov.innerHTML = `<div class="dialog"><h3>回復コード</h3>
    <p class="muted small">スマホを紛失して認証アプリが使えなくなったときに、6桁のコードの代わりに使えます。<b>この画面は二度と表示されない</b>ので、安全な場所に保存してください（各コード1回限り）。</p>
    <pre style="background:var(--bg3);border-radius:10px;padding:14px;font-size:16px;line-height:1.8;columns:2;text-align:center;user-select:all;margin:0 0 12px">${codes.map(esc).join('\n')}</pre>
    <div class="btns" style="flex-wrap:wrap"><button class="btn small" id="rc-copy">コピー</button><button class="btn small" id="rc-dl">ファイルに保存</button><button class="btn primary small" id="rc-done">保存しました</button></div></div>`;
  $('#rc-copy', ov).onclick = async () => { try { await navigator.clipboard.writeText(text); toast('コピーしました'); } catch { toast('コピーできませんでした。手で書き写してください', true); } };
  $('#rc-dl', ov).onclick = () => {
    const a = document.createElement('a');
    a.href = URL.createObjectURL(new Blob([text], { type: 'text/plain' }));
    a.download = `hako-recovery-codes-${state.me.name}.txt`;
    document.body.appendChild(a); a.click(); a.remove();
    setTimeout(() => URL.revokeObjectURL(a.href), 1000);
  };
  $('#rc-done', ov).onclick = () => { closeOverlay(); route(); };
}

function bindSecurityPanel(sec) {
  const on = $('#totp-on');
  if (on) on.onclick = () => askPassword('2段階認証を有効にする', '確認のため、現在のパスワードを入力してください。', async (password, ov) => {
    const s = await api('/me/totp/setup', { method: 'POST', body: { password } });
    $('.dialog', ov).innerHTML = `<h3>認証アプリに登録</h3>
      <p class="muted small">① Google Authenticator / Microsoft Authenticator / 1Password などの認証アプリで、このQRコードを読み取ってください。</p>
      <img src="${s.qr}" alt="QRコード" style="width:min(240px,70vw);display:block;margin:0 auto 10px;border-radius:10px;background:#fff;padding:8px">
      <details style="margin-bottom:12px"><summary class="small muted" style="cursor:pointer">QRコードを読み取れない場合（このスマホで登録する場合）</summary>
        <p class="small muted">認証アプリで「セットアップキーを入力」を選び、次のキーを入力してください（時間ベース）。</p>
        <code style="display:block;word-break:break-all;background:var(--bg3);padding:10px;border-radius:8px;user-select:all">${esc(s.secret.replace(/(.{4})/g, '$1 ').trim())}</code>
        <a class="btn small" style="margin-top:8px" href="${esc(s.uri)}">認証アプリで開く</a></details>
      <p class="muted small">② アプリに表示された6桁のコードを入力してください。</p>
      <form id="totp-f"><input class="field" name="code" inputmode="numeric" maxlength="7" autocomplete="one-time-code" placeholder="123456" required style="text-align:center;font-size:22px;letter-spacing:.3em">
      <div class="btns"><button class="btn" type="button" data-cancel>キャンセル</button><button class="btn primary" type="submit">有効にする</button></div></form>`;
    $('[data-cancel]', ov).onclick = () => closeOverlay();
    const f = $('#totp-f', ov);
    f.code.focus();
    f.onsubmit = async (e) => {
      e.preventDefault();
      try {
        const r = await api('/me/totp/enable', { method: 'POST', body: { code: f.code.value } });
        state.me.totp = true;
        toast('2段階認証を有効にしました（他の端末はログアウトしました）');
        showRecoveryCodes(r.recovery_codes, ov);
      } catch (ex) { toast(ex.message, true); f.code.value = ''; f.code.focus(); }
    };
  });

  const off = $('#totp-off');
  if (off) off.onclick = () => {
    const ov = dialog(`<h3>2段階認証を無効にする</h3><p class="muted small">セキュリティが下がります。パスワードと認証アプリのコード（または回復コード）を入力してください。</p>
      <form id="off-f"><input class="field" type="password" name="password" placeholder="パスワード" autocomplete="current-password" required>
      <input class="field" name="code" placeholder="6桁のコード または 回復コード" autocomplete="one-time-code" required>
      <div class="btns"><button class="btn" type="button" data-cancel>キャンセル</button><button class="btn danger" type="submit">無効にする</button></div></form>`);
    $('[data-cancel]', ov).onclick = () => closeOverlay();
    const f = $('#off-f', ov);
    f.password.focus();
    f.onsubmit = async (e) => {
      e.preventDefault();
      try { await api('/me/totp/disable', { method: 'POST', body: { password: f.password.value, code: f.code.value } }); state.me.totp = false; closeOverlay(); toast('2段階認証を無効にしました'); route(); } catch (ex) { toast(ex.message, true); }
    };
  };

  const regen = $('#rc-regen');
  if (regen) regen.onclick = () => askPassword('回復コードを再発行', '今までの回復コードはすべて使えなくなります。', async (password, ov) => {
    const r = await api('/me/recovery', { method: 'POST', body: { password } });
    showRecoveryCodes(r.recovery_codes, ov);
  });

  $$('[data-revoke]').forEach((b) => (b.onclick = async () => {
    try { await api(`/me/sessions/${b.dataset.revoke}`, { method: 'DELETE' }); toast('その端末をログアウトしました'); route(); } catch (ex) { toast(ex.message, true); }
  }));
  const ro = $('#revoke-others');
  if (ro) ro.onclick = async () => {
    try { await api('/me/sessions/revoke-others', { method: 'POST' }); toast('他の端末をすべてログアウトしました'); route(); } catch (ex) { toast(ex.message, true); }
  };
}

function scanText(s) {
  if (!s) return '';
  if (s.running) return `スキャン中: ${esc(s.library || '')} — ${esc(s.phase || '')} ${s.total ? `${s.done}/${s.total}` : ''}`;
  const t = s.last_finished ? new Date(s.last_finished * 1000).toLocaleString('ja-JP') : '未実行';
  return `最終スキャン: ${t}${s.last_error ? `<br><span style="color:var(--danger)">${esc(s.last_error)}</span>` : ''}`;
}

let scanTimer = null;
function pollScan() {
  clearInterval(scanTimer);
  scanTimer = setInterval(async () => {
    const el = $('#scanst');
    if (!el) { clearInterval(scanTimer); return; }
    const s = await api('/scan').catch(() => null);
    if (!s) return;
    el.innerHTML = scanText(s);
    if (!s.running) { clearInterval(scanTimer); setTimeout(() => { if ($('#scanst')) route(); }, 400); }
  }, 1500);
}

async function pickDir(start, onPick) {
  let curPath = start || '';
  const ov = dialog(`<h3>フォルダを選択</h3><div class="dir-path" id="dp"></div><div class="dir-list" id="dl"></div>
    <div class="btns"><button class="btn" data-cancel>キャンセル</button><button class="btn primary" id="dok">このフォルダを選択</button></div>`);
  const load = async (p) => {
    try {
      const r = await api('/fs?path=' + encodeURIComponent(p));
      curPath = r.path;
      $('#dp', ov).textContent = r.path;
      $('#dl', ov).innerHTML = (r.parent ? `<button data-p="${esc(r.parent)}">${I.up}<span>上のフォルダへ</span></button>` : '')
        + r.dirs.map((d) => `<button data-p="${esc(r.path.replace(/[\\/]$/, '') + (r.path.includes('\\') ? '\\' : '/') + d)}">${I.folder}<span class="ellipsis">${esc(d)}</span></button>`).join('')
        + (r.dirs.length ? '' : '<div class="empty" style="padding:16px">サブフォルダはありません</div>');
      $$('[data-p]', ov).forEach((b) => (b.onclick = () => load(b.dataset.p)));
    } catch (e) { toast(e.message, true); if (p) load(''); }
  };
  $('[data-cancel]', ov).onclick = () => closeOverlay();
  $('#dok', ov).onclick = () => { onPick(curPath); closeOverlay(); };
  load(curPath);
}

/* ============================================================
   Global event delegation
   ============================================================ */

// broken cover images (no inline handlers: the CSP forbids them)
document.addEventListener('error', (e) => {
  const img = e.target;
  if (!(img instanceof HTMLImageElement)) return;
  if (img.hasAttribute('data-rm-err')) img.remove();
  else if (img.hasAttribute('data-hide-err')) img.style.visibility = 'hidden';
}, true);

document.addEventListener('click', (e) => {
  if (e.target.closest('[data-reload]')) { location.reload(); return; }
  const el = e.target.closest('button, a, .trow, [data-open]');
  if (!el) return;
  const d = el.dataset;

  if ('pp' in d) return togglePlay();
  if ('next' in d && el.tagName === 'BUTTON') return next();
  if ('prev' in d) return prev();
  if ('shuffle' in d) return toggleShuffle();
  if ('repeat' in d) return cycleRepeat();
  if ('mute' in d) { audio.muted = !audio.muted; el.innerHTML = audio.muted ? I.mute : I.volume; return; }
  if ('open' in d) return openSheet('np');
  if (d.openTab) { if (sheetOpen && sheetTab === d.openTab) return popLayer(); return openSheet(d.openTab); }
  if ('closeSheet' in d) return popLayer();
  if (d.stab) { sheetTab = d.stab; return renderSheet(); }
  if ('back' in d) { if (history.length > 1) history.back(); else go('#/'); return; }
  if ('nav' in d && el.tagName === 'A' && sheetOpen) { e.preventDefault(); return popLayer(el.getAttribute('href')); }
  if (d.fav) { e.preventDefault(); e.stopPropagation(); return toggleFav(d.fav, +d.id, el); }
  if (d.playAlbum) { e.preventDefault(); e.stopPropagation(); return playAlbum(+d.playAlbum); }
  if ('shuffleAll' in d) return shuffleAll();
  if (d.moreTrack) { const t = cur(); if (t) trackMenu(t); return; }
  if (d.qrm !== undefined && el.tagName === 'BUTTON') { e.stopPropagation(); return removeFromQueue(+d.qrm); }
  if ('qclear' in d) { const c = cur(); P.queue = c ? [c] : []; P.orig = P.queue.slice(); P.idx = c ? 0 : -1; renderSheet(); saveQueue(); return; }
  if ('qsave' in d) return addToPlaylistDialog(P.queue.map((t) => t.id));
  if (d.more !== undefined && el.tagName === 'BUTTON') {
    e.stopPropagation();
    const row = el.closest('.trow');
    const list = state.lists.get(row.dataset.list) || [];
    const t = list[+d.more];
    if (t) trackMenu(t);
    return;
  }
  if (el.classList.contains('trow')) {
    if (el.closest('.qlist')) { const i = +el.dataset.qi; if (i === P.idx) togglePlay(); else loadTrack(i); renderQueuePanel(); return; }
    if (e.target.closest('button')) return;
    const list = state.lists.get(el.dataset.list);
    if (!list) return;
    const i = +el.dataset.i;
    const t = cur();
    if (t && list[i] && t.id === list[i].id) { togglePlay(); return; }
    playList(list, i);
  }
});

$('#sheet').addEventListener('click', (e) => {
  const p = e.target.closest('#lyr p[data-li]');
  if (p && lyr && lyr.synced) { seek(lyr.lines[+p.dataset.li].t); lyrUserScroll = 0; }
});

document.addEventListener('keydown', (e) => {
  if (e.target.closest('input, textarea, select')) return;
  if (e.code === 'Space') { e.preventDefault(); togglePlay(); }
  else if (e.key === 'ArrowRight' && e.shiftKey) next();
  else if (e.key === 'ArrowLeft' && e.shiftKey) prev();
  else if (e.key === 'ArrowRight') seek(position() + 10);
  else if (e.key === 'ArrowLeft') seek(position() - 10);
});

window.addEventListener('pagehide', saveQueue);
document.addEventListener('visibilitychange', () => { if (document.visibilityState === 'hidden') saveQueue(); });

if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => navigator.serviceWorker.register('/sw.js').catch(() => {}));
}

route();
