import type { Metadata } from "next"
import { ProfileCard } from "./_components/client"

export const metadata: Metadata = {
  title: "Profile",
}

export default function ProfilePage() {
  return (
    <div className="flex min-h-svh flex-col items-center justify-center p-6 md:p-10">
      <div className="w-full max-w-sm">
        <ProfileCard />
      </div>
    </div>
  )
}
