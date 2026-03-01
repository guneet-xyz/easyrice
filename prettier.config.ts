import { type Config } from "prettier"

const config: Config = {
  trailingComma: "none",
  semi: false,
  plugins: [
    "prettier-plugin-astro",
    "prettier-plugin-tailwindcss",
    "prettier-plugin-astro-organize-imports"
  ],
  overrides: [
    {
      files: "*.astro",
      options: {
        parser: "astro"
      }
    }
  ]
}

export default config
