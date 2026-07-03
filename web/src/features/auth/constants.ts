export enum UserRole {
  Admin = "ADMIN",
  Student = "STUDENT",
  Tutor = "TUTOR",
}

export function isUserRole(value: any): value is UserRole {
  return Object.values(UserRole).includes(value);
}

/**
 * Coerce an untrusted `redirect` search param into a safe, same-origin path.
 *
 * Prevents open-redirect abuse: a value must be a relative path with a single
 * leading slash. Absolute URLs (`https://evil.com`), protocol-relative URLs
 * (`//evil.com`) and backslash variants (`/\evil.com`, which browsers normalize
 * to `//`) all fall back to "/".
 */
export function safeRedirect(value: unknown): string {
  if (typeof value !== "string") return "/";
  if (!value.startsWith("/")) return "/";
  // Reject "//" and "/\" — both resolve cross-origin in the browser.
  if (value[1] === "/" || value[1] === "\\") return "/";
  return value;
}
