import { useState } from 'react';
import { useQuery, useMutation, useQueryClient, QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { client, type DatabaseInfo, type DestinationInfo, type RunDetail } from './api/client';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { 
  Database, 
  HardDrive, 
  Clock, 
  CheckCircle2, 
  XCircle, 
  Play, 
  ExternalLink, 
  ShieldCheck, 
  Terminal, 
  Layers,
  RefreshCw,
  FolderLock
} from 'lucide-react';
import { ModeToggle } from './components/mode-toggle';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchInterval: 5000, // Live poll every 5s
    },
  },
});

function Dashboard() {
  const qc = useQueryClient();
  const [selectedRun, setSelectedRun] = useState<RunDetail | null>(null);
  const [activeTab, setActiveTab] = useState('databases');

  const { data: stats } = useQuery({
    queryKey: ['stats'],
    queryFn: async () => {
      const res = await client.GET('/api/v1/stats');
      return res.data;
    },
  });

  const { data: status } = useQuery({
    queryKey: ['status'],
    queryFn: async () => {
      const res = await client.GET('/api/v1/status');
      return res.data;
    },
  });

  const { data: databases, isLoading: loadingDBs } = useQuery({
    queryKey: ['databases'],
    queryFn: async () => {
      const res = await client.GET('/api/v1/databases');
      return res.data || [];
    },
  });

  const { data: history, isLoading: loadingHistory } = useQuery({
    queryKey: ['history'],
    queryFn: async () => {
      const res = await client.GET('/api/v1/history', { params: { query: { limit: 50 } } });
      return res.data?.runs || [];
    },
  });

  const { data: destinations } = useQuery({
    queryKey: ['destinations'],
    queryFn: async () => {
      const res = await client.GET('/api/v1/destinations');
      return res.data || [];
    },
  });

  const triggerBackup = useMutation({
    mutationFn: async (dbName: string) => {
      await client.POST('/api/v1/databases/{id}/backup', { params: { path: { id: dbName } } });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['databases'] });
      qc.invalidateQueries({ queryKey: ['history'] });
      qc.invalidateQueries({ queryKey: ['stats'] });
    },
  });

  const successRate = stats?.total_backups 
    ? Math.round((stats.successful_count / stats.total_backups) * 100) 
    : 100;

  const formatBytes = (bytes?: number) => {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDate = (dateStr?: string | null) => {
    if (!dateStr) return 'Never';
    const d = new Date(dateStr);
    return d.toLocaleString();
  };

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      {/* Header */}
      <header className="border-b bg-card/50 backdrop-blur sticky top-0 z-40">
        <div className="container max-w-7xl mx-auto flex h-16 items-center justify-between px-4">
          <div className="flex items-center gap-3">
            <div className="h-9 w-9 rounded-lg bg-primary/10 flex items-center justify-center text-primary border border-primary/20">
              <FolderLock className="h-5 w-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-lg tracking-tight">arkbase</span>
                <Badge variant="outline" className="text-xs font-mono font-normal">v{status?.version || '1.0.0'}</Badge>
              </div>
              <p className="text-xs text-muted-foreground">GitOps database backup daemon</p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Badge variant="secondary" className="flex items-center gap-1.5 py-1 px-2.5">
              <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
              <span>Daemon Online</span>
            </Badge>
            <a href="/docs" target="_blank" rel="noreferrer">
              <Button variant="outline" size="sm" className="flex items-center gap-1.5 cursor-pointer">
                <span>API Docs</span>
                <ExternalLink className="h-3.5 w-3.5" />
              </Button>
            </a>
            <a href="https://github.com/blesswinsamuel/arkbase" target="_blank" rel="noreferrer">
              <Button variant="ghost" size="sm" className="flex items-center gap-1.5 cursor-pointer">
                <span>GitHub</span>
                <ExternalLink className="h-3.5 w-3.5" />
              </Button>
            </a>
            <ModeToggle />
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 container max-w-7xl mx-auto px-4 py-8 space-y-8">
        {/* Metric Cards */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Configured Databases</CardTitle>
              <Database className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{databases?.length || 0}</div>
              <p className="text-xs text-muted-foreground mt-1">PostgreSQL targets</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Total Backups</CardTitle>
              <CheckCircle2 className="h-4 w-4 text-emerald-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats?.total_backups || 0}</div>
              <p className="text-xs text-muted-foreground mt-1">{successRate}% success rate</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Storage Volume</CardTitle>
              <HardDrive className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{formatBytes(stats?.total_size_bytes)}</div>
              <p className="text-xs text-muted-foreground mt-1">{destinations?.length || 0} storage targets</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Last Backup</CardTitle>
              <Clock className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-sm font-semibold truncate mt-1">
                {stats?.last_backup_at ? formatDate(stats.last_backup_at) : 'No backups yet'}
              </div>
              <p className="text-xs text-muted-foreground mt-1">Automatic cron active</p>
            </CardContent>
          </Card>
        </div>

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
          <TabsList>
            <TabsTrigger value="databases" className="flex items-center gap-2">
              <Database className="h-4 w-4" />
              <span>Databases</span>
              <Badge variant="secondary" className="ml-1 text-xs">{databases?.length || 0}</Badge>
            </TabsTrigger>
            <TabsTrigger value="history" className="flex items-center gap-2">
              <Clock className="h-4 w-4" />
              <span>Backup History</span>
              <Badge variant="secondary" className="ml-1 text-xs">{history?.length || 0}</Badge>
            </TabsTrigger>
            <TabsTrigger value="destinations" className="flex items-center gap-2">
              <HardDrive className="h-4 w-4" />
              <span>Storage Destinations</span>
              <Badge variant="secondary" className="ml-1 text-xs">{destinations?.length || 0}</Badge>
            </TabsTrigger>
          </TabsList>

          {/* Databases Tab */}
          <TabsContent value="databases">
            <Card>
              <CardHeader>
                <CardTitle>Configured Databases</CardTitle>
                <CardDescription>
                  Databases defined in <code>config.yaml</code> with their scheduled backup jobs and targets.
                </CardDescription>
              </CardHeader>
              <CardContent>
                {loadingDBs ? (
                  <div className="py-8 text-center text-muted-foreground">Loading databases...</div>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Database</TableHead>
                        <TableHead>Engine</TableHead>
                        <TableHead>Host / Target</TableHead>
                        <TableHead>Schedule</TableHead>
                        <TableHead>Destinations</TableHead>
                        <TableHead>Last Run</TableHead>
                        <TableHead className="text-right">Action</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {databases?.map((db: DatabaseInfo) => (
                        <TableRow key={db.name}>
                          <TableCell className="font-semibold">{db.name}</TableCell>
                          <TableCell>
                            <Badge variant="outline" className="capitalize">
                              {db.engine}
                            </Badge>
                          </TableCell>
                          <TableCell className="font-mono text-xs text-muted-foreground">
                            {db.host}:{db.port} ({db.database})
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center gap-1.5 text-xs font-mono">
                              <span>{db.schedule || 'Manual'}</span>
                            </div>
                            {db.next_run_at && (
                              <div className="text-[11px] text-muted-foreground mt-0.5">
                                Next: {formatDate(db.next_run_at)}
                              </div>
                            )}
                          </TableCell>
                          <TableCell>
                            <div className="flex flex-wrap gap-1">
                              {db.destinations?.map((dst: string) => (
                                <Badge key={dst} variant="secondary" className="text-[11px]">
                                  {dst}
                                </Badge>
                              ))}
                            </div>
                          </TableCell>
                          <TableCell>
                            {db.last_run ? (
                              <div className="flex items-center gap-2">
                                {db.last_run.status === 'success' ? (
                                  <CheckCircle2 className="h-4 w-4 text-emerald-500 shrink-0" />
                                ) : (
                                  <XCircle className="h-4 w-4 text-red-500 shrink-0" />
                                )}
                                <div className="text-xs">
                                  <div>{formatDate(db.last_run.started_at)}</div>
                                  <div className="text-muted-foreground">{formatBytes(db.last_run.size_bytes)} • {db.last_run.duration_ms}ms</div>
                                </div>
                              </div>
                            ) : (
                              <span className="text-xs text-muted-foreground">Never</span>
                            )}
                          </TableCell>
                          <TableCell className="text-right">
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={db.is_running || triggerBackup.isPending}
                              onClick={() => triggerBackup.mutate(db.name)}
                              className="gap-1.5"
                            >
                              {db.is_running ? (
                                <>
                                  <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                                  <span>Running</span>
                                </>
                              ) : (
                                <>
                                  <Play className="h-3.5 w-3.5" />
                                  <span>Backup Now</span>
                                </>
                              )}
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

          {/* History Tab */}
          <TabsContent value="history">
            <Card>
              <CardHeader>
                <CardTitle>Execution History</CardTitle>
                <CardDescription>Historical backup runs, execution times, sizes, and raw output logs.</CardDescription>
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
                          <TableCell className="font-semibold">{run.database_name}</TableCell>
                          <TableCell className="text-xs">{formatDate(run.started_at)}</TableCell>
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
          </TabsContent>

          {/* Destinations Tab */}
          <TabsContent value="destinations">
            <div className="grid gap-4 md:grid-cols-2">
              {destinations?.map((dest: DestinationInfo) => (
                <Card key={dest.name}>
                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <CardTitle className="text-base flex items-center gap-2">
                        <HardDrive className="h-4 w-4 text-primary" />
                        <span>{dest.name}</span>
                      </CardTitle>
                      <Badge variant="outline" className="capitalize font-mono">{dest.type}</Badge>
                    </div>
                    <CardDescription className="font-mono text-xs">{dest.path}</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4 text-xs">
                    {dest.type === 's3' && (
                      <div className="space-y-1 bg-muted/50 p-2.5 rounded-md font-mono text-[11px]">
                        <div>Endpoint: {dest.endpoint || 'AWS S3 default'}</div>
                        <div>Bucket: {dest.bucket}</div>
                      </div>
                    )}

                    <div className="flex items-center gap-2">
                      <ShieldCheck className={`h-4 w-4 ${dest.encrypted ? 'text-emerald-500' : 'text-muted-foreground'}`} />
                      <span>{dest.encrypted ? 'AES-256-GCM Encryption Enabled' : 'No encryption'}</span>
                    </div>

                    {dest.retention && (
                      <div className="border-t pt-3 mt-3">
                        <div className="font-medium text-muted-foreground mb-1.5 flex items-center gap-1.5">
                          <Layers className="h-3.5 w-3.5" />
                          <span>Grandfather-Father-Son (GFS) Retention</span>
                        </div>
                        <div className="grid grid-cols-3 gap-2 font-mono text-[11px] text-muted-foreground">
                          {dest.retention.keep_last > 0 && <div>Keep Last: {dest.retention.keep_last}</div>}
                          {dest.retention.hourly > 0 && <div>Hourly: {dest.retention.hourly}h</div>}
                          {dest.retention.daily > 0 && <div>Daily: {dest.retention.daily}d</div>}
                          {dest.retention.weekly > 0 && <div>Weekly: {dest.retention.weekly}w</div>}
                          {dest.retention.monthly > 0 && <div>Monthly: {dest.retention.monthly}m</div>}
                          {dest.retention.yearly > 0 && <div>Yearly: {dest.retention.yearly}y</div>}
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
          </TabsContent>
        </Tabs>
      </main>

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

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Dashboard />
    </QueryClientProvider>
  );
}
