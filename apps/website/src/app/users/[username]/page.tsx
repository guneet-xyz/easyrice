import { db } from "@/lib/db"
import { user } from "@/lib/db/schema"
import { eq } from "drizzle-orm"
import type { Metadata } from "next"
import { notFound } from "next/navigation"
import { ProfileCard } from "./_components/profile-card"

type Props = {
  params: Promise<{ username: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { username } = await params
  const result = await db
    .select({ name: user.name })
    .from(user)
    .where(eq(user.username, username))
    .limit(1)

  if (!result[0]) return { title: "User Not Found" }

  return { title: result[0].name }
}

export default async function UserProfilePage({ params }: Props) {
  const { username } = await params
  const result = await db
    .select({
      id: user.id,
      name: user.name,
      email: user.email,
      username: user.username,
    })
    .from(user)
    .where(eq(user.username, username))
    .limit(1)

  if (!result[0]) notFound()

  return (
    <div className="flex min-h-svh flex-col items-center justify-center p-6 md:p-10">
      <div className="w-full max-w-sm">
        <ProfileCard user={result[0]} />
      </div>
    </div>
  )
}
