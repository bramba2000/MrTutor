import { type ErrorComponentProps, useRouter } from "@tanstack/react-router";
import { Button, Center, Container, Stack, Text, Title } from "@mantine/core";

import { NetworkError } from "#/lib/api";
import { BackendUnreachable } from "#/components/BackendUnreachable";

/**
 * Global router error component registered as `defaultErrorComponent` in
 * `router.tsx`. Receives any error thrown from a route's `beforeLoad`,
 * `loader`, or component.
 *
 * - NetworkError  → BackendUnreachable page (styled contact + retry)
 * - anything else → generic "Something went wrong" fallback
 *
 * `reset()` clears the error boundary; `router.invalidate()` re-runs loaders
 * and guards so the app recovers once the issue is resolved.
 */
export function RootErrorComponent({ error, reset }: ErrorComponentProps) {
  const router = useRouter();

  const retry = () => {
    reset();
    void router.invalidate();
  };

  if (error instanceof NetworkError) {
    return <BackendUnreachable onRetry={retry} />;
  }

  return (
    <Center mih="100vh">
      <Container size={480}>
        <Stack align="center" gap="xl">
          <Stack align="center" gap="xs">
            <Title order={1} ta="center">
              Something went wrong
            </Title>
            <Text c="dimmed" ta="center">
              An unexpected error occurred. You can try again or reload the page.
            </Text>
          </Stack>
          <Button onClick={retry}>Try again</Button>
        </Stack>
      </Container>
    </Center>
  );
}
