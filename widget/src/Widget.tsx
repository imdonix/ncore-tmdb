import { useEffect, useState } from "react"
import { Bell, BellOff, Check, Download, ExternalLink, Loader2 } from "lucide-react"

type SearchParams = {
  pattern?: string
  type?: string
  where?: string
  sort_by?: string
  sort_order?: string
  page?: number
}

type Follow = {
  id: number
  tmdbId: number
  name: string
  quality: string
  status: string
  skippedSeasons?: number[]
  lastCheckAt?: string
  completed?: number
  found?: number
  wanted?: number
}

type FollowItem = {
  id: number
  season: number
  episode: number
  status: string
  torrentTitle?: string
  ncoreTorrentId?: string
}

type SeasonInfo = {
  season_number: number
  episode_count: number
  name?: string
}

type MovieDetails = {
  tmdbId: number
  name: string
  year: string
  releaseDate: string
  posterPath?: string
  searchPattern: string
}

type MovieResolution = {
  quality: string
  releases: number
  seeders: number
}

type MovieDownload = {
  ncoreTorrentId: string
  torrentTitle: string
  quality: string
  qbitHash?: string
  status: string
  progress?: number
  state?: string
  size?: number
  downloaded?: number
  dlspeed?: number
  eta?: number
  ratio?: number
}

type MovieOverview = {
  movie: MovieDetails
  resolutions: MovieResolution[]
  download?: MovieDownload | null
  search?: SearchParams
  searchError?: string
}

function buildSearchResultUrl(search?: SearchParams): string {
  const params = new URLSearchParams()
  const pattern = (search?.pattern ?? "").trim()
  if (pattern) params.set("q", pattern)
  params.set("type", search?.type || "all_own")
  params.set("where", search?.where || "name")
  params.set("sort", search?.sort_by || "seeders")
  params.set("order", (search?.sort_order || "DESC").toUpperCase())
  params.set("page", String(search?.page && search.page > 0 ? search.page : 1))
  return `/ncore?${params.toString()}`
}

declare global {
  interface Window {
    __NCORE_WIDGET__?: {
      contentType: string
      tmdbID: string
    }
  }
}

function formatDate(value?: string) {
  if (!value) return "—"
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" })
}

