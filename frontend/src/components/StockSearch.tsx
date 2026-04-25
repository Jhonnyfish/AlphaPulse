import { useState, useEffect, useRef, useCallback } from 'react';
import { marketApi, type SearchSuggestion } from '@/lib/api';
import { Search, X } from 'lucide-react';

interface StockSearchProps {
  onSelect: (suggestion: SearchSuggestion) => void;
  placeholder?: string;
  className?: string;
  autoFocus?: boolean;
}

export default function StockSearch({
  onSelect,
  placeholder = '搜索股票代码或名称...',
  className = '',
  autoFocus = false,
}: StockSearchProps) {
  const [query, setQuery] = useState('');
  const [suggestions, setSuggestions] = useState<SearchSuggestion[]>([]);
  const [loading, setLoading] = useState(false);
  const [showDropdown, setShowDropdown] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>();

  // Debounced search
  const search = useCallback(async (q: string) => {
    if (q.length < 1) {
      setSuggestions([]);
      setShowDropdown(false);
      return;
    }

    setLoading(true);
    try {
      const res = await marketApi.search(q);
      setSuggestions(res.data);
      setShowDropdown(true);
      setActiveIndex(-1);
    } catch {
      setSuggestions([]);
    } finally {
      setLoading(false);
    }
  }, []);

  // Debounce input
  useEffect(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }
    debounceRef.current = setTimeout(() => {
      search(query);
    }, 300);

    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
  }, [query, search]);

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(e.target as Node)
      ) {
        setShowDropdown(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleSelect = (suggestion: SearchSuggestion) => {
    setQuery('');
    setSuggestions([]);
    setShowDropdown(false);
    onSelect(suggestion);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!showDropdown || suggestions.length === 0) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setActiveIndex((prev) => (prev < suggestions.length - 1 ? prev + 1 : 0));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setActiveIndex((prev) => (prev > 0 ? prev - 1 : suggestions.length - 1));
        break;
      case 'Enter':
        e.preventDefault();
        if (activeIndex >= 0 && activeIndex < suggestions.length) {
          handleSelect(suggestions[activeIndex]);
        }
        break;
      case 'Escape':
        setShowDropdown(false);
        setActiveIndex(-1);
        break;
    }
  };

  const clearQuery = () => {
    setQuery('');
    setSuggestions([]);
    setShowDropdown(false);
    inputRef.current?.focus();
  };

  return (
    <div className={`relative ${className}`}>
      <div className="relative">
        <Search
          className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4"
          style={{ color: 'var(--color-text-muted)' }}
        />
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onFocus={() => suggestions.length > 0 && setShowDropdown(true)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          autoFocus={autoFocus}
          className="w-full pl-9 pr-8 py-2 rounded-lg border text-sm outline-none"
          style={{
            background: 'var(--color-bg-card)',
            borderColor: showDropdown ? 'var(--color-accent)' : 'var(--color-border)',
            color: 'var(--color-text-primary)',
          }}
        />
        {query && (
          <button
            onClick={clearQuery}
            className="absolute right-3 top-1/2 -translate-y-1/2"
          >
            <X className="w-3.5 h-3.5" style={{ color: 'var(--color-text-muted)' }} />
          </button>
        )}
      </div>

      {/* Dropdown */}
      {showDropdown && suggestions.length > 0 && (
        <div
          ref={dropdownRef}
          className="absolute z-50 w-full mt-1 rounded-lg border overflow-hidden shadow-lg"
          style={{
            background: 'var(--color-bg-card)',
            borderColor: 'var(--color-border)',
          }}
        >
          {suggestions.map((item, index) => (
            <button
              key={item.code}
              onClick={() => handleSelect(item)}
              className="w-full flex items-center justify-between px-3 py-2.5 text-sm text-left transition-colors"
              style={{
                background: index === activeIndex ? 'var(--color-bg-hover)' : 'transparent',
                color: 'var(--color-text-primary)',
              }}
              onMouseEnter={() => setActiveIndex(index)}
            >
              <div className="flex items-center gap-3">
                <span className="font-mono text-xs px-1.5 py-0.5 rounded"
                  style={{ background: 'var(--color-bg-secondary)', color: 'var(--color-accent)' }}>
                  {item.code}
                </span>
                <span>{item.name}</span>
              </div>
            </button>
          ))}
        </div>
      )}

      {/* Loading indicator */}
      {showDropdown && loading && (
        <div
          className="absolute z-50 w-full mt-1 rounded-lg border p-3 text-center text-sm"
          style={{
            background: 'var(--color-bg-card)',
            borderColor: 'var(--color-border)',
            color: 'var(--color-text-muted)',
          }}
        >
          搜索中...
        </div>
      )}

      {/* No results */}
      {showDropdown && !loading && query.length >= 1 && suggestions.length === 0 && (
        <div
          className="absolute z-50 w-full mt-1 rounded-lg border p-3 text-center text-sm"
          style={{
            background: 'var(--color-bg-card)',
            borderColor: 'var(--color-border)',
            color: 'var(--color-text-muted)',
          }}
        >
          未找到匹配的股票
        </div>
      )}
    </div>
  );
}
