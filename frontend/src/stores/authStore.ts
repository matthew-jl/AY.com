import { writable } from "svelte/store";
import { getAccessToken } from "../lib/api";

// Check for existing token on initial load
const initialAuthState = !!getAccessToken();

export const isAuthenticated = writable<boolean>(initialAuthState);

// Function to update store after login/logout
export function setAuthState(isAuth: boolean) {
  isAuthenticated.set(isAuth);
}
