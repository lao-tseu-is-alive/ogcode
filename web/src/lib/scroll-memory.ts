// Tiny per-key scroll-position memory used to restore scroll inside
// non-document scroll containers (chat messages list, etc.) when the user
// navigates away and back within the SPA.
const memory = new Map<string, number>();

export function saveScroll(key: string, value: number): void {
  memory.set(key, value);
}

export function getScroll(key: string): number {
  return memory.get(key) ?? 0;
}

export function clearScroll(key: string): void {
  memory.delete(key);
}
