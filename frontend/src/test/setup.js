import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// Runs a cleanup after each test case (e.g. clearing jsdom)
afterEach(() => {
  cleanup();
  // List pages persist their state (page / filter / search) in the URL via
  // useUrlState, and BrowserRouter reads the shared jsdom window.location — so
  // reset it between tests, otherwise one test's filter leaks into the next.
  // Guarded because some lib tests opt into the node environment (no window).
  if (typeof window !== 'undefined' && window.history) {
    window.history.replaceState({}, '', '/');
  }
});
