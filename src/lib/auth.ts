import { db } from "@/lib/db"
import { user } from "@/lib/db/schema"
import { betterAuth } from "better-auth"
import { drizzleAdapter } from "better-auth/adapters/drizzle"
import { nextCookies } from "better-auth/next-js"
import { count, eq } from "drizzle-orm"
import { env } from "./env"

function sanitizeUsername(input: string): string {
  let result = ""
  for (const char of input) {
    if (/[a-zA-Z0-9_-]/.test(char)) {
      result += char
    } else if (char === " ") {
      result += "_"
    }
  }
  return result
}

async function isUsernameTaken(username: string): Promise<boolean> {
  const rows = await db
    .select({ id: user.id })
    .from(user)
    .where(eq(user.username, username.toLowerCase()))
    .limit(1)
  return rows.length > 0
}

async function generateUsername(name: string, email: string): Promise<string> {
  let base = sanitizeUsername(name).toLowerCase()
  if (base.length < 3) {
    base = sanitizeUsername(email.split("@")[0] ?? "").toLowerCase()
  }

  if (base.length < 3) {
    const [{ count: userCount }] = await db
      .select({ count: count() })
      .from(user)
    return `user-${userCount + 1}`
  }

  if (base.length > 20) {
    base = base.slice(0, 20)
  }

  if (!(await isUsernameTaken(base))) return base

  for (let i = 1; i <= 100; i++) {
    const candidate = `${base.slice(0, 16)}-${i}`
    if (!(await isUsernameTaken(candidate))) return candidate
  }

  const suffix = Math.random().toString(36).substring(2, 6)
  return `${base.slice(0, 15)}-${suffix}`
}

export const auth = betterAuth({
  database: drizzleAdapter(db, {
    provider: "pg",
  }),
  plugins: [nextCookies()],
  socialProviders: {
    google: {
      clientId: env.GOOGLE_CLIENT_ID,
      clientSecret: env.GOOGLE_CLIENT_SECRET,
    },
    github: {
      clientId: env.GITHUB_CLIENT_ID,
      clientSecret: env.GITHUB_CLIENT_SECRET,
    },
    discord: {
      clientId: env.DISCORD_CLIENT_ID,
      clientSecret: env.DISCORD_CLIENT_SECRET,
    },
  },
  user: {
    additionalFields: {
      username: {
        type: "string",
        required: true,
        unique: true,
        returned: true,
        input: false,
      },
    },
  },
  databaseHooks: {
    user: {
      create: {
        before: async (userData) => {
          const username = await generateUsername(userData.name, userData.email)
          return {
            data: { ...userData, username },
          }
        },
      },
    },
  },
})
