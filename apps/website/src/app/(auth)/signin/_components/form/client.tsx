"use client"

import { Discord, GitHub, Google } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { signIn } from "@/lib/client/auth"

export function SocialAuthProviderButtons() {
  return (
    <>
      <Button
        variant="outline"
        type="button"
        className="group"
        onClick={() => {
          signIn.social({ provider: "discord" })
        }}
      >
        <Discord />
        <span className="sr-only">Login with Discord</span>
      </Button>
      <Button
        variant="outline"
        type="button"
        className="group"
        onClick={() => {
          signIn.social({ provider: "google" })
        }}
      >
        <Google />
        <span className="sr-only">Login with Google</span>
      </Button>
      <Button
        variant="outline"
        type="button"
        className="group"
        onClick={() => {
          signIn.social({ provider: "github" })
        }}
      >
        <GitHub />
        <span className="sr-only">Login with GitHub</span>
      </Button>
    </>
  )
}
