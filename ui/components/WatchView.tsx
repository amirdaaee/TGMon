"use client";

import Link from "next/link";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { ApiError, readMedia } from "@/lib/api";
import { asId, formatDuration, mediaTitle } from "@/lib/format";
import type { MediaReadRes } from "@/lib/types";
import { MediaLike } from "./MediaLike";
import { MediaRating } from "./MediaRating";
import { VideoPlayer } from "./VideoPlayer";

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

  if (loading) {
    return (
      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-6 sm:px-6">
        <div className="aspect-video animate-pulse rounded-xl bg-zinc-800" />
        <div className="mt-4 h-5 w-1/3 animate-pulse rounded bg-zinc-800" />
      </main>
    );
  }

  if (error || !data) {
    const notFound = error && (error.status === 400 || error.status === 404);
    return (
      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-16 sm:px-6">
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
    <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-6 sm:px-6">
      <Link
        href={backHref}
        className="mb-4 inline-flex items-center gap-1 text-sm text-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
      >
        ← Library
      </Link>
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
          <p className="font-mono text-sm text-muted">{formatDuration(duration)}</p>
        ) : null}
      </div>
      <div className="mt-3 flex flex-wrap items-center gap-4">
        <MediaRating mediaId={data.Media.ID} score={data.Meta?.Score ?? 0} />
        <MediaLike mediaId={data.Media.ID} likes={data.Meta?.Likes ?? 0} />
      </div>
      <p className="mt-6 text-xs text-zinc-600">
        Double-tap sides to seek · Swipe to scrub · N/P prev/next video
      </p>
    </main>
  );
}
