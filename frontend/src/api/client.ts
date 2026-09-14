import createClient from 'openapi-fetch';
import type { paths } from './schema';

export const client = createClient<paths>({ baseUrl: '/' });

export type DatabaseInfo = NonNullable<paths['/api/v1/databases']['get']['responses']['200']['content']['application/json']>[number];
export type DestinationInfo = NonNullable<paths['/api/v1/destinations']['get']['responses']['200']['content']['application/json']>[number];
export type HistoryResponse = paths['/api/v1/history']['get']['responses']['200']['content']['application/json'];
export type HistoryRun = NonNullable<HistoryResponse['runs']>[number];
export type RunDetail = paths['/api/v1/history/{id}']['get']['responses']['200']['content']['application/json'];
export type RunLogs = paths['/api/v1/history/{id}/logs']['get']['responses']['200']['content']['application/json'];
export type SummaryStats = paths['/api/v1/stats']['get']['responses']['200']['content']['application/json'];
export type SystemStatus = paths['/api/v1/status']['get']['responses']['200']['content']['application/json'];
export type StorageBackupItem = NonNullable<paths['/api/v1/databases/{id}/backups']['get']['responses']['200']['content']['application/json']['backups']>[number];
export type RestoreBackupResponse = paths['/api/v1/databases/{id}/restore']['post']['responses']['200']['content']['application/json'];
