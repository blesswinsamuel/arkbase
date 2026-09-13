import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router';
import { Navbar } from './components/Navbar';
import { DatabasesPage } from './pages/DatabasesPage';
import { DatabaseDetailPage } from './pages/DatabaseDetailPage';
import { HistoryPage } from './pages/HistoryPage';
import { DestinationsPage } from './pages/DestinationsPage';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchInterval: 5000, // Live poll every 5s
    },
  },
});

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <div className="min-h-screen bg-background text-foreground flex flex-col">
          <Navbar />
          <main className="flex-1 container max-w-7xl mx-auto px-4 py-8">
            <Routes>
              <Route path="/" element={<Navigate to="/databases" replace />} />
              <Route path="/databases" element={<DatabasesPage />} />
              <Route path="/databases/:id" element={<DatabaseDetailPage />} />
              <Route path="/history" element={<HistoryPage />} />
              <Route path="/destinations" element={<DestinationsPage />} />
              <Route path="*" element={<Navigate to="/databases" replace />} />
            </Routes>
          </main>
        </div>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
