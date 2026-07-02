import { describe, it, expect, vi, afterEach } from "vitest";
import {
  api,
  ApiError,
  NetworkError,
  problemsToFieldErrors,
  validationProblems,
} from "#/lib/api";

describe("api client", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("NetworkError", () => {
    it("is thrown when fetch rejects (server unreachable)", async () => {
      vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));

      await expect(api.get("/test")).rejects.toThrow(NetworkError);
    });

    it("wraps the original TypeError as .cause", async () => {
      const original = new TypeError("Failed to fetch");
      vi.stubGlobal("fetch", vi.fn().mockRejectedValue(original));

      const err = await api.get("/test").catch((e: unknown) => e);
      expect(err).toBeInstanceOf(NetworkError);
      expect((err as NetworkError).cause).toBe(original);
    });

    it("has a descriptive message", async () => {
      vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));

      const err = await api.get("/test").catch((e: unknown) => e);
      expect((err as NetworkError).message).toBe("Unable to reach the server");
    });
  });

  describe("ApiError", () => {
    it("is thrown for non-ok HTTP responses", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue(
          new Response("Unauthorized", { status: 401, statusText: "Unauthorized" }),
        ),
      );

      await expect(api.get("/test")).rejects.toThrow(ApiError);
    });

    it("carries the HTTP status code", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue(
          new Response("Not Found", { status: 404, statusText: "Not Found" }),
        ),
      );

      const err = await api.get("/test").catch((e: unknown) => e);
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).status).toBe(404);
    });
  });

  describe("validationProblems", () => {
    const problems = [
      { field: "email", message: "must be valid" },
      { field: "password", message: "too short" },
    ];

    it("extracts problems from a 400 ApiError with a problems body", () => {
      const err = new ApiError(400, "Bad Request", { problems });
      expect(validationProblems(err)).toEqual(problems);
    });

    it("returns null for non-400 statuses", () => {
      const err = new ApiError(500, "Server Error", { problems });
      expect(validationProblems(err)).toBeNull();
    });

    it("returns null when the body has no problems array", () => {
      expect(validationProblems(new ApiError(400, "Bad Request", "plain text"))).toBeNull();
      expect(validationProblems(new ApiError(400, "Bad Request", {}))).toBeNull();
    });

    it("returns null for non-ApiError values", () => {
      expect(validationProblems(new Error("nope"))).toBeNull();
      expect(validationProblems(null)).toBeNull();
    });
  });

  describe("problemsToFieldErrors", () => {
    it("maps one message per field", () => {
      expect(
        problemsToFieldErrors([
          { field: "email", message: "must be valid" },
          { field: "password", message: "too short" },
        ]),
      ).toEqual({ email: "must be valid", password: "too short" });
    });

    it("joins multiple problems for the same field", () => {
      expect(
        problemsToFieldErrors([
          { field: "password", message: "too short" },
          { field: "password", message: "needs a digit" },
        ]),
      ).toEqual({ password: "too short; needs a digit" });
    });
  });
});
