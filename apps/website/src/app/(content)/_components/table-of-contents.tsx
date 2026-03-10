"use client"

import { cn } from "@/lib/utils"
import { usePathname } from "next/navigation"
import { useCallback, useEffect, useRef, useState } from "react"

type TocHeading = {
  id: string
  text: string
  level: number
}

function getHeadingsSnapshot(): TocHeading[] {
  const main = document.querySelector("main")
  if (!main) return []

  const elements = main.querySelectorAll("h2[id], h3[id]")
  return Array.from(elements).map((el) => ({
    id: el.id,
    text: el.textContent ?? "",
    level: el.tagName === "H2" ? 2 : 3,
  }))
}

const emptyHeadings: TocHeading[] = []

function useHeadings() {
  const pathname = usePathname()
  const [headings, setHeadings] = useState<TocHeading[]>(emptyHeadings)

  useEffect(() => {
    const update = () => setHeadings(getHeadingsSnapshot())

    const observer = new MutationObserver(update)
    observer.observe(document.body, { childList: true, subtree: true })

    const frame = requestAnimationFrame(update)

    return () => {
      observer.disconnect()
      cancelAnimationFrame(frame)
    }
  }, [pathname])

  return headings
}

function getHashId(): string {
  if (typeof window === "undefined") return ""
  return window.location.hash.replace(/^#/, "")
}

function useActiveHeading(headingIds: string[]) {
  const [activeId, setActiveId] = useState(getHashId)
  const observerRef = useRef<IntersectionObserver | null>(null)
  const scrollPausedRef = useRef(false)
  const pauseTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (headingIds.length === 0) return

    observerRef.current?.disconnect()

    observerRef.current = new IntersectionObserver(
      (entries) => {
        if (scrollPausedRef.current) return

        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveId(entry.target.id)
          }
        }
      },
      { rootMargin: "0px 0px -70% 0px" },
    )

    for (const id of headingIds) {
      const el = document.getElementById(id)
      if (el) observerRef.current.observe(el)
    }

    return () => observerRef.current?.disconnect()
  }, [headingIds])

  useEffect(() => {
    function onHashChange() {
      setActiveId(getHashId())
    }

    window.addEventListener("hashchange", onHashChange)
    return () => window.removeEventListener("hashchange", onHashChange)
  }, [])

  const onHeadingClick = useCallback((e: React.MouseEvent, id: string) => {
    scrollPausedRef.current = true
    if (pauseTimerRef.current) clearTimeout(pauseTimerRef.current)
    pauseTimerRef.current = setTimeout(() => {
      scrollPausedRef.current = false
    }, 1000)

    setActiveId(id)
  }, [])

  return { activeId, onHeadingClick }
}

export function TableOfContents() {
  const headings = useHeadings()
  const headingIds = headings.map((h) => h.id)
  const { activeId, onHeadingClick } = useActiveHeading(headingIds)

  if (headings.length === 0) return null

  return (
    <div className="sticky top-6">
      <h4 className="mb-3 text-sm font-semibold">On this page</h4>
      <nav className="flex flex-col gap-1">
        {headings.map((heading) => (
          <a
            key={heading.id}
            href={`#${heading.id}`}
            onClick={(e) => onHeadingClick(e, heading.id)}
            className={cn("text-sm transition-colors hover:text-foreground", {
              "pl-3": heading.level === 3,
              "font-medium text-foreground": activeId === heading.id,
              "text-muted-foreground": activeId !== heading.id,
            })}
          >
            {heading.text}
          </a>
        ))}
      </nav>
    </div>
  )
}
