import { useCallback, useEffect, useMemo, useState } from "react"
import { Link, useSearchParams } from "react-router-dom"
import { Bell, ExternalLink, Loader2, RefreshCw, Trash2 } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"

export type Follow = {
  id: number
  tmdbId: number
  name: string
  year?: string
  quality: string
  searchPattern: string
  status: string
  posterPath?: string
  lastCheckAt?: string
  lastError?: string
  wanted?: number
  found?: number
  completed?: number
}

export type FollowItem = {
  id: number
  season: number
  episode: number
  status: string
  ncoreTorrentId?: string
  torrentTitle?: string
  qbitHash?: string
  coveredBy?: number
  updatedAt?: string
}

async function parseError(res: Response) {
  try {
    const d = await res.json()
    return d.error || res.statusText
  } catch {
    return res.statusText
  }
}

export function FollowsPage() {
  const [params] = useSearchParams()
  const focusId = params.get("id")

  const [follows, setFollows] = useState<Follow[]>([])
  const [selected, setSelected] = useState<number | null>(
    focusId ? Number(focusId) : null
  )
  const [items, setItems] = useState<FollowItem[]>([])
  const [loading, setLoading] = useState(true)
  const [detailLoading, setDetailLoading] = useState(false)
  const [checking, setChecking] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadList = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch("/api/follows")
      if (!res.ok) throw new Error(await parseError(res))
      const data = await res.json()
      setFollows(data.follows ?? [])
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load")
    } finally {
      setLoading(false)
    }
  }, [])

  const loadDetail = useCallback(async (tmdbId: number) => {
    setDetailLoading(true)
    try {
      const res = await fetch(`/api/follows/${tmdbId}`)
      if (!res.ok) throw new Error(await parseError(res))
      const data = await res.json()
      setItems(data.items ?? [])
      if (data.follow) {
        setFollows((prev) => {
          const rest = prev.filter((f) => f.tmdbId !== tmdbId)
          return [data.follow, ...rest]
        })
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load detail")
    } finally {
      setDetailLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadList()
  }, [loadList])

  useEffect(() => {
    if (focusId) setSelected(Number(focusId))
  }, [focusId])

  useEffect(() => {
    if (selected) void loadDetail(selected)
    else setItems([])
  }, [selected, loadDetail])

  const active = useMemo(
    () => follows.find((f) => f.tmdbId === selected) ?? null,
    [follows, selected]
  )

  const onCheck = async (tmdbId: number) => {
    setChecking(true)
    setError(null)
    try {
      const res = await fetch(`/api/follows/${tmdbId}/check`, { method: "POST" })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || "Check failed")
      setItems(data.items ?? [])
      if (data.follow) {
        setFollows((prev) => prev.map((f) => (f.tmdbId === tmdbId ? data.follow : f)))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Check failed")
    } finally {
      setChecking(false)
    }
  }

  const onCheckAll = async () => {
    setChecking(true)
    try {
      await fetch("/api/follows/check-all", { method: "POST" })
      setTimeout(() => void loadList(), 2000)
    } finally {
      setChecking(false)
    }
  }

  const onUnfollow = async (tmdbId: number) => {
    if (!window.confirm("Stop following this series?")) return
    await fetch(`/api/follows/${tmdbId}`, { method: "DELETE" })
    if (selected === tmdbId) setSelected(null)
    await loadList()
  }

  const epLabel = (it: FollowItem) =>
    it.episode === 0
      ? `S${String(it.season).padStart(2, "0")} pack`
      : `S${String(it.season).padStart(2, "0")}E${String(it.episode).padStart(2, "0")}`

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">Follows</h1>
          <p className="text-sm text-muted-foreground">
            Series auto-download from nCore. Checks run hourly; you can also check now.
          </p>
        </div>
        <Button variant="secondary" onClick={onCheckAll} disabled={checking || follows.length === 0}>
          {checking ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
          Check all
        </Button>
      </div>

      {error && (
        <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-red-300">
          {error}
        </div>
      )}

      {loading && (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full rounded-xl" />
          ))}
        </div>
      )}

      {!loading && follows.length === 0 && (
        <div className="rounded-xl border border-dashed border-border px-6 py-16 text-center">
          <Bell className="mx-auto mb-3 h-8 w-8 text-muted-foreground" />
          <p className="text-sm font-medium">No series followed yet</p>
          <p className="mt-1 text-xs text-muted-foreground">
            Open a TV show on TMDB and use the Follow widget.
          </p>
        </div>
      )}

      {!loading && follows.length > 0 && (
        <div className="grid gap-4 lg:grid-cols-5">
          <div className="space-y-2 lg:col-span-2">
            {follows.map((f) => (
              <button
                key={f.id}
                type="button"
                onClick={() => setSelected(f.tmdbId)}
                className={`w-full rounded-xl border p-3 text-left transition-colors ${
                  selected === f.tmdbId
                    ? "border-primary/50 bg-primary/10"
                    : "border-border bg-card hover:bg-accent/40"
                }`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <div className="truncate text-sm font-semibold">
                      {f.name}
                      {f.year ? ` (${f.year})` : ""}
                    </div>
                    <div className="mt-1 flex flex-wrap gap-1.5">
                      <Badge variant="secondary">{f.quality}</Badge>
                      <Badge variant={f.status === "active" ? "success" : "muted"}>
                        {f.status}
                      </Badge>
                    </div>
                  </div>
                </div>
                <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                  <span>
                    {f.completed ?? 0} completed
                    {f.lastCheckAt
                      ? ` · checked ${new Date(f.lastCheckAt).toLocaleString()}`
                      : " · never checked"}
                  </span>
                  <a
                    href={`/tv/${f.tmdbId}`}
                    onClick={(e) => e.stopPropagation()}
                    className="inline-flex items-center gap-1 font-medium text-primary hover:underline"
                  >
                    TMDB
                    <ExternalLink className="h-3 w-3" />
                  </a>
                </div>
              </button>
            ))}
          </div>

          <div className="lg:col-span-3">
            {!selected && (
              <div className="rounded-xl border border-border px-6 py-12 text-center text-sm text-muted-foreground">
                Select a series to see downloads and run a check.
              </div>
            )}

            {selected && active && (
              <Card>
                <CardHeader className="space-y-3">
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div>
                      <CardTitle className="text-lg">
                        <a
                          href={`/tv/${active.tmdbId}`}
                          className="inline-flex items-center gap-1.5 hover:text-primary hover:underline"
                        >
                          {active.name}
                          {active.year ? ` (${active.year})` : ""}
                          <ExternalLink className="h-4 w-4 shrink-0 opacity-70" />
                        </a>
                      </CardTitle>
                      <p className="mt-1 text-xs text-muted-foreground">
                        Search: <code className="text-foreground/80">{active.searchPattern}</code>
                        {" · "}
                        <a href={`/tv/${active.tmdbId}`} className="text-primary hover:underline">
                          Open on TMDB
                        </a>
                      </p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        size="sm"
                        onClick={() => void onCheck(active.tmdbId)}
                        disabled={checking}
                      >
                        {checking ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <RefreshCw className="h-4 w-4" />
                        )}
                        Check now
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        className="text-red-400"
                        onClick={() => void onUnfollow(active.tmdbId)}
                      >
                        <Trash2 className="h-4 w-4" />
                        Unfollow
                      </Button>
                    </div>
                  </div>
                  {active.lastError && (
                    <div className="text-xs text-red-300">Last error: {active.lastError}</div>
                  )}
                </CardHeader>
                <CardContent>
                  {detailLoading && <Skeleton className="h-32 w-full" />}
                  {!detailLoading && items.length === 0 && (
                    <p className="text-sm text-muted-foreground">
                      No episode rows yet — run Check now.
                    </p>
                  )}
                  {!detailLoading && items.length > 0 && (
                    <div className="max-h-[28rem] space-y-1 overflow-y-auto">
                      {items.map((it) => (
                        <div
                          key={it.id}
                          className="flex flex-wrap items-center gap-2 rounded-lg border border-border/60 px-3 py-2 text-sm"
                        >
                          <span className="min-w-[5.5rem] font-semibold tabular-nums">
                            {epLabel(it)}
                          </span>
                          <Badge
                            variant={
                              it.status === "completed"
                                ? "success"
                                : it.status === "downloading" || it.status === "found"
                                  ? "default"
                                  : it.status === "failed" || it.status === "cannot_find"
                                    ? "danger"
                                    : "muted"
                            }
                          >
                            {it.status === "cannot_find" ? "cannot find" : it.status}
                          </Badge>
                          <span className="min-w-0 flex-1 truncate text-xs text-muted-foreground">
                            {it.torrentTitle ||
                              (it.coveredBy
                                ? "Covered by season pack"
                                : it.status === "cannot_find"
                                  ? "No matching release on nCore"
                                  : "—")}
                          </span>
                          {it.ncoreTorrentId ? (
                            <Link
                              to={`/torrent/${it.ncoreTorrentId}`}
                              className="text-xs font-medium text-primary hover:underline"
                            >
                              Open torrent
                            </Link>
                          ) : null}
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
