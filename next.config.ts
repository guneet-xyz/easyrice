// Enable calling `getCloudflareContext()` in `next dev`.
// See https://opennext.js.org/cloudflare/bindings#local-access-to-bindings.
import createMDX from "@next/mdx"
import { initOpenNextCloudflareForDev } from "@opennextjs/cloudflare"
import type { NextConfig } from "next"

const nextConfig: NextConfig = {
  pageExtensions: ["ts", "tsx", "md", "mdx"],
}

const withMDX = createMDX({
  options: {
    remarkPlugins: ["remark-gfm"],
  },
})

export default withMDX(nextConfig)

initOpenNextCloudflareForDev()
