import { useNavigate } from 'react-router-dom';
import {
  Zap,
  RefreshCw,
  Search,
  Star,
  BarChart3,
  TrendingUp,
  Briefcase,
  BookOpen,
  Target,
  Activity,
  Newspaper,
  Settings,
  Flame,
  Crown,
  Filter,
  CalendarClock,
  Gauge,
  GitBranch,
  Network,
  FlaskConical,
  Radio,
  Grid3X3,
  Building2,
  AlertTriangle,
  Monitor,
  FileText,
  Trophy,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

interface ActionItem {
  label: string;
  icon: LucideIcon;
  route?: string;
  action?: () => void;
}

interface ActionGroup {
  title: string;
  color: string;
  borderColor: string;
  items: ActionItem[];
}

export default function QuickActionsPage() {
  const navigate = useNavigate();

  const groups: ActionGroup[] = [
    {
      title: '市场速览',
      color: 'var(--color-blue-500)',
      borderColor: 'var(--color-blue-500)',
      items: [
        { label: '刷新行情', icon: RefreshCw, action: () => window.location.reload() },
        { label: '市场总览', icon: BarChart3, route: '/dashboard' },
        { label: '大盘行情', icon: TrendingUp, route: '/market' },
        { label: 'K线图', icon: Activity, route: '/kline' },
      ],
    },
    {
      title: '分析工具',
      color: 'var(--color-purple-500)',
      borderColor: 'var(--color-purple-500)',
      items: [
        { label: '个股分析', icon: Search, route: '/analyze' },
        { label: '板块轮动', icon: Grid3X3, route: '/sectors' },
        { label: '资金流向', icon: Gauge, route: '/flow' },
        { label: '趋势分析', icon: TrendingUp, route: '/trends' },
        { label: '市场广度', icon: Network, route: '/breadth' },
        { label: '市场情绪', icon: Radio, route: '/sentiment' },
        { label: '多周期趋势', icon: GitBranch, route: '/multi-trend' },
        { label: '相关性分析', icon: FlaskConical, route: '/correlation' },
      ],
    },
    {
      title: '选股工具',
      color: 'var(--color-amber-500)',
      borderColor: 'var(--color-amber-500)',
      items: [
        { label: '候选股', icon: Star, route: '/candidates' },
        { label: '选股器', icon: Filter, route: '/screener' },
        { label: '综合排名', icon: Trophy, route: '/ranking' },
        { label: '热门概念', icon: Flame, route: '/hot-concepts' },
        { label: '龙虎榜', icon: Crown, route: '/dragon-tiger' },
        { label: '形态扫描', icon: Target, route: '/pattern-scanner' },
      ],
    },
    {
      title: '交易管理',
      color: 'var(--color-green-500)',
      borderColor: 'var(--color-green-500)',
      items: [
        { label: '持仓管理', icon: Briefcase, route: '/portfolio' },
        { label: '交易日志', icon: BookOpen, route: '/journal' },
        { label: '策略管理', icon: Settings, route: '/strategies' },
        { label: '策略回测', icon: FlaskConical, route: '/backtest' },
        { label: '策略评估', icon: Monitor, route: '/strategy-eval' },
        { label: '交易日历', icon: CalendarClock, route: '/trade-calendar' },
        { label: '投资计划', icon: FileText, route: '/investment-plans' },
      ],
    },
    {
      title: '报告与监控',
      color: 'var(--color-rose-500)',
      borderColor: 'var(--color-rose-500)',
      items: [
        { label: '每日简报', icon: Newspaper, route: '/daily-brief' },
        { label: '每日报告', icon: FileText, route: '/daily-report' },
        { label: '绩效统计', icon: BarChart3, route: '/perf-stats' },
        { label: '系统诊断', icon: Monitor, route: '/diag' },
        { label: '异常检测', icon: AlertTriangle, route: '/anomalies' },
        { label: '机构动向', icon: Building2, route: '/institutions' },
      ],
    },
  ];

  const handleClick = (item: ActionItem) => {
    if (item.action) {
      item.action();
    } else if (item.route) {
      navigate(item.route);
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-6 space-y-8">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Zap className="w-7 h-7" style={{ color: 'var(--color-yellow-500)' }} />
        <h1 className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
          快捷操作
        </h1>
      </div>

      {/* Groups */}
      {groups.map((group) => (
        <section key={group.title}>
          <h2
            className="text-lg font-semibold mb-4 flex items-center gap-2"
            style={{ color: group.color }}
          >
            <span
              className="w-1 h-5 rounded-full inline-block"
              style={{ backgroundColor: group.color }}
            />
            {group.title}
          </h2>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
            {group.items.map((item) => {
              const Icon = item.icon;
              return (
                <button
                  key={item.label}
                  onClick={() => handleClick(item)}
                  className="glass-panel rounded-xl p-4 flex flex-col items-center justify-center gap-3 cursor-pointer transition-all duration-200 hover:scale-[1.03] hover:shadow-lg group"
                  style={{
                    borderColor: group.borderColor,
                    borderWidth: '1px',
                    borderStyle: 'solid',
                    borderLeftWidth: '3px',
                    background: `linear-gradient(135deg, color-mix(in srgb, ${group.borderColor} 5%, transparent), transparent)`,
                  }}
                >
                  <Icon
                    className="w-7 h-7 transition-colors duration-200"
                    style={{ color: group.color }}
                  />
                  <span
                    className="text-sm font-medium"
                    style={{ color: 'var(--color-text-primary)' }}
                  >
                    {item.label}
                  </span>
                </button>
              );
            })}
          </div>
        </section>
      ))}
    </div>
  );
}
