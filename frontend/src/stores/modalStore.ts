// src/stores/modalStore.ts
import { writable } from "svelte/store";

export const isCreateThreadModalOpen = writable(false);

export function openCreateThreadModal() {
  isCreateThreadModalOpen.set(true);
}

export function closeCreateThreadModal() {
  isCreateThreadModalOpen.set(false);
}
