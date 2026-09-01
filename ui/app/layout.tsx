import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { connection } from "next/server";
import { AuthGate } from "@/components/AuthGate";
import { AuthProvider } from "@/components/AuthProvider";
import { ConfigProvider } from "@/components/ConfigProvider";
import { readPublicRuntimeConfig } from "@/lib/config";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "TGMon",
  description: "TGMon media library",
};

export default async function RootLayout({ children }: LayoutProps<"/">) {
  await connection();
  const config = readPublicRuntimeConfig();

  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="flex min-h-full flex-col font-sans">
        <ConfigProvider config={config}>
          <AuthProvider>
            <AuthGate>{children}</AuthGate>
          </AuthProvider>
        </ConfigProvider>
      </body>
    </html>
  );
}
