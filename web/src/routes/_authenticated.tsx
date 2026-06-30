import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
import { meQueryOptions } from "#/features/auth/api";

/**
 * Pathless layout route that guards all authenticated pages.
 *
 * Any route file placed under routes/_authenticated/ is automatically
 * protected: unauthenticated visitors are redirected to /auth/login
 * with the intended destination preserved in `?redirect=`.
 *
 * The resolved user is injected into router context, so child routes can
 * read it via Route.useRouteContext().user without an extra fetch.
 */
export const Route = createFileRoute("/_authenticated")({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions());
    if (!user) {
      throw redirect({
        to: "/auth/login",
        search: { redirect: location.href },
      });
    }
    return { user };
  },
  component: () => <Outlet />,
});
