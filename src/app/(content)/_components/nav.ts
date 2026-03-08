export type NavItem = {
  label: string
  href: string
}

export type NavSection = {
  title: string
  items: NavItem[]
}

export type NavConfig = {
  title: string
  nav: NavSection[]
}

const docsNav: NavSection[] = [
  {
    title: "Getting Started",
    items: [
      { label: "Introduction", href: "/docs" },
      { label: "Installation", href: "/docs/installation" },
    ],
  },
  {
    title: "Reference",
    items: [{ label: "Commands", href: "/docs/commands" }],
  },
]

const wikiNav: NavSection[] = [
  {
    title: "Basics",
    items: [
      { label: "Introduction", href: "/wiki" },
      { label: "What is Ricing?", href: "/wiki/what-is-ricing" },
    ],
  },
  {
    title: "Topics",
    items: [
      { label: "Window Managers", href: "/wiki/window-managers" },
      { label: "Dotfiles", href: "/wiki/dotfiles" },
    ],
  },
]

export const configs: Record<string, NavConfig> = {
  "/docs": { title: "CLI Reference", nav: docsNav },
  "/wiki": { title: "Ricing Guide", nav: wikiNav },
}

export function getNavConfig(pathname: string): NavConfig | null {
  for (const [prefix, config] of Object.entries(configs)) {
    if (pathname.startsWith(prefix)) return config
  }
  return null
}

export function getPrevNext(pathname: string): {
  prev: NavItem | null
  next: NavItem | null
} {
  const config = getNavConfig(pathname)
  if (!config) return { prev: null, next: null }

  const allItems = config.nav.flatMap((section) => section.items)
  const index = allItems.findIndex((item) => item.href === pathname)

  return {
    prev: index > 0 ? allItems[index - 1] : null,
    next: index < allItems.length - 1 ? allItems[index + 1] : null,
  }
}
