import { QueryClient } from "@tanstack/react-query";

export function getContext() {
  const queryClient = new QueryClient();

  return {
    queryClient,
  } as const;
}
export default function TanstackQueryProvider() {}
