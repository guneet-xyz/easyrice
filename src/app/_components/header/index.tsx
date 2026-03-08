"use client"

import { Button } from "@/components/ui/button"
import { useSession } from "@/lib/client/auth"
import { Settings } from "lucide-react"
import dynamic from "next/dynamic"
import Link from "next/link"
import { SearchCommand } from "./search"

const ThemeToggle = dynamic(
  () => import("./theme-toggle").then((mod) => mod.ThemeToggle),
  { ssr: false },
)

export function Header() {
  const { data: session, isPending } = useSession()

  return (
    <div className="flex p-4 justify-between h-16">
      <div className="flex items-center gap-4">
        <Link href="/" className="font-semibold">
          easyrice
        </Link>
        <nav className="flex items-center gap-1">
          <Button variant="ghost" size="sm" asChild>
            <Link href="/docs">Docs</Link>
          </Button>
          <Button variant="ghost" size="sm" asChild>
            <Link href="/wiki">Wiki</Link>
          </Button>
        </nav>
      </div>
      <div className="flex items-center gap-2">
        <SearchCommand />
        {isPending ? null : session ? (
          <>
            <Button variant="ghost" asChild>
              <Link href={`/users/${session.user.username}`}>
                {session.user.username}
              </Link>
            </Button>
            <Button variant="ghost" size="icon-sm" asChild>
              <Link href="/settings">
                <Settings className="size-4" />
                <span className="sr-only">Settings</span>
              </Link>
            </Button>
          </>
        ) : (
          <Button variant="ghost" asChild>
            <Link href="/signin">Sign In</Link>
          </Button>
        )}
        <ThemeToggle />
      </div>
    </div>
  )
}
