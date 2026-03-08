"use client"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { useSession } from "@/lib/client/auth"
import Link from "next/link"

export function ProfileCard() {
  const { data: session, isPending } = useSession()

  if (isPending) {
    return null
  }

  if (!session) {
    return (
      <Card className="p-0">
        <CardContent className="p-6 md:p-8">
          <div className="flex flex-col items-center gap-6 text-center">
            <div className="flex flex-col gap-2">
              <h1 className="text-2xl font-bold">Not Signed In</h1>
              <p className="text-muted-foreground text-balance">
                You need to sign in to view your profile.
              </p>
            </div>
            <Button asChild className="w-full">
              <Link href="/signin">Sign In</Link>
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="p-0">
      <CardContent className="p-6 md:p-8">
        <div className="flex flex-col gap-6">
          <div className="flex flex-col items-center gap-2 text-center">
            <h1 className="text-2xl font-bold">Profile</h1>
          </div>
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <span className="text-muted-foreground text-sm">Username</span>
              <span className="text-sm font-medium">
                {session.user.username}
              </span>
            </div>
            <Separator />
            <div className="flex flex-col gap-1">
              <span className="text-muted-foreground text-sm">Name</span>
              <span className="text-sm font-medium">{session.user.name}</span>
            </div>
            <Separator />
            <div className="flex flex-col gap-1">
              <span className="text-muted-foreground text-sm">Email</span>
              <span className="text-sm font-medium">{session.user.email}</span>
            </div>
          </div>
          <Button variant="ghost" asChild>
            <Link href="/signout">Sign Out</Link>
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
