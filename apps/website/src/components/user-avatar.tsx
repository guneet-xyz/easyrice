import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { db } from "@/lib/db"
import { user } from "@/lib/db/schema"
import { eq } from "drizzle-orm"

export async function UserAvatar({ username }: { username: string }) {
  const result = await db
    .select({ image: user.image, name: user.name })
    .from(user)
    .where(eq(user.username, username))
    .limit(1)

  if (!result[0]) return null

  const { image, name } = result[0]
  const initials = name
    .split(" ")
    .map((word) => word[0])
    .join("")
    .toUpperCase()
    .slice(0, 2)

  return (
    <Avatar size="lg">
      {image && <AvatarImage src={image} alt={name} />}
      <AvatarFallback>{initials}</AvatarFallback>
    </Avatar>
  )
}
