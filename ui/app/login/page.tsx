import { LoginForm } from "@/components/LoginForm";

export default async function LoginPage({
  searchParams,
}: PageProps<"/login">) {
  const { next } = await searchParams;
  const nextPath = Array.isArray(next) ? next[0] : next;
  return <LoginForm next={nextPath} />;
}
