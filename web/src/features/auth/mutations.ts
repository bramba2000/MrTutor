import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { authKeys, login, logout, type LoginCredentials } from "./api";

/**
 * Mutation for logging out.
 *
 * On success: clears the /auth/me cache entry immediately (setQueryData null),
 * then invalidates to ensure a fresh fetch after next login, and navigates to /auth/login.
 */
export function useSignout() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: logout,
    onSuccess: () => {
      // Immediately mark the user as gone — no extra round-trip needed.
      queryClient.setQueryData(authKeys.me, null);
      // Drop all cached data so a subsequent login starts with a clean cache.
      queryClient.clear();
      navigate({ to: "/auth/login" });
    },
  });
}

/**
 * Mutation for logging in.
 *
 * On success: invalidates /auth/me so the next render refetches the current user.
 * The login route (or form) should redirect to the intended destination after this
 * mutation resolves.
 */
export function useLogin() {
  const queryClient = useQueryClient();

  return useMutation<void, Error, LoginCredentials>({
    mutationFn: login,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: authKeys.me });
    },
  });
}
