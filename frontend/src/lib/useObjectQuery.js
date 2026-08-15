import { useQuery } from "@tanstack/react-query";

import { isNotFound } from "@/lib/httpError";

/**
 * useObjectQuery — the canonical read lifecycle for a single object page. It is
 * a thin classifier over ONE object GET (it does NOT replace react-query and is
 * not a general-purpose fetch hook): it runs the query and buckets the result
 * into the four states every object page needs, so each page renders one
 * skeleton, one not-found, and one error via the shared components.
 *
 *   const { object, loading, notFound, isError, error, refetch } =
 *     useObjectQuery(["invoice", id], () => endpoints.getInvoice(id).then(r => r.data.data), {
 *       enabled: Boolean(id),
 *     });
 *
 * Classification (mutually exclusive, in order):
 *   - loading  — the initial load (pending, no cached data yet)
 *   - notFound — the object doesn't exist: a real 404, the API `not_found` code,
 *                OR a resolved-but-null object (the fix for the live 404-copy bug)
 *   - isError  — the request genuinely failed (network / 5xx / other)
 *   - object   — the resolved object (only when none of the above)
 *
 * `object` is null in every non-success state so a page can render its states
 * before touching object fields.
 */
export function useObjectQuery(queryKey, queryFn, options = {}) {
  const query = useQuery({ queryKey, queryFn, ...options });

  // isPending (not isLoading) is the right "no resolved result yet" signal: it
  // stays true for a still-disabled query (e.g. `enabled: Boolean(id)` before an
  // id arrives), where isLoading would be false — which would otherwise leak an
  // undefined object into the success body.
  const loading = query.isPending;
  const notFound =
    !loading &&
    isNotFound({ error: query.error, data: query.data, resolved: query.isSuccess });
  const isError = !loading && !notFound && Boolean(query.error);

  return {
    object: loading || notFound || isError ? null : query.data,
    loading,
    notFound,
    isError,
    error: query.error,
    refetch: query.refetch,
    // Escape hatch for pages that need more of the underlying query (e.g.
    // isFetching); use sparingly.
    query,
  };
}

export default useObjectQuery;
