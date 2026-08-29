"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { listMedia } from "@/lib/api";
import { PAGE_SIZE } from "@/lib/config";
import { asId } from "@/lib/format";
import type { MediaListRes } from "@/lib/types";
import { MediaCard } from "./MediaCard";
import { Pagination } from "./Pagination";

export function MediaLibrary() {
  const searchParams = useSearchParams();
  const page = Math.max(0, Number(searchParams.get("page") ?? 0) || 0);
  return <MediaLibraryPage key={page} page={page} />;
}

function MediaLibraryPage({ page }: { page: number }) {
  const searchParams = useSearchParams();
  const router = useRouter();
  const [data, setData] = useState<MediaListRes | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    listMedia(page)
      .then((res) => {
        if (cancelled) {
          return;
        }
        setData({
          Total: res.Total,
          Media: (res.Media ?? []).map((item) => ({
            ...item,
            Media: { ...item.Media, ID: asId(item.Media.ID) },
          })),
        });
        setError("");
        setLoading(false);
      })
      .catch((err: unknown) => {
        if (cancelled) {
          return;
        }
        setData(null);
        setError(err instanceof Error ? err.message : "Could not load media");
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [page]);

  function retry() {
    setLoading(true);
    setError("");
    listMedia(page)
      .then((res) => {
        setData({
          Total: res.Total,
          Media: (res.Media ?? []).map((item) => ({
            ...item,
            Media: { ...item.Media, ID: asId(item.Media.ID) },
          })),
        });
        setLoading(false);
      })
      .catch((err: unknown) => {
        setData(null);
        setError(err instanceof Error ? err.message : "Could not load media");
        setLoading(false);
      });
  }

  function goToPage(next: number) {
    const params = new URLSearchParams(searchParams.toString());
    if (next <= 0) {
      params.delete("page");
    } else {
      params.set("page", String(next));
    }
    const qs = params.toString();
    router.push(qs ? `/?${qs}` : "/");
  }

  const items = data?.Media ?? [];
  const total = data?.Total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-6 sm:px-6">
      <div className="mb-5 flex items-end justify-between gap-4">
        <div>
          <h1 className="text-lg font-semibold tracking-tight">Library</h1>
          <p className="text-sm text-muted">
            {loading ? "Loading…" : `${total} ${total === 1 ? "video" : "videos"}`}
          </p>
        </div>
        {!loading && totalPages > 1 ? (
          <p className="text-sm text-muted">
            Page {page + 1} of {totalPages}
          </p>
        ) : null}
      </div>

      {loading ? (
        <MediaGridSkeleton />
      ) : error ? (
        <div className="flex flex-col items-start gap-3 rounded-xl border border-border bg-surface p-6">
          <p className="text-sm text-danger">{error}</p>
          <button
            type="button"
            onClick={retry}
            className="rounded-md bg-surface-hover px-3 py-1.5 text-sm hover:bg-zinc-700 focus-visible:outline-2 focus-visible:outline-accent"
          >
            Retry
          </button>
        </div>
      ) : items.length === 0 ? (
        <div className="rounded-xl border border-dashed border-border p-12 text-center text-sm text-muted">
          No videos yet.
        </div>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-4">
            {items.map((item) => (
              <MediaCard key={item.Media.ID} item={item} page={page} />
            ))}
          </div>
          <Pagination page={page} total={total} onPage={goToPage} />
        </>
      )}
    </main>
  );
}

export function MediaGridSkeleton() {
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-4">
      {Array.from({ length: PAGE_SIZE }, (_, i) => (
        <div key={i} className="overflow-hidden rounded-xl bg-surface">
          <div className="aspect-video animate-pulse bg-zinc-800" />
          <div className="space-y-2 px-3 py-3">
            <div className="h-3 w-4/5 animate-pulse rounded bg-zinc-800" />
            <div className="h-3 w-2/5 animate-pulse rounded bg-zinc-800" />
          </div>
        </div>
      ))}
    </div>
  );
}
