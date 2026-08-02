import { Link } from "react-router-dom"
import { ArrowDown, ArrowUp, Calendar, HardDrive } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Card } from "@/components/ui/card"
import type { Torrent } from "@/lib/api"
import { formatDate, formatSize, typeLabel } from "@/lib/utils"

export function TorrentCard({ torrent }: { torrent: Torrent }) {
  return (
    <Link to={`/torrent/${torrent.ID}`} className="block group">
      <Card className="p-3.5 transition-all hover:border-primary/40 hover:bg-card/80 hover:shadow-md sm:p-4">
        <div className="flex flex-col gap-2.5">
          <div className="flex items-start justify-between gap-3">
            <h3 className="line-clamp-2 text-sm font-medium leading-snug text-foreground group-hover:text-primary sm:text-[15px]">
              {torrent.Title}
            </h3>
            <Badge variant="secondary" className="shrink-0">
              {typeLabel(torrent.Type)}
            </Badge>
          </div>

          <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 text-xs text-muted-foreground">
            <span className="inline-flex items-center gap-1">
              <HardDrive className="h-3.5 w-3.5" />
              {formatSize(torrent.Size)}
            </span>
            <span className="inline-flex items-center gap-1">
              <Calendar className="h-3.5 w-3.5" />
              {formatDate(torrent.Date)}
            </span>
            <span className="inline-flex items-center gap-1 text-emerald-400">
              <ArrowUp className="h-3.5 w-3.5" />
              {torrent.Seeders}
            </span>
            <span className="inline-flex items-center gap-1 text-red-400">
              <ArrowDown className="h-3.5 w-3.5" />
              {torrent.Leechers}
            </span>
          </div>
        </div>
      </Card>
    </Link>
  )
}
