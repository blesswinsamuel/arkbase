import { useQuery } from '@tanstack/react-query';
import { client, type DestinationInfo } from '@/api/client';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { HardDrive, ShieldCheck, Layers } from 'lucide-react';

export function DestinationsPage() {
  const { data: destinations = [], isLoading } = useQuery({
    queryKey: ['destinations'],
    queryFn: async () => {
      const res = await client.GET('/api/v1/destinations');
      return res.data || [];
    },
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold tracking-tight">Storage Destinations</h1>
        <p className="text-xs text-muted-foreground mt-1">
          Configured backup targets including local filesystems, object storage, encryption, and retention policies.
        </p>
      </div>

      {isLoading ? (
        <div className="py-8 text-center text-muted-foreground">Loading storage destinations...</div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {destinations.map((dest: DestinationInfo) => (
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
                  <>
                    <Separator className="my-3" />
                    <div>
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
                  </>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
