"use client"

import { Button } from "@/components/ui/button"
import { signOut } from "@/lib/client/auth"
import { useRouter } from "next/navigation"

export function SignOutButton() {
  const router = useRouter()

  return (
    <Button
      onClick={async () => {
        await signOut()
        router.push("/")
      }}
    >
      Sign Out
    </Button>
  )
}
