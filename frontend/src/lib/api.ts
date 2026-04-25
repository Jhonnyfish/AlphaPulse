import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
  timeout: 15000,
});

// Request interceptor — attach JWT token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor — handle 401
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      // Redirect to login if not already there
      if (window.location.pathname !== '/login') {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

export default api;

// --- Auth ---
export interface LoginResponse {
  token: string;
  refresh_token: string;
  user: User;
}

export interface User {
  id: string;
  username: string;
  role: string;
  created_at: string;
}

export const authApi = {
  login: (username: string, password: string) =>
    api.post<LoginResponse>('/auth/login', { username, password }),
  verify: () =>
    api.get<{ valid: boolean; user: User }>('/auth/verify'),
};

// --- Watchlist ---
export interface WatchlistItem {
  id: string;
  code: string;
  name?: string;
  group_name: string;
  added_at: string;
}

export const watchlistApi = {
  list: () => api.get<WatchlistItem[]>('/watchlist'),
  add: (code: string, group_name?: string) =>
    api.post('/watchlist', { code, group_name }),
  remove: (code: string) => api.delete(`/watchlist/${code}`),
  batchAdd: (codes: string[]) =>
    api.post('/watchlist/batch', { codes }),
};

// --- Market ---
export interface Quote {
  code: string;
  name: string;
  price: number;
  open: number;
  prev_close: number;
  high: number;
  low: number;
  change: number;
  change_percent: number;
  updated_at: string;
}

export interface KlinePoint {
  date: string;
  open: number;
  close: number;
  high: number;
  low: number;
  volume: number;
  amount: number;
}

export interface Sector {
  code: string;
  name: string;
  price: number;
  change: number;
  change_percent: number;
}

export interface OverviewIndex {
  code: string;
  name: string;
  price: number;
  change: number;
  change_percent: number;
  advance_count: number;
  decline_count: number;
  flat_count: number;
}

export interface MarketOverview {
  advance_count: number;
  decline_count: number;
  flat_count: number;
  indices: OverviewIndex[];
  updated_at: string;
}

export interface NewsItem {
  code?: string;
  title: string;
  summary?: string;
  source?: string;
  url: string;
  published_at: string;
}

export const marketApi = {
  quote: (code: string) => api.get<Quote>('/market/quote', { params: { code } }),
  kline: (code: string, period?: string) =>
    api.get<KlinePoint[]>('/market/kline', { params: { code, period } }),
  sectors: () => api.get<Sector[]>('/market/sectors'),
  overview: () => api.get<MarketOverview>('/market/overview'),
  news: () => api.get<NewsItem[]>('/market/news'),
};
