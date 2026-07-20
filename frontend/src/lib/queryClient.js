import { QueryClient } from "@tanstack/react-query";

// One shared client for the dashboard. Reads are considered fresh for a
// minute — navigating between pages reuses the cached result instead of
// refiring the same GETs on every mount (the request volume behind the
// rate-limit incident). Mutations and page "Retry" buttons invalidate or
// refetch explicitly.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60_000,
      gcTime: 5 * 60_000,
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});
