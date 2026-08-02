import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatSize(size: unknown): string {
  if (size == null) return "—"
  if (typeof size === "string") return size || "—"
  if (typeof size === "number") {
    const units = ["B", "KiB", "MiB", "GiB", "TiB"]
    let n = size
    let i = 0
    while (n >= 1024 && i < units.length - 1) {
      n /= 1024
      i++
    }
    return `${n.toFixed(2)} ${units[i]}`
  }
  return "—"
}

export function formatDate(value: string | undefined): string {
  if (!value) return "—"
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  })
}

export function typeLabel(type: string): string {
  const map: Record<string, string> = {
    hd_hun: "HD Hun",
    hd: "HD",
    xvid_hun: "SD Hun",
    xvid: "SD",
    dvd_hun: "DVD Hun",
    dvd: "DVD",
    dvd9_hun: "DVD9 Hun",
    dvd9: "DVD9",
    hdser_hun: "HD Ser Hun",
    hdser: "HD Ser",
    xvidser_hun: "SD Ser Hun",
    xvidser: "SD Ser",
    dvdser_hun: "DVD Ser Hun",
    dvdser: "DVD Ser",
    mp3_hun: "MP3 Hun",
    mp3: "MP3",
    lossless_hun: "FLAC Hun",
    lossless: "FLAC",
    clip: "Clip",
    game_iso: "Game ISO",
    game_rip: "Game Rip",
    console: "Console",
    ebook_hun: "eBook Hun",
    ebook: "eBook",
    iso: "ISO",
    misc: "Misc",
    mobil: "Mobile",
    all_own: "All",
  }
  return map[type] || type
}
