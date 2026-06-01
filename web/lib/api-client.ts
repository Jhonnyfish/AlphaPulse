const API_BASE = "/api";

export async function apiFetch<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  if (typeof window !== "undefined") {
    const token = localStorage.getItem("token");
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: { ...headers, ...(options?.headers as Record<string, string>) },
  });

  if (res.status === 401) {
    if (typeof window !== "undefined") {
      localStorage.removeItem("token");
      localStorage.removeItem("user");
      window.location.href = "/login";
    }
    throw new Error("Unauthorized");
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.message || body.error || `API Error: ${res.status}`);
  }

  return res.json();
}

// ---- Auth ----
import type {
  LoginResponse,
  VerifyResponse,
  RankingResponse,
  CandidatesResponse,
  ScreenerResponse,
  StockAnalysis,
  DeepAnalysisResponse,
  ScoreHistoryResponse,
  SearchSuggestion,
  SyncStatus,
  SyncConfig,
  PortfolioPosition,
  PortfolioPositionEnriched,
} from "./types";

export const authApi = {
  login: (username: string, password: string) =>
    apiFetch<LoginResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }),
  verify: () => apiFetch<VerifyResponse>("/auth/verify"),
};

// ---- Search ----
export const searchApi = {
  stocks: (q: string) => {
    const sp = new URLSearchParams({ q });
    return apiFetch<SearchSuggestion[]>(`/market/search?${sp}`);
  },
};

// ---- Ranking ----
export const rankingApi = {
  full: () => rankingApi.withStrategy(),
  withStrategy: (strategy?: string) => {
    const params = new URLSearchParams();
    if (strategy) params.set("strategy", strategy);
    const qs = params.toString();
    return apiFetch<RankingResponse>(
      `/watchlist-analysis/ranking${qs ? `?${qs}` : ""}`
    );
  },
};

// ---- Candidates ----
export const candidatesApi = {
  list: (params?: { limit?: number; strategy?: string }) => {
    const sp = new URLSearchParams();
    if (params?.limit) sp.set("limit", String(params.limit));
    if (params?.strategy) sp.set("strategy", params.strategy);
    const qs = sp.toString();
    return apiFetch<CandidatesResponse>(`/candidates${qs ? `?${qs}` : ""}`);
  },
};

// ---- Screener ----
export const screenerApi = {
  query: (params: {
    min_score?: number;
    max_score?: number;
    min_momentum?: number;
    min_trend?: number;
    max_volatility?: number;
    tier?: string;
    limit?: number;
  }) => {
    const sp = new URLSearchParams();
    if (params.min_score != null) sp.set("min_score", String(params.min_score));
    if (params.max_score != null) sp.set("max_score", String(params.max_score));
    if (params.min_momentum != null) sp.set("min_momentum", String(params.min_momentum));
    if (params.min_trend != null) sp.set("min_trend", String(params.min_trend));
    if (params.max_volatility != null) sp.set("max_volatility", String(params.max_volatility));
    if (params.tier) sp.set("tier", params.tier);
    if (params.limit) sp.set("limit", String(params.limit));
    const qs = sp.toString();
    return apiFetch<ScreenerResponse>(`/screener${qs ? `?${qs}` : ""}`);
  },
};

// ---- Analysis ----
export const analysisApi = {
  analyze: (code: string) =>
    apiFetch<StockAnalysis>(`/analyze?code=${code}`),
  deepAnalysis: (code: string) =>
    apiFetch<DeepAnalysisResponse>("/deep-analysis", {
      method: "POST",
      body: JSON.stringify({ code }),
    }),
  deepAnalysisStatus: (code: string) =>
    apiFetch<DeepAnalysisResponse>(`/deep-analysis/status/${code}`),
  scoreHistory: (code: string) =>
    apiFetch<ScoreHistoryResponse>(`/score-history/${code}`),
};

// ---- Data Sync ----
export const syncApi = {
  trigger: () =>
    apiFetch<{ ok: boolean; status: string }>("/system/sync", {
      method: "POST",
    }),
  status: () => apiFetch<SyncStatus>("/system/sync/status"),
  getConfig: () => apiFetch<SyncConfig>("/system/sync/config"),
  updateConfig: (config: Partial<Omit<SyncConfig, "ok">>) =>
    apiFetch<{ ok: boolean }>("/system/sync/config", {
      method: "PUT",
      body: JSON.stringify(config),
    }),
};

// ---- Portfolio ----
export const portfolioApi = {
  list: () =>
    apiFetch<{ ok: boolean; data: PortfolioPositionEnriched[] }>("/portfolio"),
  add: (position: { code: string; cost_price: number; quantity: number; notes?: string }) =>
    apiFetch<{ ok: boolean; data: PortfolioPosition }>("/portfolio", {
      method: "POST",
      body: JSON.stringify(position),
    }),
  update: (id: string, updates: Partial<{ code: string; cost_price: number; quantity: number; notes: string }>) =>
    apiFetch<{ ok: boolean }>(`/portfolio/${id}`, {
      method: "PUT",
      body: JSON.stringify(updates),
    }),
  delete: (id: string) =>
    apiFetch<{ ok: boolean }>(`/portfolio/${id}`, { method: "DELETE" }),
};
