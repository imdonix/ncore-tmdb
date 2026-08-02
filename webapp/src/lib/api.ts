export type SearchTypeOption = {
  value: string
  label: string
  group: string
}

export type Torrent = {
  ID: string
  Title: string
  Key?: string
  Size?: unknown
  Type: string
  Date: string
  Seeders: number
  Leechers: number
  Download?: string
  URL?: string
  Extra?: Record<string, unknown> | null
}

export type SearchResult = {
  Torrents: Torrent[]
  NumOfPages: number
}

export type SearchParams = {
  pattern: string
  type: string
  where: string
  sort_by: string
  sort_order: string
  page: number
}

async function parseError(res: Response): Promise<string> {
  try {
    const data = await res.json()
    return data.error || res.statusText
  } catch {
    return res.statusText || "Request failed"
  }
}

export async function fetchTypes(): Promise<SearchTypeOption[]> {
  const res = await fetch("/api/ncore/types")
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function searchTorrents(params: SearchParams): Promise<SearchResult> {
  const res = await fetch("/api/ncore/search", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
  })
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function getTorrent(id: string): Promise<Torrent> {
  const res = await fetch(`/api/ncore/torrent/${encodeURIComponent(id)}`)
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function scheduleQbit(id: string, name?: string): Promise<void> {
  const qs = name ? `?name=${encodeURIComponent(name)}` : ""
  const res = await fetch(`/api/ncore/qbit/${encodeURIComponent(id)}${qs}`, {
    method: "POST",
  })
  if (!res.ok) throw new Error(await parseError(res))
}

export function downloadTorrentUrl(id: string, name?: string): string {
  const qs = name ? `?name=${encodeURIComponent(name)}` : ""
  return `/api/ncore/download/${encodeURIComponent(id)}${qs}`
}
