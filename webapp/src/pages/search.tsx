import { useCallback, useEffect, useMemo, useState } from "react"
import { useSearchParams } from "react-router-dom"
import { ChevronLeft, ChevronRight, Filter, Loader2, Search as SearchIcon, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { TorrentCard } from "@/components/torrent-card"
import {
  fetchTypes,
  searchTorrents,
  type SearchTypeOption,
  type Torrent,
} from "@/lib/api"
import { cn } from "@/lib/utils"

const WHERE_OPTIONS = [
  { value: "name", label: "Name" },
  { value: "leiras", label: "Description" },
  { value: "imdb", label: "IMDB" },
  { value: "cimke", label: "Label" },
]

const SORT_OPTIONS = [
  { value: "seeders", label: "Seeders" },
  { value: "leechers", label: "Leechers" },
  { value: "size", label: "Size" },
  { value: "fid", label: "Upload date" },
  { value: "name", label: "Name" },
  { value: "times_completed", label: "Completed" },
]

export function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams()

  const pattern = searchParams.get("q") ?? ""
  const type = searchParams.get("type") ?? "all_own"
  const where = searchParams.get("where") ?? "name"
  const sortBy = searchParams.get("sort") ?? "seeders"
  const sortOrder = searchParams.get("order") ?? "DESC"
  const page = Math.max(1, Number(searchParams.get("page") || "1") || 1)

  const [query, setQuery] = useState(pattern)
  const [types, setTypes] = useState<SearchTypeOption[]>([])
  const [torrents, setTorrents] = useState<Torrent[]>([])
  const [numPages, setNumPages] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [filtersOpen, setFiltersOpen] = useState(false)
  const [searched, setSearched] = useState(false)

  useEffect(() => {
    setQuery(pattern)
  }, [pattern])

  useEffect(() => {
    fetchTypes()
      .then(setTypes)
      .catch(() => setTypes([]))
  }, [])

  const groupedTypes = useMemo(() => {
    const groups = new Map<string, SearchTypeOption[]>()
    for (const t of types) {
      const list = groups.get(t.group) ?? []
      list.push(t)
      groups.set(t.group, list)
    }
    return Array.from(groups.entries())
  }, [types])

  const updateParams = useCallback(
    (patch: Record<string, string | number | undefined>, replace = false) => {
      const next = new URLSearchParams(searchParams)
      for (const [k, v] of Object.entries(patch)) {
        if (v === undefined || v === "") next.delete(k)
        else next.set(k, String(v))
      }
      setSearchParams(next, { replace })
    },
    [searchParams, setSearchParams]
  )

  const runSearch = useCallback(async () => {
    // Empty pattern is allowed (browse by category)
    setLoading(true)
    setError(null)
    setSearched(true)
    try {
      const result = await searchTorrents({
        pattern,
        type,
        where,
        sort_by: sortBy,
        sort_order: sortOrder,
        page,
      })
      setTorrents(result.Torrents ?? [])
      setNumPages(result.NumOfPages || 0)
    } catch (err) {
      setTorrents([])
      setNumPages(0)
      setError(err instanceof Error ? err.message : "Search failed")
    } finally {
      setLoading(false)
    }
  }, [pattern, type, where, sortBy, sortOrder, page])

  useEffect(() => {
    // Auto-search when URL has a query or explicit filters/page
    const shouldSearch =
      pattern !== "" ||
      searchParams.has("type") ||
      searchParams.has("page") ||
      searchParams.has("sort") ||
      searchParams.has("where")
    if (shouldSearch) {
      void runSearch()
    }
  }, [pattern, type, where, sortBy, sortOrder, page, runSearch, searchParams])

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    updateParams({ q: query.trim(), page: 1 })
    if (!query.trim() && !pattern) {
      // force search even with empty query via param touch
      void runSearch()
    }
  }

  return (
    <div className="space-y-5">
      <div className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">Search torrents</h1>
        <p className="text-sm text-muted-foreground">
          Find releases on nCore. Tap a result for details and download options.
        </p>
      </div>

      <form onSubmit={onSubmit} className="space-y-3">
        <div className="flex gap-2">
          <div className="relative flex-1">
            <SearchIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Movie, series, game…"
              className="h-12 pl-10 pr-10 text-base sm:h-11"
              autoFocus
              enterKeyHint="search"
              autoComplete="off"
            />
            {query && (
              <button
                type="button"
                onClick={() => setQuery("")}
                className="absolute right-2 top-1/2 -translate-y-1/2 rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
                aria-label="Clear"
              >
                <X className="h-4 w-4" />
              </button>
            )}
          </div>
          <Button type="submit" className="h-12 shrink-0 px-4 sm:h-11 sm:px-6" disabled={loading}>
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <SearchIcon className="h-4 w-4" />}
            <span className="hidden sm:inline">Search</span>
          </Button>
          <Button
            type="button"
            variant="outline"
            className="h-12 w-12 shrink-0 sm:h-11 sm:w-auto sm:px-3"
            onClick={() => setFiltersOpen((v) => !v)}
            aria-expanded={filtersOpen}
          >
            <Filter className="h-4 w-4" />
            <span className="hidden sm:inline">Filters</span>
          </Button>
        </div>

        <div
          className={cn(
            "grid gap-3 overflow-hidden transition-all sm:grid-cols-2 lg:grid-cols-4",
            filtersOpen ? "max-h-96 opacity-100" : "max-h-0 opacity-0 sm:max-h-none sm:opacity-100"
          )}
        >
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Category</label>
            <Select value={type} onValueChange={(v) => updateParams({ type: v, page: 1 })}>
              <SelectTrigger>
                <SelectValue placeholder="Category" />
              </SelectTrigger>
              <SelectContent>
                {groupedTypes.map(([group, items]) => (
                  <SelectGroup key={group}>
                    <SelectLabel>{group}</SelectLabel>
                    {items.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Search in</label>
            <Select value={where} onValueChange={(v) => updateParams({ where: v, page: 1 })}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {WHERE_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Sort by</label>
            <Select value={sortBy} onValueChange={(v) => updateParams({ sort: v, page: 1 })}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {SORT_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Order</label>
            <Select
              value={sortOrder}
              onValueChange={(v) => updateParams({ order: v, page: 1 })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="DESC">Descending</SelectItem>
                <SelectItem value="ASC">Ascending</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </form>

      {error && (
        <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-red-300">
          {error}
        </div>
      )}

      {loading && (
        <div className="space-y-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full rounded-xl" />
          ))}
        </div>
      )}

      {!loading && searched && torrents.length === 0 && !error && (
        <div className="rounded-xl border border-dashed border-border px-6 py-16 text-center">
          <p className="text-sm font-medium">No torrents found</p>
          <p className="mt-1 text-xs text-muted-foreground">
            Try a different query or category.
          </p>
        </div>
      )}

      {!loading && torrents.length > 0 && (
        <>
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>
              Page {page}
              {numPages > 0 ? ` of ${numPages}` : ""}
              {" · "}
              {torrents.length} result{torrents.length === 1 ? "" : "s"}
            </span>
          </div>

          <div className="space-y-2.5">
            {torrents.map((t) => (
              <TorrentCard key={t.ID} torrent={t} />
            ))}
          </div>

          {(numPages > 1 || page > 1) && (
            <div className="flex items-center justify-center gap-3 pt-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1 || loading}
                onClick={() => updateParams({ page: page - 1 })}
              >
                <ChevronLeft className="h-4 w-4" />
                Prev
              </Button>
              <span className="min-w-16 text-center text-sm text-muted-foreground">
                {page}
                {numPages > 0 ? ` / ${numPages}` : ""}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={loading || (numPages > 0 && page >= numPages)}
                onClick={() => updateParams({ page: page + 1 })}
              >
                Next
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          )}
        </>
      )}

      {!loading && !searched && (
        <div className="rounded-xl border border-border/60 bg-card/40 px-6 py-14 text-center">
          <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 text-primary">
            <SearchIcon className="h-5 w-5" />
          </div>
          <p className="text-sm font-medium">Start typing to search nCore</p>
          <p className="mt-1 text-xs text-muted-foreground">
            Or open filters and browse by category.
          </p>
          <Button
            className="mt-4"
            variant="secondary"
            onClick={() => {
              updateParams({ type: "hd_hun", page: 1 })
            }}
          >
            Browse HD Hun movies
          </Button>
        </div>
      )}
    </div>
  )
}
