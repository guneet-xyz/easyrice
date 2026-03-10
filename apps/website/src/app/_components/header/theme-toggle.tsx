"use client"

import { DynamicIcon } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { useTheme } from "next-themes"
import { PiMoon, PiMoonDuotone, PiSun, PiSunDuotone } from "react-icons/pi"

export function ThemeToggle() {
  const { theme, setTheme } = useTheme()
  return (
    <div>
      <Button
        variant="secondary"
        onClick={() => {
          setTheme((theme) => (theme === "light" ? "dark" : "light"))
        }}
        className="aspect-square cursor-pointer group"
      >
        {theme === "light" ? (
          <DynamicIcon noHover={<PiSun />} hover={<PiSunDuotone />} />
        ) : (
          <DynamicIcon noHover={<PiMoon />} hover={<PiMoonDuotone />} />
        )}
      </Button>
    </div>
  )
}
