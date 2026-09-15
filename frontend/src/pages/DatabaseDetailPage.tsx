import { useState } from 'react';
import { useParams, Link, useSearchParams } from 'react-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { client, type DatabaseInfo, type HistoryRun, type StorageBackupItem } from '@/api/client';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { ScrollArea } from '@/components/ui/scroll-area';
import { 
  Database, 
  HardDrive, 
  Clock, 
  CheckCircle2, 
  XCircle, 
  Play, 
  ShieldCheck, 
  Terminal, 
  RefreshCw,
  RotateCcw,
  ArrowLeft,
  Filter,
  ChevronLeft,
  ChevronRight,
  Loader2
} from 'lucide-react';
import { DatabasusBackupGraph } from '@/components/DatabasusBackupGraph';
import { RestoreDialog } from '@/components/RestoreDialog';
import { formatBytes, formatDate, formatTimeAgo } from '@/lib/formatters';

export function DatabaseDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const qc = useQueryClient();

  const activeTab = searchParams.get('tab') || 'history';
  const setActiveTab = (tab: string) => {
    setSearchParams({ tab });
  };

  const [selectedRun, setSelectedRun] = useState<HistoryRun | null>(null);
  const [selectedBackupForRestore, setSelectedBackupForRestore] = useState<StorageBackupItem | null>(null);
  const [destinationFilter, setDestinationFilter] = useState<string>('all');
  const [historyPage, setHistoryPage] = useState(1);
  const historyPageSize = 20;

  // Fetch databases to find the current database configuration
  const { data: databases } = useQuery({
    queryKey: ['databases'],
    queryFn: async () => {
      const res = await client.GET('/api/v1/databases');
      return res.data || [];
    },
  });

  const dbInfo = databases?.find((d: DatabaseInfo) => d.name === id);

  // Fetch runs specifically for this database
  const { data: historyData, isLoading: loadingRuns } = useQuery({
    queryKey: ['history', id, historyPage, historyPageSize],
    queryFn: async () => {
      if (!id) return { runs: [], total: 0 };
      const res = await client.GET('/api/v1/history', { 
        params: { query: { database: id, limit: historyPageSize, offset: (historyPage - 1) * historyPageSize } } 
      });
      return {
        runs: res.data?.runs || [],
        total: res.data?.total || 0,
      };
    },
    enabled: !!id,
  });

  const runs = historyData?.runs || [];
  const totalRuns = historyData?.total || 0;
  const totalHistoryPages = Math.max(1, Math.ceil(totalRuns / historyPageSize));

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

  // Fetch storage destination backups for this database
  const { data: storageBackups = [], isLoading: loadingBackups, refetch: refetchStorageBackups } = useQuery({
    queryKey: ['database-backups', id],
    queryFn: async () => {
      if (!id) return [];
      const res = await client.GET('/api/v1/databases/{id}/backups', {
        params: { path: { id } },
      });
      return res.data?.backups || [];
    },
    enabled: !!id,
  });

  // Trigger on-demand backup
  const triggerBackup = useMutation({
    mutationFn: async () => {
      if (!id) return;
      await client.POST('/api/v1/databases/{id}/backup', { params: { path: { id } } });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['databases'] });
      qc.invalidateQueries({ queryKey: ['history', id] });
      qc.invalidateQueries({ queryKey: ['database-backups', id] });
      qc.invalidateQueries({ queryKey: ['stats'] });
    },
  });

  if (!id) {
    return <div>Invalid database</div>;
  }

  const filteredStorageBackups = destinationFilter === 'all'
    ? storageBackups
    : storageBackups.filter(b => b.destination === destinationFilter);

  return (
    <div className="space-y-6">
      {/* Breadcrumb & Navigation */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Link to="/databases" className="hover:text-foreground flex items-center gap-1">
            <ArrowLeft className="h-4 w-4" />
            <span>Databases</span>
          </Link>
          <span>/</span>
          <span className="font-semibold text-foreground">{id}</span>
        </div>

        <Button
          size="sm"
          disabled={dbInfo?.is_running || triggerBackup.isPending}
          onClick={() => triggerBackup.mutate()}
          className="gap-1.5"
        >
          {dbInfo?.is_running ? (
            <>
              <RefreshCw className="h-3.5 w-3.5 animate-spin" />
              <span>Running Backup</span>
            </>
          ) : (
            <>
              <Play className="h-3.5 w-3.5" />
              <span>Backup Now</span>
            </>
          )}
        </Button>
      </div>

      {/* Database Overview Header Card */}
      <Card>
        <CardContent>
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div className="space-y-1.5">
              <div className="flex items-center gap-2.5">
                <Database className="h-5 w-5 text-primary" />
                <h1 className="text-xl font-bold tracking-tight">{id}</h1>
                <Badge variant="outline" className="capitalize font-mono">
                  {dbInfo?.engine || 'postgres'}
                </Badge>
                {dbInfo?.is_running ? (
                  <Badge variant="secondary" className="animate-pulse flex items-center gap-1">
                    <RefreshCw className="h-3 w-3 animate-spin" />
                    <span>Backup in progress</span>
                  </Badge>
                ) : (
                  <Badge className="bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border-emerald-500/20">
                    Ready
                  </Badge>
                )}
              </div>
              <p className="text-xs text-muted-foreground font-mono">
                {dbInfo ? `${dbInfo.host}:${dbInfo.port} (${dbInfo.database})` : 'Target connection'}
              </p>
            </div>

            <div className="flex flex-wrap gap-4 text-xs text-muted-foreground">
              <div className="bg-muted/40 p-2.5 rounded-md border min-w-[130px]">
                <div className="text-[11px]">Schedule</div>
                <div className="font-mono font-medium text-foreground mt-0.5">
                  {dbInfo?.schedule || 'Manual trigger'}
                </div>
              </div>

              <div className="bg-muted/40 p-2.5 rounded-md border min-w-[150px]">
                <div className="text-[11px]">Next Scheduled</div>
                <div className="font-medium text-foreground mt-0.5">
                  {dbInfo?.next_run_at ? formatDate(dbInfo.next_run_at) : 'None'}
                </div>
              </div>

              <div className="bg-muted/40 p-2.5 rounded-md border min-w-[140px]">
                <div className="text-[11px]">Configured Storage</div>
                <div className="flex flex-wrap gap-1 mt-0.5">
                  {dbInfo?.destinations?.map(d => (
                    <Badge key={d} variant="secondary" className="text-[10px]">
                      {d}
                    </Badge>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Databasus-Style Backup Health Graph */}
      <DatabasusBackupGraph 
        runs={runs} 
        onSelectRun={(run) => setSelectedRun(run)} 
      />

      {/* Database Detail Tabs (Default: History) */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList>
          <TabsTrigger value="history" className="flex items-center gap-2">
            <Clock className="h-4 w-4" />
            <span>Backup History</span>
            <Badge variant="secondary" className="ml-1 text-xs">{runs.length}</Badge>
          </TabsTrigger>
          <TabsTrigger value="storage" className="flex items-center gap-2">
            <HardDrive className="h-4 w-4" />
            <span>Storage Backups</span>
            <Badge variant="secondary" className="ml-1 text-xs">{storageBackups.length}</Badge>
          </TabsTrigger>
        </TabsList>

        {/* Tab 1: Backup History (Default) */}
        <TabsContent value="history">
          <Card>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-base">Execution History: {id}</CardTitle>
                  <CardDescription>
                    All scheduled and manual backup attempts with execution duration, size, and logs.
                  </CardDescription>
                </div>
                <Button 
                  variant="outline" 
                  size="sm" 
                  onClick={() => qc.invalidateQueries({ queryKey: ['history', id] })}
                  className="gap-1.5"
                >
                  <RefreshCw className="h-3.5 w-3.5" />
                  <span>Refresh</span>
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {loadingRuns ? (
                <div className="py-8 text-center text-muted-foreground">Loading history...</div>
              ) : runs.length === 0 ? (
                <div className="py-12 text-center text-muted-foreground">
                  No backup runs recorded for this database yet.
                </div>
              ) : (
                <>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Status</TableHead>
                        <TableHead>Started</TableHead>
                        <TableHead>Duration</TableHead>
                        <TableHead>Size</TableHead>
                        <TableHead>Destinations</TableHead>
                        <TableHead className="text-right">Action</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {runs.map((run: HistoryRun) => (
                        <TableRow key={run.id}>
                          <TableCell>
                            {run.status === 'success' ? (
                              <Badge className="bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 flex w-fit items-center gap-1">
                                <CheckCircle2 className="h-3 w-3" />
                                <span>Success</span>
                              </Badge>
                            ) : run.status === 'running' ? (
                              <Badge variant="secondary" className="animate-pulse flex w-fit items-center gap-1">
                                <RefreshCw className="h-3 w-3 animate-spin" />
                                <span>Running</span>
                              </Badge>
                            ) : (
                              <Badge variant="destructive" className="flex w-fit items-center gap-1">
                                <XCircle className="h-3 w-3" />
                                <span>Failed</span>
                              </Badge>
                            )}
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
                  {totalRuns > 0 && (
                    <div className="flex flex-col sm:flex-row items-center justify-between gap-3 pt-4 border-t border-border mt-4 text-xs text-muted-foreground">
                      <div>
                        Showing <span className="font-medium text-foreground">{(historyPage - 1) * historyPageSize + 1}</span> to{' '}
                        <span className="font-medium text-foreground">{Math.min(historyPage * historyPageSize, totalRuns)}</span> of{' '}
                        <span className="font-medium text-foreground">{totalRuns}</span> runs
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="mr-2">
                          Page {historyPage} of {totalHistoryPages}
                        </span>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setHistoryPage((p) => Math.max(1, p - 1))}
                          disabled={historyPage <= 1}
                          className="h-8 px-2.5 gap-1"
                        >
                          <ChevronLeft className="h-3.5 w-3.5" />
                          <span>Previous</span>
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setHistoryPage((p) => Math.min(totalHistoryPages, p + 1))}
                          disabled={historyPage >= totalHistoryPages}
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
        </TabsContent>

        {/* Tab 2: Storage Backups (List from storage destinations) */}
        <TabsContent value="storage">
          <Card>
            <CardHeader className="pb-3">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div>
                  <CardTitle className="text-base">Storage Destination Backups</CardTitle>
                  <CardDescription>
                    Physical backup files stored across destinations for {id}. You can restore any backup directly to this database.
                  </CardDescription>
                </div>

                <div className="flex items-center gap-2">
                  {dbInfo?.destinations && dbInfo.destinations.length > 1 && (
                    <div className="flex items-center gap-1.5 text-xs">
                      <Filter className="h-3.5 w-3.5 text-muted-foreground" />
                      <Select
                        value={destinationFilter}
                        onValueChange={(val) => {
                          if (val) setDestinationFilter(val);
                        }}
                      >
                        <SelectTrigger className="w-[160px] h-8 text-xs">
                          <SelectValue placeholder="All Destinations" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="all">All Destinations</SelectItem>
                          {dbInfo.destinations.map((d) => (
                            <SelectItem key={d} value={d}>
                              {d}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}

                  <Button 
                    variant="outline" 
                    size="sm" 
                    onClick={() => refetchStorageBackups()}
                    className="gap-1.5"
                  >
                    <RefreshCw className="h-3.5 w-3.5" />
                    <span>Refresh</span>
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              {loadingBackups ? (
                <div className="py-8 text-center text-muted-foreground">Scanning storage destinations...</div>
              ) : filteredStorageBackups.length === 0 ? (
                <div className="py-12 text-center text-muted-foreground space-y-2">
                  <div>No backup files found in configured destinations.</div>
                  <Button 
                    size="sm" 
                    variant="outline" 
                    onClick={() => triggerBackup.mutate()}
                    disabled={triggerBackup.isPending}
                  >
                    Create First Backup
                  </Button>
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Destination</TableHead>
                      <TableHead>Filename & Path</TableHead>
                      <TableHead>Snapshot Size</TableHead>
                      <TableHead>Created / Modified</TableHead>
                      <TableHead>Security</TableHead>
                      <TableHead className="text-right">Action</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredStorageBackups.map((backup: StorageBackupItem) => (
                      <TableRow key={`${backup.destination}-${backup.path}`}>
                        <TableCell>
                          <Badge variant="secondary" className="font-mono text-xs">
                            {backup.destination}
                          </Badge>
                          <div className="text-[10px] text-muted-foreground capitalize mt-0.5">
                            {backup.destination_type}
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="font-mono font-medium text-xs text-foreground">
                            {backup.filename}
                          </div>
                          <div className="font-mono text-[11px] text-muted-foreground truncate max-w-md">
                            {backup.path}
                          </div>
                        </TableCell>
                        <TableCell className="text-xs font-mono">
                          {formatBytes(backup.size_bytes)}
                        </TableCell>
                        <TableCell className="text-xs">
                          <div>{formatDate(backup.mod_time)}</div>
                          <div className="text-muted-foreground text-[11px]">{formatTimeAgo(backup.mod_time)}</div>
                        </TableCell>
                        <TableCell>
                          {backup.encrypted ? (
                            <Badge variant="outline" className="text-emerald-600 dark:text-emerald-400 border-emerald-500/30 gap-1 text-[11px]">
                              <ShieldCheck className="h-3 w-3" />
                              <span>AES-256</span>
                            </Badge>
                          ) : (
                            <span className="text-xs text-muted-foreground">Standard</span>
                          )}
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => setSelectedBackupForRestore(backup)}
                            className="gap-1.5 hover:bg-destructive/10 hover:text-destructive hover:border-destructive/30"
                          >
                            <RotateCcw className="h-3.5 w-3.5" />
                            <span>Restore</span>
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

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
          <ScrollArea className="flex-1 max-h-[60vh] bg-zinc-950 p-4 rounded-md font-mono text-xs whitespace-pre-wrap leading-relaxed border border-zinc-800 min-h-[200px]">
            {loadingLogs ? (
              <div className="flex items-center justify-center py-12 text-zinc-400 gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                <span>Loading execution logs...</span>
              </div>
            ) : (
              <div className="text-zinc-100 font-mono text-xs whitespace-pre-wrap leading-relaxed">
                {logsData || 'No log output captured.'}
              </div>
            )}
          </ScrollArea>
        </DialogContent>
      </Dialog>

      {/* Restore Dialog */}
      <RestoreDialog
        open={!!selectedBackupForRestore}
        onOpenChange={(open) => !open && setSelectedBackupForRestore(null)}
        databaseName={id}
        backup={selectedBackupForRestore}
        onSuccess={() => {
          qc.invalidateQueries({ queryKey: ['history', id] });
        }}
      />
    </div>
  );
}
