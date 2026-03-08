import { env } from "@/lib/env"
import axios from "axios"
import { drizzle } from "drizzle-orm/pg-proxy"
import * as schema from "./schema"

const endpoint = `${env.DRIZZLE_PROXY_URL}/query`
const authHeader = `Bearer ${env.DRIZZLE_PROXY_TOKEN}`

export const db = drizzle(
  async (sql, params, method) => {
    try {
      const rows: { data: unknown[] } = await axios.post(
        endpoint,
        { sql, params, method, database: "easyrice" },
        {
          headers: {
            Authorization: authHeader,
          },
        },
      )

      return { rows: rows.data }
    } catch (e) {
      console.error("Error during db query", e)
      return { rows: [] }
    }
  },
  {
    schema: schema,
  },
)
