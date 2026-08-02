import { useEffect, useState } from "react"
import { ArrowDown, ArrowUp, Calendar, ExternalLink, Loader2 } from "lucide-react"

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
  activeDownload?: {
    id: string
    progress: number
    status: string
  } | null
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
        <a
          className="nw-browse"
          href={searchHref}
          title="Open the same nCore search in the full client"
        >
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

const widgetCss = `
.nw-panel {
  margin: 0 0 16px;
  border: 1px solid var(--nw-border);
  border-radius: 12px;
  background: linear-gradient(180deg, rgba(30,41,59,.75), rgba(15,23,42,.7));
  box-shadow: 0 8px 24px rgba(0,0,0,.18);
  overflow: hidden;
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
.nw-browse svg {
  display: block;
  flex-shrink: 0;
}
.nw-body { padding: 8px; }
.nw-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 20px 12px;
  color: var(--nw-muted);
  font-size: 12px;
}
.nw-error { color: #fca5a5; }
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
  padding: 10px 10px;
  border-radius: 8px;
  background: transparent;
  transition: background .15s ease;
}
.nw-item:hover {
  background: rgba(255,255,255,.05);
}
.nw-item-main {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
}
.nw-provider {
  margin-top: 2px;
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  border-radius: 2px;
  opacity: .9;
}
.nw-item-title {
  font-size: 12.5px;
  font-weight: 500;
  color: #f1f5f9;
  word-break: break-word;
  line-height: 1.35;
}
.nw-item:hover .nw-item-title {
  color: #fff;
  text-decoration: underline;
  text-underline-offset: 2px;
}
.nw-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding-left: 22px;
}
.nw-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 11px;
  color: var(--nw-muted);
}
.nw-seed { color: var(--nw-seed); }
.nw-leech { color: var(--nw-leech); }

@media (min-width: 720px) {
  .nw-item {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }
  .nw-item-main { flex: 1; min-width: 0; }
  .nw-item-title {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .nw-meta {
    padding-left: 0;
    flex-shrink: 0;
  }
}
`
