import { Link } from 'react-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { client, type DatabaseInfo } from '@/api/client';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { 
  Database, 
  HardDrive, 
  Clock, 
  CheckCircle2, 
  XCircle, 
  Play, 
  RefreshCw,
  ChevronRight
} from 'lucide-react';
import { formatBytes, formatDate } from '@/lib/formatters';

export function DatabasesPage() {
  const qc = useQueryClient();

  const { data: stats } = useQuery({
    queryKey: ['stats'],
    queryFn: async () => {
      const res = await client.GET('/api/v1/stats');
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

  return (
    <div className="space-y-8">
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

      {/* Databases Table */}
      <Card>
        <CardHeader>
          <CardTitle>Configured Databases</CardTitle>
          <CardDescription>
            Databases defined in <code>config.yaml</code>. Click on any database to view backup history, inspect storage backups, or restore.
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
                  <TableRow key={db.name} className="group">
                    <TableCell>
                      <Link 
                        to={`/databases/${db.name}`}
                        className="font-semibold text-primary hover:underline flex items-center gap-1.5"
                      >
                        <span>{db.name}</span>
                        <ChevronRight className="h-3.5 w-3.5 opacity-0 group-hover:opacity-100 transition-opacity" />
                      </Link>
                    </TableCell>
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
    </div>
  );
}
