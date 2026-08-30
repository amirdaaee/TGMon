"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { deleteMedia, listMedia } from "@/lib/api";
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
  const [selected, setSelected] = useState<Set<string>>(() => new Set());
  const [deleting, setDeleting] = useState(false);
  const [actionError, setActionError] = useState("");

  const applyList = useCallback((res: MediaListRes) => {
    setData({
      Total: res.Total,
      Media: (res.Media ?? []).map((item) => ({
        ...item,
        Media: { ...item.Media, ID: asId(item.Media.ID) },
      })),
    });
  }, []);

  useEffect(() => {
    let cancelled = false;
    listMedia(page)
      .then((res) => {
        if (cancelled) {
          return;
        }
        applyList(res);
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
  }, [page, applyList]);

  function retry() {
    setLoading(true);
    setError("");
    listMedia(page)
      .then((res) => {
        applyList(res);
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
  const pageIds = useMemo(
    () => items.map((item) => String(item.Media.ID)),
    [items],
  );
  const selectedCount = selected.size;
  const allPageSelected =
    pageIds.length > 0 && pageIds.every((id) => selected.has(id));

  function toggleSelect(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
    setActionError("");
  }

  function toggleSelectAllPage() {
    setSelected((prev) => {
      if (pageIds.every((id) => prev.has(id))) {
        const next = new Set(prev);
        for (const id of pageIds) {
          next.delete(id);
        }
        return next;
      }
      const next = new Set(prev);
      for (const id of pageIds) {
        next.add(id);
      }
      return next;
    });
    setActionError("");
  }

  function clearSelection() {
    setSelected(new Set());
    setActionError("");
  }

  async function deleteSelected() {
    const ids = [...selected];
    if (ids.length === 0) {
      return;
    }
    if (
      !window.confirm(
        `Delete ${ids.length} selected ${ids.length === 1 ? "video" : "videos"}? This cannot be undone.`,
      )
    ) {
      return;
    }
    setDeleting(true);
    setActionError("");
    const results = await Promise.allSettled(ids.map((id) => deleteMedia(id)));
    const failed = results.filter((r) => r.status === "rejected").length;
    const deletedIds = new Set(
      ids.filter((_, i) => results[i]?.status === "fulfilled"),
    );
    const remaining = (data?.Media ?? []).filter(
      (item) => !deletedIds.has(String(item.Media.ID)),
    );

    setData((prev) => {
      if (!prev) {
        return prev;
      }
      return {
        Media: (prev.Media ?? []).filter(
          (item) => !deletedIds.has(String(item.Media.ID)),
        ),
        Total: Math.max(0, prev.Total - deletedIds.size),
      };
    });
    setSelected((prev) => {
      const next = new Set(prev);
      for (const id of deletedIds) {
        next.delete(id);
      }
      return next;
    });
    setDeleting(false);

    if (failed > 0) {
      setActionError(
        `Deleted ${deletedIds.size}, but ${failed} failed. Try again.`,
      );
    }

    if (deletedIds.size > 0 && remaining.length === 0) {
      setLoading(true);
      try {
        const res = await listMedia(page);
        applyList(res);
        setSelected(new Set());
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Could not load media");
      } finally {
        setLoading(false);
      }
    }
  }

  return (
    <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-6 sm:px-6">
      <div className="mb-5 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-lg font-semibold tracking-tight">Library</h1>
          <p className="text-sm text-muted">
            {loading
              ? "Loading…"
              : `${total} ${total === 1 ? "video" : "videos"}`}
          </p>
        </div>
        {!loading && totalPages > 1 ? (
          <p className="text-sm text-muted">
            Page {page + 1} of {totalPages}
          </p>
        ) : null}
      </div>

      {!loading && items.length > 0 ? (
        <div className="mb-4 flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={toggleSelectAllPage}
            className="rounded-md px-2.5 py-1.5 text-sm text-muted outline-none hover:bg-surface-hover hover:text-foreground focus-visible:ring-2 focus-visible:ring-accent"
          >
            {allPageSelected ? "Deselect page" : "Select page"}
          </button>
          {selectedCount > 0 ? (
            <>
              <span className="font-mono text-sm tabular-nums text-muted">
                {selectedCount} selected
              </span>
              <button
                type="button"
                onClick={clearSelection}
                className="rounded-md px-2.5 py-1.5 text-sm text-muted outline-none hover:bg-surface-hover hover:text-foreground focus-visible:ring-2 focus-visible:ring-accent"
              >
                Clear
              </button>
              <button
                type="button"
                disabled={deleting}
                onClick={() => void deleteSelected()}
                className="rounded-md bg-danger/15 px-2.5 py-1.5 text-sm text-danger outline-none hover:bg-danger/25 focus-visible:ring-2 focus-visible:ring-danger disabled:opacity-60"
              >
                {deleting ? "Deleting…" : "Delete selected"}
              </button>
            </>
          ) : null}
        </div>
      ) : null}

      {actionError ? (
        <p className="mb-3 text-sm text-danger">{actionError}</p>
      ) : null}

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
            {items.map((item) => {
              const id = String(item.Media.ID);
              return (
                <MediaCard
                  key={id}
                  item={item}
                  page={page}
                  selected={selected.has(id)}
                  onToggleSelect={toggleSelect}
                />
              );
            })}
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
