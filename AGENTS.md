# Agent Instructions

## Repository Structure

This is an npm workspaces monorepo. The website lives in `apps/website/`.

```
├── apps/
│   └── website/          # Next.js app (Cloudflare Workers via OpenNext)
├── compose.yml           # Docker Compose for local dev services
├── prettier.config.ts    # Shared Prettier config (root-level)
├── AGENTS.md
└── README.md
```

## Build, Lint & Format Commands

```bash
npm run dev -w @easyrice/website     # Start Next.js dev server
npm run build -w @easyrice/website   # Production build (Next.js + OpenNext for Cloudflare)
npm run format:write                 # Format all files with Prettier (root)
npm run format:check                 # Check formatting (root)
npm run lint -w @easyrice/website    # ESLint
```

There is no test runner configured in this project.

After every file change, run these commands and fix any issues before continuing:

```bash
npm run format:write && npm run lint -w @easyrice/website
```

## Code Style

### No Semicolons

Prettier is configured with `semi: false`. Never add semicolons.

### Imports

Import ordering is enforced automatically by `@ianvs/prettier-plugin-sort-imports` —
running `format:write` handles it. Key rules:

- Use the `@/*` path alias for anything under `src/`. Never use deep relative paths like `../../lib/utils`.
- Use relative imports (`./`, `../`) only for sibling or child files within the same feature folder.
- Use `import type { X }` for type-only imports. When mixing values and types from the same module, use the inline form: `import { value, type Type } from "..."`.
- Access env vars through `@/lib/env`, never via `process.env` directly in app code.

### Types

- Use `type`, not `interface`, for all type definitions.
- Never use `React.FC`. Use function declarations with typed parameters.
- Props can be typed inline for simple cases, or via a `type Props = { ... }` alias for complex ones:

```tsx
// Inline — for 1-2 props
export function ProfileCard({ user }: { user: User }) { ... }

// Separate type — for complex props
type UserSettingsProps = {
  name: string
  username: string
  image: string | null
}
export function UserSettings({ name, username, image }: UserSettingsProps) { ... }
```

- For wrapper components, extend with `React.ComponentProps<"element">`:

```tsx
function Input({ className, type, ...props }: React.ComponentProps<"input">) { ... }
```

### Naming Conventions

| Item                | Convention        | Example                        |
| ------------------- | ----------------- | ------------------------------ |
| Files & folders     | kebab-case        | `user-avatar.tsx`, `auth.ts`   |
| Components          | PascalCase        | `UserSettings`, `ProfileCard`  |
| Functions & vars    | camelCase         | `changeName`, `authedAction`   |
| Types               | PascalCase        | `ActionResult`, `ProviderId`   |
| Server actions      | camelCase verb    | `changeName`, `changeUsername` |
| DB tables (Drizzle) | singular          | `user`, `session`, `account`   |
| DB columns          | snake_case in SQL | `created_at`, `provider_id`    |
| Route-local folders | `_components/`    | `src/app/settings/_components` |

### Components

- Always use `function` declarations, never arrow-function components.
- Always destructure props in the function signature.
- Page components use `export default function`.
- Reusable components use named exports: `export function X() { ... }`.
- shadcn/ui components declare with `function X() { ... }` and group exports at the bottom: `export { X, Y, Z }`.

### Conditional Classes with `cn()`

Use the object syntax for conditional classes — never the `&&` short-circuit pattern.

```tsx
// ✅ Correct — object syntax
cn("base-class", {
  "text-white": isActive,
  "text-gray-500": !isActive,
})

// ❌ Incorrect — short-circuit
cn("base-class", isActive && "text-white", !isActive && "text-gray-500")
```

### Comments

Write clean, self-documenting code. Do not add comments that restate what the code
already expresses. Use comments only to provide context that cannot be inferred from
the code itself — explain _why_, not _what_.

```tsx
// ❌ Bad — restates the code
// Check if the user is authenticated
if (session.user) { ... }

// ✅ Good — adds context not present in the code
// The provider returns null during the prefetch phase on Cloudflare Workers
if (!ctx.provider) return null
```

## Architecture

### Server vs Client Components

- Server components are the default — no directive needed.
- Client components have `"use client"` on the very first line.
- Extract interactive logic into dedicated `client.tsx` files or `_components/` folders.
  The server component handles data and layout; the client component handles interactivity.

### Server Actions

All actions live in `src/lib/server/actions/`. Every action file starts with `"use server"`.

- Wrap with `authedAction()` from the barrel `index.ts` for auth-guarded mutations.
- Return `ActionResult` (`{ success: boolean; message: string }`) — never throw.
- Validate inline with early returns; call `revalidatePath()` after mutations.

```tsx
export const changeUsername = authedAction(
  async (session, username: string) => {
    const trimmed = username.trim()
    if (trimmed.length < 3) {
      return {
        success: false,
        message: "Username must be at least 3 characters.",
      }
    }
    await db
      .update(user)
      .set({ username: trimmed })
      .where(eq(user.id, session.user.id))
    revalidatePath("/settings")
    return { success: true, message: "Username updated successfully." }
  },
)
```

### Error Handling

- Server actions: return `{ success: false, message }` — never throw.
- Client components: display results with `toast.success()` / `toast.error()` from `sonner`.
- Auth guards on pages: `if (!session) redirect("/signin")`.

### Data Fetching

- Server components fetch data directly with Drizzle queries — no API layer.
- Auth state on the server: `await auth.api.getSession({ headers: await headers() })`.
- Auth state on the client: `useSession()` hook from `@/lib/client/auth`.
- Client mutations: call server actions directly, wrap in `useTransition` for pending state.

### Database (Drizzle ORM)

- Uses `pg-proxy` driver (HTTP proxy for Cloudflare Workers compatibility).
- Schema in `src/lib/db/schema.ts` — singular table names, `easyrice_*` prefix filter.
- Use the SQL-like select API (`db.select().from().where()`), not the relational query builder.

### shadcn/ui

Add components via the shadcn CLI. They live in `src/components/ui/`. The project uses
the `new-york` style, `zinc` base color, and Radix UI v2 imports (`from "radix-ui"`).
Customizations are made directly in the component files.
