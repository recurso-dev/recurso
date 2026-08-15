// Shared HTTP/error classification for the dashboard's read flows. The app's
// axios instance attaches the server response to a rejected request unchanged
// (there is no response interceptor that reshapes it), and the API's canonical
// error envelope is { error: { code, message } }. A network failure has no
// response. These helpers key off those real shapes — no page should hand-roll
// status/error detection.

// httpStatus — the HTTP status of a rejected request, or null (e.g. a network
// failure, which has no response).
export function httpStatus(error) {
  return error?.response?.status ?? error?.status ?? null;
}

// apiError — the API's { code, message } error object, if present.
export function apiError(error) {
  return error?.response?.data?.error ?? null;
}

// isNotFound — did this request resolve to "the object does not exist"? True for
//   1. a real HTTP 404,
//   2. the API's `not_found` code (defensive: covers any status the envelope
//      carries a not_found on), or
//   3. a resolved-but-null object — a successful response that carried no object
//      (e.g. a 200 with data:null, or a body the queryFn unwrapped to null/
//      undefined).
// Case 3 is the root of the old live 404-copy bug: the per-page guard
// `if (error || !obj)` treated a resolved-null as a generic error. Callers pass
// { error, data, resolved } where `resolved` means the query settled without
// throwing (react-query isSuccess).
export function isNotFound({ error, data, resolved }) {
  if (error) {
    if (httpStatus(error) === 404) return true;
    return apiError(error)?.code === "not_found";
  }
  return Boolean(resolved) && (data === null || data === undefined);
}

const GENERIC = "Something went wrong on our end. Please try again.";

// errorMessage — safe, operator-facing text for a failed request. Preserves the
// API's known human message (the backend's error envelope is written for
// operators and never carries SQL/stack/gateway detail — respondInternalError
// returns a fixed message), maps auth/server errors to safe copy, and falls back
// to a generic line. It never echoes a raw error object, a stack, or a status
// code. Not used for not-found (that has its own state) — this is the ERROR
// branch only.
export function errorMessage(error, fallback = GENERIC) {
  const status = httpStatus(error);
  if (status === 401 || status === 403) {
    return "You don’t have permission to view this.";
  }
  // Server errors: don't surface internal detail, even if a message leaked.
  if (status && status >= 500) return fallback || GENERIC;
  const msg = apiError(error)?.message;
  if (typeof msg === "string" && msg.trim()) return msg.trim();
  return fallback || GENERIC;
}
