import { useQuery } from "@tanstack/react-query";
import { meQueryOptions, type Principal } from "./api";

export interface AuthState {
  user: Principal | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

/**
 * Read the current auth state from the React Query cache.
 *
 * Backed by GET /api/v0/auth/me; the query is shared across the app
 * so components subscribe to the same cache entry — no extra fetches.
 *
 * Use inside the QueryClientProvider tree (i.e. any route component).
 * For route-level guards use queryClient.ensureQueryData(meQueryOptions())
 * directly in beforeLoad (see _authenticated.tsx).
 */
export function useAuth(): AuthState {
  const { data, isPending } = useQuery(meQueryOptions());
  return {
    user: data ?? null,
    isAuthenticated: !!data,
    isLoading: isPending,
  };
}
