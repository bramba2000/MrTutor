import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { MantineProvider } from "@mantine/core";

import { BackendUnreachable } from "#/components/BackendUnreachable";
import { SUPPORT_EMAIL, SUPPORT_HOURS } from "#/config";

function renderWithMantine(ui: React.ReactElement) {
  return render(<MantineProvider>{ui}</MantineProvider>);
}

// Explicitly call cleanup after each test so DOM doesn't accumulate between
// tests (required when vitest globals are not enabled).
afterEach(cleanup);

describe("BackendUnreachable", () => {
  it("displays the support email as a mailto link", () => {
    renderWithMantine(<BackendUnreachable onRetry={vi.fn()} />);

    const link = screen.getByRole("link", { name: SUPPORT_EMAIL });
    expect(link).toBeTruthy();
    expect(link.getAttribute("href")).toBe(`mailto:${SUPPORT_EMAIL}`);
  });

  it("displays the support hours", () => {
    renderWithMantine(<BackendUnreachable onRetry={vi.fn()} />);

    // Use tagName check to avoid matching ancestor container nodes whose
    // textContent also includes SUPPORT_HOURS.
    const el = screen.getByText((_content, node) => {
      return node?.tagName === "P" && !!node.textContent?.includes(SUPPORT_HOURS);
    });
    expect(el).toBeTruthy();
  });

  it("calls onRetry when the 'Try again' button is clicked", () => {
    const onRetry = vi.fn();
    renderWithMantine(<BackendUnreachable onRetry={onRetry} />);

    fireEvent.click(screen.getByRole("button", { name: /try again/i }));
    expect(onRetry).toHaveBeenCalledOnce();
  });
});
