"use client";

import { ApiError, getRandomMedia } from "@/lib/api";
import { asId } from "@/lib/format";
import { useRouter } from "next/navigation";
import { useState } from "react";

type Props = {
  variant: "label" | "icon";
  fromQs?: string;
  currentId?: string;
  disabled?: boolean;
  onError?: (message: string) => void;
};

function errorMessage(err: unknown): string {
  if (err instanceof ApiError && err.status === 404) {
    return "No videos yet";
  }
  if (err instanceof Error) {
    return err.message;
  }
  return "Could not pick a video";
}

export function RandomMediaButton({
  variant,
  fromQs = "",
  currentId,
  disabled,
  onError,
}: Props) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function pickRandom() {
    setLoading(true);
    setError("");
    onError?.("");
    try {
      let id = asId((await getRandomMedia()).MediaID);
      if (currentId && id && id === currentId) {
        const again = asId((await getRandomMedia()).MediaID);
        if (again) {
          id = again;
        }
      }
      if (!id) {
        throw new Error("Could not pick a video");
      }
      router.push(`/watch/${encodeURIComponent(id)}${fromQs}`);
      if (id === currentId) {
        setLoading(false);
      }
    } catch (err: unknown) {
      const message = errorMessage(err);
      if (onError) {
        onError(message);
      } else {
        setError(message);
      }
      setLoading(false);
    }
  }

  const isDisabled = disabled || loading;

  return (
    <div className={variant === "label" ? "flex flex-col items-end gap-1" : undefined}>
      {variant === "label" ? (
        <button
          type="button"
          disabled={isDisabled}
          onClick={() => void pickRandom()}
          className="rounded-md px-2.5 py-1.5 text-sm text-muted outline-none hover:bg-surface-hover hover:text-foreground focus-visible:ring-2 focus-visible:ring-accent disabled:opacity-60"
        >
          {loading ? "Picking…" : "Random"}
        </button>
      ) : (
        <button
          type="button"
          disabled={isDisabled}
          onClick={() => void pickRandom()}
          aria-label="Watch random video"
          title="Random"
          className="inline-flex h-8 w-8 items-center justify-center rounded text-zinc-500 outline-none hover:text-accent focus-visible:text-accent focus-visible:ring-2 focus-visible:ring-accent disabled:opacity-60 motion-safe:transition-colors"
        >
          <ShuffleIcon />
        </button>
      )}
      {error ? <p className="text-sm text-danger">{error}</p> : null}
    </div>
  );
}

function ShuffleIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-5 w-5"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <path d="m18 14 4 4-4 4M18 2l4 4-4 4" />
      <path d="M2 18h2a4 4 0 0 0 3.3-1.7l5.4-8.6A4 4 0 0 1 16 6h6" />
      <path d="M2 6h2a4 4 0 0 1 3.3 1.7l5.4 8.6A4 4 0 0 0 16 18h6" />
    </svg>
  );
}
