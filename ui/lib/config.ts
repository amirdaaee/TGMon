export type RuntimeConfig = {
  apiUrl: string;
  minioUrl: string;
};

function stripSlash(value: string): string {
  return value.replace(/\/$/, "");
}

let apiUrl = "http://localhost:8080";
let minioUrl = "";

export function readPublicRuntimeConfig(): RuntimeConfig {
  return {
    apiUrl: stripSlash(
      process.env["NEXT_PUBLIC_API_URL"] || "http://localhost:8080",
    ),
    minioUrl: stripSlash(process.env["NEXT_PUBLIC_MINIO_URL"] || ""),
  };
}

export function setRuntimeConfig(config: RuntimeConfig): void {
  apiUrl = stripSlash(config.apiUrl);
  minioUrl = stripSlash(config.minioUrl);
}

export function getApiUrl(): string {
  return apiUrl;
}

export function getMinioUrl(): string {
  return minioUrl;
}

export const PAGE_SIZE = 12;

export const RESUME_MIN_SEC = 15;
export const RESUME_END_GAP_SEC = 10;

export function assetUrl(key: string | null | undefined): string | null {
  if (!key || !getMinioUrl()) {
    return null;
  }
  const encoded = key
    .replace(/^\//, "")
    .split("/")
    .map(encodeURIComponent)
    .join("/");
  return `/minio/${encoded}`;
}

export function streamUrl(id: string): string {
  return `${getApiUrl()}/stream/${id}`;
}

export function downloadUrl(id: string): string {
  return `${streamUrl(id)}?d=true`;
}
