# Agent Guidelines

## Frontend & shadcn/ui Standards

### 1. Never Hand-Edit shadcn Components
- **NEVER manually edit files in `frontend/src/components/ui/*` under any circumstances.**
- Files in `src/components/ui/` are official shadcn components generated and maintained by the shadcn CLI.
- Do not modify them to change sizing, typography, spacing, padding, borders, or colors.
- If styling adjustments are needed:
  - Pass Tailwind classes through the `className` prop at the call site (which merges cleanly using `cn(...)`).
  - Adjust global theme CSS variables in `frontend/src/index.css`.
  - Create a custom wrapper component outside `src/components/ui/` (e.g., in `src/components/`) if advanced composition or specific defaults are required.

### 2. Use shadcn Components As Much As Possible
- Always prefer shadcn primitives over raw HTML elements or bespoke Tailwind containers:
  - Use `Button` instead of `<button>`.
  - Use `Badge` instead of custom tag pills.
  - Use `Select` (`SelectTrigger`, `SelectValue`, `SelectContent`, `SelectItem`) instead of `<select>`.
  - Use `Alert`, `AlertTitle`, `AlertDescription` instead of ad-hoc alert/banner boxes with manual border/background colors.
  - Use `Separator` instead of `<hr>` or manual `border-t`/`border-b` divider divs.
  - Use `ScrollArea` instead of raw `overflow-auto` containers for logs and lists.
  - Use `Tabs`, `TabsList`, `TabsTrigger`, `TabsContent` instead of manual button group toggles.
  - Use `Table`, `TableHeader`, `TableBody`, `TableRow`, `TableHead`, `TableCell` for tabular data.
  - Use `Dialog`, `DialogContent`, `DialogHeader`, `DialogTitle`, `DialogDescription` for modals.
  - Use `Tooltip`, `TooltipTrigger`, `TooltipContent`, `TooltipProvider` for tooltips.

### 3. Adding and Updating Components via CLI
- When adding a component:
  ```bash
  pnpm --prefix frontend dlx shadcn add <component-name>
  ```
- When updating or resetting components to upstream versions:
  ```bash
  pnpm --prefix frontend dlx shadcn add -y --overwrite <component-name>
  ```
- Check component differences against the registry:
  ```bash
  pnpm --prefix frontend dlx shadcn diff
  ```

### 4. Configuration & Path Aliases
- `frontend/components.json` is configured with style `base-mira` and Tailwind CSS.
- `frontend/tsconfig.json` maintains the `@/* -> ./src/*` path mapping to ensure the shadcn CLI resolves imports and paths correctly. Do not remove this mapping.

## Code Quality & Git
- Fix root causes, not symptoms.
- Follow existing project conventions across Go and React / TypeScript.
- Always run `pnpm run build && pnpm run lint` in `frontend` before completing tasks.
- Use conventional commits (`feat:`, `fix:`, `chore:`, `refactor:`, `docs:`, `test:`).
