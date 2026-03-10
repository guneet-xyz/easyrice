"use client"

import { Button } from "@/components/ui/button"
import { signIn } from "@/lib/client/auth"
import { Check } from "lucide-react"

type ProviderId = "google" | "github" | "discord"

export function ProviderStatus({
  providerId,
  connected,
}: {
  providerId: ProviderId
  connected: boolean
}) {
  if (connected) {
    return (
      <span className="text-muted-foreground flex items-center gap-1.5 text-sm">
        <Check className="size-3.5" />
        Connected
      </span>
    )
  }

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => {
        signIn.social({
          provider: providerId,
          callbackURL: "/settings",
        })
      }}
    >
      Connect
    </Button>
  )
}
