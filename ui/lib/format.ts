import { RESUME_END_GAP_SEC, RESUME_MIN_SEC } from "./config";
import type { MediaFileDoc } from "./types";

export function asId(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }
  if (value && typeof value === "object" && "$oid" in value) {
    return String((value as { $oid: string }).$oid);
  }
  if (value == null) {
    return "";
  }
  return String(value);
}

export function stripExtension(fileName: string): string {
  return fileName.replace(/\.[^/.]+$/, "");
}

export function mediaTitle(media: MediaFileDoc): string {
  if (media.UName) {
    return media.UName;
  }
  const id = asId(media.ID);
  const base =
    stripExtension(media.Meta?.FileName ?? "") || id || "Untitled";
  if (!id) {
    return base;
  }
  return `${base}-${id.slice(-8)}`;
}

export function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) {
    return "0:00";
  }
  const total = Math.floor(seconds);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  if (h > 0) {
    return `${h}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  }
  return `${m}:${String(s).padStart(2, "0")}`;
}

export function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let n = bytes;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i += 1;
  }
  return `${n < 10 && i > 0 ? n.toFixed(1) : Math.round(n)} ${units[i]}`;
}

export function isResumable(checkpoint: number, duration: number): boolean {
  return (
    checkpoint >= RESUME_MIN_SEC &&
    duration > 0 &&
    checkpoint < duration - RESUME_END_GAP_SEC
  );
}

export function checkpointToSave(
  currentTime: number,
  duration: number,
): number {
  if (
    currentTime < RESUME_MIN_SEC ||
    (duration > 0 && currentTime >= duration - RESUME_END_GAP_SEC)
  ) {
    return 0;
  }
  return Math.floor(currentTime);
}
