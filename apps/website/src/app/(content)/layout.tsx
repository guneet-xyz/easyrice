import { ContentLayout } from "./_components/content-layout"

export default function ContentSectionLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return <ContentLayout>{children}</ContentLayout>
}
