/**
 * Technical indicator calculation utilities
 * All formulas follow standard A-share technical analysis conventions
 */

// ── Moving Average (MA) ──────────────────────────────────────────────
export function calcMA(closes: number[], period: number): (number | null)[] {
  const result: (number | null)[] = [];
  for (let i = 0; i < closes.length; i++) {
    if (i < period - 1) {
      result.push(null);
    } else {
      let sum = 0;
      for (let j = i - period + 1; j <= i; j++) {
        sum += closes[j];
      }
      result.push(+(sum / period).toFixed(2));
    }
  }
  return result;
}

// ── MACD (Moving Average Convergence Divergence) ─────────────────────
// MACD = DIF - DEA
// DIF = EMA(12) - EMA(26)
// DEA = EMA(DIF, 9)
// MACD bar = 2 * (DIF - DEA)
export interface MACDData {
  dif: number[];
  dea: number[];
  macd: number[];
}

export function calcMACD(closes: number[], fast = 12, slow = 26, signal = 9): MACDData {
  const emaFast = calcEMA(closes, fast);
  const emaSlow = calcEMA(closes, slow);

  const dif: number[] = [];
  for (let i = 0; i < closes.length; i++) {
    dif.push(+(emaFast[i] - emaSlow[i]).toFixed(4));
  }

  const dea = calcEMA(dif, signal).map((v) => +v.toFixed(4));

  const macd: number[] = [];
  for (let i = 0; i < closes.length; i++) {
    macd.push(+((dif[i] - dea[i]) * 2).toFixed(4));
  }

  return { dif, dea, macd };
}

// ── EMA helper ───────────────────────────────────────────────────────
function calcEMA(data: number[], period: number): number[] {
  const k = 2 / (period + 1);
  const result: number[] = [data[0]];
  for (let i = 1; i < data.length; i++) {
    result.push(data[i] * k + result[i - 1] * (1 - k));
  }
  return result;
}

// ── KDJ ──────────────────────────────────────────────────────────────
// RSV = (close - lowest_low) / (highest_high - lowest_low) * 100
// K = 2/3 * K_prev + 1/3 * RSV
// D = 2/3 * D_prev + 1/3 * K
// J = 3 * K - 2 * D
export interface KDJData {
  k: number[];
  d: number[];
  j: number[];
}

export function calcKDJ(
  highs: number[],
  lows: number[],
  closes: number[],
  period = 9,
): KDJData {
  const len = closes.length;
  const k: number[] = [];
  const d: number[] = [];
  const j: number[] = [];

  let prevK = 50;
  let prevD = 50;

  for (let i = 0; i < len; i++) {
    const start = Math.max(0, i - period + 1);
    let highest = -Infinity;
    let lowest = Infinity;
    for (let s = start; s <= i; s++) {
      if (highs[s] > highest) highest = highs[s];
      if (lows[s] < lowest) lowest = lows[s];
    }
    const rsv =
      highest === lowest ? 50 : ((closes[i] - lowest) / (highest - lowest)) * 100;

    const curK = +(2 / 3 * prevK + 1 / 3 * rsv).toFixed(2);
    const curD = +(2 / 3 * prevD + 1 / 3 * curK).toFixed(2);
    const curJ = +(3 * curK - 2 * curD).toFixed(2);

    k.push(curK);
    d.push(curD);
    j.push(curJ);

    prevK = curK;
    prevD = curD;
  }

  return { k, d, j };
}

// ── RSI (Relative Strength Index) ────────────────────────────────────
// RSI = 100 - 100 / (1 + RS)
// RS = average_gain / average_loss over period
export function calcRSI(closes: number[], period: number): (number | null)[] {
  const result: (number | null)[] = [];

  if (closes.length < period + 1) {
    return closes.map(() => null);
  }

  // First RSI calculation uses simple average
  let gainSum = 0;
  let lossSum = 0;
  for (let i = 1; i <= period; i++) {
    const diff = closes[i] - closes[i - 1];
    if (diff > 0) gainSum += diff;
    else lossSum += Math.abs(diff);
  }

  // Fill nulls for indices 0..period-1
  for (let i = 0; i < period; i++) {
    result.push(null);
  }

  let avgGain = gainSum / period;
  let avgLoss = lossSum / period;

  // First RSI
  if (avgLoss === 0) result.push(100);
  else result.push(+(100 - 100 / (1 + avgGain / avgLoss)).toFixed(2));

  // Subsequent values use Wilder's smoothing
  for (let i = period + 1; i < closes.length; i++) {
    const diff = closes[i] - closes[i - 1];
    const gain = diff > 0 ? diff : 0;
    const loss = diff < 0 ? Math.abs(diff) : 0;

    avgGain = (avgGain * (period - 1) + gain) / period;
    avgLoss = (avgLoss * (period - 1) + loss) / period;

    if (avgLoss === 0) result.push(100);
    else result.push(+(100 - 100 / (1 + avgGain / avgLoss)).toFixed(2));
  }

  return result;
}
