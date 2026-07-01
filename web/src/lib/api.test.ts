import { describe, it, expect, vi, afterEach } from "vitest";
import { api, ApiError, NetworkError } from "#/lib/api";

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
});
