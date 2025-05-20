import { writable } from "svelte/store";
import type { UserProfileBasic } from "../lib/api";

export const user = writable<UserProfileBasic | null>(null);

// Function to update the store after login/fetch
export function setUser(userData: UserProfileBasic | null) {
  if (userData) {
    console.log("Setting user store:", userData);
  } else {
    console.log("Clearing user store.");
  }
  user.set(userData);
}

// Function to clear the store on logout
export function clearUser() {
  setUser(null);
}
