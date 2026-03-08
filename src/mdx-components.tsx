import { cn } from "@/lib/utils"
import type { MDXComponents } from "mdx/types"

export function useMDXComponents(components: MDXComponents): MDXComponents {
  return {
    h1: ({ className, ...props }) => (
      <h1
        className={cn("text-3xl font-bold tracking-tight mb-6", className)}
        {...props}
      />
    ),
    h2: ({ className, ...props }) => (
      <h2
        className={cn(
          "text-2xl font-semibold tracking-tight mt-10 mb-4 border-b border-border pb-2",
          className,
        )}
        {...props}
      />
    ),
    h3: ({ className, ...props }) => (
      <h3
        className={cn(
          "text-xl font-semibold tracking-tight mt-8 mb-3",
          className,
        )}
        {...props}
      />
    ),
    h4: ({ className, ...props }) => (
      <h4
        className={cn("text-lg font-semibold mt-6 mb-2", className)}
        {...props}
      />
    ),
    p: ({ className, ...props }) => (
      <p
        className={cn("leading-7 [&:not(:first-child)]:mt-4", className)}
        {...props}
      />
    ),
    a: ({ className, ...props }) => (
      <a
        className={cn(
          "font-medium text-primary underline underline-offset-4",
          className,
        )}
        {...props}
      />
    ),
    ul: ({ className, ...props }) => (
      <ul className={cn("my-4 ml-6 list-disc", className)} {...props} />
    ),
    ol: ({ className, ...props }) => (
      <ol className={cn("my-4 ml-6 list-decimal", className)} {...props} />
    ),
    li: ({ className, ...props }) => (
      <li className={cn("mt-2", className)} {...props} />
    ),
    blockquote: ({ className, ...props }) => (
      <blockquote
        className={cn("mt-4 border-l-2 border-border pl-6 italic", className)}
        {...props}
      />
    ),
    hr: ({ className, ...props }) => (
      <hr className={cn("my-6 border-border", className)} {...props} />
    ),
    table: ({ className, ...props }) => (
      <div className="my-6 w-full overflow-x-auto">
        <table className={cn("w-full", className)} {...props} />
      </div>
    ),
    th: ({ className, ...props }) => (
      <th
        className={cn(
          "border border-border px-4 py-2 text-left font-bold",
          className,
        )}
        {...props}
      />
    ),
    td: ({ className, ...props }) => (
      <td
        className={cn("border border-border px-4 py-2", className)}
        {...props}
      />
    ),
    pre: ({ className, ...props }) => (
      <pre
        className={cn(
          "my-4 overflow-x-auto rounded-lg bg-muted p-4 font-mono text-sm",
          className,
        )}
        {...props}
      />
    ),
    code: ({ className, ...props }) => (
      <code
        className={cn(
          "rounded bg-muted px-1.5 py-0.5 font-mono text-sm",
          className,
        )}
        {...props}
      />
    ),
    ...components,
  }
}
