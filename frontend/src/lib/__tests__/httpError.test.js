import { describe, it, expect } from "vitest";
import { httpStatus, apiError, isNotFound, errorMessage } from "../httpError";

// Shapes mirroring the app's real axios errors (response attached unchanged;
// API envelope { error: { code, message } }).
const axiosError = (status, body) => ({ response: { status, data: body } });
const notFound404 = axiosError(404, { error: { code: "not_found", message: "payment not found" } });
const notFoundCode = axiosError(400, { error: { code: "not_found", message: "no such object" } });
const serverError = axiosError(500, { error: { code: "internal", message: "boom" } });
const badRequest = axiosError(400, { error: { code: "validation_failed", message: "invalid id" } });
const forbidden = axiosError(403, { error: { code: "forbidden", message: "nope" } });
const networkError = new Error("Network Error"); // no response

describe("httpStatus", () => {
  it("reads the response status, or null for a network failure", () => {
    expect(httpStatus(notFound404)).toBe(404);
    expect(httpStatus(networkError)).toBeNull();
    expect(httpStatus({ status: 418 })).toBe(418);
    expect(httpStatus(undefined)).toBeNull();
  });
});

describe("apiError", () => {
  it("returns the API error envelope or null", () => {
    expect(apiError(badRequest)).toEqual({ code: "validation_failed", message: "invalid id" });
    expect(apiError(networkError)).toBeNull();
  });
});

describe("isNotFound", () => {
  it("is true for a real HTTP 404", () => {
    expect(isNotFound({ error: notFound404 })).toBe(true);
  });
  it("is true for the API not_found code on a non-404 status (defensive)", () => {
    expect(isNotFound({ error: notFoundCode })).toBe(true);
  });
  it("is true for a resolved-but-null object (the live 404-copy bug)", () => {
    expect(isNotFound({ error: null, data: null, resolved: true })).toBe(true);
    expect(isNotFound({ error: null, data: undefined, resolved: true })).toBe(true);
  });
  it("is false while still resolving (no data yet, not resolved)", () => {
    expect(isNotFound({ error: null, data: undefined, resolved: false })).toBe(false);
  });
  it("is false for a resolved object", () => {
    expect(isNotFound({ error: null, data: { id: "x" }, resolved: true })).toBe(false);
  });
  it("is false for a genuine (non-404) error", () => {
    expect(isNotFound({ error: serverError })).toBe(false);
    expect(isNotFound({ error: networkError })).toBe(false);
  });
});

describe("errorMessage — safe operator-facing text", () => {
  it("maps 401/403 to a permission message", () => {
    expect(errorMessage(forbidden)).toMatch(/permission/i);
    expect(errorMessage(axiosError(401, {}))).toMatch(/permission/i);
  });
  it("never echoes server-error detail (5xx → generic)", () => {
    const msg = errorMessage(serverError);
    expect(msg).not.toContain("boom");
    expect(msg).toMatch(/went wrong/i);
  });
  it("preserves a known 4xx API message", () => {
    expect(errorMessage(badRequest)).toBe("invalid id");
  });
  it("falls back to a safe generic for a network failure", () => {
    expect(errorMessage(networkError)).toMatch(/went wrong/i);
  });
  it("falls back for an unknown error shape", () => {
    expect(errorMessage({ weird: true })).toMatch(/went wrong/i);
    expect(errorMessage(null)).toMatch(/went wrong/i);
  });
  it("honors a custom fallback", () => {
    expect(errorMessage(networkError, "Couldn’t load this invoice")).toBe("Couldn’t load this invoice");
  });
});
