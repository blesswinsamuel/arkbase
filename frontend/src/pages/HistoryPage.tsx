import { useState } from 'react';
import { Link } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import { client, type RunDetail } from '@/api/client';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Terminal } from 'lucide-react';
import { formatBytes, formatDate, formatTimeAgo } from '@/lib/formatters';

export function HistoryPage() {
  const [selectedRun, setSelectedRun] = useState<RunDetail | null>(null);

  const { data: history, isLoading: loadingHistory } = useQuery({
    queryKey: ['history'],
    queryFn: async () => {
      const res = await client.GET('/api/v1/history', { params: { query: { limit: 100 } } });
      return res.data?.runs || [];
    },
  });

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Global Execution History</CardTitle>
          <CardDescription>
            Historical backup runs across all databases, execution times, sizes, and raw output logs.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loadingHistory ? (
            <div className="py-8 text-center text-muted-foreground">Loading history...</div>
          ) : history?.length === 0 ? (
            <div className="py-12 text-center text-muted-foreground">No backup runs recorded yet.</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Database</TableHead>
                  <TableHead>Started</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead>Output Size</TableHead>
                  <TableHead>Destinations</TableHead>
                  <TableHead className="text-right">Logs</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {history?.map((run: RunDetail) => (
                  <TableRow key={run.id}>
                    <TableCell>
                      {run.status === 'success' ? (
                        <Badge className="bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border-emerald-500/20">Success</Badge>
                      ) : run.status === 'running' ? (
                        <Badge variant="secondary" className="animate-pulse">Running</Badge>
                      ) : (
                        <Badge variant="destructive">Failed</Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      <Link 
                        to={`/databases/${run.database_name}`}
                        className="font-semibold text-primary hover:underline"
                      >
                        {run.database_name}
                      </Link>
                    </TableCell>
                    <TableCell className="text-xs">
                      <div>{formatDate(run.started_at)}</div>
                      <div className="text-muted-foreground text-[11px]">{formatTimeAgo(run.started_at)}</div>
                    </TableCell>
                    <TableCell className="text-xs font-mono">{run.duration_ms} ms</TableCell>
                    <TableCell className="text-xs font-mono">{formatBytes(run.size_bytes)}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {run.destinations?.map((dst: string) => (
                          <Badge key={dst} variant="outline" className="text-[10px]">
                            {dst}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setSelectedRun(run)}
                        className="h-8 px-2"
                      >
                        <Terminal className="h-4 w-4 mr-1 text-muted-foreground" />
                        <span className="text-xs">Logs</span>
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Log Viewer Dialog */}
      <Dialog open={!!selectedRun} onOpenChange={(open: boolean) => !open && setSelectedRun(null)}>
        <DialogContent className="max-w-5xl sm:max-w-5xl w-[calc(100%-2rem)] max-h-[85vh] flex flex-col">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Terminal className="h-5 w-5 text-primary" />
              <span>Execution Logs: {selectedRun?.database_name}</span>
            </DialogTitle>
            <DialogDescription>
              Run ID: <span className="font-mono">{selectedRun?.id}</span> • Started: {selectedRun?.started_at ? formatDate(selectedRun.started_at) : ''}
            </DialogDescription>
          </DialogHeader>
          <div className="flex-1 overflow-auto bg-zinc-950 text-zinc-100 p-4 rounded-md font-mono text-xs whitespace-pre-wrap leading-relaxed border border-zinc-800">
            {selectedRun?.logs || 'No log output captured.'}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
