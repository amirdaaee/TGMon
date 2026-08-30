"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { readMedia } from "@/lib/api";
import { assetUrl } from "@/lib/config";
import { asId, formatDuration, mediaTitle } from "@/lib/format";
import type { MediaReadRes, MediaWithMeta } from "@/lib/types";
import { PlaceholderThumb } from "./PlaceholderThumb";

type Props = {
  currentId: string;
  current: MediaReadRes;
  fromQs: string;
  className?: string;
};

async function walkNeighbors(
  startId: string | null,
  direction: "prev" | "next",
  steps: number,
  signal: { cancelled: boolean },
): Promise<MediaWithMeta[]> {
  const out: MediaWithMeta[] = [];
  let nextId = startId;
  for (let i = 0; i < steps && nextId; i++) {
    if (signal.cancelled) {
      return out;
    }
    try {
      const res = await readMedia(nextId);
      if (signal.cancelled) {
        return out;
      }
      const item: MediaWithMeta = {
        Media: { ...res.Media, ID: asId(res.Media.ID) },
        Meta: res.Meta,
      };
      out.push(item);
      nextId =
        direction === "prev"
          ? res.pervID
            ? asId(res.pervID)
            : null
          : res.nextID
            ? asId(res.nextID)
            : null;
    } catch {
      break;
    }
  }
  return out;
}

export function WatchPlaylist({
  currentId,
  current,
  fromQs,
  className = "",
}: Props) {
  const [items, setItems] = useState<MediaWithMeta[]>([
    { Media: current.Media, Meta: current.Meta },
  ]);
  const [loading, setLoading] = useState(true);
  const listRef = useRef<HTMLDivElement>(null);
  const currentRef = useRef<HTMLAnchorElement>(null);

  useEffect(() => {
    const signal = { cancelled: false };
    const currentItem = { Media: current.Media, Meta: current.Meta };
    const prevId = current.pervID;
    const nextId = current.nextID;
    setLoading(true);
    setItems([currentItem]);

    void (async () => {
      const [prevItems, nextItems] = await Promise.all([
        walkNeighbors(prevId, "prev", 2, signal),
        walkNeighbors(nextId, "next", 2, signal),
      ]);
      if (signal.cancelled) {
        return;
      }
      setItems([...prevItems.reverse(), currentItem, ...nextItems]);
      setLoading(false);
    })();

    return () => {
      signal.cancelled = true;
    };
  }, [currentId, current]);

  useEffect(() => {
    if (loading) {
      return;
    }
    currentRef.current?.scrollIntoView({
      block: "nearest",
      behavior: "smooth",
    });
  }, [loading, currentId, items]);

  const onlyCurrent = items.length <= 1;

  return (
    <aside className={className} aria-label="Nearby videos">
      <h2 className="mb-3 text-sm font-medium text-zinc-300">Nearby</h2>
      <div ref={listRef} className="flex flex-col gap-1.5">
        {loading && onlyCurrent ? (
          <>
            <PlaylistSkeleton />
            <PlaylistSkeleton />
          </>
        ) : null}
        {items.map((item) => {
          const id = String(item.Media.ID);
          const isCurrent = id === currentId;
          const title = mediaTitle(item.Media);
          const duration = item.Media.Meta?.Duration ?? 0;
          const thumb = assetUrl(item.Media.Thumbnail);
          const score = item.Meta?.Score ?? 0;
          const likes = item.Meta?.Likes ?? 0;
          return (
            <Link
              key={id}
              ref={isCurrent ? currentRef : undefined}
              href={`/watch/${encodeURIComponent(id)}${fromQs}`}
              aria-current={isCurrent ? "true" : undefined}
              className={`flex gap-2.5 rounded-lg p-1.5 outline-none focus-visible:ring-2 focus-visible:ring-accent motion-safe:transition-colors ${
                isCurrent
                  ? "bg-accent/15 ring-1 ring-accent"
                  : "hover:bg-surface-hover"
              }`}
            >
              <div className="relative w-28 shrink-0 overflow-hidden rounded-md bg-zinc-900 aspect-video">
                {thumb ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={thumb}
                    alt=""
                    loading="lazy"
                    className="h-full w-full object-cover"
                  />
                ) : (
                  <PlaceholderThumb
                    duration={duration}
                    className="h-full w-full"
                  />
                )}
                {duration > 0 ? (
                  <span className="absolute right-1 bottom-1 rounded bg-black/75 px-1 py-0.5 font-mono text-[10px] text-zinc-100">
                    {formatDuration(duration)}
                  </span>
                ) : null}
              </div>
              <div className="min-w-0 flex-1 py-0.5">
                <p
                  className={`line-clamp-2 text-sm leading-snug ${
                    isCurrent ? "font-medium text-accent" : "text-zinc-100"
                  }`}
                  title={title}
                >
                  {title}
                </p>
                {score > 0 || likes > 0 ? (
                  <div className="mt-1.5 flex items-center gap-2 text-[11px] text-muted">
                    {score > 0 ? (
                      <span className="inline-flex items-center gap-0.5 text-accent">
                        <StarIcon />
                        <span className="font-mono tabular-nums">{score}</span>
                      </span>
                    ) : null}
                    {likes > 0 ? (
                      <span className="inline-flex items-center gap-0.5">
                        <HeartIcon />
                        <span className="font-mono tabular-nums">{likes}</span>
                      </span>
                    ) : null}
                  </div>
                ) : null}
              </div>
            </Link>
          );
        })}
        {!loading && onlyCurrent ? (
          <p className="px-1 py-2 text-xs text-muted">No nearby videos</p>
        ) : null}
      </div>
    </aside>
  );
}

function PlaylistSkeleton() {
  return (
    <div className="flex gap-2.5 rounded-lg p-1.5">
      <div className="aspect-video w-28 shrink-0 animate-pulse rounded-md bg-zinc-800" />
      <div className="flex-1 space-y-2 py-1">
        <div className="h-3 w-4/5 animate-pulse rounded bg-zinc-800" />
        <div className="h-3 w-2/5 animate-pulse rounded bg-zinc-800" />
      </div>
    </div>
  );
}

function StarIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-3 w-3 fill-current" aria-hidden>
      <path d="m12 2.5 2.7 5.5 6.1.9-4.4 4.3 1 6.1L12 16.4 6.6 19.3l1-6.1-4.4-4.3 6.1-.9L12 2.5Z" />
    </svg>
  );
}

function HeartIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-3 w-3 fill-current" aria-hidden>
      <path d="M12 21s-7.2-4.6-9.4-9.1C1.2 8.7 3.1 5.5 6.6 5.2c1.8-.2 3.4.7 4.4 2.1 1-1.4 2.6-2.3 4.4-2.1 3.5.3 5.4 3.5 4 6.7C19.2 16.4 12 21 12 21Z" />
    </svg>
  );
}
