"use client";

import "@vidstack/react/player/styles/default/theme.css";
import "@vidstack/react/player/styles/default/layouts/video.css";

import {
  MediaPlayer,
  MediaProvider,
  Track,
  type MediaPlayerInstance,
} from "@vidstack/react";
import {
  DefaultVideoLayout,
  defaultLayoutIcons,
} from "@vidstack/react/player/layouts/default";
import { useCallback, useEffect, useRef, useState } from "react";
import { patchMediaMeta } from "@/lib/api";
import { assetUrl, streamUrl } from "@/lib/config";
import { checkpointToSave, isResumable } from "@/lib/format";
import type { MediaFileDoc } from "@/lib/types";
import { rewriteSpriteVttPaths } from "@/lib/vtt";

const CHECKPOINT_DEBOUNCE_MS = 5000;

type Props = {
  media: MediaFileDoc;
  checkpoint: number;
  prevId: string | null;
  nextId: string | null;
  onPrev: () => void;
  onNext: () => void;
};

export function VideoPlayer({
  media,
  checkpoint,
  prevId,
  nextId,
  onPrev,
  onNext,
}: Props) {
  const id = String(media.ID);
  const playerRef = useRef<MediaPlayerInstance>(null);
  const lastPlayedSent = useRef(false);
  const saveTimer = useRef<number | null>(null);
  const lastSaved = useRef<number | null>(null);
  const durationRef = useRef(0);
  const timeRef = useRef(0);
  const dirtyRef = useRef(false);
  const resumedRef = useRef(false);

  const [thumbnails, setThumbnails] = useState<string | undefined>();

  const poster = assetUrl(media.Thumbnail);
  const spriteUrl = assetUrl(media.Sprite);
  const srtUrl = assetUrl(media.Srt);
  const src = streamUrl(id);

  const persistCheckpoint = useCallback(() => {
    if (!dirtyRef.current) {
      return;
    }
    const t = timeRef.current;
    const d = durationRef.current;
    const value = checkpointToSave(t, d);
    if (lastSaved.current === value) {
      return;
    }
    lastSaved.current = value;
    void patchMediaMeta(id, { Checkpoint: value }).catch(() => {
      lastSaved.current = null;
    });
  }, [id]);

  const queueCheckpoint = useCallback(() => {
    if (saveTimer.current) {
      window.clearTimeout(saveTimer.current);
    }
    saveTimer.current = window.setTimeout(() => {
      persistCheckpoint();
    }, CHECKPOINT_DEBOUNCE_MS);
  }, [persistCheckpoint]);

  useEffect(() => {
    const onHide = () => persistCheckpoint();
    window.addEventListener("pagehide", onHide);
    return () => {
      window.removeEventListener("pagehide", onHide);
      if (saveTimer.current) {
        window.clearTimeout(saveTimer.current);
      }
      persistCheckpoint();
    };
  }, [persistCheckpoint]);

  useEffect(() => {
    const vttUrl = assetUrl(media.Vtt);
    if (!vttUrl) {
      setThumbnails(undefined);
      return;
    }

    let blobUrl: string | null = null;
    let cancelled = false;

    fetch(vttUrl)
      .then((res) => (res.ok ? res.text() : Promise.reject()))
      .then((text) => {
        if (cancelled) {
          return;
        }
        const rewritten = rewriteSpriteVttPaths(text, spriteUrl);
        blobUrl = URL.createObjectURL(
          new Blob([rewritten], { type: "text/vtt" }),
        );
        setThumbnails(blobUrl);
      })
      .catch(() => {
        if (!cancelled) {
          setThumbnails(undefined);
        }
      });

    return () => {
      cancelled = true;
      if (blobUrl) {
        URL.revokeObjectURL(blobUrl);
      }
    };
  }, [media.Vtt, spriteUrl]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      if (target && ["INPUT", "TEXTAREA"].includes(target.tagName)) {
        return;
      }
      if (e.key === "n" || e.key === "N") {
        if (nextId) {
          e.preventDefault();
          onNext();
        }
      } else if (e.key === "p" || e.key === "P") {
        if (prevId) {
          e.preventDefault();
          onPrev();
        }
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [nextId, prevId, onNext, onPrev]);

  return (
    <div className="overflow-hidden rounded-xl bg-black ring-accent focus-within:ring-2">
      <MediaPlayer
        ref={playerRef}
        src={src}
        poster={poster ?? undefined}
        viewType="video"
        crossOrigin
        playsInline
        fullscreenOrientation="landscape"
        className="aspect-video w-full bg-black"
        onLoadedMetadata={() => {
          const player = playerRef.current;
          if (!player) {
            return;
          }
          const dur = Number.isFinite(player.state.duration)
            ? player.state.duration
            : (media.Meta?.Duration ?? 0);
          durationRef.current = dur;
        }}
        onCanPlay={() => {
          if (resumedRef.current) {
            return;
          }
          const player = playerRef.current;
          if (!player) {
            return;
          }
          const dur =
            durationRef.current ||
            (Number.isFinite(player.state.duration)
              ? player.state.duration
              : (media.Meta?.Duration ?? 0));
          durationRef.current = dur;
          if (isResumable(checkpoint, dur)) {
            player.currentTime = checkpoint;
            timeRef.current = checkpoint;
          }
          resumedRef.current = true;
        }}
        onPlay={() => {
          if (!lastPlayedSent.current) {
            lastPlayedSent.current = true;
            void patchMediaMeta(id, {
              LastPlayedAt: new Date().toISOString(),
            }).catch(() => {
              lastPlayedSent.current = false;
            });
          }
        }}
        onPause={() => {
          persistCheckpoint();
        }}
        onTimeUpdate={(detail) => {
          timeRef.current = detail.currentTime;
          dirtyRef.current = true;
          queueCheckpoint();
        }}
        onEnded={() => {
          persistCheckpoint();
          if (nextId) {
            onNext();
          }
        }}
      >
        <MediaProvider>
          {srtUrl ? (
            <Track
              src={srtUrl}
              kind="subtitles"
              label="Subtitles"
              language="und"
              type="srt"
              default
            />
          ) : null}
        </MediaProvider>
        <DefaultVideoLayout
          icons={defaultLayoutIcons}
          colorScheme="dark"
          thumbnails={thumbnails}
          seekStep={5}
          slots={{
            beforeMuteButton: (
              <>
                <NavBtn label="Previous video" disabled={!prevId} onClick={onPrev}>
                  <SkipPrevIcon />
                </NavBtn>
                <NavBtn label="Next video" disabled={!nextId} onClick={onNext}>
                  <SkipNextIcon />
                </NavBtn>
              </>
            ),
          }}
        />
      </MediaPlayer>
    </div>
  );
}

function NavBtn({
  label,
  onClick,
  disabled,
  children,
}: {
  label: string;
  onClick: () => void;
  disabled?: boolean;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
      className="vds-button flex h-9 w-9 items-center justify-center rounded-md text-white hover:bg-white/10 disabled:opacity-30"
    >
      {children}
    </button>
  );
}

function SkipPrevIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden>
      <path d="M6 6h2v12H6V6Zm3.5 6L18 18V6l-8.5 6Z" />
    </svg>
  );
}

function SkipNextIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden>
      <path d="M16 6h2v12h-2V6ZM6 18l8.5-6L6 6v12Z" />
    </svg>
  );
}
