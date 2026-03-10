"use server"

import { db } from "@/lib/db"
import { user } from "@/lib/db/schema"
import { eq } from "drizzle-orm"
import { revalidatePath } from "next/cache"
import { authedAction } from "."

export const changeName = authedAction(async (session, name: string) => {
  const trimmed = name.trim()
  if (trimmed.length === 0) {
    return { success: false, message: "Name cannot be empty." }
  }
  if (trimmed.length > 50) {
    return { success: false, message: "Name must be 50 characters or fewer." }
  }

  await db
    .update(user)
    .set({ name: trimmed })
    .where(eq(user.id, session.user.id))

  revalidatePath("/settings")
  return { success: true, message: "Name updated successfully." }
})
