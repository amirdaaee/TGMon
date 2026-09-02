"use client";

import { ApiError, deleteMedia, readMedia } from "@/lib/api";
import { downloadUrl } from "@/lib/config";
import { asId, formatDuration, mediaTitle } from "@/lib/format";
import type { MediaReadRes } from "@/lib/types";
import Link from "next/link";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { MediaLike } from "./MediaLike";
import { MediaRating } from "./MediaRating";
import { RandomMediaButton } from "./RandomMediaButton";
import { VideoPlayer } from "./VideoPlayer";
import { WatchPlaylist } from "./WatchPlaylist";

export function WatchView() {
  const params = useParams<{ id: string }>();
  const searchParams = useSearchParams();
  const rawId = params.id;
  const id = decodeURIComponent(
    Array.isArray(rawId) ? (rawId[0] ?? "") : (rawId ?? ""),
  );
  const from = searchParams.get("from");
  if (!id) {
    return null;
  }
  return <WatchInner key={id} id={id} from={from} />;
}

function WatchInner({ id, from }: { id: string; from: string | null }) {
  const router = useRouter();
  const backHref =
    from != null && from !== "" && /^\d+$/.test(from) ? `/?page=${from}` : "/";
  const fromQs =
    from != null && from !== "" ? `?from=${encodeURIComponent(from)}` : "";

  const [data, setData] = useState<MediaReadRes | null>(null);
  const [error, setError] = useState<{ status: number; message: string } | null>(
    null,
  );
  const [loading, setLoading] = useState(true);
  const [deleting, setDeleting] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    readMedia(id)
      .then((res) => {
        if (cancelled) {
          return;
        }
        setData({
          ...res,
          Media: { ...res.Media, ID: asId(res.Media.ID) },
          pervID: res.pervID ? asId(res.pervID) : null,
          nextID: res.nextID ? asId(res.nextID) : null,
        });
        setError(null);
        setLoading(false);
      })
      .catch((err: unknown) => {
        if (cancelled) {
          return;
        }
        setData(null);
        if (err instanceof ApiError) {
          setError({ status: err.status, message: err.message });
        } else {
          setError({
            status: 0,
            message: err instanceof Error ? err.message : "Could not load video",
          });
        }
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  const prevId = data?.pervID ?? null;
  const nextId = data?.nextID ?? null;

  const goPrev = useCallback(() => {
    if (prevId) {
      router.push(`/watch/${encodeURIComponent(prevId)}${fromQs}`);
    }
  }, [prevId, fromQs, router]);

  const goNext = useCallback(() => {
    if (nextId) {
      router.push(`/watch/${encodeURIComponent(nextId)}${fromQs}`);
    }
  }, [nextId, fromQs, router]);

  const handleDelete = useCallback(async () => {
    if (
      !window.confirm(
        "Delete this video? This cannot be undone.",
      )
    ) {
      return;
    }
    setDeleting(true);
    setActionError(null);
    try {
      await deleteMedia(id);
      router.push(backHref);
    } catch (err: unknown) {
      setActionError(
        err instanceof ApiError
          ? err.message
          : err instanceof Error
            ? err.message
            : "Could not delete video",
      );
      setDeleting(false);
    }
  }, [id, router, backHref]);

  if (loading) {
    return (
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6 sm:px-6">
        <div className="lg:grid lg:grid-cols-[minmax(0,1fr)_18rem] lg:gap-6">
          <div>
            <div className="aspect-video animate-pulse rounded-xl bg-zinc-800" />
            <div className="mt-4 h-5 w-1/3 animate-pulse rounded bg-zinc-800" />
          </div>
          <div className="mt-6 hidden space-y-2 lg:mt-0 lg:block">
            <div className="h-4 w-20 animate-pulse rounded bg-zinc-800" />
            <div className="h-16 animate-pulse rounded-lg bg-zinc-800" />
            <div className="h-16 animate-pulse rounded-lg bg-zinc-800" />
          </div>
        </div>
      </main>
    );
  }

  if (error || !data) {
    const notFound = error && (error.status === 400 || error.status === 404);
    return (
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-16 sm:px-6">
        <h1 className="text-lg font-semibold">
          {notFound ? "Video not found" : "Could not load video"}
        </h1>
        <p className="mt-2 text-sm text-muted">{error?.message}</p>
        <Link
          href="/"
          className="mt-6 inline-block text-sm text-accent hover:underline focus-visible:outline-2 focus-visible:outline-accent"
        >
          Back to library
        </Link>
      </main>
    );
  }

  const title = mediaTitle(data.Media);
  const duration = data.Media.Meta?.Duration ?? 0;

  return (
    <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6 sm:px-6">
      <Link
        href={backHref}
        className="mb-4 inline-flex items-center gap-1 text-sm text-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
      >
        ← Library
      </Link>
      <div className="lg:grid lg:grid-cols-[minmax(0,1fr)_18rem] lg:items-start lg:gap-6">
        <div>
          <VideoPlayer
            key={data.Media.ID}
            media={data.Media}
            checkpoint={data.Meta?.Checkpoint ?? 0}
            prevId={prevId}
            nextId={nextId}
            onPrev={goPrev}
            onNext={goNext}
          />
          <div className="mt-4 flex flex-wrap items-baseline justify-between gap-2">
            <h1 className="text-lg font-semibold tracking-tight">{title}</h1>
            {duration > 0 ? (
              <p className="font-mono text-sm text-muted">
                {formatDuration(duration)}
              </p>
            ) : null}
          </div>
          <div className="mt-3 flex flex-wrap items-center gap-4">
            <MediaRating mediaId={data.Media.ID} score={data.Meta?.Score ?? 0} />
            <MediaLike mediaId={data.Media.ID} likes={data.Meta?.Likes ?? 0} />
            <div className="ml-auto flex items-center gap-1">
              <RandomMediaButton
                variant="icon"
                fromQs={fromQs}
                currentId={id}
                onError={(msg) => setActionError(msg || null)}
              />
              <a
                href={downloadUrl(data.Media.ID)}
                download
                aria-label="Download video"
                title="Download"
                className="inline-flex h-8 w-8 items-center justify-center rounded text-zinc-500 outline-none hover:text-accent focus-visible:text-accent focus-visible:ring-2 focus-visible:ring-accent motion-safe:transition-colors"
              >
                <DownloadIcon />
              </a>
              <button
                type="button"
                disabled={deleting}
                onClick={() => void handleDelete()}
                aria-label="Delete video"
                title="Delete"
                className="inline-flex h-8 w-8 items-center justify-center rounded text-zinc-500 outline-none hover:text-danger focus-visible:text-danger focus-visible:ring-2 focus-visible:ring-danger disabled:opacity-60 motion-safe:transition-colors"
              >
                <TrashIcon />
              </button>
            </div>
          </div>
          {actionError ? (
            <p className="mt-2 text-sm text-danger">{actionError}</p>
          ) : null}
        </div>
        <WatchPlaylist
          currentId={id}
          current={data}
          fromQs={fromQs}
          className="mt-8 lg:mt-0 lg:sticky lg:top-4 lg:max-h-[calc(100vh-2rem)] lg:overflow-y-auto"
        />
      </div>
    </main>
  );
}

function DownloadIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-5 w-5"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      aria-hidden
    >
      <path d="M12 3v12m0 0 4-4m-4 4-4-4M5 21h14" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-5 w-5"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      aria-hidden
    >
      <path d="M4 7h16M10 11v6M14 11v6M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
    </svg>
  );
}
