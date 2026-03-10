import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { UserAvatar } from "@/components/user-avatar"
import { SignOut } from "./sign-out"

type User = {
  id: string
  name: string
  email: string
  username: string
}

export function ProfileCard({ user }: { user: User }) {
  return (
    <Card className="p-0">
      <CardContent className="p-6 md:p-8">
        <div className="flex flex-col gap-6">
          <div className="flex items-center gap-3">
            <UserAvatar username={user.username} />
            <h1 className="text-2xl font-bold">{user.name}</h1>
          </div>
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <span className="text-muted-foreground text-sm">Username</span>
              <span className="text-sm font-medium">{user.username}</span>
            </div>
            <Separator />
            <div className="flex flex-col gap-1">
              <span className="text-muted-foreground text-sm">Name</span>
              <span className="text-sm font-medium">{user.name}</span>
            </div>
            <SignOut userId={user.id} email={user.email} />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
