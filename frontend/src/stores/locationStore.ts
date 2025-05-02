import { writable } from "svelte/store";

// Initialize with null or current path if on client
const initialPath =
  typeof window !== "undefined" ? window.location.pathname : null;

// Writable store holding the current pathname string
export const currentPathname = writable<string | null>(initialPath);
