import { useCallback, useEffect, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import {
  ArrowDown,
  ArrowLeft,
  ArrowUp,
  Calendar,
  Check,
  Download,
  ExternalLink,
  HardDrive,
  Loader2,
  Trash2,
} from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  deleteQbitTorrent,
  downloadTorrentUrl,
  formatBytes,
  formatEta,
  formatSpeed,
  getQbitByNcoreId,
  getTorrent,
  qbitStateLabel,
  scheduleQbit,
  type QbitTorrent,
  type Torrent,
} from "@/lib/api"
import { formatDate, formatSize, typeLabel } from "@/lib/utils"

export function TorrentPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [torrent, setTorrent] = useState<Torrent | null>(null)
  const [qbit, setQbit] = useState<QbitTorrent | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [scheduling, setScheduling] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)

  const refreshQbit = useCallback(async () => {
    if (!id) return
    try {
      const t = await getQbitByNcoreId(id)
      setQbit(t)
    } catch {
      // ignore poll errors
    }
  }, [id])

  useEffect(() => {
    if (!id) return
    setLoading(true)
    setError(null)
    setActionError(null)
    getTorrent(id)
      .then(setTorrent)
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load"))
      .finally(() => setLoading(false))
  }, [id])

  useEffect(() => {
    void refreshQbit()
    const timer = window.setInterval(() => void refreshQbit(), 3000)
    return () => window.clearInterval(timer)
  }, [refreshQbit])

  const onQbit = async () => {
    if (!torrent) return
    setScheduling(true)
    setActionError(null)
    try {
      await scheduleQbit(torrent.ID, torrent.Title)
      await refreshQbit()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to schedule")
    } finally {
      setScheduling(false)
    }
  }

  const onDelete = async () => {
    if (!qbit) return
    const ok = window.confirm(
      `Delete "${qbit.name}" from qBittorrent and remove files on disk?`
    )
    if (!ok) return
    setDeleting(true)
    setActionError(null)
    try {
      await deleteQbitTorrent(qbit.hash, true)
      setQbit(null)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Delete failed")
    } finally {
      setDeleting(false)
    }
  }

  const goBack = () => {
    if (window.history.length > 1) navigate(-1)
    else navigate("/")
  }

  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-40 w-full rounded-xl" />
        <Skeleton className="h-24 w-full rounded-xl" />
      </div>
    )
  }

  if (error || !torrent) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" size="sm" onClick={goBack}>
          <ArrowLeft className="h-4 w-4" />
          Back
        </Button>
        <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-red-300">
          {error || "Torrent not found"}
        </div>
      </div>
    )
  }

  const inQbit = !!qbit
  const pct = qbit ? Math.min(100, Math.max(0, qbit.progress * 100)) : 0
  const done = pct >= 99.9

  return (
    <div className="space-y-5">
      <Button variant="ghost" size="sm" className="-ml-2" onClick={goBack}>
        <ArrowLeft className="h-4 w-4" />
        Back
      </Button>

      {qbit && (
        <Card className="border-primary/30 bg-primary/5">
          <CardHeader className="space-y-2 pb-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={done ? "success" : "default"}>
                {qbitStateLabel(qbit.state)}
              </Badge>
              <span className="text-xs text-muted-foreground">qBittorrent</span>
            </div>
            <CardTitle className="text-base sm:text-lg">
              {done ? "Downloaded" : "Download in progress"}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-1.5">
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>{pct.toFixed(1)}%</span>
                <span>
                  {formatBytes(qbit.downloaded || (pct / 100) * qbit.size)} /{" "}
                  {formatBytes(qbit.size)}
                </span>
              </div>
              <div className="h-2.5 overflow-hidden rounded-full bg-muted">
                <div
                  className={`h-full rounded-full transition-all ${
                    done ? "bg-emerald-500" : "bg-primary"
                  }`}
                  style={{ width: `${pct}%` }}
                />
              </div>
            </div>

            <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
              <span>↓ {formatSpeed(qbit.dlspeed)}</span>
              <span>↑ {formatSpeed(qbit.upspeed)}</span>
              <span>ETA {formatEta(qbit.eta)}</span>
              <span>Ratio {(qbit.ratio ?? 0).toFixed(2)}</span>
            </div>

            <div className="flex flex-col gap-2 sm:flex-row">
              <Button asChild variant="secondary" className="h-10 flex-1">
                <Link to="/downloads">View all downloads</Link>
              </Button>
              <Button
                variant="outline"
                className="h-10 flex-1 text-red-400 hover:text-red-300"
                onClick={onDelete}
                disabled={deleting}
              >
                {deleting ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Trash2 className="h-4 w-4" />
                )}
                Delete + files
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="secondary">{typeLabel(torrent.Type)}</Badge>
            <Badge variant="outline">#{torrent.ID}</Badge>
          </div>
          <CardTitle className="text-lg leading-snug sm:text-xl break-words">
            {torrent.Title}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <Meta
              icon={<HardDrive className="h-4 w-4" />}
              label="Size"
              value={formatSize(torrent.Size)}
            />
            <Meta
              icon={<Calendar className="h-4 w-4" />}
              label="Uploaded"
              value={formatDate(torrent.Date)}
            />
            <Meta
              icon={<ArrowUp className="h-4 w-4 text-emerald-400" />}
              label="Seeders"
              value={String(torrent.Seeders)}
              valueClass="text-emerald-400"
            />
            <Meta
              icon={<ArrowDown className="h-4 w-4 text-red-400" />}
              label="Leechers"
              value={String(torrent.Leechers)}
              valueClass="text-red-400"
            />
          </div>

          {actionError && (
            <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-red-300">
              {actionError}
            </div>
          )}

          <div className="flex flex-col gap-2 sm:flex-row">
            <Button
              className="h-11 flex-1"
              onClick={onQbit}
              disabled={scheduling || inQbit}
            >
              {scheduling ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : inQbit ? (
                <Check className="h-4 w-4" />
              ) : (
                <Download className="h-4 w-4" />
              )}
              {inQbit ? "In qBittorrent" : "Send to qBittorrent"}
            </Button>
            <Button variant="outline" className="h-11 flex-1" asChild>
              <a href={downloadTorrentUrl(torrent.ID, torrent.Title)} download>
                <Download className="h-4 w-4" />
                Download .torrent
              </a>
            </Button>
          </div>

          {torrent.URL && (
            <a
              href={torrent.URL}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
            >
              <ExternalLink className="h-3.5 w-3.5" />
              Open on nCore
            </a>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function Meta({
  icon,
  label,
  value,
  valueClass,
}: {
  icon: React.ReactNode
  label: string
  value: string
  valueClass?: string
}) {
  return (
    <div className="rounded-lg border border-border/70 bg-muted/40 px-3 py-2.5">
      <div className="mb-1 flex items-center gap-1.5 text-[11px] uppercase tracking-wide text-muted-foreground">
        {icon}
        {label}
      </div>
      <div className={`text-sm font-semibold ${valueClass ?? ""}`}>{value}</div>
    </div>
  )
}
