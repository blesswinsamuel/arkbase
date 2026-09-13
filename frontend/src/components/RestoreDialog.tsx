import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { client, type StorageBackupItem, type RestoreBackupResponse } from '@/api/client';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { AlertTriangle, CheckCircle2, XCircle, RotateCcw, Terminal, RefreshCw, ShieldCheck } from 'lucide-react';
import { formatBytes, formatDate } from '@/lib/formatters';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  databaseName: string;
  backup: StorageBackupItem | null;
  onSuccess?: () => void;
}

export function RestoreDialog({ open, onOpenChange, databaseName, backup, onSuccess }: Props) {
  const qc = useQueryClient();
  const [restoreResult, setRestoreResult] = useState<RestoreBackupResponse | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const restoreMutation = useMutation({
    mutationFn: async () => {
      if (!backup) throw new Error('No backup selected');
      setErrorMessage(null);
      setRestoreResult(null);

      const res = await client.POST('/api/v1/databases/{id}/restore', {
        params: { path: { id: databaseName } },
        body: {
          destination: backup.destination,
          backup_path: backup.path,
        },
      });

      if (res.error) {
        const detail = (res.error as any)?.detail || (res.error as any)?.title || 'Failed to restore database';
        throw new Error(detail);
      }

      return res.data;
    },
    onSuccess: (data) => {
      setRestoreResult(data);
      qc.invalidateQueries({ queryKey: ['history'] });
      qc.invalidateQueries({ queryKey: ['databases'] });
      qc.invalidateQueries({ queryKey: ['stats'] });
      onSuccess?.();
    },
    onError: (err: any) => {
      setErrorMessage(err.message || 'An error occurred during restore.');
    },
  });

  const handleClose = () => {
    if (restoreMutation.isPending) return;
    setRestoreResult(null);
    setErrorMessage(null);
    onOpenChange(false);
  };

  if (!backup) return null;

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-2xl sm:max-w-2xl max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <RotateCcw className="h-5 w-5 text-primary" />
            <span>Restore Database: {databaseName}</span>
          </DialogTitle>
          <DialogDescription>
            Restore this database from a stored snapshot.
          </DialogDescription>
        </DialogHeader>

        {/* Content View */}
        <div className="space-y-4 py-2 flex-1 overflow-y-auto">
          {/* Metadata Card */}
          <div className="bg-muted/40 p-3.5 rounded-lg border text-xs space-y-2">
            <div className="grid grid-cols-2 gap-2">
              <div>
                <span className="text-muted-foreground">Target Database:</span>
                <div className="font-semibold text-foreground text-sm mt-0.5">{databaseName}</div>
              </div>
              <div>
                <span className="text-muted-foreground">Destination:</span>
                <div className="flex items-center gap-1.5 mt-0.5">
                  <Badge variant="secondary" className="font-mono text-[11px]">{backup.destination}</Badge>
                  <span className="text-muted-foreground capitalize">({backup.destination_type})</span>
                </div>
              </div>
            </div>

            <div className="border-t pt-2 grid grid-cols-2 gap-2">
              <div>
                <span className="text-muted-foreground">File Size:</span>
                <div className="font-mono font-medium text-foreground">{formatBytes(backup.size_bytes)}</div>
              </div>
              <div>
                <span className="text-muted-foreground">Snapshot Time:</span>
                <div className="text-foreground">{formatDate(backup.mod_time)}</div>
              </div>
            </div>

            <div className="border-t pt-2">
              <span className="text-muted-foreground">Source Path:</span>
              <div className="font-mono text-[11px] text-muted-foreground break-all mt-0.5 bg-background p-1.5 rounded border">
                {backup.path}
              </div>
            </div>

            {backup.encrypted && (
              <div className="flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400 text-[11px] pt-1">
                <ShieldCheck className="h-3.5 w-3.5" />
                <span>AES-256-GCM encrypted snapshot (will be decrypted automatically)</span>
              </div>
            )}
          </div>

          {/* Result view if finished */}
          {restoreResult && (
            <div className="space-y-3">
              <div className="flex items-center gap-2 p-3 bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 rounded-lg text-xs font-medium">
                <CheckCircle2 className="h-4 w-4 shrink-0" />
                <div className="flex-1">
                  <span>{restoreResult.message}</span>
                  <span className="ml-2 font-mono text-[11px] text-muted-foreground">
                    ({restoreResult.duration_ms} ms)
                  </span>
                </div>
              </div>

              {restoreResult.logs && (
                <div className="space-y-1">
                  <div className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                    <Terminal className="h-3.5 w-3.5" />
                    <span>Restore Execution Output</span>
                  </div>
                  <pre className="p-3 bg-zinc-950 text-zinc-100 rounded-md font-mono text-[11px] max-h-48 overflow-auto border border-zinc-800 whitespace-pre-wrap leading-relaxed">
                    {restoreResult.logs}
                  </pre>
                </div>
              )}
            </div>
          )}

          {/* Error view */}
          {errorMessage && (
            <div className="flex items-start gap-2 p-3 bg-red-500/10 border border-red-500/20 text-red-600 dark:text-red-400 rounded-lg text-xs">
              <XCircle className="h-4 w-4 shrink-0 mt-0.5" />
              <div className="space-y-1">
                <div className="font-semibold">Restore Operation Failed</div>
                <div className="text-muted-foreground font-mono text-[11px] whitespace-pre-wrap">{errorMessage}</div>
              </div>
            </div>
          )}

          {/* Caution Alert prior to restoring */}
          {!restoreResult && (
            <div className="flex items-start gap-2.5 p-3.5 bg-amber-500/10 border border-amber-500/25 rounded-lg text-xs text-amber-700 dark:text-amber-300">
              <AlertTriangle className="h-4 w-4 shrink-0 text-amber-500 mt-0.5" />
              <div>
                <span className="font-semibold">Caution: </span>
                Restoring will clean, drop, and overwrite existing schema and data in database{' '}
                <strong className="font-semibold underline">{databaseName}</strong>. Make sure you want to revert to this snapshot before continuing.
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          {restoreResult ? (
            <Button onClick={handleClose}>Close</Button>
          ) : (
            <>
              <Button
                variant="outline"
                onClick={handleClose}
                disabled={restoreMutation.isPending}
              >
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={() => restoreMutation.mutate()}
                disabled={restoreMutation.isPending}
                className="gap-2"
              >
                {restoreMutation.isPending ? (
                  <>
                    <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                    <span>Restoring Database...</span>
                  </>
                ) : (
                  <>
                    <RotateCcw className="h-3.5 w-3.5" />
                    <span>Confirm & Restore</span>
                  </>
                )}
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
