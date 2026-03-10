import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import type { Metadata } from "next"
import Link from "next/link"
import { SignOutButton } from "./_components/client"

export const metadata: Metadata = {
  title: "Sign Out",
}

export default function SignOutPage() {
  return (
    <div className="flex min-h-svh flex-col items-center justify-center p-6 md:p-10">
      <div className="w-full max-w-sm">
        <Card className="p-0">
          <CardContent className="p-6 md:p-8">
            <div className="flex flex-col items-center gap-6 text-center">
              <div className="flex flex-col gap-2">
                <h1 className="text-2xl font-bold">Sign Out</h1>
                <p className="text-muted-foreground text-balance">
                  Are you sure you want to sign out?
                </p>
              </div>
              <div className="flex w-full flex-col gap-2">
                <SignOutButton />
                <Button variant="ghost" asChild>
                  <Link href="/">Cancel</Link>
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
