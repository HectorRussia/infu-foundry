// Visible-list moves. The server keeps one global order per pin group and a
// category filter only hides rows, so placing an item after a visible neighbour
// preserves that relative order.

export type OrderedItem = { id: string; pinned: boolean };

/** null means the top of the item's pin group. undefined means the move is a no-op or crosses groups. */
export function afterIdForDrop<T extends OrderedItem>(
  items: T[],
  id: string,
  targetId: string,
  placeAfter: boolean,
): string | null | undefined {
  const item = items.find((entry) => entry.id === id);
  const target = items.find((entry) => entry.id === targetId);
  if (!item || !target || item.pinned !== target.pinned || id === targetId) return undefined;
  const group = items.filter((entry) => entry.pinned === item.pinned);
  const from = group.findIndex((entry) => entry.id === id);
  const rest = group.filter((entry) => entry.id !== id);
  let insertAt = rest.findIndex((entry) => entry.id === targetId);
  if (from < 0 || insertAt < 0) return undefined;
  if (placeAfter) insertAt += 1;
  if (insertAt === from) return undefined;
  if (insertAt === 0) return null;
  return rest[insertAt - 1].id;
}

export function afterIdForStep<T extends OrderedItem>(
  items: T[],
  id: string,
  direction: -1 | 1,
): string | null | undefined {
  const item = items.find((entry) => entry.id === id);
  if (!item) return undefined;
  const group = items.filter((entry) => entry.pinned === item.pinned);
  const from = group.findIndex((entry) => entry.id === id);
  const target = from + direction;
  if (from < 0 || target < 0 || target >= group.length) return undefined;
  return afterIdForDrop(items, id, group[target].id, direction > 0);
}

export function reorder<T extends OrderedItem>(items: T[], id: string, afterId: string | null): T[] {
  const item = items.find((entry) => entry.id === id);
  if (!item) return items;
  const without = items.filter((entry) => entry.id !== id);
  if (afterId === null) {
    const at = without.findIndex((entry) => entry.pinned === item.pinned);
    const next = without.slice();
    next.splice(at < 0 ? next.length : at, 0, item);
    return next;
  }
  const after = without.find((entry) => entry.id === afterId);
  if (!after || after.pinned !== item.pinned) return items;
  const at = without.findIndex((entry) => entry.id === afterId);
  const next = without.slice();
  next.splice(at + 1, 0, item);
  return next;
}

export function withPinned<T extends OrderedItem>(items: T[], id: string, pinned: boolean): T[] {
  const item = items.find((entry) => entry.id === id);
  if (!item || item.pinned === pinned) return items;
  const nextItem = { ...item, pinned };
  const without = items.filter((entry) => entry.id !== id);
  if (pinned) return [nextItem, ...without];
  const at = without.findIndex((entry) => !entry.pinned);
  if (at < 0) return [...without, nextItem];
  const next = without.slice();
  next.splice(at, 0, nextItem);
  return next;
}
