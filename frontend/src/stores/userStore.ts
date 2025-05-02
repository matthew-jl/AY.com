import { writable } from "svelte/store";

// Interface matching the userpb.User structure (adjust if needed)
// Note: Timestamps from proto might become string|Date depending on JSON marshal
export interface UserData {
  id: number; // Use number for JS IDs
  name: string;
  username: string;
  email: string;
  gender: string;
  profile_picture: string | null; // Allow null if empty
  banner: string | null; // Allow null if empty
  date_of_birth: string; // YYYY-MM-DD string
  account_status: string;
  account_privacy: string;
  created_at: string; // ISO String date from backend
}

// Writable store, initialized to null (no user logged in)
export const user = writable<UserData | null>(null);

// Function to update the store after login/fetch
export function setUser(userData: UserData | null) {
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
