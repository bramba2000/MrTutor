import { createFileRoute } from "@tanstack/react-router";

// Placeholder home page for authenticated users.
// Replace with a real dashboard or redirect once the app grows.
export const Route = createFileRoute("/_authenticated/")({
  component: () => <p>Welcome! (authenticated home placeholder)</p>,
});
