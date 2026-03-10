import { Discord, GitHub, Google } from "@/components/icons"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Suspense } from "react"
import { ProviderStatus } from "./provider-status"

type Providers = {
  google: boolean
  github: boolean
  discord: boolean
}

const providerConfig = [
  {
    id: "discord" as const,
    name: "Discord",
    icon: Discord,
  },
  {
    id: "google" as const,
    name: "Google",
    icon: Google,
  },
  {
    id: "github" as const,
    name: "GitHub",
    icon: GitHub,
  },
]

function StatusSkeleton() {
  return <div className="bg-muted h-8 w-20 animate-pulse rounded-md" />
}

export function ProviderCards({ providers }: { providers: Providers }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Social Provider Integrations</CardTitle>
        <CardDescription>
          Connect your social accounts to sign in with them.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {providerConfig.map((provider, index) => {
          const Icon = provider.icon
          return (
            <div key={provider.id}>
              {index > 0 && <Separator className="mb-4" />}
              <div className="group flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="flex size-9 items-center justify-center">
                    <Icon />
                  </div>
                  <span className="text-sm font-medium">{provider.name}</span>
                </div>
                <Suspense fallback={<StatusSkeleton />}>
                  <ProviderStatus
                    providerId={provider.id}
                    connected={providers[provider.id]}
                  />
                </Suspense>
              </div>
            </div>
          )
        })}
      </CardContent>
    </Card>
  )
}
