"use server"

import { db } from "@/lib/db"
import { user } from "@/lib/db/schema"
import { eq } from "drizzle-orm"
import { revalidatePath } from "next/cache"
import { authedAction } from "."

export const changeUsername = authedAction(
  async (session, username: string) => {
    const trimmed = username.trim()
    if (trimmed.length < 3) {
      return {
        success: false,
        message: "Username must be at least 3 characters.",
      }
    }
    if (trimmed.length > 20) {
      return {
        success: false,
        message: "Username must be 20 characters or fewer.",
      }
    }
    if (!/^[a-zA-Z0-9_-]+$/.test(trimmed)) {
      return {
        success: false,
        message:
          "Username can only contain letters, numbers, hyphens, and underscores.",
      }
    }

    const existing = await db
      .select({ id: user.id })
      .from(user)
      .where(eq(user.username, trimmed.toLowerCase()))
      .limit(1)

    if (existing[0] && existing[0].id !== session.user.id) {
      return { success: false, message: "Username is already taken." }
    }

    await db
      .update(user)
      .set({ username: trimmed.toLowerCase() })
      .where(eq(user.id, session.user.id))

    revalidatePath("/settings")
    return { success: true, message: "Username updated successfully." }
  },
)
