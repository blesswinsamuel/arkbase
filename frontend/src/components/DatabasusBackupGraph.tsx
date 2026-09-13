import { useState } from 'react';
import type { RunDetail } from '@/api/client';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { CheckCircle2, XCircle, Clock, Activity, Zap, RefreshCw, Terminal } from 'lucide-react';
import { formatBytes, formatDate, formatTimeAgo } from '@/lib/formatters';

interface Props {
  runs: RunDetail[];
  onSelectRun: (run: RunDetail) => void;
}

export function DatabasusBackupGraph({ runs, onSelectRun }: Props) {
  const [filterPeriod, setFilterPeriod] = useState<'30' | '50' | 'all'>('50');

  const displayedRuns = filterPeriod === 'all' 
    ? [...runs].reverse() 
    : [...runs].slice(0, parseInt(filterPeriod, 10)).reverse();

  const totalRuns = displayedRuns.length;
  const successCount = displayedRuns.filter(r => r.status === 'success').length;
  const failedCount = displayedRuns.filter(r => r.status === 'failed').length;
  const successRate = totalRuns > 0 ? Math.round((successCount / totalRuns) * 100) : 100;
  
  const avgDuration = totalRuns > 0
    ? Math.round(displayedRuns.reduce((acc, r) => acc + (r.duration_ms || 0), 0) / totalRuns)
    : 0;

  return (
    <Card className="border bg-card/60 backdrop-blur">
      <CardHeader className="pb-3">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <Activity className="h-4 w-4 text-primary" />
            <CardTitle className="text-base font-semibold">Backup Health & Execution Timeline</CardTitle>
            <Badge variant="outline" className="text-xs font-mono">
              {successRate}% health
            </Badge>
          </div>

          <div className="flex items-center gap-2">
            <div className="flex items-center bg-muted/60 p-0.5 rounded-md border text-xs">
              {(['30', '50', 'all'] as const).map(p => (
                <button
                  key={p}
                  type="button"
                  onClick={() => setFilterPeriod(p)}
                  className={`px-2.5 py-1 rounded-[4px] font-medium transition-colors cursor-pointer ${
                    filterPeriod === p
                      ? 'bg-background text-foreground shadow-xs font-semibold'
                      : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {p === 'all' ? 'All' : `Last ${p}`}
                </button>
              ))}
            </div>
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Quick summary metric row */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs bg-muted/30 p-3 rounded-lg border border-border/50">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-emerald-500 shrink-0" />
            <div>
              <div className="text-muted-foreground">Successful</div>
              <div className="font-semibold text-foreground">{successCount} runs</div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <XCircle className="h-4 w-4 text-red-500 shrink-0" />
            <div>
              <div className="text-muted-foreground">Failed</div>
              <div className="font-semibold text-foreground">{failedCount} runs</div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Clock className="h-4 w-4 text-muted-foreground shrink-0" />
            <div>
              <div className="text-muted-foreground">Avg Duration</div>
              <div className="font-semibold font-mono text-foreground">{avgDuration} ms</div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Zap className="h-4 w-4 text-amber-500 shrink-0" />
            <div>
              <div className="text-muted-foreground">Health Rate</div>
              <div className="font-semibold text-foreground">{successRate}%</div>
            </div>
          </div>
        </div>

        {/* Databasus Block Graph Grid */}
        <TooltipProvider delay={50}>
          <div className="space-y-2">
            <div className="text-xs text-muted-foreground flex items-center justify-between">
              <span>Oldest</span>
              <span>Latest</span>
            </div>

            {displayedRuns.length === 0 ? (
              <div className="py-8 text-center text-xs text-muted-foreground bg-muted/20 rounded-md border border-dashed">
                No backup executions recorded yet.
              </div>
            ) : (
              <div className="flex flex-wrap gap-1.5 p-3 bg-zinc-950/5 dark:bg-zinc-950/40 rounded-lg border border-border/40 min-h-[60px] items-center">
                {displayedRuns.map((run) => {
                  const isSuccess = run.status === 'success';
                  const isRunning = run.status === 'running';

                  return (
                    <Tooltip key={run.id}>
                      <TooltipTrigger
                        onClick={() => onSelectRun(run)}
                        className={`h-4 w-4 sm:h-5 sm:w-5 rounded-[4px] transition-transform hover:scale-125 cursor-pointer outline-hidden focus-visible:ring-2 focus-visible:ring-primary ${
                          isRunning
                            ? 'bg-sky-500 animate-pulse ring-1 ring-sky-400'
                            : isSuccess
                            ? 'bg-emerald-500 hover:bg-emerald-400'
                            : 'bg-red-500 hover:bg-red-400'
                        }`}
                      />
                      <TooltipContent 
                        side="top" 
                        sideOffset={6}
                        className="w-64 p-3 space-y-2 rounded-lg border shadow-xl bg-popover text-popover-foreground text-xs"
                      >
                        {/* Header: Status & Relative Time */}
                        <div className="flex items-center justify-between pb-1.5 border-b border-border/60">
                          <div className="flex items-center gap-1.5 font-semibold">
                            {isSuccess ? (
                              <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 shrink-0" />
                            ) : isRunning ? (
                              <RefreshCw className="h-3.5 w-3.5 text-sky-500 animate-spin shrink-0" />
                            ) : (
                              <XCircle className="h-3.5 w-3.5 text-red-500 shrink-0" />
                            )}
                            <span className="capitalize">{run.status}</span>
                          </div>
                          <span className="text-[11px] text-muted-foreground font-mono">
                            {formatTimeAgo(run.started_at)}
                          </span>
                        </div>

                        {/* Timestamp & Metrics */}
                        <div className="space-y-1">
                          <div className="text-[11px] text-muted-foreground">
                            {formatDate(run.started_at)}
                          </div>
                          <div className="flex items-center justify-between pt-0.5 text-xs font-mono">
                            <span className="text-muted-foreground">Duration:</span>
                            <span className="font-semibold text-foreground">{run.duration_ms} ms</span>
                          </div>
                          <div className="flex items-center justify-between text-xs font-mono">
                            <span className="text-muted-foreground">Output Size:</span>
                            <span className="font-semibold text-foreground">{formatBytes(run.size_bytes)}</span>
                          </div>
                        </div>

                        {/* Destinations */}
                        {run.destinations && run.destinations.length > 0 && (
                          <div className="pt-1.5 border-t border-border/50 text-[11px] flex items-center justify-between">
                            <span className="text-muted-foreground">Targets:</span>
                            <span className="font-mono text-foreground font-medium truncate max-w-[150px]">
                              {run.destinations.join(', ')}
                            </span>
                          </div>
                        )}

                        {/* Call to action */}
                        <div className="pt-1 border-t border-border/50 text-[10px] text-primary font-medium flex items-center gap-1">
                          <Terminal className="h-3 w-3" />
                          <span>Click to inspect logs</span>
                        </div>
                      </TooltipContent>
                    </Tooltip>
                  );
                })}
              </div>
            )}
          </div>
        </TooltipProvider>

        {/* Legend */}
        <div className="flex items-center justify-between text-[11px] text-muted-foreground border-t pt-2">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-1.5">
              <span className="h-2.5 w-2.5 rounded-[2px] bg-emerald-500" />
              <span>Success</span>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="h-2.5 w-2.5 rounded-[2px] bg-red-500" />
              <span>Failed</span>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="h-2.5 w-2.5 rounded-[2px] bg-sky-500 animate-pulse" />
              <span>Running</span>
            </div>
          </div>
          <div>Click any block to inspect run output logs</div>
        </div>
      </CardContent>
    </Card>
  );
}
