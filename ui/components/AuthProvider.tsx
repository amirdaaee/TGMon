"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { loginRequest, sessionRequest, setUnauthorizedHandler } from "@/lib/api";
import { clearToken, getToken, setToken } from "@/lib/auth";

type AuthStatus = "loading" | "anon" | "authed";

type AuthContextValue = {
  status: AuthStatus;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading");

  const logout = useCallback(() => {
    clearToken();
    setStatus("anon");
  }, []);

  useEffect(() => {
    setUnauthorizedHandler(logout);
    const token = getToken();
    if (!token) {
      const t = window.setTimeout(() => setStatus("anon"), 0);
      return () => {
        window.clearTimeout(t);
        setUnauthorizedHandler(null);
      };
    }
    let cancelled = false;
    sessionRequest()
      .then(() => {
        if (!cancelled) {
          setStatus("authed");
        }
      })
      .catch((err: unknown) => {
        if (cancelled) {
          return;
        }
        const statusCode =
          err && typeof err === "object" && "status" in err
            ? Number((err as { status: number }).status)
            : 0;
        if (statusCode === 401) {
          clearToken();
          setStatus("anon");
        } else {
          setStatus("authed");
        }
      });
    return () => {
      cancelled = true;
      setUnauthorizedHandler(null);
    };
  }, [logout]);

  const login = useCallback(async (username: string, password: string) => {
    const res = await loginRequest(username, password);
    setToken(res.Token);
    setStatus("authed");
  }, []);

  const value = useMemo(
    () => ({ status, login, logout }),
    [status, login, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
