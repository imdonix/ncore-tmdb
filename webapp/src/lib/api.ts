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

export type QbitTorrent = {
  hash: string
  name: string
  progress: number
  state: string
  size: number
  downloaded?: number
  uploaded?: number
  dlspeed: number
  upspeed: number
  eta: number
  ratio: number
  tags: string
  category?: string
  save_path?: string
  added_on?: number
  completion_on?: number
  num_seeds?: number
  num_leechs?: number
  ncoreId?: string
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

export async function listQbitTorrents(): Promise<QbitTorrent[]> {
  const res = await fetch("/api/qbit/torrents")
  if (!res.ok) throw new Error(await parseError(res))
  const data = await res.json()
  return data.torrents ?? []
}

export async function getQbitByNcoreId(id: string): Promise<QbitTorrent | null> {
  const res = await fetch(`/api/qbit/torrents/ncore/${encodeURIComponent(id)}`)
  if (!res.ok) throw new Error(await parseError(res))
  const data = await res.json()
  return data.torrent ?? null
}

export async function deleteQbitTorrent(
  hash: string,
  deleteFiles = true
): Promise<void> {
  const qs = deleteFiles ? "?deleteFiles=true" : "?deleteFiles=false"
  const res = await fetch(`/api/qbit/torrents/${encodeURIComponent(hash)}${qs}`, {
    method: "DELETE",
  })
  if (!res.ok) throw new Error(await parseError(res))
}

export function formatBytes(n: number): string {
  if (!n || n < 0) return "0 B"
  const units = ["B", "KiB", "MiB", "GiB", "TiB"]
  let v = n
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

export function formatSpeed(n: number): string {
  if (!n || n <= 0) return "—"
  return `${formatBytes(n)}/s`
}

export function formatEta(seconds: number): string {
  if (seconds == null || seconds < 0 || seconds >= 8640000) return "—"
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  if (seconds < 86400) {
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    return `${h}h ${m}m`
  }
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  return `${d}d ${h}h`
}

export function qbitStateLabel(state: string): string {
  const map: Record<string, string> = {
    downloading: "Downloading",
    metaDL: "Fetching metadata",
    forcedDL: "Downloading",
    allocating: "Allocating",
    stalledDL: "Stalled",
    checkingDL: "Checking",
    checkingUP: "Checking",
    checkingResumeData: "Checking",
    moving: "Moving",
    uploading: "Seeding",
    forcedUP: "Seeding",
    stalledUP: "Seeding (stalled)",
    queuedDL: "Queued",
    queuedUP: "Queued seed",
    pausedDL: "Paused",
    pausedUP: "Paused seed",
    stoppedDL: "Stopped",
    stoppedUP: "Stopped seed",
    error: "Error",
    missingFiles: "Missing files",
    unknown: "Unknown",
  }
  return map[state] || state || "Unknown"
}
