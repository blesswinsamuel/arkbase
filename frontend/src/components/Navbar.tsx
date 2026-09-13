import { NavLink } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import { client } from '@/api/client';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { 
  FolderLock, 
  Database, 
  Clock, 
  HardDrive, 
  ExternalLink 
} from 'lucide-react';
import { ModeToggle } from './mode-toggle';

export function Navbar() {
  const { data: status } = useQuery({
    queryKey: ['status'],
    queryFn: async () => {
      const res = await client.GET('/api/v1/status');
      return res.data;
    },
  });

  return (
    <header className="border-b bg-card/50 backdrop-blur sticky top-0 z-40">
      <div className="container max-w-7xl mx-auto flex h-16 items-center justify-between px-4">
        {/* Left: Brand & NavLinks */}
        <div className="flex items-center gap-6">
          <NavLink to="/databases" className="flex items-center gap-3">
            <div className="h-9 w-9 rounded-lg bg-primary/10 flex items-center justify-center text-primary border border-primary/20">
              <FolderLock className="h-5 w-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-lg tracking-tight">arkbase</span>
                <Badge variant="outline" className="text-xs font-mono font-normal">v{status?.version || '1.0.0'}</Badge>
              </div>
              <p className="text-[11px] text-muted-foreground">GitOps database backup daemon</p>
            </div>
          </NavLink>

          {/* Navigation Links with Active Indicator */}
          <nav className="hidden md:flex items-center gap-1 pl-4 border-l">
            <NavLink
              to="/databases"
              className={({ isActive }) =>
                `flex items-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-primary/10 text-primary font-semibold'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
                }`
              }
            >
              <Database className="h-4 w-4" />
              <span>Databases</span>
            </NavLink>

            <NavLink
              to="/history"
              className={({ isActive }) =>
                `flex items-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-primary/10 text-primary font-semibold'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
                }`
              }
            >
              <Clock className="h-4 w-4" />
              <span>Backup History</span>
            </NavLink>

            <NavLink
              to="/destinations"
              className={({ isActive }) =>
                `flex items-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-primary/10 text-primary font-semibold'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
                }`
              }
            >
              <HardDrive className="h-4 w-4" />
              <span>Storage Destinations</span>
            </NavLink>
          </nav>
        </div>

        {/* Right: Status & Actions */}
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
  );
}
