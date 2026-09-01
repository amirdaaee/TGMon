import { getToken } from "./auth";
import { getApiUrl } from "./config";
import type {
  ApiErrorBody,
  JobReqDoc,
  LoginPostRes,
  MediaListRes,
  MediaMetaPatchReq,
  MediaReadRes,
} from "./types";

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

let unauthorizedHandler: (() => void) | null = null;

export function setUnauthorizedHandler(handler: (() => void) | null): void {
  unauthorizedHandler = handler;
}

function errorMessage(status: number, body: unknown): string {
  if (body && typeof body === "object") {
    const b = body as ApiErrorBody;
    if (b.msg) {
      return b.msg;
    }
    if (b.message) {
      return b.message;
    }
  }
  if (status === 401) {
    return "Invalid username or password";
  }
  if (status === 0) {
    return "Could not reach the server";
  }
  return `Request failed (${status})`;
}

async function parseBody(res: Response): Promise<unknown> {
  const text = await res.text();
  if (!text) {
    return null;
  }
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

type FetchOpts = {
  method?: string;
  body?: unknown;
  auth?: boolean;
};

async function request<T>(path: string, opts: FetchOpts = {}): Promise<T> {
  const headers: Record<string, string> = {};
  if (opts.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (opts.auth !== false) {
    const token = getToken();
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
  }

  let res: Response;
  try {
    res = await fetch(`${getApiUrl()}${path}`, {
      method: opts.method ?? "GET",
      headers,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    });
  } catch {
    throw new ApiError(0, errorMessage(0, null));
  }

  if (res.status === 401 && opts.auth !== false) {
    unauthorizedHandler?.();
    throw new ApiError(401, "Session expired");
  }

  const body = await parseBody(res);
  if (!res.ok) {
    throw new ApiError(res.status, errorMessage(res.status, body));
  }
  return body as T;
}

export function loginRequest(
  username: string,
  password: string,
): Promise<LoginPostRes> {
  return request<LoginPostRes>("/api/auth/login/", {
    method: "POST",
    auth: false,
    body: { Username: username, Password: password },
  });
}

export function sessionRequest(): Promise<LoginPostRes> {
  return request<LoginPostRes>("/api/auth/session/");
}

export function listMedia(page: number): Promise<MediaListRes> {
  return request<MediaListRes>(`/api/media/?page=${page}`);
}

export function readMedia(id: string): Promise<MediaReadRes> {
  return request<MediaReadRes>(`/api/media/${encodeURIComponent(id)}`);
}

export function patchMediaMeta(
  id: string,
  patch: MediaMetaPatchReq,
): Promise<void> {
  return request(`/api/media/${encodeURIComponent(id)}/meta/`, {
    method: "PATCH",
    body: patch,
  }).then(() => undefined);
}

export function deleteMedia(id: string): Promise<void> {
  return request(`/api/media/${encodeURIComponent(id)}/`, {
    method: "DELETE",
  }).then(() => undefined);
}

export function listJobReqs(): Promise<JobReqDoc[]> {
  return request<JobReqDoc[]>("/api/jobReq/");
}
