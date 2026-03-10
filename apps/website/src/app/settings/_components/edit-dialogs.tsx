"use client"

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useSession } from "@/lib/client/auth"
import { changeName } from "@/lib/server/actions/change-name"
import { changeUsername } from "@/lib/server/actions/change-username"
import { Pencil } from "lucide-react"
import { useState, useTransition } from "react"
import { toast } from "sonner"

function getInitials(name: string): string {
  return name
    .split(" ")
    .map((word) => word[0])
    .join("")
    .toUpperCase()
    .slice(0, 2)
}

export function AvatarEdit({
  name,
  image,
}: {
  name: string
  image: string | null
}) {
  return (
    <Dialog>
      <DialogTrigger asChild>
        <button
          type="button"
          className="group/avatar-edit relative size-20 rounded-full focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        >
          <Avatar className="size-20">
            {image && <AvatarImage src={image} alt={name} />}
            <AvatarFallback className="text-lg">
              {getInitials(name)}
            </AvatarFallback>
          </Avatar>
          <div className="absolute inset-0 flex items-center justify-center rounded-full bg-black/50 opacity-0 transition-opacity group-hover/avatar-edit:opacity-100">
            <Pencil className="size-5 text-white" />
          </div>
        </button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Change Avatar</DialogTitle>
          <DialogDescription>
            This feature isn&apos;t available yet. Check back soon!
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">OK</Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function EditNameDialog({ currentName }: { currentName: string }) {
  const [open, setOpen] = useState(false)
  const [value, setValue] = useState(currentName)
  const [isPending, startTransition] = useTransition()
  const { refetch } = useSession()

  function handleOpen(isOpen: boolean) {
    setOpen(isOpen)
    if (isOpen) {
      setValue(currentName)
    }
  }

  function handleSave() {
    const trimmed = value.trim()
    if (trimmed.length === 0) {
      toast.error("Name cannot be empty.")
      return
    }
    if (trimmed.length > 50) {
      toast.error("Name must be 50 characters or fewer.")
      return
    }

    startTransition(async () => {
      const result = await changeName(trimmed)
      if (result.success) {
        toast.success(result.message)
        setOpen(false)
        await refetch()
      } else {
        toast.error(result.message)
      }
    })
  }

  return (
    <Dialog open={open} onOpenChange={handleOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="xs">
          Change
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Change Name</DialogTitle>
          <DialogDescription>
            Enter your new display name below.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="edit-name">Name</Label>
          <Input
            id="edit-name"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="Your name"
            maxLength={50}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleSave()
            }}
          />
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline" disabled={isPending}>
              Cancel
            </Button>
          </DialogClose>
          <Button
            onClick={handleSave}
            disabled={isPending || value.trim() === currentName}
          >
            {isPending ? "Saving..." : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function EditUsernameDialog({
  currentUsername,
}: {
  currentUsername: string
}) {
  const [open, setOpen] = useState(false)
  const [value, setValue] = useState(currentUsername)
  const [error, setError] = useState<string | null>(null)
  const [isPending, startTransition] = useTransition()
  const { refetch } = useSession()

  function handleOpen(isOpen: boolean) {
    setOpen(isOpen)
    if (isOpen) {
      setValue(currentUsername)
      setError(null)
    }
  }

  function validate(input: string): string | null {
    if (input.length < 3) return "Must be at least 3 characters."
    if (input.length > 20) return "Must be 20 characters or fewer."
    if (!/^[a-zA-Z0-9_-]+$/.test(input)) {
      return "Only letters, numbers, hyphens, and underscores."
    }
    return null
  }

  function handleChange(input: string) {
    setValue(input)
    setError(validate(input))
  }

  function handleSave() {
    const trimmed = value.trim()
    const validationError = validate(trimmed)
    if (validationError) {
      setError(validationError)
      return
    }

    startTransition(async () => {
      const result = await changeUsername(trimmed)
      if (result.success) {
        toast.success(result.message)
        setOpen(false)
        await refetch()
      } else {
        toast.error(result.message)
      }
    })
  }

  return (
    <Dialog open={open} onOpenChange={handleOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="xs">
          Change
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Change Username</DialogTitle>
          <DialogDescription>
            Your username must be 3-20 characters and can only contain letters,
            numbers, hyphens, and underscores.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="edit-username">Username</Label>
          <Input
            id="edit-username"
            value={value}
            onChange={(e) => handleChange(e.target.value)}
            placeholder="your-username"
            maxLength={20}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !error) handleSave()
            }}
          />
          {error && <p className="text-destructive text-sm">{error}</p>}
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline" disabled={isPending}>
              Cancel
            </Button>
          </DialogClose>
          <Button
            onClick={handleSave}
            disabled={
              isPending ||
              !!error ||
              value.trim().toLowerCase() === currentUsername
            }
          >
            {isPending ? "Saving..." : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
