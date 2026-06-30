import { createFileRoute, redirect } from "@tanstack/react-router";
import { LoginPage } from "#/features/auth/pages/LoginPage";
import { meQueryOptions } from "#/features/auth/api";

export const Route = createFileRoute("/auth/login")({
  validateSearch: (search: Record<string, unknown>) => ({
    redirect: typeof search.redirect === "string" ? search.redirect : undefined,
  }),
  beforeLoad: async ({ context }) => {
    // Already logged-in users get bounced to the app root.
    const user = await context.queryClient.ensureQueryData(meQueryOptions());
    if (user) {
      throw redirect({ to: "/" });
    }
  },
  component: LoginPage,
});
