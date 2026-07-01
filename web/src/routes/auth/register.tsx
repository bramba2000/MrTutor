import {
  createFileRoute,
  redirect,
  type SearchSchemaInput,
} from "@tanstack/react-router";
import { RegisterPage } from "#/features/auth/pages/RegisterPage";
import { meQueryOptions } from "#/features/auth/api";

export const Route = createFileRoute("/auth/register")({
  validateSearch: (search: Record<string, unknown> & SearchSchemaInput) => ({
    redirect: typeof search.redirect === "string" ? search.redirect : undefined,
  }),
  beforeLoad: async ({ context }) => {
    // Already logged-in users get bounced to the app root.
    const user = await context.queryClient.ensureQueryData(meQueryOptions());
    if (user) {
      throw redirect({ to: "/" });
    }
  },
  component: RegisterPage,
});
