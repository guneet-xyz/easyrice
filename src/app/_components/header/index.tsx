"use client"

import { Button } from "@/components/ui/button"
import { useSession } from "@/lib/client/auth"
import dynamic from "next/dynamic"
import Link from "next/link"

const ThemeToggle = dynamic(() => import("./theme-toggle"), { ssr: false })

export default function Header() {
  const { data: session, isPending } = useSession()

  return (
    <div className="flex p-4 justify-between">
      <div>
        <div>easyrice</div>
      </div>
      <div className="flex items-center gap-2">
        {isPending ? null : session ? (
          <Button variant="ghost" asChild>
            <Link href="/profile">{session.user.username}</Link>
          </Button>
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
