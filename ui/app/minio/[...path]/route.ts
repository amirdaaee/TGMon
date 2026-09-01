import { NextResponse } from "next/server";
import { readPublicRuntimeConfig } from "@/lib/config";

const PASS_HEADERS = [
  "content-type",
  "content-length",
  "cache-control",
  "etag",
  "last-modified",
  "accept-ranges",
  "content-range",
];

type RouteParams = { params: Promise<{ path: string[] }> };

async function proxy(
  req: Request,
  path: string[],
  method: "GET" | "HEAD",
): Promise<Response> {
  const { minioUrl } = readPublicRuntimeConfig();
  if (!minioUrl) {
    return new NextResponse(null, { status: 404 });
  }

  const encoded = path.map(encodeURIComponent).join("/");
  const target = `${minioUrl}/${encoded}${new URL(req.url).search}`;

  let upstream: Response;
  try {
    upstream = await fetch(target, { method });
  } catch {
    return new NextResponse(null, { status: 502 });
  }

  const headers = new Headers();
  for (const name of PASS_HEADERS) {
    const value = upstream.headers.get(name);
    if (value) {
      headers.set(name, value);
    }
  }

  return new NextResponse(method === "HEAD" ? null : upstream.body, {
    status: upstream.status,
    headers,
  });
}

export async function GET(req: Request, ctx: RouteParams): Promise<Response> {
  const { path } = await ctx.params;
  return proxy(req, path, "GET");
}

export async function HEAD(req: Request, ctx: RouteParams): Promise<Response> {
  const { path } = await ctx.params;
  return proxy(req, path, "HEAD");
}
