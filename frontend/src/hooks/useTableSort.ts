import { useState, useMemo, useCallback } from 'react';

export type SortDirection = 'asc' | 'desc' | null;

export interface SortState {
  key: string;
  direction: SortDirection;
}

/**
 * Generic table sort, search, filter and pagination hook.
 * Works with any array of objects — no API dependency.
 */
export function useTableSort<T extends Record<string, unknown>>(
  data: T[],
  options?: {
    defaultSortKey?: string;
    defaultSortDir?: SortDirection;
    pageSize?: number;
    searchFields?: (keyof T & string)[];
  }
) {
  const {
    defaultSortKey = '',
    defaultSortDir = null,
    pageSize = 20,
    searchFields = [],
  } = options ?? {};

  const [sort, setSort] = useState<SortState>({
    key: defaultSortKey,
    direction: defaultSortDir,
  });
  const [searchQuery, setSearchQuery] = useState('');
  const [page, setPage] = useState(1);
  const [filter, setFilter] = useState<string>('all');

  /** Cycle sort: none → asc → desc → none */
  const toggleSort = useCallback((key: string) => {
    setSort((prev) => {
      if (prev.key !== key) return { key, direction: 'asc' };
      if (prev.direction === 'asc') return { key, direction: 'desc' };
      return { key: '', direction: null };
    });
    setPage(1);
  }, []);

  /** Apply search filter */
  const searched = useMemo(() => {
    if (!searchQuery.trim()) return data;
    const q = searchQuery.trim().toLowerCase();
    return data.filter((item) =>
      searchFields.some((field) => {
        const val = item[field];
        return val != null && String(val).toLowerCase().includes(q);
      })
    );
  }, [data, searchQuery, searchFields]);

  /** Apply custom filter callback — to be extended by consumers */
  const filtered = useMemo(() => {
    if (filter === 'all') return searched;
    return searched; // override in consumer by passing filter function externally
  }, [searched, filter]);

  /** Sort */
  const sorted = useMemo(() => {
    if (!sort.key || !sort.direction) return filtered;

    return [...filtered].sort((a, b) => {
      const aVal = a[sort.key];
      const bVal = b[sort.key];

      // Handle nulls
      if (aVal == null && bVal == null) return 0;
      if (aVal == null) return 1;
      if (bVal == null) return -1;

      let cmp = 0;
      if (typeof aVal === 'number' && typeof bVal === 'number') {
        cmp = aVal - bVal;
      } else {
        // String comparison (works for Chinese via localeCompare)
        cmp = String(aVal).localeCompare(String(bVal), 'zh-CN');
      }

      return sort.direction === 'asc' ? cmp : -cmp;
    });
  }, [filtered, sort]);

  /** Pagination */
  const totalPages = Math.max(1, Math.ceil(sorted.length / pageSize));
  const currentPage = Math.min(page, totalPages);

  const paginated = useMemo(() => {
    const start = (currentPage - 1) * pageSize;
    return sorted.slice(start, start + pageSize);
  }, [sorted, currentPage, pageSize]);

  const goToPage = useCallback(
    (p: number) => setPage(Math.max(1, Math.min(p, totalPages))),
    [totalPages]
  );

  /** Reset page when search changes */
  const updateSearch = useCallback((q: string) => {
    setSearchQuery(q);
    setPage(1);
  }, []);

  const updateFilter = useCallback((f: string) => {
    setFilter(f);
    setPage(1);
  }, []);

  return {
    // State
    sort,
    searchQuery,
    page: currentPage,
    filter,
    totalPages,
    totalItems: sorted.length,
    // Derived
    paginated,
    // Actions
    toggleSort,
    updateSearch,
    updateFilter,
    goToPage,
    setPage: updateFilter, // alias
  };
}
