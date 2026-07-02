import { queryOptions } from "@tanstack/react-query";
import { api, ApiError } from "#/lib/api";

// Principal mirrors the /auth/me JSON response (lowercase field names from the server).
export interface Principal {
  id: number;
  username: string;
  email: string;
  createdAt: string;
  modifiedAt: string;
  role: string;
}

export const authKeys = {
  me: ["auth", "me"] as const,
};

// --- /auth/me ----------------------------------------------------------

async function getMe(): Promise<Principal | null> {
  try {
    return await api.get<Principal>("/auth/me");
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      return null; // unauthenticated — not an error, guards branch on null
    }
    throw e;
  }
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
  token: string; // username or email (PascalCase to match Go LoginRequest)
  password: string;
}

export async function login(credentials: LoginCredentials): Promise<void> {
  try {
    await api.post("/auth/login", credentials);
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      throw new Error("Invalid credentials");
    }
    throw e;
  }
}

// --- /auth/logout ------------------------------------------------------

export async function logout(): Promise<void> {
  await api.post("/auth/logout");
}

// --- /auth/register ----------------------------------------------------

export interface RegisterCredentials {
  username: string;
  email: string;
  password: string;
}

export async function register(
  credentials: RegisterCredentials,
): Promise<Principal> {
  try {
    return await api.post<Principal>("/auth/register", credentials);
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      throw new Error("Invalid credentials");
    }
    throw e;
  }
}
