/**
 * Web Vitals collection module.
 * Collects Core Web Vitals (LCP, FCP, CLS, TTFB, INP) and stores them
 * in memory + localStorage. Optionally beacons to backend.
 */
import { onLCP, onFCP, onCLS, onTTFB, onINP } from 'web-vitals';
import type { Metric } from 'web-vitals';

export type VitalRating = 'good' | 'needs-improvement' | 'poor';

export interface VitalEntry {
  name: string;
  value: number;
  rating: VitalRating;
  timestamp: number;
  id: string;
}

const STORAGE_KEY = 'alphapulse_vitals';
const MAX_ENTRIES = 100;

// In-memory ring buffer
let entries: VitalEntry[] = [];
let initialized = false;

/** Rating thresholds per metric */
const THRESHOLDS: Record<string, [number, number]> = {
  LCP: [2500, 4000],   // ms
  FCP: [1800, 3000],    // ms
  CLS: [0.1, 0.25],     // score
  TTFB: [800, 1800],    // ms
  INP: [200, 500],      // ms
};

export function getRating(name: string, value: number): VitalRating {
  const t = THRESHOLDS[name];
  if (!t) return 'good';
  if (value <= t[0]) return 'good';
  if (value <= t[1]) return 'needs-improvement';
  return 'poor';
}

function addEntry(metric: Metric) {
  const entry: VitalEntry = {
    name: metric.name,
    value: metric.value,
    rating: getRating(metric.name, metric.value),
    timestamp: Date.now(),
    id: metric.id,
  };

  // Deduplicate by id (web-vitals fires updates for same metric)
  const idx = entries.findIndex((e) => e.id === entry.id);
  if (idx >= 0) {
    entries[idx] = entry;
  } else {
    entries.push(entry);
  }

  // Trim to max
  if (entries.length > MAX_ENTRIES) {
    entries = entries.slice(-MAX_ENTRIES);
  }

  // Persist
  persistToStorage();

  // Beacon to backend (non-blocking)
  beaconToBackend(entry);
}

function persistToStorage() {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(entries));
  } catch {
    // localStorage full or unavailable — silently ignore
  }
}

function loadFromStorage(): VitalEntry[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed)) return parsed as VitalEntry[];
    }
  } catch {
    // corrupted storage
  }
  return [];
}

function beaconToBackend(entry: VitalEntry) {
  try {
    const token = localStorage.getItem('token');
    if (!token) return;

    const url = '/api/system/vitals';
    const body = JSON.stringify(entry);
    // Always use fetch(keepalive) — sendBeacon doesn't support custom headers
    // so Authorization header would be lost, causing 401 from auth middleware.
    // keepalive survives page unload (same as sendBeacon) and is supported in all modern browsers.
    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body,
      keepalive: true,
    }).catch(() => {
      // Non-blocking, ignore errors
    });
  } catch {
    // Non-blocking, ignore errors
  }
}

/** Initialize web vitals collection. Call once from App on mount. */
export function initVitals() {
  if (initialized) return;
  initialized = true;

  // Load persisted entries
  entries = loadFromStorage();

  // Register collectors
  onLCP(addEntry);
  onFCP(addEntry);
  onCLS(addEntry);
  onTTFB(addEntry);
  onINP(addEntry);
}

/** Get all collected vitals entries. */
export function getVitals(): VitalEntry[] {
  return [...entries];
}

/** Get the latest value for a specific metric. */
export function getLatestVital(name: string): VitalEntry | undefined {
  // Walk backwards to find the most recent
  for (let i = entries.length - 1; i >= 0; i--) {
    if (entries[i].name === name) return entries[i];
  }
  return undefined;
}

/** Get all entries for a specific metric, sorted by timestamp. */
export function getVitalsByName(name: string): VitalEntry[] {
  return entries
    .filter((e) => e.name === name)
    .sort((a, b) => a.timestamp - b.timestamp);
}

/** Clear all collected vitals. */
export function clearVitals() {
  entries = [];
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    // ignore
  }
}

/** Generate mock data for demo/preview purposes. */
export function getMockVitals(): VitalEntry[] {
  const now = Date.now();
  const names = ['LCP', 'FCP', 'CLS', 'TTFB', 'INP'];
  const mockEntries: VitalEntry[] = [];

  for (let i = 0; i < 30; i++) {
    const ts = now - (30 - i) * 60_000; // 1 minute intervals
    for (const name of names) {
      let value: number;
      switch (name) {
        case 'LCP':
          value = 1200 + Math.random() * 3000;
          break;
        case 'FCP':
          value = 800 + Math.random() * 2000;
          break;
        case 'CLS':
          value = Math.random() * 0.35;
          break;
        case 'TTFB':
          value = 200 + Math.random() * 1500;
          break;
        case 'INP':
          value = 50 + Math.random() * 400;
          break;
        default:
          value = 0;
      }
      mockEntries.push({
        name,
        value: Math.round(value * 100) / 100,
        rating: getRating(name, value),
        timestamp: ts + Math.random() * 10_000,
        id: `mock-${name}-${i}`,
      });
    }
  }
  return mockEntries;
}
