"use client"

import { Button } from "@/components/ui/button"
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import { FileText, Search } from "lucide-react"
import MiniSearch from "minisearch"
import { useRouter } from "next/navigation"
import type React from "react"
import { useCallback, useEffect, useMemo, useRef, useState } from "react"

type SearchEntry = {
  id: string
  href: string
  title: string
  section: string
  content: string
}

function useSearchIndex() {
  const [entries, setEntries] = useState<SearchEntry[]>([])
  const fetchedRef = useRef(false)

  const load = useCallback(() => {
    if (fetchedRef.current) return
    fetchedRef.current = true

    fetch("/search-index.json")
      .then((res) => res.json() as Promise<SearchEntry[]>)
      .then(setEntries)
  }, [])

  const miniSearch = useMemo(() => {
    const ms = new MiniSearch<SearchEntry>({
      fields: ["title", "content"],
      storeFields: ["href", "title", "section", "content"],
      searchOptions: {
        boost: { title: 3 },
        prefix: true,
        fuzzy: 0.2,
      },
    })
    if (entries.length > 0) ms.addAll(entries)
    return ms
  }, [entries])

  return { entries, miniSearch, load }
}

type SearchResult = {
  href: string
  title: string
  section: string
  snippet?: string
}

const SNIPPET_WINDOW = 120

function getSnippet(content: string, query: string): string | undefined {
  const terms = query
    .toLowerCase()
    .split(/\s+/)
    .filter((t) => t.length > 0)
  if (terms.length === 0) return undefined

  const lowerContent = content.toLowerCase()

  let earliest = -1
  for (const term of terms) {
    const idx = lowerContent.indexOf(term)
    if (idx !== -1 && (earliest === -1 || idx < earliest)) {
      earliest = idx
    }
  }

  if (earliest === -1) return content.slice(0, SNIPPET_WINDOW) + "..."

  const start = Math.max(0, earliest - SNIPPET_WINDOW / 2)
  const end = Math.min(content.length, earliest + SNIPPET_WINDOW / 2)
  const prefix = start > 0 ? "..." : ""
  const suffix = end < content.length ? "..." : ""

  return prefix + content.slice(start, end).trim() + suffix
}

function groupBySection(items: SearchResult[]): Record<string, SearchResult[]> {
  const groups: Record<string, SearchResult[]> = {}
  for (const item of items) {
    const group = (groups[item.section] ??= [])
    group.push(item)
  }
  return groups
}

function highlightMatches(text: string, query: string): React.ReactNode {
  const terms = query
    .split(/\s+/)
    .filter((t) => t.length > 0)
    .map((t) => t.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))

  if (terms.length === 0) return text

  const regex = new RegExp(`(${terms.join("|")})`, "gi")
  const parts = text.split(regex)

  return parts.map((part, i) => {
    if (terms.some((t) => part.toLowerCase() === t.toLowerCase())) {
      return (
        <mark
          key={i}
          className="bg-transparent font-medium text-red-500 dark:text-red-400"
        >
          {part}
        </mark>
      )
    }
    return part
  })
}

export function SearchCommand() {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const router = useRouter()
  const { entries, miniSearch, load } = useSearchIndex()

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        setOpen((prev) => {
          if (prev) setQuery("")
          return !prev
        })
      }
    }

    document.addEventListener("keydown", onKeyDown)
    return () => document.removeEventListener("keydown", onKeyDown)
  }, [])

  useEffect(() => {
    if (open) load()
  }, [open, load])

  const onOpenChange = useCallback((value: boolean) => {
    setOpen(value)
    if (!value) setQuery("")
  }, [])

  const onSelect = useCallback(
    (href: string) => {
      setOpen(false)
      router.push(href)
    },
    [router],
  )

  const results = useMemo(() => {
    if (!query.trim()) {
      return groupBySection(
        entries.map((e) => ({
          href: e.href,
          title: e.title,
          section: e.section,
        })),
      )
    }
    const hits = miniSearch.search(query)
    return groupBySection(
      hits.map((hit) => ({
        href: hit.href as string,
        title: hit.title as string,
        section: hit.section as string,
        snippet: getSnippet(hit.content as string, query),
      })),
    )
  }, [query, entries, miniSearch])

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        className="hidden gap-2 text-muted-foreground sm:flex"
        onClick={() => setOpen(true)}
      >
        <Search className="size-4" />
        <span className="text-sm">Search...</span>
        <kbd className="pointer-events-none ml-2 inline-flex h-5 items-center gap-0.5 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground">
          <span className="text-xs">⌘</span>K
        </kbd>
      </Button>
      <Button
        variant="outline"
        size="icon-sm"
        className="sm:hidden"
        onClick={() => setOpen(true)}
      >
        <Search className="size-4" />
        <span className="sr-only">Search</span>
      </Button>
      <CommandDialog
        open={open}
        onOpenChange={onOpenChange}
        shouldFilter={false}
        className="sm:max-w-2xl"
        title="Search"
        description="Search across docs and wiki pages."
      >
        <CommandInput
          placeholder="Search docs and wiki..."
          value={query}
          onValueChange={setQuery}
        />
        <CommandList className="max-h-[400px]">
          <CommandEmpty>No results found.</CommandEmpty>
          {Object.entries(results).map(([section, items]) => (
            <CommandGroup key={section} heading={section}>
              {items.map((item) => (
                <CommandItem
                  key={item.href}
                  value={item.href}
                  onSelect={() => onSelect(item.href)}
                >
                  <FileText className="mt-0.5 size-4 shrink-0 self-start" />
                  <div className="flex min-w-0 flex-col">
                    <span>{item.title}</span>
                    {item.snippet && (
                      <span className="line-clamp-1 text-xs text-muted-foreground">
                        {highlightMatches(item.snippet, query)}
                      </span>
                    )}
                  </div>
                </CommandItem>
              ))}
            </CommandGroup>
          ))}
        </CommandList>
      </CommandDialog>
    </>
  )
}
