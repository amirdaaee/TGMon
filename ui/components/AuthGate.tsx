"use client";

import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import { isSafeNext } from "@/lib/auth";
import { useAuth } from "./AuthProvider";
import { BootScreen } from "./BootScreen";
import { Header } from "./Header";

export function AuthGate({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const pathname = usePathname();
  const router = useRouter();
  const isLogin = pathname === "/login";

  useEffect(() => {
    if (status === "loading") {
      return;
    }
    const params = new URLSearchParams(window.location.search);
    if (isLogin && status === "authed") {
      const next = params.get("next") ?? "/";
      router.replace(isSafeNext(next) ? next : "/");
      return;
    }
    if (!isLogin && status === "anon") {
      const qs = params.toString();
      const current = qs ? `${pathname}?${qs}` : pathname;
      const next = isSafeNext(current) ? current : "/";
      router.replace(`/login?next=${encodeURIComponent(next)}`);
    }
  }, [status, isLogin, pathname, router]);

  // Render the login page during the initial "loading" status so SSR HTML matches hydration.
  if (isLogin) {
    if (status === "authed") {
      return <BootScreen />;
    }
    return <>{children}</>;
  }

  if (status === "loading") {
    return <BootScreen />;
  }

  if (status === "anon") {
    return <BootScreen />;
  }

  return (
    <div className="flex min-h-full flex-col">
      <Header />
      <div className="flex flex-1 flex-col">{children}</div>
    </div>
  );
}
