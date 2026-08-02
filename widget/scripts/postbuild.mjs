import { copyFileSync, existsSync, mkdirSync, readdirSync, unlinkSync, writeFileSync } from "fs"
import path from "path"
import { fileURLToPath } from "url"

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const outDir = path.resolve(__dirname, "../../internal/static/widget")
const pngSrc = path.resolve(__dirname, "../public/ncore.png")

mkdirSync(outDir, { recursive: true })

// Drop Vite scaffold leftovers if publicDir copied anything
for (const name of readdirSync(outDir)) {
  if (name === "favicon.svg" || name === "icons.svg") {
    unlinkSync(path.join(outDir, name))
  }
}

if (!existsSync(pngSrc)) {
  throw new Error(`missing icon: ${pngSrc}`)
}
copyFileSync(pngSrc, path.join(outDir, "ncore.png"))

const cssPath = path.join(outDir, "widget.css")
if (!existsSync(cssPath)) {
  writeFileSync(cssPath, "/* ncore widget */\n")
}

const snippet = `<section class="panel top_billed scroller ncore-widget-panel">
  <div id="ncore-widget-root"></div>
</section>
<script>
  window.__NCORE_WIDGET__ = {
    contentType: "#CONTENT_TYPE#",
    tmdbID: "#CONTENT_TMDBID#"
  };
</script>
<link rel="stylesheet" href="/widget/widget.css" />
<script type="module" src="/widget/widget.js"></script>
`

writeFileSync(path.join(outDir, "snippet.html"), snippet)
console.log("widget postbuild: wrote snippet.html + assets to", outDir)
