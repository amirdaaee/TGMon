import Link from "next/link";
import { assetUrl } from "@/lib/config";
import { formatDuration, isResumable, mediaTitle } from "@/lib/format";
import type { MediaWithMeta } from "@/lib/types";
import { PlaceholderThumb } from "./PlaceholderThumb";

export function MediaCard({
  item,
  page,
}: {
  item: MediaWithMeta;
  page: number;
}) {
  const media = item.Media;
  const meta = item.Meta;
  const id = String(media.ID);
  const title = mediaTitle(media);
  const duration = media.Meta?.Duration ?? 0;
  const checkpoint = meta?.Checkpoint ?? 0;
  const thumb = assetUrl(media.Thumbnail);
  const resumable = isResumable(checkpoint, duration);
  const progress = duration > 0 ? Math.min(1, checkpoint / duration) : 0;

  return (
    <Link
      href={`/watch/${encodeURIComponent(id)}?from=${page}`}
      className="group overflow-hidden rounded-xl bg-surface outline-none ring-accent focus-visible:ring-2 motion-safe:transition-transform motion-safe:hover:-translate-y-0.5"
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
        <h2 className="line-clamp-2 text-sm font-medium leading-snug text-zinc-100">
          {title}
        </h2>
        {resumable ? (
          <p className="mt-1 text-xs text-accent">Continue watching</p>
        ) : null}
      </div>
    </Link>
  );
}

function PlayIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden>
      <path d="M8 5.14v13.72L19 12 8 5.14Z" />
    </svg>
  );
}
