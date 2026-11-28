"use client";

import { useEffect, useState } from "react";
import { api } from "./api";

// "loading" until /auth/me answers, then "in" on success and "out" on any error,
// so a failing API never makes the page pretend you are logged in.
export type Session = "loading" | "in" | "out";

export function useSession(): Session {
  const [s, setS] = useState<Session>("loading");
  useEffect(() => {
    api("/auth/me").then(() => setS("in"), () => setS("out"));
  }, []);
  return s;
}

// Where to send someone after login. Only same-site paths, so ?next= cannot redirect off-site.
export function safeNext(raw: string | null): string {
  // Browsers read "//host" and "/\\host" as another site, so both are refused.
  return raw && /^\/(?![/\\])/.test(raw) ? raw : "/upload";
}

export async function logout() {
  await api("/auth/logout", { method: "POST" }).catch(() => {});
  window.location.href = "/";
}
