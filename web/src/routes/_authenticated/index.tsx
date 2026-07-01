import { useSignout } from "#/features/auth/mutations";
import { createFileRoute } from "@tanstack/react-router";

// Placeholder home page for authenticated users.
// Replace with a real dashboard or redirect once the app grows.
export const Route = createFileRoute("/_authenticated/")({
  component: AuthenticatedHome,
});

function AuthenticatedHome() {
  const signout = useSignout();

  return (
    <div>
      <h1>Welcome! (authenticated home placeholder)</h1>
      <button onClick={() => signout.mutate()}>Logout</button>
    </div>
  );
}
