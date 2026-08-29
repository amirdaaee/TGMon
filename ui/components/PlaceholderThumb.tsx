import { formatDuration } from "@/lib/format";

export function PlaceholderThumb({
  duration,
  className = "",
}: {
  duration?: number;
  className?: string;
}) {
  return (
    <div
      className={`flex items-center justify-center bg-gradient-to-br from-zinc-800 to-zinc-950 ${className}`}
    >
      {duration && duration > 0 ? (
        <span className="font-mono text-sm text-zinc-400">
          {formatDuration(duration)}
        </span>
      ) : (
        <span className="text-2xl text-zinc-600" aria-hidden>
          ▶
        </span>
      )}
    </div>
  );
}
