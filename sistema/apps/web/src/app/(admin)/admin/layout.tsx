import { ShellAdmin } from "@/components/ShellAdmin";

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return <ShellAdmin>{children}</ShellAdmin>;
}
