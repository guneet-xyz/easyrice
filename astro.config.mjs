// @ts-check
import { defineConfig } from "astro/config"

import cloudflare from "@astrojs/cloudflare"
import tailwindcss from "@tailwindcss/vite"

import starlight from "@astrojs/starlight"

export default defineConfig({
  vite: { plugins: [tailwindcss()] },

  adapter: cloudflare({
    platformProxy: {
      enabled: true
    },

    imageService: "cloudflare"
  }),

  integrations: [
    starlight({
      title: "easyrice",
      sidebar: [
        {
          label: "Command-Line Docs",
          autogenerate: { directory: "docs" }
        },
        {
          label: "Ricing Wiki",
          autogenerate: { directory: "wiki" }
        }
      ]
    })
  ]
})
