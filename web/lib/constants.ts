// ---- Dimension labels (Chinese) ----
export const DIM_LABELS: Record<string, string> = {
  volume_price: "量价",
  valuation: "估值",
  volatility: "波动",
  money_flow: "资金",
  technical: "技术",
  sector: "板块",
  sentiment: "情绪",
  fundamentals: "基本面",
  northbound: "北向",
  margin: "融资",
};

// Analysis dimension key → icon label for the detail page
export const DIM_DETAIL_LABELS: Record<string, string> = {
  volume_price: "量价分析",
  valuation: "估值分析",
  volatility: "波动分析",
  money_flow: "资金流向",
  technical: "技术指标",
  sector: "板块分析",
  sentiment: "市场情绪",
  fundamentals: "基本面",
  northbound: "北向资金",
  margin: "融资融券",
};

// ---- Tier config ----
export const TIER_CONFIG: Record<string, { label: string }> = {
  strong_buy: { label: "强推" },
  buy: { label: "推荐" },
  focus: { label: "重点关注" },
  observe: { label: "观察池" },
  watch: { label: "关注" },
};

// ---- Signal helpers ----
export function signalLabel(signal: string): string {
  switch (signal) {
    case "strong_buy":
      return "强买";
    case "buy":
      return "买入";
    case "sell":
      return "卖出";
    case "strong_sell":
      return "强卖";
    case "neutral":
      return "中性";
    default:
      return signal || "—";
  }
}

export function formatPct(pct: number): string {
  const sign = pct > 0 ? "+" : "";
  return `${sign}${pct.toFixed(2)}%`;
}

export function formatPrice(price: number): string {
  return price.toFixed(2);
}

export function formatVolume(v: number): string {
  if (v >= 1e8) return `${(v / 1e8).toFixed(2)}亿`;
  if (v >= 1e4) return `${(v / 1e4).toFixed(0)}万`;
  return v.toFixed(0);
}

export function formatMoney(v: number): string {
  const abs = Math.abs(v);
  const sign = v < 0 ? "-" : v > 0 ? "+" : "";
  if (abs >= 1e8) return `${sign}${(abs / 1e8).toFixed(2)}亿`;
  if (abs >= 1e4) return `${sign}${(abs / 1e4).toFixed(0)}万`;
  return `${sign}${abs.toFixed(0)}`;
}
