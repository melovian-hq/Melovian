// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// A Map/Set with a fixed capacity, evicting the least-recently-used entry
// once the capacity is exceeded. Prevents module-level caches from growing
// without bound over long-running sessions.

export function createBoundedMap<K, V>(maxEntries: number): Map<K, V> {
  const map = new Map<K, V>();
  const set = map.set.bind(map);
  const get = map.get.bind(map);

  map.set = (key: K, value: V) => {
    map.delete(key);
    set(key, value);
    while (map.size > maxEntries) {
      const oldest = map.keys().next().value;
      if (oldest === undefined) break;
      map.delete(oldest);
    }
    return map;
  };

  map.get = (key: K) => {
    const value = get(key);
    if (value !== undefined && map.size > 0) {
      map.delete(key);
      set(key, value);
    }
    return value;
  };

  return map;
}

export function createBoundedSet<T>(maxEntries: number): Set<T> {
  const set = new Set<T>();
  const add = set.add.bind(set);

  set.add = (value: T) => {
    set.delete(value);
    add(value);
    while (set.size > maxEntries) {
      const oldest = set.values().next().value;
      if (oldest === undefined) break;
      set.delete(oldest);
    }
    return set;
  };

  return set;
}
