import { useState } from 'react';
import { Link } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import { client, type HistoryRun } from '@/api/client';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Terminal, ChevronLeft, ChevronRight, Loader2 } from 'lucide-react';
import { formatBytes, formatDate, formatTimeAgo } from '@/lib/formatters';

export function HistoryPage() {
  const [selectedRun, setSelectedRun] = useState<HistoryRun | null>(null);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const { data, isLoading: loadingHistory } = useQuery({
    queryKey: ['history', page, pageSize],
    queryFn: async () => {
      const res = await client.GET('/api/v1/history', { 
        params: { query: { limit: pageSize, offset: (page - 1) * pageSize } } 
      });
      return {
        runs: res.data?.runs || [],
        total: res.data?.total || 0,
      };
    },
  });

  const runs = data?.runs || [];
  const total = data?.total || 0;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  // Query logs on demand when a run is selected
  const { data: logsData, isLoading: loadingLogs } = useQuery({
    queryKey: ['run-logs', selectedRun?.id],
    queryFn: async () => {
      if (!selectedRun?.id) return null;
      const res = await client.GET('/api/v1/history/{id}/logs', {
        params: { path: { id: selectedRun.id } },
      });
      return res.data?.logs || '';
    },
    enabled: !!selectedRun?.id,
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
          ) : runs.length === 0 ? (
            <div className="py-12 text-center text-muted-foreground">No backup runs recorded yet.</div>
          ) : (
            <>
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
                  {runs.map((run: HistoryRun) => (
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

              {/* Pagination Bar */}
              {total > 0 && (
                <div className="flex flex-col sm:flex-row items-center justify-between gap-3 pt-4 border-t border-border mt-4 text-xs text-muted-foreground">
                  <div>
                    Showing <span className="font-medium text-foreground">{(page - 1) * pageSize + 1}</span> to{' '}
                    <span className="font-medium text-foreground">{Math.min(page * pageSize, total)}</span> of{' '}
                    <span className="font-medium text-foreground">{total}</span> runs
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="mr-2">
                      Page {page} of {totalPages}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage((p) => Math.max(1, p - 1))}
                      disabled={page <= 1}
                      className="h-8 px-2.5 gap-1"
                    >
                      <ChevronLeft className="h-3.5 w-3.5" />
                      <span>Previous</span>
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                      disabled={page >= totalPages}
                      className="h-8 px-2.5 gap-1"
                    >
                      <span>Next</span>
                      <ChevronRight className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              )}
            </>
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
          <div className="flex-1 overflow-auto bg-zinc-950 text-zinc-100 p-4 rounded-md font-mono text-xs whitespace-pre-wrap leading-relaxed border border-zinc-800 min-h-[200px]">
            {loadingLogs ? (
              <div className="flex items-center justify-center py-12 text-zinc-400 gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                <span>Loading execution logs...</span>
              </div>
            ) : (
              logsData || 'No log output captured.'
            )}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
