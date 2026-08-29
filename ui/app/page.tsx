import { Suspense } from "react";
import { MediaGridSkeleton, MediaLibrary } from "@/components/MediaLibrary";

export default function Home() {
  return (
    <Suspense
      fallback={
        <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-6 sm:px-6">
          <div className="mb-5 h-6 w-28 animate-pulse rounded bg-zinc-800" />
          <MediaGridSkeleton />
        </main>
      }
    >
      <MediaLibrary />
    </Suspense>
  );
}
