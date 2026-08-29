export const API_URL = (
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"
).replace(/\/$/, "");

export const MINIO_URL = (process.env.NEXT_PUBLIC_MINIO_URL ?? "").replace(
  /\/$/,
  "",
);

export const PAGE_SIZE = 12;

export const RESUME_MIN_SEC = 15;
export const RESUME_END_GAP_SEC = 10;

export function assetUrl(key: string | null | undefined): string | null {
  if (!key || !MINIO_URL) {
    return null;
  }
  const encoded = key
    .replace(/^\//, "")
    .split("/")
    .map(encodeURIComponent)
    .join("/");
  return `/minio/${encoded}`;
}
// https://tgmon.milloo.top/tgmon-thumb/6a92a56f5d9a6fe72e73f189_THUMBNAIL.jpeg

export function streamUrl(id: string): string {
  return `${API_URL}/stream/${id}`;
}
