import { useCallback, useEffect, useRef } from "react";
import { useSearchParams } from "react-router";

/**
 * useUrlState — like useState, but the value lives in the URL query string, so
 * a list page's context (page, search, filter, sort) survives navigating into a
 * detail and back (audit §5: back-nav used to reset to page 1 / All). Drop-in
 * for useState: returns `[value, setValue]` and setValue accepts a value or an
 * updater function.
 *
 * URL params are strings, so pass `parse`/`serialize` for non-string values
 * (e.g. numbers). The default value is omitted from the URL to keep it clean,
 * and updates use `replace` so typing/filtering never floods the back button.
 *
 *   const [page, setPage] = useUrlState("page", 1, { parse: Number });
 *   const [status, setStatus] = useUrlState("status", "all");
 *
 * The returned setter is STABLE (safe in effect deps) even though the default
 * parse/serialize params are recreated each render — config is read from a ref.
 */
export function useUrlState(key, defaultValue, options = {}) {
  const { parse = (s) => s, serialize = (v) => String(v) } = options;
  const [params, setParams] = useSearchParams();

  const raw = params.get(key);
  const value = raw === null ? defaultValue : parse(raw);

  // Latest config in a ref so setValue's identity never changes (its deps are
  // only the stable [key, setParams]) — otherwise it would recreate every render
  // and loop any effect that lists it as a dependency.
  const cfg = useRef(null);
  // eslint-disable-next-line react-hooks/refs -- latest-ref idiom: written during render on purpose (see above)
  cfg.current = { defaultValue, parse, serialize };

  const setValue = useCallback(
    (next) => {
      setParams(
        (prev) => {
          const { defaultValue: dv, parse: p, serialize: s } = cfg.current;
          const np = new URLSearchParams(prev);
          const current = np.get(key) === null ? dv : p(np.get(key));
          const resolved = typeof next === "function" ? next(current) : next;
          if (
            resolved === dv ||
            resolved === "" ||
            resolved === null ||
            resolved === undefined
          ) {
            np.delete(key);
          } else {
            np.set(key, s(resolved));
          }
          return np;
        },
        { replace: true },
      );
    },
    [key, setParams],
  );

  return [value, setValue];
}

/**
 * useResetPageOnChange — reset a list back to page 1 when its filter/search
 * values change, done correctly for URL-persisted state:
 *
 *  - runs in an effect (a SEPARATE tick from the filter's own URL write), so the
 *    two useUrlState setters don't clobber each other in one navigation;
 *  - skips the first run, so a page restored from the URL on mount (back
 *    navigation) survives;
 *  - reads `setPage` from a ref, so its identity — which `useSearchParams`
 *    churns after every URL write — never spuriously re-triggers the reset (which
 *    would snap the user back to page 1 the instant they paged forward).
 *
 * `deps` are the filter values (strings); they're serialized so the dependency
 * array stays a stable literal.
 *
 *   useResetPageOnChange(setPage, [search, status]);
 */
export function useResetPageOnChange(setPage, deps) {
  const setPageRef = useRef(setPage);
  // eslint-disable-next-line react-hooks/refs -- latest-ref idiom: keeps the effect's deps to the serialized key only
  setPageRef.current = setPage;
  const firstRun = useRef(true);
  const key = JSON.stringify(deps);

  useEffect(() => {
    if (firstRun.current) {
      firstRun.current = false;
      return;
    }
    setPageRef.current(1);
  }, [key]);
}
