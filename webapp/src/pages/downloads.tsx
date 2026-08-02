import { useCallback, useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { ExternalLink, Loader2, Trash2 } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  deleteQbitTorrent,
  formatBytes,
  formatEta,
  formatSpeed,
  listQbitTorrents,
  qbitStateLabel,
  type QbitTorrent,
} from "@/lib/api"

export function DownloadsPage() {
  const [torrents, setTorrents] = useState<QbitTorrent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<string | null>(null)

  const refresh = useCallback(async (silent = false) => {
    if (!silent) setLoading(true)
    try {
      const list = await listQbitTorrents()
      // Active / incomplete first, then by added time desc
      list.sort((a, b) => {
        const aDone = a.progress >= 1 ? 1 : 0
        const bDone = b.progress >= 1 ? 1 : 0
        if (aDone !== bDone) return aDone - bDone
        return (b.added_on ?? 0) - (a.added_on ?? 0)
      })
      setTorrents(list)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load downloads")
    } finally {
      if (!silent) setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
    const id = window.setInterval(() => void refresh(true), 3000)
    return () => window.clearInterval(id)
  }, [refresh])

  const onDelete = async (t: QbitTorrent) => {
    const ok = window.confirm(
      `Delete "${t.name}" and remove files from disk?`
    )
    if (!ok) return
    setDeleting(t.hash)
    try {
      await deleteQbitTorrent(t.hash, true)
      await refresh(true)
    } catch (err) {
      alert(err instanceof Error ? err.message : "Delete failed")
    } finally {
      setDeleting(null)
    }
  }

  return (
    <div className="space-y-5">
      <div className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">qBittorrent</h1>
        <p className="text-sm text-muted-foreground">
          Active and completed torrents. Open nCore details when the download was
          started from this client.
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-red-300">
          {error}
        </div>
      )}

      {loading && (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-28 w-full rounded-xl" />
          ))}
        </div>
      )}

      {!loading && torrents.length === 0 && !error && (
        <div className="rounded-xl border border-dashed border-border px-6 py-16 text-center">
          <p className="text-sm font-medium">No torrents in qBittorrent</p>
          <p className="mt-1 text-xs text-muted-foreground">
            Send something from Search or a torrent detail page.
          </p>
          <Button asChild className="mt-4" variant="secondary">
            <Link to="/">Go to search</Link>
          </Button>
        </div>
      )}

      {!loading && torrents.length > 0 && (
        <div className="space-y-3">
          {torrents.map((t) => (
            <DownloadRow
              key={t.hash}
              torrent={t}
              deleting={deleting === t.hash}
              onDelete={() => void onDelete(t)}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function DownloadRow({
  torrent: t,
  deleting,
  onDelete,
}: {
  torrent: QbitTorrent
  deleting: boolean
  onDelete: () => void
}) {
  const pct = Math.min(100, Math.max(0, (t.progress || 0) * 100))
  const done = pct >= 99.9
  const ncoreHref = t.ncoreId ? `/torrent/${t.ncoreId}` : null

  return (
    <Card className="p-4 space-y-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-1.5">
          {ncoreHref ? (
            <Link
              to={ncoreHref}
              className="block text-sm font-medium leading-snug text-foreground hover:text-primary hover:underline break-words"
            >
              {t.name}
            </Link>
          ) : (
            <div className="text-sm font-medium leading-snug break-words">{t.name}</div>
          )}
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={done ? "success" : "secondary"}>
              {qbitStateLabel(t.state)}
            </Badge>
            {t.ncoreId && (
              <Badge variant="outline" className="gap-1">
                nCore #{t.ncoreId}
              </Badge>
            )}
          </div>
        </div>
        <Button
          variant="outline"
          size="icon"
          className="shrink-0 text-red-400 hover:text-red-300 hover:bg-red-500/10"
          onClick={onDelete}
          disabled={deleting}
          title="Delete torrent and files"
        >
          {deleting ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Trash2 className="h-4 w-4" />
          )}
        </Button>
      </div>

      <div className="space-y-1.5">
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span>{pct.toFixed(1)}%</span>
          <span>
            {formatBytes(t.downloaded || pct * t.size)} / {formatBytes(t.size)}
          </span>
        </div>
        <div className="h-2 overflow-hidden rounded-full bg-muted">
          <div
            className={`h-full rounded-full transition-all ${
              done ? "bg-emerald-500" : "bg-primary"
            }`}
            style={{ width: `${pct}%` }}
          />
        </div>
      </div>

      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <span>↓ {formatSpeed(t.dlspeed)}</span>
        <span>↑ {formatSpeed(t.upspeed)}</span>
        <span>ETA {formatEta(t.eta)}</span>
        <span>Ratio {(t.ratio ?? 0).toFixed(2)}</span>
      </div>

      {ncoreHref && (
        <Link
          to={ncoreHref}
          className="inline-flex items-center gap-1.5 text-xs font-medium text-primary hover:underline"
        >
          Open nCore torrent
          <ExternalLink className="h-3 w-3" />
        </Link>
      )}
    </Card>
  )
}
