import { auth } from "@/lib/auth"
import { headers } from "next/headers"

export type ActionResult = { success: boolean; message: string }

type SessionData = NonNullable<Awaited<ReturnType<typeof auth.api.getSession>>>

export function authedAction<TArgs extends unknown[]>(
  fn: (session: SessionData, ...args: TArgs) => Promise<ActionResult>,
): (...args: TArgs) => Promise<ActionResult> {
  return async (...args: TArgs) => {
    const session = await auth.api.getSession({
      headers: await headers(),
    })
    if (!session) {
      return { success: false, message: "Not logged in." }
    }
    return fn(session, ...args)
  }
}
