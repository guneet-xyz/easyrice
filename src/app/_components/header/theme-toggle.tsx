"use client"

import { Button } from "@/components/ui/button"
import { useTheme } from "next-themes"
import { PiMoon, PiMoonDuotone, PiSun, PiSunDuotone } from "react-icons/pi"
import DynamicIcon from "../dynamic-icon"

export default function ThemeToggle() {
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
        {theme == "light" ? (
          <DynamicIcon normal={<PiSun />} hover={<PiSunDuotone />} />
        ) : (
          <DynamicIcon normal={<PiMoon />} hover={<PiMoonDuotone />} />
        )}
      </Button>
    </div>
  )
}
