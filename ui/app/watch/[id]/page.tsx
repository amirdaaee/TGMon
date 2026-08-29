import { Suspense } from "react";
import { WatchView } from "@/components/WatchView";

export default function WatchPage() {
  return (
    <Suspense
      fallback={
        <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-6 sm:px-6">
          <div className="aspect-video animate-pulse rounded-xl bg-zinc-800" />
        </main>
      }
    >
      <WatchView />
    </Suspense>
  );
}
