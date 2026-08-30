"use client";

import { useCallback, useEffect, useState } from "react";
import { patchMediaMeta } from "@/lib/api";

type Props = {
  mediaId: string;
  likes: number;
};

export function MediaLike({ mediaId, likes: initialLikes }: Props) {
  const [likes, setLikes] = useState(initialLikes);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setLikes(initialLikes);
  }, [initialLikes, mediaId]);

  const like = useCallback(() => {
    const prev = likes;
    const next = likes + 1;
    setLikes(next);
    setSaving(true);
    void patchMediaMeta(mediaId, { Likes: next })
      .catch(() => {
        setLikes(prev);
      })
      .finally(() => {
        setSaving(false);
      });
  }, [mediaId, likes]);

  return (
    <button
      type="button"
      disabled={saving}
      onClick={like}
      aria-label="Like this video"
      className="inline-flex items-center gap-1.5 rounded text-zinc-500 outline-none hover:text-accent focus-visible:text-accent focus-visible:ring-2 focus-visible:ring-accent disabled:opacity-60 motion-safe:transition-colors"
    >
      <HeartIcon filled={likes > 0} />
      <span className="font-mono text-xs tabular-nums">{likes}</span>
    </button>
  );
}

function HeartIcon({ filled }: { filled: boolean }) {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-5 w-5"
      fill={filled ? "currentColor" : "none"}
      stroke="currentColor"
      strokeWidth="1.5"
      aria-hidden
    >
      <path d="M12 21s-7.2-4.6-9.4-9.1C1.2 8.7 3.1 5.5 6.6 5.2c1.8-.2 3.4.7 4.4 2.1 1-1.4 2.6-2.3 4.4-2.1 3.5.3 5.4 3.5 4 6.7C19.2 16.4 12 21 12 21Z" />
    </svg>
  );
}
