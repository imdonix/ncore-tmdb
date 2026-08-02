import { createRoot } from "react-dom/client"
import { Widget } from "./Widget"
import "./index.css"

const mount = () => {
  const el = document.getElementById("ncore-widget-root")
  if (!el) {
    console.warn("[ncore-widget] mount point #ncore-widget-root not found")
    return
  }
  createRoot(el).render(<Widget />)
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", mount)
} else {
  mount()
}
