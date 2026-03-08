import { cn } from "@/lib/utils"
import {
  PiAppleLogo,
  PiAppleLogoBold,
  PiAppleLogoDuotone,
  PiDiscordLogo,
  PiDiscordLogoDuotone,
  PiGithubLogo,
  PiGithubLogoDuotone,
  PiGoogleLogo,
  PiGoogleLogoDuotone,
} from "react-icons/pi"

export function DynamicIconThemed({
  lightNoHover,
  lightHover,
  darkNoHover,
  darkHover,
}: {
  lightNoHover: React.ReactNode
  lightHover: React.ReactNode
  darkNoHover: React.ReactNode
  darkHover: React.ReactNode
}) {
  return (
    <div className="relative">
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 opacity-100 group-hover:opacity-0 transition-opacity dark:opacity-0">
        {lightNoHover}
      </div>
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 dark:opacity-0 transition-opacity">
        {lightHover}
      </div>
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 opacity-0 dark:group-hover:opacity-0 transition-opacity dark:opacity-100">
        {darkNoHover}
      </div>
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 opacity-0 dark:group-hover:opacity-100 dark:opacity-100 transition-opacity-100">
        {darkHover}
      </div>
    </div>
  )
}

export function DynamicIcon({
  className,
  noHover,
  hover,
}: {
  className?: string
  noHover: React.ReactNode
  hover: React.ReactNode
}) {
  return (
    <div className={cn("relative", className)}>
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 opacity-100 group-hover:opacity-0 transition-opacity">
        {noHover}
      </div>
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity">
        {hover}
      </div>
    </div>
  )
}

export function Discord() {
  return (
    <DynamicIcon noHover={<PiDiscordLogo />} hover={<PiDiscordLogoDuotone />} />
  )
}

export function Google() {
  return (
    <DynamicIcon noHover={<PiGoogleLogo />} hover={<PiGoogleLogoDuotone />} />
  )
}

export function GitHub() {
  return (
    <DynamicIcon noHover={<PiGithubLogo />} hover={<PiGithubLogoDuotone />} />
  )
}
