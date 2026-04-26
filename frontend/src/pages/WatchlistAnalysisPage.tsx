import { useState, useEffect, useCallback } from 'react';
import {
  watchlistAnalysisApi,
  type HeatmapItem,
  type WatchlistRanking,
  type WatchlistGroup,
} from '@/lib/api';
import { Grid3X3, PieChart, Trophy, FolderOpen, Plus, Trash2, Edit2, RefreshCw } from 'lucide-react';
import ErrorState from '@/components/ErrorState';

const TABS = ['热力图', '板块分布', '排名', '分组管理'] as const;
type Tab = (typeof TABS)[number];

export default function WatchlistAnalysisPage() {
  const [tab, setTab] = useState<Tab>('热力图');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Heatmap
  const [heatmap, setHeatmap] = useState<HeatmapItem[]>([]);
  // Sectors
  const [sectors, setSectors] = useState<Record<string, number>>([]);
  // Ranking
  const [ranking, setRanking] = useState<WatchlistRanking[]>([]);
  // Groups
  const [groups, setGroups] = useState<WatchlistGroup[]>([]);
  const [newGroupName, setNewGroupName] = useState('');
  const [editGroupId, setEditGroupId] = useState<string | null>(null);
  const [editGroupName, setEditGroupName] = useState('');

  const fetchHeatmap = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await watchlistAnalysisApi.heatmap();
      const hmData = res.data;
      setHeatmap(hmData && typeof hmData === 'object' && 'items' in hmData ? hmData.items : []);
    } catch {
      setError('加载热力图数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchSectors = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await watchlistAnalysisApi.sectors();
      setSectors(res.data && typeof res.data === 'object' ? res.data : {});
    } catch {
      setError('加载板块分布失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchRanking = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await watchlistAnalysisApi.ranking();
      setRanking(Array.isArray(res.data) ? res.data : []);
    } catch {
      setError('加载排名失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchGroups = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await watchlistAnalysisApi.groups();
      setGroups(Array.isArray(res.data) ? res.data : []);
    } catch {
      setError('加载分组失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (tab === '热力图') fetchHeatmap();
    else if (tab === '板块分布') fetchSectors();
    else if (tab === '排名') fetchRanking();
    else if (tab === '分组管理') fetchGroups();
  }, [tab, fetchHeatmap, fetchSectors, fetchRanking, fetchGroups]);

  const handleCreateGroup = async () => {
    if (!newGroupName.trim()) return;
    try {
      await watchlistAnalysisApi.createGroup(newGroupName.trim());
      setNewGroupName('');
      fetchGroups();
    } catch {
      setError('创建分组失败');
    }
  };

  const handleUpdateGroup = async (id: string) => {
    if (!editGroupName.trim()) return;
    try {
      await watchlistAnalysisApi.updateGroup(id, editGroupName.trim());
      setEditGroupId(null);
      setEditGroupName('');
      fetchGroups();
    } catch {
      setError('更新分组失败');
    }
  };

  const handleDeleteGroup = async (id: string) => {
    if (!confirm('确定删除该分组？')) return;
    try {
      await watchlistAnalysisApi.deleteGroup(id);
      fetchGroups();
    } catch {
      setError('删除分组失败');
    }
  };

  const changeColor = (n: number) =>
    n > 0 ? 'var(--color-danger)' : n < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';

  // Heatmap: compute min/max for color scaling
  const maxAbsChange = heatmap.length
    ? Math.max(...heatmap.map((h) => Math.abs(h.change_pct)), 1)
    : 1;

  // Sectors: compute total for percentage
  const sectorEntries = Object.entries(sectors);
  const sectorTotal = sectorEntries.reduce((s, [, v]) => s + v, 0) || 1;

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold flex items-center gap-2">
          <Grid3X3 size={22} /> 自选分析
        </h1>
        <button
          onClick={() => {
            if (tab === '热力图') fetchHeatmap();
            else if (tab === '板块分布') fetchSectors();
            else if (tab === '排名') fetchRanking();
            else fetchGroups();
          }}
          className="p-2 rounded-lg hover:bg-gray-700 transition"
          title="刷新"
        >
          <RefreshCw size={16} />
        </button>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-gray-800 rounded-lg p-1">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`flex-1 py-2 px-3 rounded-md text-sm font-medium transition ${
              tab === t ? 'bg-gray-700 text-white' : 'text-gray-400 hover:text-gray-200'
            }`}
          >
            {t === '热力图' && <Grid3X3 size={14} className="inline mr-1" />}
            {t === '板块分布' && <PieChart size={14} className="inline mr-1" />}
            {t === '排名' && <Trophy size={14} className="inline mr-1" />}
            {t === '分组管理' && <FolderOpen size={14} className="inline mr-1" />}
            {t}
          </button>
        ))}
      </div>

      {error && (
        <ErrorState
          title="加载失败"
          description={error}
          onRetry={() => {
            setError('');
            if (tab === '热力图') fetchHeatmap();
            else if (tab === '板块分布') fetchSectors();
            else if (tab === '排名') fetchRanking();
            else fetchGroups();
          }}
        />
      )}

      {/* Heatmap Tab */}
      {tab === '热力图' && (
        <div className="bg-gray-800 rounded-xl p-4">
          {loading ? (
            <div className="text-center text-gray-400 py-12">加载中...</div>
          ) : heatmap.length === 0 ? (
            <div className="text-center text-gray-400 py-12">暂无自选股数据</div>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-2">
              {heatmap.map((item) => {
                const intensity = Math.min(Math.abs(item.change_pct) / maxAbsChange, 1);
                const bg =
                  item.change_pct > 0
                    ? `rgba(239,68,68,${0.15 + intensity * 0.55})`
                    : item.change_pct < 0
                    ? `rgba(34,197,94,${0.15 + intensity * 0.55})`
                    : 'rgba(107,114,128,0.2)';
                return (
                  <div
                    key={item.code}
                    className="rounded-lg p-3 text-center cursor-pointer hover:ring-1 hover:ring-gray-500 transition"
                    style={{ backgroundColor: bg }}
                    title={`${item.name} (${item.code})\n涨跌: ${item.change_pct.toFixed(2)}%\n成交量: ${item.volume.toLocaleString()}\n板块: ${item.sector}`}
                  >
                    <div className="text-sm font-medium text-gray-100 truncate">{item.name}</div>
                    <div className="text-xs text-gray-300 mt-0.5">{item.code}</div>
                    <div
                      className="text-sm font-bold mt-1"
                      style={{ color: changeColor(item.change_pct) }}
                    >
                      {item.change_pct > 0 ? '+' : ''}
                      {item.change_pct.toFixed(2)}%
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* Sectors Tab */}
      {tab === '板块分布' && (
        <div className="bg-gray-800 rounded-xl p-4">
          {loading ? (
            <div className="text-center text-gray-400 py-12">加载中...</div>
          ) : sectorEntries.length === 0 ? (
            <div className="text-center text-gray-400 py-12">暂无板块数据</div>
          ) : (
            <div className="space-y-3">
              {sectorEntries
                .sort((a, b) => b[1] - a[1])
                .map(([name, count]) => {
                  const pct = (count / sectorTotal) * 100;
                  return (
                    <div key={name} className="flex items-center gap-3">
                      <div className="w-28 text-sm text-gray-300 truncate">{name}</div>
                      <div className="flex-1 bg-gray-700 rounded-full h-5 overflow-hidden">
                        <div
                          className="h-full rounded-full bg-blue-500/70 transition-all"
                          style={{ width: `${Math.min(pct, 100)}%` }}
                        />
                      </div>
                      <div className="w-16 text-right text-sm text-gray-400">
                        {count}只 ({pct.toFixed(1)}%)
                      </div>
                    </div>
                  );
                })}
            </div>
          )}
        </div>
      )}

      {/* Ranking Tab */}
      {tab === '排名' && (
        <div className="bg-gray-800 rounded-xl overflow-hidden">
          {loading ? (
            <div className="text-center text-gray-400 py-12">加载中...</div>
          ) : ranking.length === 0 ? (
            <div className="text-center text-gray-400 py-12">暂无排名数据</div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-700 text-gray-400">
                  <th className="px-4 py-3 text-left">排名</th>
                  <th className="px-4 py-3 text-left">代码</th>
                  <th className="px-4 py-3 text-left">名称</th>
                  <th className="px-4 py-3 text-right">评分</th>
                  <th className="px-4 py-3 text-right">涨跌幅</th>
                </tr>
              </thead>
              <tbody>
                {ranking.map((item) => (
                  <tr
                    key={item.code}
                    className="border-b border-gray-700/50 hover:bg-gray-700/30 transition"
                  >
                    <td className="px-4 py-3">
                      <span
                        className={`inline-flex items-center justify-center w-6 h-6 rounded-full text-xs font-bold ${
                          item.rank <= 3
                            ? 'bg-yellow-500/20 text-yellow-400'
                            : 'bg-gray-700 text-gray-400'
                        }`}
                      >
                        {item.rank}
                      </span>
                    </td>
                    <td className="px-4 py-3 font-mono text-gray-300">{item.code}</td>
                    <td className="px-4 py-3">{item.name}</td>
                    <td className="px-4 py-3 text-right font-medium text-blue-400">
                      {item.score.toFixed(1)}
                    </td>
                    <td
                      className="px-4 py-3 text-right font-medium"
                      style={{ color: changeColor(item.change_pct) }}
                    >
                      {item.change_pct > 0 ? '+' : ''}
                      {item.change_pct.toFixed(2)}%
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {/* Groups Tab */}
      {tab === '分组管理' && (
        <div className="space-y-4">
          {/* Add group */}
          <div className="bg-gray-800 rounded-xl p-4">
            <div className="flex gap-2">
              <input
                type="text"
                value={newGroupName}
                onChange={(e) => setNewGroupName(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleCreateGroup()}
                placeholder="新建分组名称..."
                className="flex-1 bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-sm text-gray-100 placeholder:text-gray-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
              <button
                onClick={handleCreateGroup}
                disabled={!newGroupName.trim()}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg text-sm font-medium transition flex items-center gap-1"
              >
                <Plus size={14} /> 创建
              </button>
            </div>
          </div>

          {/* Groups list */}
          <div className="bg-gray-800 rounded-xl overflow-hidden">
            {loading ? (
              <div className="text-center text-gray-400 py-12">加载中...</div>
            ) : groups.length === 0 ? (
              <div className="text-center text-gray-400 py-12">暂无分组</div>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-700 text-gray-400">
                    <th className="px-4 py-3 text-left">分组名称</th>
                    <th className="px-4 py-3 text-right">股票数</th>
                    <th className="px-4 py-3 text-right">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {groups.map((group) => (
                    <tr
                      key={group.id}
                      className="border-b border-gray-700/50 hover:bg-gray-700/30 transition"
                    >
                      <td className="px-4 py-3">
                        {editGroupId === group.id ? (
                          <div className="flex gap-2">
                            <input
                              type="text"
                              value={editGroupName}
                              onChange={(e) => setEditGroupName(e.target.value)}
                              onKeyDown={(e) => {
                                if (e.key === 'Enter') handleUpdateGroup(group.id);
                                if (e.key === 'Escape') {
                                  setEditGroupId(null);
                                  setEditGroupName('');
                                }
                              }}
                              className="bg-gray-700 border border-gray-600 rounded px-2 py-1 text-sm text-gray-100 focus:outline-none focus:ring-1 focus:ring-blue-500"
                              autoFocus
                            />
                            <button
                              onClick={() => handleUpdateGroup(group.id)}
                              className="text-blue-400 hover:text-blue-300 text-xs"
                            >
                              保存
                            </button>
                            <button
                              onClick={() => {
                                setEditGroupId(null);
                                setEditGroupName('');
                              }}
                              className="text-gray-400 hover:text-gray-300 text-xs"
                            >
                              取消
                            </button>
                          </div>
                        ) : (
                          <div className="flex items-center gap-2">
                            <FolderOpen size={14} className="text-gray-500" />
                            <span>{group.name}</span>
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-3 text-right text-gray-400">{group.stock_count}</td>
                      <td className="px-4 py-3 text-right">
                        {editGroupId !== group.id && (
                          <div className="flex items-center justify-end gap-2">
                            <button
                              onClick={() => {
                                setEditGroupId(group.id);
                                setEditGroupName(group.name);
                              }}
                              className="p-1.5 rounded hover:bg-gray-600 transition text-gray-400 hover:text-gray-200"
                              title="编辑"
                            >
                              <Edit2 size={14} />
                            </button>
                            <button
                              onClick={() => handleDeleteGroup(group.id)}
                              className="p-1.5 rounded hover:bg-gray-600 transition text-red-400 hover:text-red-300"
                              title="删除"
                            >
                              <Trash2 size={14} />
                            </button>
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
