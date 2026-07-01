/**
 * Vitest global test setup.
 *
 * Stubs browser APIs that jsdom does not implement but are required by Mantine
 * (and other UI libraries) when running component tests.
 */

// Mantine's MantineProvider calls window.matchMedia to detect the system color
// scheme. jsdom does not implement it, so we stub it with a minimal object.
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string): MediaQueryList =>
    ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }) as MediaQueryList,
});
