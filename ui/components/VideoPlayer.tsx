"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { patchMediaMeta } from "@/lib/api";
import { assetUrl, streamUrl } from "@/lib/config";
import {
  checkpointToSave,
  formatDuration,
  isResumable,
} from "@/lib/format";
import type { MediaFileDoc } from "@/lib/types";
import { findSpriteCue, parseSpriteVtt, type SpriteCue } from "@/lib/vtt";
import { PlaceholderThumb } from "./PlaceholderThumb";

const CHECKPOINT_DEBOUNCE_MS = 5000;
const CONTROLS_HIDE_MS = 2500;

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
  const videoRef = useRef<HTMLVideoElement>(null);
  const shellRef = useRef<HTMLDivElement>(null);
  const pendingSeek = useRef<number | null>(null);
  const lastPlayedSent = useRef(false);
  const saveTimer = useRef<number | null>(null);
  const hideTimer = useRef<number | null>(null);
  const lastSaved = useRef<number | null>(null);
  const durationRef = useRef(0);
  const timeRef = useRef(0);
  const dirtyRef = useRef(false);

  const [paused, setPaused] = useState(true);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(media.Meta?.Duration ?? 0);
  const [showControls, setShowControls] = useState(true);
  const [buffering, setBuffering] = useState(false);
  const [muted, setMuted] = useState(false);
  const [cues, setCues] = useState<SpriteCue[]>([]);
  const [hasPlayed, setHasPlayed] = useState(false);

  const poster = assetUrl(media.Thumbnail);
  const spriteUrl = assetUrl(media.Sprite);
  const src = streamUrl(id);

  useEffect(() => {
    durationRef.current = duration;
  }, [duration]);

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
      return;
    }
    let cancelled = false;
    fetch(vttUrl)
      .then((res) => (res.ok ? res.text() : Promise.reject()))
      .then((text) => {
        if (!cancelled) {
          setCues(parseSpriteVtt(text));
        }
      })
      .catch(() => {
        if (!cancelled) {
          setCues([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [media.Vtt]);

  function scheduleHide() {
    if (hideTimer.current) {
      window.clearTimeout(hideTimer.current);
    }
    hideTimer.current = window.setTimeout(() => {
      if (videoRef.current && !videoRef.current.paused) {
        setShowControls(false);
      }
    }, CONTROLS_HIDE_MS);
  }

  function revealControls() {
    setShowControls(true);
    scheduleHide();
  }

  async function applyPendingSeek() {
    const v = videoRef.current;
    if (!v || pendingSeek.current == null) {
      return;
    }
    const t = pendingSeek.current;
    pendingSeek.current = null;
    if (Math.abs(v.currentTime - t) < 0.3) {
      return;
    }
    await new Promise<void>((resolve) => {
      const onSeeked = () => {
        v.removeEventListener("seeked", onSeeked);
        resolve();
      };
      v.addEventListener("seeked", onSeeked);
      v.currentTime = t;
    });
  }

  async function togglePlay() {
    const v = videoRef.current;
    if (!v) {
      return;
    }
    if (v.paused) {
      await applyPendingSeek();
      await v.play();
    } else {
      v.pause();
    }
  }

  function seekBy(delta: number) {
    const v = videoRef.current;
    if (!v) {
      return;
    }
    pendingSeek.current = null;
    const d = v.duration || duration;
    v.currentTime = Math.min(Math.max(0, v.currentTime + delta), d || v.currentTime + delta);
    revealControls();
  }

  function seekTo(time: number) {
    const v = videoRef.current;
    if (!v) {
      return;
    }
    pendingSeek.current = null;
    v.currentTime = time;
    timeRef.current = time;
    dirtyRef.current = true;
    setCurrentTime(time);
    queueCheckpoint();
  }

  function toggleFullscreen() {
    const el = shellRef.current;
    if (!el) {
      return;
    }
    if (document.fullscreenElement) {
      void document.exitFullscreen();
    } else {
      void el.requestFullscreen();
    }
  }

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      if (target && ["INPUT", "TEXTAREA"].includes(target.tagName)) {
        return;
      }
      const key = e.key;
      if (key === " " || key === "k" || key === "K") {
        e.preventDefault();
        void togglePlay();
      } else if (key === "ArrowLeft") {
        e.preventDefault();
        seekBy(e.shiftKey ? -15 : -5);
      } else if (key === "ArrowRight") {
        e.preventDefault();
        seekBy(e.shiftKey ? 15 : 5);
      } else if (key === "f" || key === "F") {
        e.preventDefault();
        toggleFullscreen();
      } else if (key === "m" || key === "M") {
        const v = videoRef.current;
        if (v) {
          v.muted = !v.muted;
          setMuted(v.muted);
        }
      } else if (key === "n" || key === "N") {
        if (nextId) {
          onNext();
        }
      } else if (key === "p" || key === "P") {
        if (prevId) {
          onPrev();
        }
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [nextId, prevId, onNext, onPrev, duration]);

  const d = duration || 0;

  return (
    <div
      ref={shellRef}
      className="relative overflow-hidden rounded-xl bg-black outline-none focus-visible:ring-2 focus-visible:ring-accent"
      tabIndex={0}
      onMouseMove={revealControls}
      onMouseLeave={() => {
        if (videoRef.current && !videoRef.current.paused) {
          setShowControls(false);
        }
      }}
    >
      {!hasPlayed && !poster ? (
        <PlaceholderThumb
          duration={media.Meta?.Duration}
          className="pointer-events-none absolute inset-0 z-10"
        />
      ) : null}
      <video
        ref={videoRef}
        src={src}
        poster={poster ?? undefined}
        preload="metadata"
        playsInline
        className="aspect-video w-full bg-black"
        onClick={() => void togglePlay()}
        onLoadedMetadata={(e) => {
          const v = e.currentTarget;
          const dur = Number.isFinite(v.duration) ? v.duration : media.Meta?.Duration ?? 0;
          durationRef.current = dur;
          setDuration(dur);
          if (isResumable(checkpoint, dur)) {
            pendingSeek.current = checkpoint;
            if (!v.paused) {
              v.currentTime = checkpoint;
              pendingSeek.current = null;
            }
          }
        }}
        onPlay={() => {
          setPaused(false);
          setHasPlayed(true);
          scheduleHide();
        }}
        onPlaying={() => {
          setBuffering(false);
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
          setPaused(true);
          setShowControls(true);
          persistCheckpoint();
        }}
        onTimeUpdate={(e) => {
          const t = e.currentTarget.currentTime;
          timeRef.current = t;
          dirtyRef.current = true;
          setCurrentTime(t);
          queueCheckpoint();
        }}
        onWaiting={() => setBuffering(true)}
        onEnded={() => {
          persistCheckpoint();
          if (nextId) {
            onNext();
          }
        }}
      />

      {buffering ? (
        <div className="pointer-events-none absolute inset-0 z-20 flex items-center justify-center">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-white/30 border-t-white" />
        </div>
      ) : null}

      {paused ? (
        <button
          type="button"
          aria-label="Play"
          onClick={() => void togglePlay()}
          className="absolute top-1/2 left-1/2 z-20 flex h-16 w-16 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full bg-black/70 text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
        >
          <PlayIcon className="h-8 w-8" />
        </button>
      ) : null}

      <div
        className={`absolute inset-x-0 bottom-0 z-30 bg-gradient-to-t from-black/80 via-black/40 to-transparent px-3 pb-3 pt-10 transition-opacity ${
          showControls || paused ? "opacity-100" : "pointer-events-none opacity-0"
        }`}
      >
        <SeekBar
          duration={d}
          currentTime={currentTime}
          cues={cues}
          spriteUrl={spriteUrl}
          onSeek={seekTo}
        />
        <div className="mt-2 flex items-center gap-2">
          <IconBtn
            label={paused ? "Play" : "Pause"}
            onClick={() => void togglePlay()}
          >
            {paused ? <PlayIcon /> : <PauseIcon />}
          </IconBtn>
          <IconBtn label="Previous" disabled={!prevId} onClick={onPrev}>
            <SkipPrevIcon />
          </IconBtn>
          <IconBtn label="Next" disabled={!nextId} onClick={onNext}>
            <SkipNextIcon />
          </IconBtn>
          <span className="ml-1 font-mono text-xs text-zinc-200">
            {formatDuration(currentTime)} / {formatDuration(d)}
          </span>
          <div className="flex-1" />
          <IconBtn
            label={muted ? "Unmute" : "Mute"}
            onClick={() => {
              const v = videoRef.current;
              if (!v) {
                return;
              }
              v.muted = !v.muted;
              setMuted(v.muted);
            }}
          >
            {muted ? <MuteIcon /> : <VolumeIcon />}
          </IconBtn>
          <IconBtn label="Fullscreen" onClick={toggleFullscreen}>
            <FullIcon />
          </IconBtn>
        </div>
      </div>
    </div>
  );
}

function SeekBar({
  duration,
  currentTime,
  cues,
  spriteUrl,
  onSeek,
}: {
  duration: number;
  currentTime: number;
  cues: SpriteCue[];
  spriteUrl: string | null;
  onSeek: (time: number) => void;
}) {
  const barRef = useRef<HTMLDivElement>(null);
  const dragging = useRef(false);
  const [hover, setHover] = useState<{
    time: number;
    x: number;
    width: number;
  } | null>(null);

  const timeFromClientX = useCallback(
    (clientX: number) => {
      const el = barRef.current;
      if (!el || duration <= 0) {
        return 0;
      }
      const rect = el.getBoundingClientRect();
      const ratio = Math.min(1, Math.max(0, (clientX - rect.left) / rect.width));
      return ratio * duration;
    },
    [duration],
  );

  useEffect(() => {
    const onMove = (e: PointerEvent) => {
      if (!dragging.current) {
        return;
      }
      const t = timeFromClientX(e.clientX);
      const el = barRef.current;
      if (el) {
        const rect = el.getBoundingClientRect();
        setHover({ time: t, x: e.clientX - rect.left, width: rect.width });
      }
      onSeek(t);
    };
    const onUp = () => {
      dragging.current = false;
    };
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    return () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
  }, [onSeek, timeFromClientX]);

  const ratio = duration > 0 ? currentTime / duration : 0;
  const cue = hover && spriteUrl ? findSpriteCue(cues, hover.time) : null;
  const previewW = cue ? Math.min(192, cue.w || 160) : 0;
  const previewH = cue && cue.w ? (cue.h / cue.w) * previewW : 0;

  return (
    <div
      ref={barRef}
      role="slider"
      aria-label="Seek"
      aria-valuemin={0}
      aria-valuemax={Math.floor(duration)}
      aria-valuenow={Math.floor(currentTime)}
      tabIndex={0}
      className="relative h-5 cursor-pointer"
      onPointerDown={(e) => {
        dragging.current = true;
        (e.target as HTMLElement).setPointerCapture?.(e.pointerId);
        const t = timeFromClientX(e.clientX);
        const rect = barRef.current?.getBoundingClientRect();
        if (rect) {
          setHover({ time: t, x: e.clientX - rect.left, width: rect.width });
        }
        onSeek(t);
      }}
      onPointerMove={(e) => {
        if (dragging.current) {
          return;
        }
        const t = timeFromClientX(e.clientX);
        const rect = barRef.current?.getBoundingClientRect();
        if (rect) {
          setHover({ time: t, x: e.clientX - rect.left, width: rect.width });
        }
      }}
      onPointerLeave={() => {
        if (!dragging.current) {
          setHover(null);
        }
      }}
      onKeyDown={(e) => {
        if (e.key === "ArrowLeft") {
          e.preventDefault();
          onSeek(Math.max(0, currentTime - 5));
        } else if (e.key === "ArrowRight") {
          e.preventDefault();
          onSeek(Math.min(duration, currentTime + 5));
        }
      }}
    >
      {hover && cue && spriteUrl ? (
        <div
          className="seek-preview pointer-events-none absolute bottom-7 z-40 overflow-hidden rounded-md border border-white/20 bg-black"
          style={{
            width: previewW,
            height: previewH + 22,
            left: Math.min(
              Math.max(0, hover.x - previewW / 2),
              hover.width - previewW,
            ),
          }}
        >
          <div
            className="relative overflow-hidden"
            style={{ width: previewW, height: previewH }}
          >
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={spriteUrl}
              alt=""
              className="absolute top-0 left-0 max-w-none"
              style={{
                transform: `translate(-${cue.x}px, -${cue.y}px) scale(${previewW / (cue.w || 1)})`,
                transformOrigin: "top left",
              }}
            />
          </div>
          <div className="py-0.5 text-center font-mono text-[10px] text-zinc-200">
            {formatDuration(hover.time)}
          </div>
        </div>
      ) : hover ? (
        <div
          className="pointer-events-none absolute bottom-7 z-40 rounded bg-black/80 px-1.5 py-0.5 font-mono text-[10px] text-zinc-200"
          style={{
            left: Math.max(0, hover.x - 18),
          }}
        >
          {formatDuration(hover.time)}
        </div>
      ) : null}
      <div className="absolute inset-x-0 top-1/2 h-1 -translate-y-1/2 rounded-full bg-white/20">
        <div
          className="h-full rounded-full bg-accent"
          style={{ width: `${ratio * 100}%` }}
        />
      </div>
      <div
        className="absolute top-1/2 h-3 w-3 -translate-x-1/2 -translate-y-1/2 rounded-full bg-accent"
        style={{ left: `${ratio * 100}%` }}
      />
    </div>
  );
}

function IconBtn({
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
      className="flex h-9 w-9 items-center justify-center rounded-md text-white hover:bg-white/10 disabled:opacity-30 focus-visible:outline-2 focus-visible:outline-accent"
    >
      {children}
    </button>
  );
}

function PlayIcon({ className = "h-5 w-5" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={`${className} fill-current`} aria-hidden>
      <path d="M8 5.14v13.72L19 12 8 5.14Z" />
    </svg>
  );
}
function PauseIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden>
      <path d="M7 5h3v14H7V5Zm7 0h3v14h-3V5Z" />
    </svg>
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
function VolumeIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden>
      <path d="M5 10v4h3l4 4V6L8 10H5Zm11.5 2a3.5 3.5 0 0 0-1.8-3.05v6.1A3.5 3.5 0 0 0 16.5 12Z" />
    </svg>
  );
}
function MuteIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden>
      <path d="M16.5 12 19 14.5 17.5 16 15 13.5 12.5 16 11 14.5 13.5 12 11 9.5 12.5 8 15 10.5 17.5 8 19 9.5 16.5 12ZM5 10v4h3l4 4V6L8 10H5Z" />
    </svg>
  );
}
function FullIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden>
      <path d="M7 14H5v5h5v-2H7v-3Zm12 0h-2v3h-3v2h5v-5ZM7 7h3V5H5v5h2V7Zm7-2v2h3v3h2V5h-5Z" />
    </svg>
  );
}
