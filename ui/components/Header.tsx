"use client";

import Link from "next/link";
import { useAuth } from "./AuthProvider";

export function Header() {
  const { logout } = useAuth();

  return (
    <header className="sticky top-0 z-20 border-b border-border/80 bg-background/85 backdrop-blur-md">
      <div className="mx-auto flex h-14 w-full max-w-7xl items-center justify-between px-4 sm:px-6">
        <Link
          href="/"
          className="text-sm font-semibold tracking-tight text-foreground focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
        >
          TGMon
        </Link>
        <button
          type="button"
          onClick={logout}
          className="rounded-md px-3 py-1.5 text-sm text-muted transition-colors hover:bg-surface-hover hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
        >
          Sign out
        </button>
      </div>
    </header>
  );
}
