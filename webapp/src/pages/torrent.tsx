import { useEffect, useState } from "react"
import { useNavigate, useParams } from "react-router-dom"
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
} from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  downloadTorrentUrl,
  getTorrent,
  scheduleQbit,
  type Torrent,
} from "@/lib/api"
import { formatDate, formatSize, typeLabel } from "@/lib/utils"

export function TorrentPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [torrent, setTorrent] = useState<Torrent | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [scheduling, setScheduling] = useState(false)
  const [scheduled, setScheduled] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    setLoading(true)
    setError(null)
    setScheduled(false)
    setActionError(null)
    getTorrent(id)
      .then(setTorrent)
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load"))
      .finally(() => setLoading(false))
  }, [id])

  const onQbit = async () => {
    if (!torrent) return
    setScheduling(true)
    setActionError(null)
    try {
      await scheduleQbit(torrent.ID, torrent.Title)
      setScheduled(true)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to schedule")
    } finally {
      setScheduling(false)
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

  return (
    <div className="space-y-5">
      <Button variant="ghost" size="sm" className="-ml-2" onClick={goBack}>
        <ArrowLeft className="h-4 w-4" />
        Back
      </Button>

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

          {scheduled && (
            <div className="flex items-center gap-2 rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-300">
              <Check className="h-4 w-4" />
              Scheduled in qBittorrent
            </div>
          )}

          <div className="flex flex-col gap-2 sm:flex-row">
            <Button
              className="h-11 flex-1"
              onClick={onQbit}
              disabled={scheduling || scheduled}
            >
              {scheduling ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : scheduled ? (
                <Check className="h-4 w-4" />
              ) : (
                <Download className="h-4 w-4" />
              )}
              {scheduled ? "Added to qBit" : "Send to qBittorrent"}
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
