import { queryOptions } from "@tanstack/react-query";

// Principal mirrors the /auth/me JSON response (lowercase field names from the server).
export interface Principal {
  id: number;
  username: string;
  email: string;
}

export const authKeys = {
  me: ["auth", "me"] as const,
};

// --- /auth/me ----------------------------------------------------------

async function getMe(): Promise<Principal | null> {
  const res = await fetch("/api/v0/auth/me", { credentials: "include" });
  if (res.status === 401) {
    return null; // unauthenticated — not an error, guards branch on null
  }
  if (!res.ok) {
    throw new Error(`GET /auth/me failed: ${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<Principal>;
}

export function meQueryOptions() {
  return queryOptions({
    queryKey: authKeys.me,
    queryFn: getMe,
    retry: false,
    staleTime: 5 * 60 * 1000, // 5 min — refresh on nav, not on every render
  });
}

// --- /auth/login -------------------------------------------------------

export interface LoginCredentials {
  Token: string; // username or email (PascalCase to match Go LoginRequest)
  Password: string;
}

export async function login(credentials: LoginCredentials): Promise<void> {
  const res = await fetch("/api/v0/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(credentials),
    credentials: "include",
  });
  if (res.status === 401) {
    throw new Error("Invalid credentials");
  }
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(text || `Login failed: ${res.status}`);
  }
}

// --- /auth/logout ------------------------------------------------------

export async function logout(): Promise<void> {
  const res = await fetch("/api/v0/auth/logout", {
    method: "POST",
    credentials: "include",
  });
  if (!res.ok) {
    throw new Error(`Logout failed: ${res.status}`);
  }
}
