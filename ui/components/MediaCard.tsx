import { assetUrl } from "@/lib/config";
import {
  formatDuration,
  formatFileSize,
  isResumable,
  mediaTitle,
} from "@/lib/format";
import type { MediaWithMeta } from "@/lib/types";
import Link from "next/link";
import { PlaceholderThumb } from "./PlaceholderThumb";

export function MediaCard({
  item,
  page,
  selected,
  onToggleSelect,
}: {
  item: MediaWithMeta;
  page: number;
  selected: boolean;
  onToggleSelect: (id: string) => void;
}) {
  const media = item.Media;
  const meta = item.Meta;
  const id = String(media.ID);
  const title = mediaTitle(media);
  const duration = media.Meta?.Duration ?? 0;
  const fileSize = formatFileSize(media.Meta?.FileSize ?? 0);
  const score = meta?.Score ?? 0;
  const likes = meta?.Likes ?? 0;
  const isFavorite = Boolean(meta?.IsFavorite);
  const checkpoint = meta?.Checkpoint ?? 0;
  const thumb = assetUrl(media.Thumbnail);
  const resumable = isResumable(checkpoint, duration);
  const progress = duration > 0 ? Math.min(1, checkpoint / duration) : 0;

  return (
    <article
      className={`group relative overflow-hidden rounded-xl bg-surface outline-none motion-safe:transition-[transform,box-shadow] motion-safe:hover:-translate-y-0.5 ${
        selected ? "ring-2 ring-accent" : ""
      }`}
    >
      <button
        type="button"
        aria-label={selected ? "Deselect video" : "Select video"}
        aria-pressed={selected}
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          onToggleSelect(id);
        }}
        className={`absolute top-2 left-2 z-20 flex h-7 w-7 items-center justify-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-accent ${
          selected
            ? "bg-accent text-accent-fg opacity-100"
            : "bg-black/75 text-white opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
        }`}
      >
        <CheckIcon />
      </button>
      <Link
        href={`/watch/${encodeURIComponent(id)}?from=${page}`}
        className="block outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-inset"
      >
        <div className="relative aspect-video overflow-hidden bg-zinc-900">
          {thumb ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={thumb}
              alt=""
              loading="lazy"
              className="h-full w-full object-cover"
            />
          ) : (
            <PlaceholderThumb duration={duration} className="h-full w-full" />
          )}
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/0 motion-safe:transition-colors group-hover:bg-black/35">
            <span className="flex h-11 w-11 items-center justify-center rounded-full bg-black/70 text-white opacity-0 group-hover:opacity-100 motion-safe:transition-opacity">
              <PlayIcon />
            </span>
          </div>
          {isFavorite ? (
            <span
              className="absolute top-2 right-2 flex h-7 w-7 items-center justify-center rounded-full bg-black/75 text-accent"
              title="Favorite"
              aria-label="Favorite"
            >
              <HeartIcon filled />
            </span>
          ) : null}
          {duration > 0 ? (
            <span className="absolute right-2 bottom-2 rounded bg-black/75 px-1.5 py-0.5 font-mono text-[11px] text-zinc-100">
              {formatDuration(duration)}
            </span>
          ) : null}
          {resumable ? (
            <div className="absolute inset-x-0 bottom-0 h-1 bg-zinc-800">
              <div
                className="h-full bg-accent"
                style={{ width: `${progress * 100}%` }}
              />
            </div>
          ) : null}
        </div>
        <div className="px-3 py-2.5">
          <h2
            className="truncate text-sm font-medium text-zinc-100"
            title={title}
          >
            {title}
          </h2>
          <div className="mt-1.5 flex items-center justify-between gap-2 text-xs text-muted">
            <span className="truncate">{fileSize}</span>
            {score > 0 || likes > 0 ? (
              <span className="inline-flex shrink-0 items-center gap-2">
                {score > 0 ? (
                  <span className="inline-flex items-center gap-0.5 text-accent">
                    <StarIcon />
                    <span className="font-mono tabular-nums">{score}</span>
                  </span>
                ) : null}
                {likes > 0 ? (
                  <span className="inline-flex items-center gap-0.5">
                    <HeartIcon filled />
                    <span className="font-mono tabular-nums">{likes}</span>
                  </span>
                ) : null}
              </span>
            ) : null}
          </div>
        </div>
      </Link>
    </article>
  );
}

function CheckIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current" aria-hidden>
      <path d="m9.5 16.2-3.7-3.7 1.4-1.4 2.3 2.3 5.3-5.3 1.4 1.4-6.7 6.7Z" />
    </svg>
  );
}

function PlayIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden>
      <path d="M8 5.14v13.72L19 12 8 5.14Z" />
    </svg>
  );
}

function HeartIcon({ filled }: { filled?: boolean }) {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-3.5 w-3.5"
      fill={filled ? "currentColor" : "none"}
      stroke="currentColor"
      strokeWidth="2"
      aria-hidden
    >
      <path d="M12 21s-7.2-4.6-9.4-9.1C1.2 8.7 3.1 5.5 6.6 5.2c1.8-.2 3.4.7 4.4 2.1 1-1.4 2.6-2.3 4.4-2.1 3.5.3 5.4 3.5 4 6.7C19.2 16.4 12 21 12 21Z" />
    </svg>
  );
}

function StarIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-3 w-3 fill-current" aria-hidden>
      <path d="m12 2.5 2.7 5.5 6.1.9-4.4 4.3 1 6.1L12 16.4 6.6 19.3l1-6.1-4.4-4.3 6.1-.9L12 2.5Z" />
    </svg>
  );
}
