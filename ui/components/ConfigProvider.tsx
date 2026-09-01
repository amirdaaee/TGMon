"use client";

import { setRuntimeConfig, type RuntimeConfig } from "@/lib/config";

export function ConfigProvider({
  config,
  children,
}: {
  config: RuntimeConfig;
  children: React.ReactNode;
}) {
  setRuntimeConfig(config);
  return children;
}
