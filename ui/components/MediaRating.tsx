"use client";

import { useCallback, useEffect, useState } from "react";
import { patchMediaMeta } from "@/lib/api";

type Props = {
  mediaId: string;
  score: number;
};

export function MediaRating({ mediaId, score: initialScore }: Props) {
  const [score, setScore] = useState(initialScore);
  const [hover, setHover] = useState<number | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setScore(initialScore);
  }, [initialScore, mediaId]);

  const display = hover ?? score;

  const rate = useCallback(
    (value: number) => {
      const prev = score;
      setScore(value);
      setSaving(true);
      void patchMediaMeta(mediaId, { Score: value })
        .catch(() => {
          setScore(prev);
        })
        .finally(() => {
          setSaving(false);
        });
    },
    [mediaId, score],
  );

  return (
    <div className="flex items-center gap-2">
      <div
        role="group"
        aria-label="Rating"
        className="inline-flex items-center gap-0.5"
        onMouseLeave={() => setHover(null)}
      >
        {[1, 2, 3, 4, 5].map((value) => {
          const filled = display >= value;
          return (
            <button
              key={value}
              type="button"
              disabled={saving}
              onClick={() => rate(value)}
              onMouseEnter={() => setHover(value)}
              aria-label={`Rate ${value} out of 5`}
              aria-pressed={score >= value}
              className={`rounded p-0.5 outline-none focus-visible:ring-2 focus-visible:ring-accent disabled:opacity-60 motion-safe:transition-colors ${
                filled
                  ? "text-accent"
                  : "text-zinc-600 hover:text-accent focus-visible:text-accent"
              }`}
            >
              <StarIcon filled={filled} />
            </button>
          );
        })}
      </div>
      {score > 0 ? (
        <span className="font-mono text-xs tabular-nums text-muted">
          {score}/5
        </span>
      ) : (
        <span className="text-xs text-muted">Rate this video</span>
      )}
    </div>
  );
}

function StarIcon({ filled }: { filled: boolean }) {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-5 w-5"
      fill={filled ? "currentColor" : "none"}
      stroke="currentColor"
      strokeWidth="1.5"
      aria-hidden
    >
      <path d="m12 2.5 2.7 5.5 6.1.9-4.4 4.3 1 6.1L12 16.4 6.6 19.3l1-6.1-4.4-4.3 6.1-.9L12 2.5Z" />
    </svg>
  );
}
