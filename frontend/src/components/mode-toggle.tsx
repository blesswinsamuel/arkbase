import { Moon, Sun, Monitor, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useTheme } from "@/hooks/use-theme";

export function ModeToggle() {
  const { theme, setTheme, resolvedTheme } = useTheme();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            className="h-8 w-8 px-0 cursor-pointer text-muted-foreground hover:text-foreground"
            aria-label="Toggle theme"
          />
        }
      >
        {resolvedTheme === "dark" ? (
          <Moon className="h-4 w-4 transition-all" />
        ) : (
          <Sun className="h-4 w-4 transition-all" />
        )}
        <span className="sr-only">Toggle theme</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-36">
        <DropdownMenuItem
          onClick={() => setTheme("light")}
          className="cursor-pointer flex items-center justify-between"
        >
          <span className="flex items-center gap-2">
            <Sun className="h-4 w-4 text-muted-foreground" />
            <span>Light</span>
          </span>
          {theme === "light" && <Check className="h-3.5 w-3.5 text-primary" />}
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => setTheme("dark")}
          className="cursor-pointer flex items-center justify-between"
        >
          <span className="flex items-center gap-2">
            <Moon className="h-4 w-4 text-muted-foreground" />
            <span>Dark</span>
          </span>
          {theme === "dark" && <Check className="h-3.5 w-3.5 text-primary" />}
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => setTheme("system")}
          className="cursor-pointer flex items-center justify-between"
        >
          <span className="flex items-center gap-2">
            <Monitor className="h-4 w-4 text-muted-foreground" />
            <span>System</span>
          </span>
          {theme === "system" && <Check className="h-3.5 w-3.5 text-primary" />}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
