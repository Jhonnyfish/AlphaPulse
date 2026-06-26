"use client";

import { useState, useEffect, useCallback } from "react";
import { portfolioApi, searchApi } from "@/lib/api-client";
import { useAuth } from "@/lib/auth";
import type { PortfolioPositionEnriched, SearchSuggestion } from "@/lib/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Briefcase, Plus, Trash2, Loader2, Search, X, Pencil, Check,
} from "lucide-react";

export default function PortfolioPage() {
  const { user, loading: authLoading } = useAuth();
  const [positions, setPositions] = useState<PortfolioPositionEnriched[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // Add form state
  const [showForm, setShowForm] = useState(false);
  const [formCode, setFormCode] = useState("");
  const [formName, setFormName] = useState("");
  const [formCost, setFormCost] = useState("");
  const [formQty, setFormQty] = useState("");
  const [formNotes, setFormNotes] = useState("");
  const [adding, setAdding] = useState(false);
  const [suggestions, setSuggestions] = useState<SearchSuggestion[]>([]);

  // Edit form state
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editCost, setEditCost] = useState("");
  const [editQty, setEditQty] = useState("");
  const [saving, setSaving] = useState(false);

  const fetchPositions = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const res = await portfolioApi.list();
      setPositions(res.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!authLoading && user) fetchPositions();
  }, [authLoading, user, fetchPositions]);

  const handleSearch = async (q: string) => {
    setFormCode(q);
    if (q.length >= 2) {
      try {
        const results = await searchApi.stocks(q);
        setSuggestions(results);
      } catch { /* ignore */ }
    } else {
      setSuggestions([]);
    }
  };

  const selectStock = (s: SearchSuggestion) => {
    setFormCode(s.code);
    setFormName(s.name);
    setSuggestions([]);
  };

  const handleAdd = async () => {
    if (!formCode || !formCost || !formQty) return;
    setAdding(true);
    setError("");
    try {
      await portfolioApi.add({
        code: formCode,
        cost_price: parseFloat(formCost),
        quantity: parseInt(formQty),
        notes: formNotes,
      });
      setShowForm(false);
      setFormCode("");
      setFormName("");
      setFormCost("");
      setFormQty("");
      setFormNotes("");
      fetchPositions();
    } catch (err) {
      setError(err instanceof Error ? err.message : "添加失败");
    } finally {
      setAdding(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await portfolioApi.delete(id);
      fetchPositions();
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除失败");
    }
  };

  const startEdit = (pos: PortfolioPositionEnriched) => {
    setEditingId(pos.id);
    setEditCost(String(pos.cost_price));
    setEditQty(String(pos.quantity));
    setError("");
  };

  const cancelEdit = () => {
    setEditingId(null);
    setEditCost("");
    setEditQty("");
  };

  const handleSaveEdit = async () => {
    if (!editingId || !editCost || !editQty) return;
    setSaving(true);
    setError("");
    try {
      await portfolioApi.update(editingId, {
        cost_price: parseFloat(editCost),
        quantity: parseInt(editQty),
      });
      setEditingId(null);
      fetchPositions();
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存失败");
    } finally {
      setSaving(false);
    }
  };

  if (authLoading) return null;
  if (!user) return null;

  const totalMV = positions.reduce((s, p) => s + p.market_value, 0);
  const totalCost = positions.reduce((s, p) => s + p.total_cost, 0);
  const totalPnL = positions.reduce((s, p) => s + p.pnl, 0);

  return (
    <div className="mx-auto max-w-5xl space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Briefcase className="h-6 w-6 text-primary" />
          <div>
            <h1 className="text-2xl font-bold text-foreground">持仓管理</h1>
            <p className="text-sm text-muted-foreground">
              管理您的股票持仓，添加后在做T建议中将显示具体数量
            </p>
          </div>
        </div>
        <Button onClick={() => setShowForm(!showForm)} size="sm">
          {showForm ? <X className="mr-1 h-4 w-4" /> : <Plus className="mr-1 h-4 w-4" />}
          {showForm ? "取消" : "添加持仓"}
        </Button>
      </div>

      {/* Add form */}
      {showForm && (
        <Card>
          <CardHeader className="pb-2 pt-4 px-4">
            <CardTitle className="text-base">新增持仓</CardTitle>
          </CardHeader>
          <CardContent className="px-4 pb-4 space-y-4">
            {/* Stock search */}
            <div className="space-y-1">
              <Label className="text-sm">股票代码/名称</Label>
              <div className="relative">
                <div className="relative">
                  <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    value={formName ? `${formName} (${formCode})` : formCode}
                    onChange={(e) => handleSearch(e.target.value)}
                    placeholder="输入代码或名称搜索，如 600519 或 茅台"
                    className="pl-9"
                  />
                </div>
                {suggestions.length > 0 && (
                  <div className="absolute z-10 mt-1 w-full rounded-md border border-border bg-popover shadow-md max-h-48 overflow-y-auto">
                    {suggestions.map((s) => (
                      <button
                        key={s.code}
                        type="button"
                        className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent text-left"
                        onClick={() => selectStock(s)}
                      >
                        <span className="font-mono text-muted-foreground">{s.code}</span>
                        <span>{s.name}</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1">
                <Label className="text-sm">成本价</Label>
                <Input
                  type="number"
                  step="0.01"
                  value={formCost}
                  onChange={(e) => setFormCost(e.target.value)}
                  placeholder="48.50"
                />
              </div>
              <div className="space-y-1">
                <Label className="text-sm">持仓数量（股）</Label>
                <Input
                  type="number"
                  step="100"
                  value={formQty}
                  onChange={(e) => setFormQty(e.target.value)}
                  placeholder="300"
                />
              </div>
            </div>

            <div className="space-y-1">
              <Label className="text-sm">备注（可选）</Label>
              <Input
                value={formNotes}
                onChange={(e) => setFormNotes(e.target.value)}
                placeholder="如：第一仓位"
              />
            </div>

            {error && (
              <div className="rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>
            )}

            <Button onClick={handleAdd} disabled={adding || !formCode || !formCost || !formQty} className="w-full">
              {adding ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Plus className="mr-2 h-4 w-4" />}
              确认添加
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Summary */}
      {positions.length > 0 && (
        <div className="grid grid-cols-3 gap-4">
          <Card>
            <CardContent className="px-4 py-3">
              <div className="text-xs text-muted-foreground">总市值</div>
              <div className="text-lg font-semibold font-mono">
                {totalMV >= 10000 ? `${(totalMV / 10000).toFixed(2)}万` : totalMV.toFixed(2)}
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="px-4 py-3">
              <div className="text-xs text-muted-foreground">总成本</div>
              <div className="text-lg font-semibold font-mono">
                {totalCost >= 10000 ? `${(totalCost / 10000).toFixed(2)}万` : totalCost.toFixed(2)}
              </div>
            </CardContent>
          </Card>
          <Card className={totalPnL >= 0 ? "border-green-500/30" : "border-red-500/30"}>
            <CardContent className="px-4 py-3">
              <div className="text-xs text-muted-foreground">总盈亏</div>
              <div className={`text-lg font-semibold font-mono ${totalPnL >= 0 ? "text-green-500" : "text-red-500"}`}>
                {totalPnL >= 0 ? "+" : ""}
                {totalPnL >= 10000 ? `${(totalPnL / 10000).toFixed(2)}万` : totalPnL.toFixed(2)}
                {totalCost > 0 && (
                  <span className="text-xs ml-1">
                    ({totalPnL >= 0 ? "+" : ""}{((totalPnL / totalCost) * 100).toFixed(2)}%)
                  </span>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Positions list */}
      {loading ? (
        <div className="flex justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : positions.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <Briefcase className="mx-auto h-10 w-10 text-muted-foreground/50" />
            <p className="mt-3 text-sm text-muted-foreground">
              暂无持仓，点击上方「添加持仓」开始管理
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              添加持仓后，在个股分析页面将显示持仓盈亏，做T建议将显示具体操作股数
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader className="pb-2 pt-4 px-4">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              持仓明细 ({positions.length}只)
            </CardTitle>
          </CardHeader>
          <CardContent className="px-4 pb-4">
            {error && (
              <div className="mb-3 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>
            )}
            <div className="space-y-3">
              {positions.map((pos) => (
                <div
                  key={pos.id}
                  className="flex items-center gap-4 rounded-lg border border-border p-3"
                >
                  {/* Name + code */}
                  <div className="min-w-[100px]">
                    <div className="text-sm font-medium">{pos.name || pos.code}</div>
                    <div className="text-xs text-muted-foreground font-mono">{pos.code}</div>
                  </div>

                  {pos.id === editingId ? (
                    <div className="flex-1 grid grid-cols-2 gap-4">
                      <div className="space-y-1">
                        <Label className="text-xs">成本价</Label>
                        <Input
                          type="number"
                          step="0.01"
                          value={editCost}
                          onChange={(e) => setEditCost(e.target.value)}
                          placeholder="48.50"
                        />
                      </div>
                      <div className="space-y-1">
                        <Label className="text-xs">持仓数量（股）</Label>
                        <Input
                          type="number"
                          step="100"
                          value={editQty}
                          onChange={(e) => setEditQty(e.target.value)}
                          placeholder="300"
                        />
                      </div>
                    </div>
                  ) : (
                    <div className="flex-1 grid grid-cols-5 gap-4 text-center">
                      <div>
                        <div className="text-xs text-muted-foreground">持仓</div>
                        <div className="text-sm font-mono">{pos.quantity.toLocaleString()}</div>
                      </div>
                      <div>
                        <div className="text-xs text-muted-foreground">成本</div>
                        <div className="text-sm font-mono">{pos.cost_price.toFixed(2)}</div>
                      </div>
                      <div>
                        <div className="text-xs text-muted-foreground">现价</div>
                        <div className="text-sm font-mono">{pos.current_price.toFixed(2)}</div>
                      </div>
                      <div>
                        <div className="text-xs text-muted-foreground">市值</div>
                        <div className="text-sm font-mono">
                          {pos.market_value >= 10000
                            ? `${(pos.market_value / 10000).toFixed(2)}万`
                            : pos.market_value.toFixed(2)}
                        </div>
                      </div>
                      <div>
                        <div className="text-xs text-muted-foreground">盈亏</div>
                        <div className={`text-sm font-semibold font-mono ${pos.pnl >= 0 ? "text-green-500" : "text-red-500"}`}>
                          {pos.pnl >= 0 ? "+" : ""}{pos.pnl.toFixed(0)}
                          <Badge
                            variant="outline"
                            className={`ml-1 text-[10px] ${pos.pnl >= 0 ? "text-green-500 border-green-500/30" : "text-red-500 border-red-500/30"}`}
                          >
                            {pos.pnl >= 0 ? "+" : ""}{pos.pnl_pct.toFixed(2)}%
                          </Badge>
                        </div>
                      </div>
                    </div>
                  )}

                  {pos.id === editingId ? (
                    <div className="flex items-center gap-1 shrink-0">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-green-500 hover:text-green-500"
                        onClick={handleSaveEdit}
                        disabled={saving || !editCost || !editQty}
                      >
                        {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-muted-foreground"
                        onClick={cancelEdit}
                        disabled={saving}
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    </div>
                  ) : (
                    <div className="flex items-center gap-1 shrink-0">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-muted-foreground hover:text-primary"
                        onClick={() => startEdit(pos)}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-muted-foreground hover:text-red-500"
                        onClick={() => handleDelete(pos.id)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
