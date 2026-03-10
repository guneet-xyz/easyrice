import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Suspense } from "react"
import { AvatarEdit, EditNameDialog, EditUsernameDialog } from "./edit-dialogs"

type UserSettingsProps = {
  name: string
  username: string
  image: string | null
}

function AvatarSkeleton() {
  return <div className="bg-muted size-20 animate-pulse rounded-full" />
}

function ButtonSkeleton() {
  return <div className="bg-muted h-6 w-14 animate-pulse rounded-md" />
}

export function UserSettings({ name, username, image }: UserSettingsProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>User Settings</CardTitle>
        <CardDescription>Manage your profile information.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <Suspense fallback={<AvatarSkeleton />}>
          <AvatarEdit name={name} image={image} />
        </Suspense>

        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground text-sm">Name</span>
            <Suspense fallback={<ButtonSkeleton />}>
              <EditNameDialog currentName={name} />
            </Suspense>
          </div>
          <p className="text-sm font-medium">{name}</p>
        </div>

        <Separator />

        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground text-sm">Username</span>
            <Suspense fallback={<ButtonSkeleton />}>
              <EditUsernameDialog currentUsername={username} />
            </Suspense>
          </div>
          <p className="text-sm font-medium">{username}</p>
        </div>
      </CardContent>
    </Card>
  )
}
