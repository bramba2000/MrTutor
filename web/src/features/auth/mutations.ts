import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import {
  authKeys,
  login,
  logout,
  register,
  type LoginCredentials,
  type RegisterCredentials,
  type Principal,
} from "./api";

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
      navigate({ to: "/auth/login", search: { redirect: "/" } });
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
      // Remove the pre-login null from cache so the next ensureQueryData
      // (in _authenticated beforeLoad) fetches fresh instead of returning null.
      queryClient.removeQueries({ queryKey: authKeys.me });
    },
  });
}

/**
 * Mutation for registering a new user.
 *
 * The onSuccess callback sets the /auth/me cache entry to the newly registered user.
 */
export function useRegister() {
  const queryClient = useQueryClient();

  return useMutation<Principal, Error, RegisterCredentials>({
    mutationFn: register,
    onSuccess: (principal) => {
      queryClient.setQueryData(authKeys.me, principal);
    },
  });
}
