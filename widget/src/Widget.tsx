import { useEffect, useState } from "react"
import { ArrowDown, ArrowUp, Calendar, ExternalLink, Loader2, Bell, BellOff } from "lucide-react"

type Torrent = {
  ID: string
  Title: string
  Type?: string
  Date?: string
  Seeders?: number
  Leechers?: number
  Provider?: string
}

type SearchParams = {
  pattern?: string
  type?: string
  where?: string
  sort_by?: string
  sort_order?: string
  page?: number
}

type FetchResponse = {
  torrents?: Torrent[]
  search?: SearchParams
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

export function Widget() {
  const cfg = window.__NCORE_WIDGET__
  if (cfg?.contentType === "tv") {
    return <FollowPanel tmdbId={cfg.tmdbID} />
  }
  return <MovieTorrentPanel />
}

function MovieTorrentPanel() {
  const cfg = window.__NCORE_WIDGET__
  const [torrents, setTorrents] = useState<Torrent[]>([])
  const [searchHref, setSearchHref] = useState("/ncore")
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!cfg?.contentType || !cfg?.tmdbID) {
      setError("Missing widget config")
      setLoading(false)
      return
    }

    const url = `/api/${cfg.contentType}/${cfg.tmdbID}`
    let cancelled = false

    ;(async () => {
      try {
        const res = await fetch(url)
        if (!res.ok) throw new Error("Failed to load torrents")
        const data: FetchResponse = await res.json()
        if (!cancelled) {
          setTorrents(data.torrents ?? [])
          setSearchHref(buildSearchResultUrl(data.search))
          setLoading(false)
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Error")
          setLoading(false)
        }
      }
    })()

    return () => {
      cancelled = true
    }
  }, [cfg?.contentType, cfg?.tmdbID])

  return (
    <section className="nw-panel">
      <div className="nw-header">
        <span className="nw-title">Download with Torrent</span>
        <a className="nw-browse" href={searchHref} title="Open the same nCore search">
          Open search result
          <ExternalLink size={12} aria-hidden />
        </a>
      </div>

      <div className="nw-body">
        {loading && (
          <div className="nw-state">
            <Loader2 size={16} className="nw-spin" />
            Loading torrents…
          </div>
        )}
        {error && <div className="nw-state nw-error">{error}</div>}
        {!loading && !error && torrents.length === 0 && (
          <div className="nw-state">No torrents found.</div>
        )}
        {!loading && !error && torrents.length > 0 && (
          <ul className="nw-list">
            {torrents.map((t) => (
              <li key={t.ID}>
                <a className="nw-item" href={`/ncore/torrent/${t.ID}`} title="Open in NCore client">
                  <div className="nw-item-main">
                    <img
                      src="/widget/ncore.png"
                      alt={t.Provider || "NCORE"}
                      className="nw-provider"
                      width={14}
                      height={14}
                    />
                    <span className="nw-item-title">{t.Title}</span>
                  </div>
                  <div className="nw-meta">
                    <span className="nw-meta-item" title="Upload date">
                      <Calendar size={11} />
                      {formatDate(t.Date)}
                    </span>
                    <span className="nw-meta-item nw-seed" title="Seeders">
                      <ArrowUp size={11} />
                      {t.Seeders ?? 0}
                    </span>
                    <span className="nw-meta-item nw-leech" title="Leechers">
                      <ArrowDown size={11} />
                      {t.Leechers ?? 0}
                    </span>
                  </div>
                </a>
              </li>
            ))}
          </ul>
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
.nw-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  border-radius: 8px;
  transition: background .15s ease;
  text-decoration: none;
  color: inherit;
}
.nw-item:hover { background: rgba(255,255,255,.05); }
.nw-item-main { display: flex; align-items: flex-start; gap: 8px; min-width: 0; }
.nw-provider { margin-top: 2px; width: 14px; height: 14px; flex-shrink: 0; border-radius: 2px; }
.nw-item-title { font-size: 12.5px; font-weight: 500; color: #f1f5f9; word-break: break-word; line-height: 1.35; }
.nw-item:hover .nw-item-title { color: #fff; text-decoration: underline; text-underline-offset: 2px; }
.nw-meta { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; padding-left: 22px; }
.nw-meta-item { display: inline-flex; align-items: center; gap: 3px; font-size: 11px; color: var(--nw-muted); }
.nw-seed { color: var(--nw-seed); }
.nw-leech { color: var(--nw-leech); }
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
@media (min-width: 720px) {
  .nw-item { flex-direction: row; align-items: center; justify-content: space-between; }
  .nw-item-main { flex: 1; min-width: 0; }
  .nw-item-title { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .nw-meta { padding-left: 0; flex-shrink: 0; }
}
`
