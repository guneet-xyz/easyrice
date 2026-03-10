"use client"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { useSession } from "@/lib/client/auth"
import Link from "next/link"

export function SignOut({ userId, email }: { userId: string; email: string }) {
  const { data: session } = useSession()

  if (!session || session.user.id !== userId) return null

  return (
    <>
      <Separator />
      <div className="flex flex-col gap-1">
        <span className="text-muted-foreground text-sm">Email</span>
        <span className="text-sm font-medium">{email}</span>
      </div>
      <Button variant="ghost" asChild>
        <Link href="/signout">Sign Out</Link>
      </Button>
    </>
  )
}
