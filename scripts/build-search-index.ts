import { readdirSync, readFileSync, statSync, writeFileSync } from "fs"
import { dirname, join, relative } from "path"

type SearchEntry = {
  id: string
  href: string
  title: string
  section: string
  content: string
}

const CONTENT_DIR = join(import.meta.dirname, "../src/app/(content)")
const OUTPUT_PATH = join(import.meta.dirname, "../public/search-index.json")

const SECTION_TITLES: Record<string, string> = {
  docs: "CLI Reference",
  wiki: "Ricing Guide",
}

function collectMdxFiles(dir: string): string[] {
  const results: string[] = []

  for (const entry of readdirSync(dir)) {
    if (entry.startsWith("_")) continue

    const fullPath = join(dir, entry)
    const stat = statSync(fullPath)

    if (stat.isDirectory()) {
      results.push(...collectMdxFiles(fullPath))
    } else if (entry === "page.mdx") {
      results.push(fullPath)
    }
  }

  return results
}

function extractTitle(raw: string): string {
  const match = raw.match(/title:\s*"([^"]+)"/)
  return match?.[1] ?? "Untitled"
}

function stripMdx(raw: string): string {
  // Remove the metadata export block (may span multiple lines)
  const withoutMeta = raw.replace(
    /export\s+const\s+metadata\s*=\s*\{[^}]*\}\s*/g,
    "",
  )

  return withoutMeta
    .split("\n")
    .filter((line) => {
      const trimmed = line.trimStart()
      if (trimmed.startsWith("export ")) return false
      if (trimmed.startsWith("import ")) return false
      if (trimmed === "```" || trimmed.startsWith("```")) return false
      if (/^\|?\s*[-:]+\s*\|/.test(trimmed)) return false
      return true
    })
    .join("\n")
    .replace(/#{1,6}\s+/g, "")
    .replace(/\*\*([^*]+)\*\*/g, "$1")
    .replace(/\*([^*]+)\*/g, "$1")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/^\|/gm, "")
    .replace(/\|$/gm, "")
    .replace(/\|/g, " ")
    .replace(/\s+/g, " ")
    .trim()
}

function filePathToHref(filePath: string): string {
  const rel = relative(CONTENT_DIR, dirname(filePath))
  return "/" + rel.replace(/\\/g, "/")
}

function getSection(href: string): string {
  const topLevel = href.split("/")[1]
  return SECTION_TITLES[topLevel] ?? topLevel
}

const files = collectMdxFiles(CONTENT_DIR)
const entries: SearchEntry[] = []

for (const filePath of files) {
  const raw = readFileSync(filePath, "utf-8")
  const href = filePathToHref(filePath)
  const title = extractTitle(raw)
  const section = getSection(href)
  const content = stripMdx(raw)

  entries.push({ id: href, href, title, section, content })
}

entries.sort((a, b) => a.href.localeCompare(b.href))

writeFileSync(OUTPUT_PATH, JSON.stringify(entries, null, 2))
console.log(`Search index built: ${entries.length} pages → ${OUTPUT_PATH}`)
