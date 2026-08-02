import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { Layout } from "@/components/layout"
import { SearchPage } from "@/pages/search"
import { TorrentPage } from "@/pages/torrent"

export default function App() {
  return (
    <BrowserRouter basename="/ncore">
      <Layout>
        <Routes>
          <Route path="/" element={<SearchPage />} />
          <Route path="/torrent/:id" element={<TorrentPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  )
}
