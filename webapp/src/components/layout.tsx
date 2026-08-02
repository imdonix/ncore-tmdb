import { Link, useLocation } from "react-router-dom"
import { Clapperboard, ListVideo, Search, Download } from "lucide-react"
import { cn } from "@/lib/utils"

export function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation()
  const isHome = location.pathname === "/"
  const isDownloads = location.pathname.startsWith("/downloads")

  return (
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-40 border-b border-border/80 bg-background/80 backdrop-blur-md">
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between gap-3 px-4 sm:h-16 sm:px-6">
          <Link to="/" className="flex items-center gap-2.5 min-w-0">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/15 text-primary ring-1 ring-primary/25">
              <Download className="h-4 w-4" />
            </span>
            <div className="min-w-0">
              <div className="truncate text-sm font-semibold tracking-tight sm:text-base">
                NCore
              </div>
              <div className="hidden text-[11px] text-muted-foreground sm:block">
                Torrent search
              </div>
            </div>
          </Link>

          <nav className="flex items-center gap-1 sm:gap-2">
            <Link
              to="/"
              className={cn(
                "inline-flex items-center gap-1.5 rounded-md px-2.5 py-2 text-sm font-medium transition-colors sm:px-3",
                isHome
                  ? "bg-secondary text-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-foreground"
              )}
            >
              <Search className="h-4 w-4" />
              <span className="hidden sm:inline">Search</span>
            </Link>
            <Link
              to="/downloads"
              className={cn(
                "inline-flex items-center gap-1.5 rounded-md px-2.5 py-2 text-sm font-medium transition-colors sm:px-3",
                isDownloads
                  ? "bg-secondary text-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-foreground"
              )}
            >
              <ListVideo className="h-4 w-4" />
              <span className="hidden sm:inline">qBit</span>
            </Link>
            <a
              href="/"
              className="inline-flex items-center gap-1.5 rounded-md px-2.5 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground sm:px-3"
              title="Back to TMDB"
            >
              <Clapperboard className="h-4 w-4" />
              <span className="hidden sm:inline">TMDB</span>
            </a>
          </nav>
        </div>
      </header>

      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-4 sm:px-6 sm:py-8">
        {children}
      </main>
    </div>
  )
}
