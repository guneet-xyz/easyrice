"use client"

import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"
import { cn } from "@/lib/utils"
import { ChevronLeft, ChevronRight, Menu } from "lucide-react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import { useState } from "react"
import { getNavConfig, getPrevNext, type NavConfig } from "./nav"

type NavLinksProps = {
  config: NavConfig
  pathname: string
  onNavigate?: () => void
}

function NavLinks({ config, pathname, onNavigate }: NavLinksProps) {
  return (
    <nav className="flex flex-col gap-6">
      {config.nav.map((section) => (
        <div key={section.title} className="flex flex-col gap-1">
          <h4 className="px-3 text-sm font-semibold text-foreground">
            {section.title}
          </h4>
          {section.items.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              onClick={onNavigate}
              className={cn(
                "rounded-md px-3 py-1.5 text-sm transition-colors",
                {
                  "bg-accent font-medium text-accent-foreground":
                    pathname === item.href,
                  "text-muted-foreground hover:bg-accent/50 hover:text-foreground":
                    pathname !== item.href,
                },
              )}
            >
              {item.label}
            </Link>
          ))}
        </div>
      ))}
    </nav>
  )
}

function PrevNextNav({ pathname }: { pathname: string }) {
  const { prev, next } = getPrevNext(pathname)

  if (!prev && !next) return null

  return (
    <nav className="mt-12 flex items-stretch border-t border-border pt-6">
      {prev ? (
        <Button variant="ghost" className="h-auto items-start py-2" asChild>
          <Link href={prev.href}>
            <ChevronLeft className="mt-0.5 size-4 shrink-0" />
            <div className="flex flex-col text-left">
              <span className="text-xs text-muted-foreground">Previous</span>
              <span className="text-sm font-medium">{prev.label}</span>
            </div>
          </Link>
        </Button>
      ) : (
        <div />
      )}
      {next ? (
        <Button
          variant="ghost"
          className="ml-auto h-auto items-start py-2"
          asChild
        >
          <Link href={next.href}>
            <div className="flex flex-col text-right">
              <span className="text-xs text-muted-foreground">Next</span>
              <span className="text-sm font-medium">{next.label}</span>
            </div>
            <ChevronRight className="mt-0.5 size-4 shrink-0" />
          </Link>
        </Button>
      ) : (
        <div />
      )}
    </nav>
  )
}

export function ContentLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const config = getNavConfig(pathname)
  const [sheetOpen, setSheetOpen] = useState(false)

  if (!config) return <>{children}</>

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-1 gap-8 p-6">
      <aside className="hidden w-56 shrink-0 md:block">
        <div className="sticky top-6">
          <h3 className="mb-4 text-lg font-bold">{config.title}</h3>
          <ScrollArea className="h-[calc(100svh-8rem)]">
            <NavLinks config={config} pathname={pathname} />
          </ScrollArea>
        </div>
      </aside>

      <div className="md:hidden">
        <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
          <SheetTrigger asChild>
            <Button variant="outline" size="icon-sm" className="mb-4">
              <Menu className="size-4" />
              <span className="sr-only">Open navigation</span>
            </Button>
          </SheetTrigger>
          <SheetContent side="left">
            <SheetHeader>
              <SheetTitle>{config.title}</SheetTitle>
            </SheetHeader>
            <div className="px-4">
              <NavLinks
                config={config}
                pathname={pathname}
                onNavigate={() => setSheetOpen(false)}
              />
            </div>
          </SheetContent>
        </Sheet>
      </div>

      <main className="min-w-0 flex-1">
        {children}
        <PrevNextNav pathname={pathname} />
      </main>
    </div>
  )
}
