import { useState, useEffect } from 'react';
import api, { reportsApi } from '@/lib/api';
import { FileText, RefreshCw, Calendar, ChevronRight, ArrowLeft, Download, Clock } from 'lucide-react';

interface ReportItem {
  filename: string;
  date: string;
  size: number;
  mtime: string;
  type: string;
  preview: string;
}

interface ReportContent {
  ok: boolean;
  filename: string;
  date: string;
  size: number;
  mtime: string;
  content: string;
}

export default function DailyReportPage() {
  const [reports, setReports] = useState<ReportItem[]>([]);
  const [selected, setSelected] = useState<ReportContent | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingContent, setLoadingContent] = useState(false);
  const [error, setError] = useState('');

  const fetchList = () => {
    setLoading(true);
    setError('');
    api.get<{ ok: boolean; reports: ReportItem[] }>('/daily-report/list')
      .then((res) => setReports(res.data.reports || []))
      .catch(() => setError('加载日报列表失败'))
      .finally(() => setLoading(false));
  };

  const fetchReport = (filename: string) => {
    setLoadingContent(true);
    api.get<ReportContent>(`/reports/${filename}`)
      .then((res) => setSelected(res.data))
      .catch(() => setError('加载日报内容失败'))
      .finally(() => setLoadingContent(false));
  };

  const fetchLatest = () => {
    setLoadingContent(true);
    api.get<ReportContent>('/daily-report/latest')
      .then((res) => setSelected(res.data))
      .catch(() => setError('暂无日报'))
      .finally(() => setLoadingContent(false));
  };

  useEffect(() => {
    fetchList();
    fetchLatest();
  }, []);

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes}B`;
    return `${(bytes / 1024).toFixed(1)}KB`;
  };

  const formatDate = (date: string) => {
    if (date.length === 8) {
      return `${date.slice(0, 4)}-${date.slice(4, 6)}-${date.slice(6, 8)}`;
    }
    return date;
  };

  // Simple markdown → HTML (handles tables, headers, bold, blockquotes, lists)
  const renderMarkdown = (md: string) => {
    let html = md
      // Tables
      .replace(/\|(.+)\|\n\|[-|\s]+\|\n((?:\|.+\|\n?)*)/g, (_match, header, body) => {
        const ths = header.split('|').filter((s: string) => s.trim()).map((s: string) => `<th style="padding:6px 12px;border-bottom:1px solid var(--color-border);text-align:left;font-size:12px;color:var(--color-text-secondary)">${s.trim()}</th>`).join('');
        const rows = body.trim().split('\n').map((row: string) => {
          const tds = row.split('|').filter((s: string) => s.trim()).map((s: string) => `<td style="padding:6px 12px;border-bottom:1px solid var(--color-border);font-size:13px">${s.trim()}</td>`).join('');
          return `<tr>${tds}</tr>`;
        }).join('');
        return `<div style="overflow-x:auto;margin:12px 0"><table style="width:100%;border-collapse:collapse"><thead><tr>${ths}</tr></thead><tbody>${rows}</tbody></table></div>`;
      })
      // Headers
      .replace(/^### (.+)$/gm, '<h4 style="font-size:14px;font-weight:600;margin:16px 0 8px;color:var(--color-text-primary)">$1</h4>')
      .replace(/^## (.+)$/gm, '<h3 style="font-size:16px;font-weight:700;margin:20px 0 10px;color:var(--color-text-primary)">$1</h3>')
      .replace(/^# (.+)$/gm, '<h2 style="font-size:20px;font-weight:700;margin:24px 0 12px;color:var(--color-text-primary)">$1</h2>')
      // Blockquote
      .replace(/^> (.+)$/gm, '<blockquote style="border-left:3px solid var(--color-accent);padding-left:12px;margin:8px 0;color:var(--color-text-secondary);font-size:13px">$1</blockquote>')
      // Bold
      .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
      // Horizontal rule
      .replace(/^---$/gm, '<hr style="border:none;border-top:1px solid var(--color-border);margin:16px 0" />')
      // Line breaks (non-empty lines become paragraphs)
      .replace(/^(?!<[hbt]|<hr|<div|<blockquote)(.+)$/gm, '<p style="margin:4px 0;line-height:1.7;font-size:14px">$1</p>');

    return html;
  };

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <FileText className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">每日报告</h1>
        </div>
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '60px' }} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <FileText className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">每日报告</h1>
        </div>
        <div className="text-sm px-4 py-3 rounded-lg" style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
          {error}
          <button onClick={() => { fetchList(); fetchLatest(); }} className="ml-3 underline">重试</button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <FileText className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">每日报告</h1>
          <span className="text-xs px-2 py-0.5 rounded-full font-mono" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}>
            {reports.length} 份
          </span>
        </div>
        <button onClick={() => { fetchList(); fetchLatest(); }} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
          <RefreshCw className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Report list */}
        <div className="glass-panel rounded-xl p-4 lg:col-span-1" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-medium mb-3 flex items-center gap-2" style={{ color: 'var(--color-text-secondary)' }}>
            <Calendar className="w-4 h-4" />
            报告列表
          </h3>
          {reports.length === 0 ? (
            <div className="text-center py-8" style={{ color: 'var(--color-text-muted)' }}>
              <FileText className="w-8 h-8 mx-auto mb-2" />
              <p className="text-sm">暂无报告</p>
            </div>
          ) : (
            <div className="space-y-2">
              {reports.map((r) => (
                <button
                  key={r.filename}
                  onClick={() => fetchReport(r.filename)}
                  className="w-full text-left p-3 rounded-lg transition-all hover:scale-[1.01]"
                  style={{
                    background: selected?.filename === r.filename ? 'rgba(59,130,246,0.12)' : 'var(--color-bg-hover)',
                    border: selected?.filename === r.filename ? '1px solid var(--color-accent)' : '1px solid transparent',
                  }}
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-sm font-medium">{formatDate(r.date)}</div>
                      <div className="text-xs mt-0.5" style={{ color: 'var(--color-text-muted)' }}>{r.preview}</div>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-mono" style={{ color: 'var(--color-text-muted)' }}>{formatSize(r.size)}</span>
                      <ChevronRight className="w-3.5 h-3.5" style={{ color: 'var(--color-text-muted)' }} />
                    </div>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Report content */}
        <div className="glass-panel rounded-xl p-6 lg:col-span-2" style={{ borderColor: 'var(--color-border)', minHeight: '400px' }}>
          {loadingContent ? (
            <div className="space-y-3">
              {Array.from({ length: 8 }).map((_, i) => (
                <div key={i} className="animate-pulse rounded" style={{ height: '16px', width: `${60 + Math.random() * 40}%`, background: 'var(--color-bg-hover)' }} />
              ))}
            </div>
          ) : selected ? (
            <div>
              <div className="flex items-center justify-between mb-4 pb-3" style={{ borderBottom: '1px solid var(--color-border)' }}>
                <div>
                  <h2 className="font-bold text-lg">{formatDate(selected.date)} 日报</h2>
                  <div className="flex items-center gap-3 mt-1 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                    <span className="flex items-center gap-1"><Clock className="w-3 h-3" />{selected.mtime}</span>
                    <span>{formatSize(selected.size)}</span>
                  </div>
                </div>
              </div>
              <div
                style={{ color: 'var(--color-text-primary)' }}
                dangerouslySetInnerHTML={{ __html: renderMarkdown(selected.content || '') }}
              />
            </div>
          ) : (
            <div className="text-center py-16" style={{ color: 'var(--color-text-muted)' }}>
              <FileText className="w-12 h-12 mx-auto mb-3" />
              <p>选择一份报告查看内容</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
