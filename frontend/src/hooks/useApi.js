import { useState, useCallback, useRef, useEffect } from 'react';

/**
 * Generic hook for API calls with loading, error, and retry handling.
 *
 * Usage:
 *   const { data, loading, error, refetch } = useApi(() => endpoints.getCustomers())
 *   const { execute, loading } = useApi(() => endpoints.createCustomer(data), { immediate: false })
 */
export function useApi(apiFn, { immediate = true, initialData = null } = {}) {
  const [data, setData] = useState(initialData);
  const [loading, setLoading] = useState(immediate);
  const [error, setError] = useState(null);
  const mountedRef = useRef(true);

  const execute = useCallback(async (...args) => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiFn(...args);
      if (mountedRef.current) {
        const result = response?.data?.data ?? response?.data ?? response;
        setData(result);
        return result;
      }
    } catch (err) {
      if (mountedRef.current) {
        const apiError = err?.response?.data?.error;
        const message = apiError?.message
          || (typeof apiError === 'string' ? apiError : null)
          || err?.message
          || 'Something went wrong';
        setError(message);
      }
      throw err;
    } finally {
      if (mountedRef.current) {
        setLoading(false);
      }
    }
  }, [apiFn]);

  const refetch = useCallback(() => execute(), [execute]);

  useEffect(() => {
    mountedRef.current = true;
    if (immediate) {
      execute();
    }
    return () => { mountedRef.current = false; };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  return { data, loading, error, execute, refetch, setData };
}

/**
 * Hook for multiple parallel API calls
 *
 * Usage:
 *   const { data, loading, error } = useApis({
 *     customers: () => endpoints.getCustomers(),
 *     plans: () => endpoints.getPlans(),
 *   })
 */
export function useApis(apiMap) {
  const keys = Object.keys(apiMap);
  const [data, setData] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const execute = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const results = await Promise.all(keys.map(k => apiMap[k]()));
      const combined = {};
      keys.forEach((k, i) => {
        combined[k] = results[i]?.data?.data ?? results[i]?.data ?? results[i];
      });
      setData(combined);
    } catch (err) {
      setError(err?.response?.data?.error?.message || err?.message || 'Failed to load data');
    } finally {
      setLoading(false);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => { execute(); }, [execute]);

  return { data, loading, error, refetch: execute };
}

export default useApi;
