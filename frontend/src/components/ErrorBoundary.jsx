import React from "react";

import { ErrorState } from "@/components/patterns/ErrorState";

/**
 * isChunkLoadError — true when a lazy route/chunk failed to load. This is
 * almost always a *stale deploy*: the tab holds an old app shell that asks for
 * a chunk hash the server has since replaced, so the import 404s. A state reset
 * can't recover it (the chunk is gone) — only a full reload, which fetches the
 * new index and its chunks.
 */
function isChunkLoadError(error) {
  if (!error) return false;
  const msg = String(error.message || "");
  return (
    error.name === "ChunkLoadError" ||
    /failed to fetch dynamically imported module/i.test(msg) ||
    /error loading dynamically imported module/i.test(msg) ||
    /importing a module script failed/i.test(msg) ||
    /is not a valid javascript mime type/i.test(msg)
  );
}

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }

  componentDidCatch(error, errorInfo) {
    this.setState({ errorInfo });
    console.error("[ErrorBoundary]", error, errorInfo);
  }

  // Generic errors may be transient — resetting re-renders the subtree.
  handleRetry = () => {
    this.setState({ hasError: false, error: null, errorInfo: null });
  };

  // Stale-chunk errors only recover on a full reload.
  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      const chunk = isChunkLoadError(this.state.error);
      return (
        <ErrorState
          title={chunk ? "Update available" : "Something went wrong"}
          message={
            chunk
              ? "A new version of Recurso was just deployed. Reload to continue — your place is preserved."
              : this.state.error?.message ||
                "An unexpected error occurred. Please try again."
          }
          retryLabel={chunk ? "Reload" : "Try again"}
          onRetry={chunk ? this.handleReload : this.handleRetry}
        />
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
