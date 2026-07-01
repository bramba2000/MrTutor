/**
 * Shared HTTP client for the mrtutor API.
 *
 * Usage:
 *   import { api, ApiError } from "#/lib/api";
 *
 *   const user = await api.get<User>("/auth/me");
 *   await api.post("/auth/login", { Token: "alice", Password: "…" });
 *   await api.delete("/items/42");
 *
 * All methods:
 *   - prepend the API base path (/api/v0 by default, overridable via VITE_API_BASE_PATH)
 *   - include credentials (session cookie)
 *   - serialize plain-object bodies as JSON with Content-Type: application/json;
 *     FormData / Blob / string / etc. are passed through untouched
 *   - deserialize JSON responses automatically; return undefined for 204/empty bodies
 *   - throw ApiError on non-ok responses (status + body available for branching)
 */

// Vite proxies /api → :8080 in dev; same origin in prod.
const BASE_PATH = import.meta.env.VITE_API_BASE_PATH ?? "/api/v0";

// ---------------------------------------------------------------------------
// Error type
// ---------------------------------------------------------------------------

/** Thrown by every api.* helper on a non-ok HTTP response. */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly statusText: string,
    /** Parsed JSON body, or plain text if the response wasn't JSON. */
    readonly body: unknown,
  ) {
    super(typeof body === "string" && body ? body : `${status} ${statusText}`);
    this.name = "ApiError";
  }
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Everything RequestInit accepts except method and body, which we manage. */
export type RequestOptions = Omit<RequestInit, "method" | "body">;

interface InternalOptions extends RequestOptions {
  body?: unknown;
}

// ---------------------------------------------------------------------------
// Core
// ---------------------------------------------------------------------------

function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (value === null || typeof value !== "object") return false;
  const proto = Object.getPrototypeOf(value);
  return proto === Object.prototype || proto === null;
}

async function parseBody(res: Response): Promise<unknown> {
  // 204 No Content or truly empty body
  const contentLength = res.headers.get("content-length");
  if (res.status === 204 || contentLength === "0") return undefined;

  const ct = res.headers.get("content-type") ?? "";
  if (ct.includes("application/json")) {
    return res.json();
  }
  const text = await res.text();
  return text || undefined;
}

async function request<T>(
  method: string,
  path: string,
  { body, headers, credentials = "include", ...rest }: InternalOptions = {},
): Promise<T> {
  const resolvedHeaders = new Headers(headers);
  let resolvedBody: BodyInit | undefined;

  if (body !== undefined) {
    if (isPlainObject(body) || Array.isArray(body)) {
      resolvedHeaders.set("Content-Type", "application/json");
      resolvedBody = JSON.stringify(body);
    } else {
      // FormData, Blob, URLSearchParams, ArrayBuffer, string — pass through;
      // the browser sets the correct Content-Type (e.g. multipart boundary for FormData).
      resolvedBody = body as BodyInit;
    }
  }

  const res = await fetch(`${BASE_PATH}${path}`, {
    method,
    body: resolvedBody,
    headers: resolvedHeaders,
    credentials,
    ...rest,
  });

  if (!res.ok) {
    const errorBody = await parseBody(res).catch(() => res.statusText);
    throw new ApiError(res.status, res.statusText, errorBody);
  }

  return parseBody(res) as Promise<T>;
}

// ---------------------------------------------------------------------------
// Exported verb helpers
// ---------------------------------------------------------------------------

export const api = {
  /** GET {base}/{path} */
  get<T>(path: string, options?: RequestOptions): Promise<T> {
    return request<T>("GET", path, options);
  },

  /** POST {base}/{path} with optional body */
  post<T = void>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return request<T>("POST", path, { ...options, body });
  },

  /** PUT {base}/{path} with optional body */
  put<T = void>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return request<T>("PUT", path, { ...options, body });
  },

  /** PATCH {base}/{path} with optional body */
  patch<T = void>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return request<T>("PATCH", path, { ...options, body });
  },

  /** DELETE {base}/{path} */
  delete<T = void>(path: string, options?: RequestOptions): Promise<T> {
    return request<T>("DELETE", path, options);
  },
};
