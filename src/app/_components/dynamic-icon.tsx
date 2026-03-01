export default function DynamicIcon({
  normal,
  hover
}: {
  normal: React.ReactNode
  hover: React.ReactNode
}) {
  return (
    <div className="relative">
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 opacity-100 group-hover:opacity-0">
        {normal}
      </div>
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100">
        {hover}
      </div>
    </div>
  )
}
