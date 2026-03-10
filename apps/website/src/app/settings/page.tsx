import { auth } from "@/lib/auth"
import { db } from "@/lib/db"
import { account } from "@/lib/db/schema"
import { eq } from "drizzle-orm"
import type { Metadata } from "next"
import { headers } from "next/headers"
import { redirect } from "next/navigation"
import { ProviderCards } from "./_components/provider-cards"
import { UserSettings } from "./_components/user-settings"

export const metadata: Metadata = {
  title: "Settings",
}

export default async function SettingsPage() {
  const session = await auth.api.getSession({
    headers: await headers(),
  })

  if (!session) {
    redirect("/signin")
  }

  const linkedAccounts = await db
    .select({ providerId: account.providerId })
    .from(account)
    .where(eq(account.userId, session.user.id))

  const providers = {
    google: linkedAccounts.some((a) => a.providerId === "google"),
    github: linkedAccounts.some((a) => a.providerId === "github"),
    discord: linkedAccounts.some((a) => a.providerId === "discord"),
  }

  return (
    <div className="mx-auto w-full max-w-2xl space-y-6 p-6">
      <h1 className="text-2xl font-bold">Settings</h1>
      <UserSettings
        name={session.user.name}
        username={session.user.username}
        image={session.user.image ?? null}
      />
      <ProviderCards providers={providers} />
    </div>
  )
}
