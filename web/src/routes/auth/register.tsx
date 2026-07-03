import { createFileRoute, redirect } from "@tanstack/react-router";
import {
  RegisterPage,
  validateSearchParams,
} from "#/features/auth/pages/RegisterPage";
import { meQueryOptions } from "#/features/auth/api";

export const Route = createFileRoute("/auth/register")({
  validateSearch: validateSearchParams,
  beforeLoad: async ({ context }) => {
    // Already logged-in users get bounced to the app root.
    const user = await context.queryClient.ensureQueryData(meQueryOptions());
    if (user) {
      throw redirect({ to: "/" });
    }
  },
  component: RegisterPage,
});
