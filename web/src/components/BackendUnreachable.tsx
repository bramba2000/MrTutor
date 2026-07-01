import { Anchor, Button, Center, Container, Paper, Stack, Text, Title } from "@mantine/core";
import { ServerCrash } from "lucide-react";

import { SUPPORT_EMAIL, SUPPORT_HOURS } from "#/config";

interface BackendUnreachableProps {
  onRetry: () => void;
}

/**
 * Full-screen page shown when the backend is not reachable (network error,
 * server down, connection refused). Displays contact details so the user
 * knows who to write and when to get help.
 *
 * Rendered by RootErrorComponent when it receives a NetworkError.
 */
export function BackendUnreachable({ onRetry }: BackendUnreachableProps) {
  return (
    <Center mih="100vh">
      <Container size={480}>
        <Stack align="center" gap="xl">
          <ServerCrash size={64} color="var(--mantine-color-red-6)" />

          <Stack align="center" gap="xs">
            <Title order={1} ta="center">
              Can&apos;t reach the server
            </Title>
            <Text c="dimmed" ta="center">
              The service may be temporarily down. Please try again in a moment.
            </Text>
          </Stack>

          <Paper withBorder shadow="sm" p="lg" radius="md" w="100%">
            <Stack gap="xs">
              <Text fw={500}>Need help?</Text>
              <Text>
                Write to{" "}
                <Anchor href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</Anchor>
              </Text>
              <Text size="sm" c="dimmed">
                Available {SUPPORT_HOURS}
              </Text>
            </Stack>
          </Paper>

          <Button onClick={onRetry}>Try again</Button>
        </Stack>
      </Container>
    </Center>
  );
}
