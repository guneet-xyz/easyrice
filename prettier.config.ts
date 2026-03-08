import { type Config } from "prettier"

const config: Config = {
  trailingComma: "all",
  semi: false,
  plugins: [
    "prettier-package-json",
    "prettier-plugin-tailwindcss",
    "@ianvs/prettier-plugin-sort-imports",
  ],
}

export default config
