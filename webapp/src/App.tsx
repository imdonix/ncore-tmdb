import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { Layout } from "@/components/layout"
import { DownloadsPage } from "@/pages/downloads"
import { FollowsPage } from "@/pages/follows"
import { SearchPage } from "@/pages/search"
import { TorrentPage } from "@/pages/torrent"

export default function App() {
  return (
    <BrowserRouter basename="/ncore">
      <Layout>
        <Routes>
          <Route path="/" element={<SearchPage />} />
          <Route path="/downloads" element={<DownloadsPage />} />
          <Route path="/follows" element={<FollowsPage />} />
          <Route path="/torrent/:id" element={<TorrentPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  )
}
