import { useState, useEffect } from 'react';
import api from '@/lib/api';
import { Building2, RefreshCw, TrendingUp, TrendingDown } from 'lucide-react';

interface Institution {
  name: string;
  appearances: number;
  total_net: number;
  dates: string[];
}

interface InstitutionResponse {
  ok: boolean;
  institutions: Institution[];
}

export default function InstitutionPage() {
  const [data, setData] = useState<Institution[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [sortBy, setSortBy] = useState<'total_net' | 'appearances'>('total_net');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');

  const fetchData = () => {
    setLoading(true);
    setError('');
    api
      .get<InstitutionResponse>('/institution-tracker')
      .then((res) => setData(res.data.institutions || []))
      .catch(() => setError('加载机构数据失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchData();
  }, []);

  const handleSort = (col: 'total_net' | 'appearances') => {
    if (sortBy === col) {
      setSortDir((d) => (d === 'desc' ? 'asc' : 'desc'));
    } else {
      setSortBy(col);
      setSortDir('desc');
    }
  };

  const sorted = [...data].sort((a, b) => {
    const mul = sortDir === 'desc' ? -1 : 1;
    return (a[sortBy] - b[sortBy]) * mul;
  });

  const formatAmount = (val: number) => {
    const abs = Math.abs(val);
    if (abs >= 1e8) return `${(val / 1e8).toFixed(2)}亿`;
    if (abs >= 1e4) return `${(val / 1e4).toFixed(2)}万`;
    return val.toFixed(2);
  };

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Building2 className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">机构动向</h1>
        </div>
        <div className="space-y-3">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '50px' }} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Building2 className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">机构动向</h1>
        </div>
        <div className="text-sm px-4 py-3 rounded-lg" style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
          {error}
          <button onClick={fetchData} className="ml-3 underline">重试</button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Building2 className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">机构动向</h1>
          <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}>
            {data.length} 家机构
          </span>
        </div>
        <button onClick={fetchData} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      {sorted.length > 0 ? (
        <div className="glass-panel rounded-xl overflow-hidden" style={{ borderColor: 'var(--color-border)' }}>
          <table className="data-table">
            <thead>
              <tr>
                <th>#</th>
                <th>机构名称</th>
                <th className={sortBy === 'appearances' ? 'sorted' : ''} onClick={() => handleSort('appearances')}>
                  出现次数 {sortBy === 'appearances' ? (sortDir === 'desc' ? '↓' : '↑') : ''}
                </th>
                <th className={sortBy === 'total_net' ? 'sorted' : ''} onClick={() => handleSort('total_net')}>
                  净买入 {sortBy === 'total_net' ? (sortDir === 'desc' ? '↓' : '↑') : ''}
                </th>
                <th>日期</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((inst, i) => (
                <tr key={inst.name}>
                  <td className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{i + 1}</td>
                  <td className="font-medium text-sm">{inst.name}</td>
                  <td className="font-mono">{inst.appearances}</td>
                  <td>
                    <div className="flex items-center gap-1">
                      {inst.total_net > 0 ? (
                        <TrendingUp className="w-3.5 h-3.5" style={{ color: 'var(--color-danger)' }} />
                      ) : (
                        <TrendingDown className="w-3.5 h-3.5" style={{ color: 'var(--color-success)' }} />
                      )}
                      <span
                        className="font-mono font-medium"
                        style={{ color: inst.total_net > 0 ? 'var(--color-danger)' : 'var(--color-success)' }}
                      >
                        {formatAmount(inst.total_net)}
                      </span>
                    </div>
                  </td>
                  <td>
                    <div className="flex flex-wrap gap-1">
                      {inst.dates.slice(0, 3).map((d) => (
                        <span
                          key={d}
                          className="text-[10px] px-1.5 py-0.5 rounded"
                          style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}
                        >
                          {d.slice(5)}
                        </span>
                      ))}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div
          className="text-center py-16 rounded-xl"
          style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}
        >
          <Building2 className="w-12 h-12 mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p style={{ color: 'var(--color-text-muted)' }}>暂无机构数据</p>
        </div>
      )}
    </div>
  );
}