function formatBytes(n?: number) {
  if (!n || n < 0) return "0 B"
  const units = ["B", "KiB", "MiB", "GiB", "TiB"]
  let v = n
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

function formatEta(seconds?: number) {
  if (seconds == null || seconds < 0 || seconds >= 8640000) return "—"
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  if (seconds < 86400) {
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    return `${h}h ${m}m`
  }
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  return `${d}d ${h}h`
}

function qbitStateLabel(state?: string) {
  const map: Record<string, string> = {
    downloading: "Downloading",
    metaDL: "Fetching metadata",
    forcedDL: "Downloading",
    allocating: "Allocating",
    stalledDL: "Stalled",
    checkingDL: "Checking",
    checkingUP: "Checking",
    checkingResumeData: "Checking",
    moving: "Moving",
    uploading: "Seeding",
    forcedUP: "Seeding",
    stalledUP: "Seeding (stalled)",
    queuedDL: "Queued",
    queuedUP: "Queued seed",
    pausedDL: "Paused",
    pausedUP: "Paused seed",
    stoppedDL: "Stopped",
    stoppedUP: "Stopped seed",
    error: "Error",
    missingFiles: "Missing files",
  }
  return (state && map[state]) || state || ""
}

function pickDefaultQuality(resolutions: MovieResolution[]) {
  if (resolutions.some((r) => r.quality === "1080p")) return "1080p"
  return resolutions[0]?.quality ?? ""
}

export function Widget() {
  const cfg = window.__NCORE_WIDGET__
  if (cfg?.contentType === "tv") {
    return <FollowPanel tmdbId={cfg.tmdbID} />
  }
  return <MovieDownloadPanel tmdbId={cfg?.tmdbID ?? ""} />
}

function MovieDownloadPanel({ tmdbId }: { tmdbId: string }) {
  const [movie, setMovie] = useState<MovieDetails | null>(null)
  const [resolutions, setResolutions] = useState<MovieResolution[]>([])
  const [download, setDownload] = useState<MovieDownload | null>(null)
  const [quality, setQuality] = useState("")
  const [searchHref, setSearchHref] = useState("/ncore")
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  useEffect(() => {
    if (!tmdbId) {
      setError("Missing widget config")
      setLoading(false)
      return
    }

    let cancelled = false

    ;(async () => {
      try {
        const res = await fetch(`/api/movies/${encodeURIComponent(tmdbId)}`)
        const data: MovieOverview & { error?: string } = await res.json()
        if (!res.ok) throw new Error(data.error || "Failed to load movie")
        if (cancelled) return

        const available = data.resolutions ?? []
        setMovie(data.movie ?? null)
        setResolutions(available)
        setDownload(data.download ?? null)
        setSearchHref(buildSearchResultUrl(data.search))

        if (data.download?.quality) {
          setQuality(data.download.quality)
        } else if (available.length > 0) {
          setQuality(pickDefaultQuality(available))
        }
        setError(data.searchError && !data.download ? data.searchError : null)
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : "Error")
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()

    return () => {
      cancelled = true
    }
  }, [tmdbId])

  const downloading = download?.status === "downloading"
  const rawPct = Math.min(100, Math.max(0, download?.progress ?? 0))
  const done = download?.status === "completed" || rawPct >= 99.9
  const pct = done ? 100 : rawPct
  const locked = downloading || done

  // Poll only while a download is active — no nCore searches in the loop.
  useEffect(() => {
    if (!downloading || !tmdbId) return
    const timer = window.setInterval(() => {
      void fetch(`/api/movies/${encodeURIComponent(tmdbId)}/download`)
        .then(async (res) => {
          if (!res.ok) return
          const data = await res.json()
          setDownload(data.download ?? null)
        })
        .catch(() => {
          // ignore transient qBittorrent errors
        })
    }, 3000)
    return () => window.clearInterval(timer)
  }, [downloading, tmdbId])

  const onDownload = async () => {
    if (!tmdbId || !quality) return
    setBusy(true)
    setError(null)
    setMessage(null)
    try {
      const res = await fetch(`/api/movies/${encodeURIComponent(tmdbId)}/download`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ quality }),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || "Download failed")
      setDownload(data.download ?? null)
      setMessage(
        data.download?.status === "completed"
          ? "Already downloaded."
          : "Added to qBittorrent — it will keep going in the background."
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : "Download failed")
    } finally {
      setBusy(false)
    }
  }

  const selected = resolutions.find((r) => r.quality === quality)
  const statusText = done
    ? "ready to watch"
    : download?.status === "missing"
      ? "removed from qBittorrent"
      : downloading
        ? qbitStateLabel(download?.state) || "downloading"
        : download?.status

  return (
    <section className="nw-panel">
      <div className="nw-header">
        <span className="nw-title">Download movie</span>
        <a className="nw-browse" href={searchHref} title="Open the same nCore search">
          Open search result
          <ExternalLink size={12} aria-hidden />
        </a>
      </div>

      <div className="nw-body nw-follow-body">
        {loading && (
          <div className="nw-state">
            <Loader2 size={16} className="nw-spin" />
            Checking nCore…
          </div>
        )}

        {error && <div className="nw-state nw-error">{error}</div>}

        {!loading && (
          <div className="nw-follow-setup">
            {movie && (
              <p className="nw-follow-desc">
                {movie.name}
                {movie.year ? ` (${movie.year})` : ""} — pick a resolution and let qBittorrent
                handle the rest.
              </p>
            )}

            {resolutions.length > 0 && (
              <div className="nw-quality">
                <span className="nw-quality-label">Resolution</span>
                <div className="nw-quality-btns">
                  {resolutions.map((r) => (
                    <button
                      key={r.quality}
                      type="button"
                      className={`nw-qbtn ${quality === r.quality ? "active" : ""}`}
                      onClick={() => setQuality(r.quality)}
                      disabled={busy || locked}
                      title={`${r.releases} release${r.releases === 1 ? "" : "s"} · best ${r.seeders} seeders`}
                    >
                      {r.quality}
                    </button>
                  ))}
                </div>
                {selected && (
                  <p className="nw-skip-hint">
                    {selected.releases} release{selected.releases === 1 ? "" : "s"} · best{" "}
                    {selected.seeders} seeders
                  </p>
                )}
              </div>
            )}

            {resolutions.length === 0 && !download && !error && (
              <div className="nw-state">No releases found on nCore yet.</div>
            )}

            {download && (
              <div className="nw-download">
                <div className="nw-follow-stats">
                  <span className="nw-pill">{download.quality}</span>
                  <span className="nw-muted">{statusText}</span>
                  {!!download.size && (
                    <span className="nw-muted">
                      {formatBytes(download.downloaded || (pct / 100) * download.size)} /{" "}
                      {formatBytes(download.size)}
                    </span>
                  )}
                </div>

                <div className="nw-progress">
                  <div
                    className={`nw-progress-fill ${done ? "done" : ""}`}
                    style={{ width: `${pct}%` }}
                  />
                </div>

                {downloading && (
                  <div className="nw-download-meta">
                    <span>↓ {formatBytes(download.dlspeed)}/s</span>
                    <span>ETA {formatEta(download.eta)}</span>
                    <span>{pct.toFixed(1)}%</span>
                  </div>
                )}

                <div className="nw-download-title" title={download.torrentTitle}>
                  {download.torrentTitle}
                </div>

                <div className="nw-follow-actions">
                  <a
                    className="nw-secondary-btn"
                    href={`/ncore/torrent/${download.ncoreTorrentId}`}
                  >
                    Open torrent
                  </a>
                  <a className="nw-secondary-btn" href="/ncore/downloads">
                    All downloads
                  </a>
                </div>
              </div>
            )}

            {message && <div className="nw-state nw-ok">{message}</div>}

            {(resolutions.length > 0 || download) && (
              <button
                type="button"
                className="nw-primary-btn"
                disabled={busy || locked || !quality}
                onClick={onDownload}
              >
                {busy ? (
                  <Loader2 size={14} className="nw-spin" />
                ) : done ? (
                  <Check size={14} />
                ) : (
                  <Download size={14} />
                )}
                {busy
                  ? "Starting…"
                  : done
                    ? "Downloaded"
                    : downloading
                      ? `Downloading ${pct.toFixed(0)}%`
                      : download?.status === "missing"
                        ? "Download again"
                        : "Download movie now"}
              </button>
            )}
          </div>
        )}
      </div>
      <style>{widgetCss}</style>
    </section>
  )
}

function FollowPanel({ tmdbId }: { tmdbId: string }) {
  const [follow, setFollow] = useState<Follow | null>(null)
  const [items, setItems] = useState<FollowItem[]>([])
  const [seasons, setSeasons] = useState<SeasonInfo[]>([])
  const [quality, setQuality] = useState<"720p" | "1080p">("1080p")
  const [skipped, setSkipped] = useState<number[]>([])
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const load = async () => {
    try {
      const res = await fetch(`/api/follows/${tmdbId}`)
      if (!res.ok) throw new Error("Failed to load follow status")
      const data = await res.json()
      setFollow(data.follow ?? null)
      setItems(data.items ?? [])
      setSeasons(data.seasons ?? [])
      if (data.follow?.quality === "720p" || data.follow?.quality === "1080p") {
        setQuality(data.follow.quality)
      }
      if (Array.isArray(data.follow?.skippedSeasons)) {
        setSkipped(data.follow.skippedSeasons)
      }
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Error")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [tmdbId])

  const toggleSkip = (season: number) => {
    setSkipped((prev) =>
      prev.includes(season) ? prev.filter((s) => s !== season) : [...prev, season].sort((a, b) => a - b)
    )
  }

  const onFollow = async () => {
    setBusy(true)
    setMessage(null)
    try {
      const res = await fetch("/api/follows", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          tmdbId: Number(tmdbId),
          quality,
          skippedSeasons: skipped,
        }),
      })
      if (!res.ok) {
        const e = await res.json().catch(() => ({}))
        throw new Error(e.error || "Follow failed")
      }
      setMessage("Following — searching nCore in the background…")
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Follow failed")
    } finally {
      setBusy(false)
    }
  }

  const onSaveSkips = async () => {
    setBusy(true)
    setMessage(null)
    try {
      const res = await fetch(`/api/follows/${tmdbId}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ skippedSeasons: skipped }),
      })
      if (!res.ok) {
        const e = await res.json().catch(() => ({}))
        throw new Error(e.error || "Update failed")
      }
      setMessage("Skipped seasons updated")
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Update failed")
    } finally {
      setBusy(false)
    }
  }

  const onUnfollow = async () => {
    if (!window.confirm("Stop following this series?")) return
    setBusy(true)
    try {
      const res = await fetch(`/api/follows/${tmdbId}`, { method: "DELETE" })
      if (!res.ok) throw new Error("Unfollow failed")
      setFollow(null)
      setItems([])
      setMessage(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unfollow failed")
    } finally {
      setBusy(false)
    }
  }

  const seasonButtons =
    seasons.length > 0
      ? seasons
      : // fallback from items if TMDB seasons missing
        Array.from(new Set(items.map((i) => i.season).filter((s) => s > 0)))
          .sort((a, b) => a - b)
          .map((n) => ({ season_number: n, episode_count: 0, name: `Season ${n}` }))

  return (
    <section className="nw-panel">
      <div className="nw-header">
        <span className="nw-title">Follow series</span>
        <a className="nw-browse" href="/ncore/follows" title="Open follows dashboard">
          Open follows
          <ExternalLink size={12} aria-hidden />
        </a>
      </div>

      <div className="nw-body nw-follow-body">
        {loading && (
          <div className="nw-state">
            <Loader2 size={16} className="nw-spin" />
            Loading…
          </div>
        )}

        {error && <div className="nw-state nw-error">{error}</div>}
        {message && <div className="nw-state nw-ok">{message}</div>}

        {!loading && (
          <div className="nw-follow-setup">
            {!follow && (
              <p className="nw-follow-desc">
                Auto-download episodes from nCore. Skip seasons you already have — they are
                treated as already downloaded.
              </p>
            )}

            <div className="nw-quality">
              <span className="nw-quality-label">Quality</span>
              <div className="nw-quality-btns">
                {(["720p", "1080p"] as const).map((q) => (
                  <button
                    key={q}
                    type="button"
                    className={`nw-qbtn ${quality === q ? "active" : ""}`}
                    onClick={() => setQuality(q)}
                    disabled={!!follow}
                  >
                    {q}
                  </button>
                ))}
              </div>
            </div>

            {seasonButtons.length > 0 && (
              <div className="nw-skip">
                <span className="nw-quality-label">
                  Skip seasons (already have)
                </span>
                <div className="nw-skip-btns">
                  {seasonButtons.map((s) => {
                    const n = s.season_number
                    const on = skipped.includes(n)
                    return (
                      <button
                        key={n}
                        type="button"
                        className={`nw-skip-btn ${on ? "active" : ""}`}
                        onClick={() => toggleSkip(n)}
                        title={
                          on
                            ? `S${String(n).padStart(2, "0")} skipped (treated as owned)`
                            : `Skip S${String(n).padStart(2, "0")}`
                        }
                      >
                        S{String(n).padStart(2, "0")}
                        {s.episode_count > 0 ? (
                          <span className="nw-skip-ep">{s.episode_count}ep</span>
                        ) : null}
                      </button>
                    )
                  })}
                </div>
                <p className="nw-skip-hint">
                  Highlighted seasons are ignored by the scanner (as if already downloaded).
                </p>
              </div>
            )}

            {!follow ? (
              <button type="button" className="nw-primary-btn" disabled={busy} onClick={onFollow}>
                {busy ? <Loader2 size={14} className="nw-spin" /> : <Bell size={14} />}
                Follow series
              </button>
            ) : (
              <div className="nw-follow-active">
                <div className="nw-follow-stats">
                  <span className="nw-pill">
                    {follow.quality} · {follow.status}
                  </span>
                  {skipped.length > 0 && (
                    <span className="nw-muted">
                      skip {skipped.map((s) => `S${String(s).padStart(2, "0")}`).join(", ")}
                    </span>
                  )}
                </div>
                {follow.lastCheckAt && (
                  <div className="nw-muted nw-small">
                    Last check: {formatDate(follow.lastCheckAt)}
                  </div>
                )}
                {items.length > 0 && (
                  <ul className="nw-list nw-follow-list">
                    {items
                      .filter(
                        (it) =>
                          it.status === "skipped" ||
                          it.status === "cannot_find" ||
                          it.status === "downloading" ||
                          it.status === "found" ||
                          it.status === "completed" ||
                          !!it.ncoreTorrentId
                      )
                      .map((it) => (
                        <li key={it.id} className="nw-follow-item">
                          <span className="nw-ep-label">
                            {it.episode === 0
                              ? `S${String(it.season).padStart(2, "0")} (season pack)`
                              : `S${String(it.season).padStart(2, "0")}E${String(it.episode).padStart(2, "0")}`}
                          </span>
                          <span className="nw-ep-status">
                            {it.status === "cannot_find"
                              ? "cannot find"
                              : it.status === "skipped"
                                ? "skipped"
                                : it.status}
                          </span>
                          {it.ncoreTorrentId ? (
                            <a className="nw-ep-link" href={`/ncore/torrent/${it.ncoreTorrentId}`}>
                              open torrent
                            </a>
                          ) : null}
                        </li>
                      ))}
                  </ul>
                )}
                <div className="nw-follow-actions">
                  <button type="button" className="nw-secondary-btn" disabled={busy} onClick={onSaveSkips}>
                    Save skip list
                  </button>
                  <a className="nw-secondary-btn" href={`/ncore/follows?id=${tmdbId}`}>
                    Manage
                  </a>
                  <button type="button" className="nw-danger-btn" disabled={busy} onClick={onUnfollow}>
                    {busy ? <Loader2 size={14} className="nw-spin" /> : <BellOff size={14} />}
                    Unfollow
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
      <style>{widgetCss}</style>
    </section>
  )
}

const widgetCss = `
.nw-panel {
  margin: 0 0 16px;
  border: 1px solid var(--nw-border);
  border-radius: 12px;
  background: linear-gradient(180deg, rgba(30,41,59,.75), rgba(15,23,42,.7));
  box-shadow: 0 8px 24px rgba(0,0,0,.18);
  overflow: hidden;
  --nw-text: #e2e8f0;
  --nw-muted: #94a3b8;
  --nw-primary: #34d399;
  --nw-seed: #34d399;
  --nw-leech: #f87171;
  --nw-border: rgba(148, 163, 184, 0.18);
  color: var(--nw-text);
  font-family: Inter, ui-sans-serif, system-ui, sans-serif;
}
.nw-header {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--nw-border);
  min-height: 44px;
  box-sizing: border-box;
}
.nw-title {
  display: block;
  margin: 0 !important;
  padding: 0 !important;
  border: 0 !important;
  float: none !important;
  line-height: 1.25 !important;
  font-size: 14px !important;
  font-weight: 650 !important;
  letter-spacing: .01em;
  color: var(--nw-text) !important;
  text-align: left !important;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
  flex: 1 1 auto;
}
.nw-browse {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  flex: 0 0 auto;
  margin: 0;
  padding: 0;
  line-height: 1;
  font-size: 11px;
  font-weight: 600;
  color: var(--nw-primary);
  opacity: .9;
  white-space: nowrap;
  text-decoration: none;
}
.nw-browse:hover { opacity: 1; text-decoration: underline; }
.nw-browse svg { display: block; flex-shrink: 0; }
.nw-body { padding: 8px; }
.nw-follow-body { padding: 12px 14px 14px; }
.nw-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 16px 12px;
  color: var(--nw-muted);
  font-size: 12px;
}
.nw-error { color: #fca5a5; }
.nw-ok { color: #6ee7b7; }
.nw-spin { animation: nw-spin 1s linear infinite; }
@keyframes nw-spin { to { transform: rotate(360deg); } }
.nw-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.nw-follow-desc { margin: 0 0 12px; font-size: 12px; line-height: 1.45; color: var(--nw-muted); }
.nw-quality { margin-bottom: 12px; }
.nw-quality-label { display: block; font-size: 11px; color: var(--nw-muted); margin-bottom: 6px; }
.nw-quality-btns { display: flex; gap: 8px; flex-wrap: wrap; }
.nw-qbtn {
  border: 1px solid var(--nw-border);
  background: rgba(255,255,255,.04);
  color: var(--nw-text);
  border-radius: 8px;
  padding: 8px 14px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}
.nw-qbtn:disabled { opacity: .55; cursor: default; }
.nw-qbtn.active { border-color: var(--nw-primary); color: var(--nw-primary); background: rgba(52,211,153,.12); }
.nw-skip { margin-bottom: 12px; }
.nw-skip-btns { display: flex; flex-wrap: wrap; gap: 6px; }
.nw-skip-btn {
  border: 1px solid var(--nw-border);
  background: rgba(255,255,255,.04);
  color: var(--nw-text);
  border-radius: 8px;
  padding: 7px 10px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  min-width: 3.2rem;
}
.nw-skip-btn.active {
  border-color: #fbbf24;
  color: #fcd34d;
  background: rgba(251, 191, 36, .12);
  text-decoration: line-through;
}
.nw-skip-ep { font-size: 9px; font-weight: 500; opacity: .75; text-decoration: none !important; }
.nw-skip-hint { margin: 6px 0 0; font-size: 10px; color: var(--nw-muted); line-height: 1.35; }
.nw-primary-btn, .nw-secondary-btn, .nw-danger-btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  border-radius: 8px; padding: 10px 14px; font-size: 12px; font-weight: 600; cursor: pointer;
  border: 0; text-decoration: none;
}
.nw-primary-btn { width: 100%; background: #34d399; color: #032541; }
.nw-primary-btn:disabled { opacity: .6; cursor: wait; }
.nw-secondary-btn { background: rgba(255,255,255,.08); color: var(--nw-text); border: 1px solid var(--nw-border); }
.nw-danger-btn { background: transparent; color: #fca5a5; border: 1px solid rgba(248,113,113,.35); }
.nw-follow-actions { display: flex; gap: 8px; margin-top: 12px; flex-wrap: wrap; }
.nw-download {
  margin: 4px 0 12px;
  padding: 10px;
  border: 1px solid var(--nw-border);
  border-radius: 10px;
  background: rgba(255,255,255,.03);
}
.nw-download .nw-follow-actions { margin-top: 0; }
.nw-progress {
  height: 8px;
  margin: 4px 0 8px;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(148, 163, 184, .22);
}
.nw-progress-fill {
  height: 100%;
  border-radius: 999px;
  background: var(--nw-primary);
  transition: width .25s ease;
}
.nw-progress-fill.done { background: #34d399; }
.nw-download-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 8px;
  font-size: 11px;
  color: var(--nw-muted);
}
.nw-download-title {
  margin-bottom: 8px;
  font-size: 11px;
  line-height: 1.35;
  color: var(--nw-muted);
  word-break: break-word;
}
.nw-follow-stats { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-bottom: 6px; }
.nw-pill {
  font-size: 11px; font-weight: 600; padding: 3px 8px; border-radius: 999px;
  background: rgba(52,211,153,.15); color: #6ee7b7;
}
.nw-muted { color: var(--nw-muted); font-size: 12px; }
.nw-small { font-size: 11px; margin-bottom: 8px; }
.nw-follow-list { margin: 8px 0; gap: 2px; }
.nw-follow-item {
  display: flex; align-items: center; gap: 8px; padding: 6px 4px;
  font-size: 12px; border-bottom: 1px solid rgba(255,255,255,.04);
}
.nw-ep-label { font-weight: 600; min-width: 7rem; }
.nw-ep-status { color: var(--nw-muted); text-transform: capitalize; flex: 1; }
.nw-ep-link { color: var(--nw-primary); text-decoration: none; font-size: 11px; }
`
