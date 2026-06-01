"use client";

import { useState, useEffect, useCallback } from "react";
import { syncApi } from "@/lib/api-client";
import { useAuth } from "@/lib/auth";
import type { SyncStatus, SyncConfig } from "@/lib/types";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import {
  Settings,
  Database,
  Clock,
  RefreshCw,
  CheckCircle2,
  XCircle,
  Save,
  Loader2,
} from "lucide-react";

export default function SettingsPage() {
  const { user, loading: authLoading } = useAuth();
  const [config, setConfig] = useState<SyncConfig | null>(null);
  const [syncStatus, setSyncStatus] = useState<SyncStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [saveMsg, setSaveMsg] = useState("");

  // Local form state
  const [syncTime, setSyncTime] = useState("21:00");
  const [retryTime, setRetryTime] = useState("23:00");
  const [syncEnabled, setSyncEnabled] = useState(true);
  const [retryEnabled, setRetryEnabled] = useState(true);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [cfg, status] = await Promise.all([
        syncApi.getConfig(),
        syncApi.status(),
      ]);
      setConfig(cfg);
      setSyncStatus(status);
      setSyncTime(cfg.sync_time);
      setRetryTime(cfg.retry_time);
      setSyncEnabled(cfg.sync_enabled);
      setRetryEnabled(cfg.retry_enabled);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载配置失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!authLoading && user) {
      fetchData();
    }
  }, [authLoading, user, fetchData]);

  const handleSave = async () => {
    setSaving(true);
    setSaveMsg("");
    setError("");
    try {
      await syncApi.updateConfig({
        sync_enabled: syncEnabled,
        sync_time: syncTime,
        retry_enabled: retryEnabled,
        retry_time: retryTime,
      });
      setSaveMsg("保存成功");
      setTimeout(() => setSaveMsg(""), 3000);
      fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存失败");
    } finally {
      setSaving(false);
    }
  };

  const handleManualSync = async () => {
    setError("");
    try {
      await syncApi.trigger();
      // Poll for completion
      const poll = setInterval(async () => {
        try {
          const status = await syncApi.status();
          setSyncStatus(status);
          if (status.status === "completed" || status.status === "failed") {
            clearInterval(poll);
          }
        } catch {
          clearInterval(poll);
        }
      }, 2000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "触发同步失败");
    }
  };

  if (authLoading) return null;
  if (!user) return null;

  const statusBadge = () => {
    if (!syncStatus) return null;
    switch (syncStatus.status) {
      case "running":
        return (
          <Badge variant="outline" className="gap-1 text-blue-400 border-blue-400/30 bg-blue-400/10">
            <RefreshCw className="h-3 w-3 animate-spin" />
            同步中
          </Badge>
        );
      case "completed":
        return (
          <Badge variant="outline" className="gap-1 text-green-400 border-green-400/30 bg-green-400/10">
            <CheckCircle2 className="h-3 w-3" />
            已完成
          </Badge>
        );
      case "failed":
        return (
          <Badge variant="outline" className="gap-1 text-red-400 border-red-400/30 bg-red-400/10">
            <XCircle className="h-3 w-3" />
            失败
          </Badge>
        );
      default:
        return (
          <Badge variant="outline" className="gap-1 text-muted-foreground">
            空闲
          </Badge>
        );
    }
  };

  return (
    <div className="mx-auto max-w-2xl space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Settings className="h-6 w-6 text-primary" />
        <div>
          <h1 className="text-2xl font-bold text-foreground">数据同步设置</h1>
          <p className="text-sm text-muted-foreground">
            配置 Tushare 数据自动同步时间和手动触发同步
          </p>
        </div>
      </div>

      {/* Sync Status */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Database className="h-4 w-4" />
            同步状态
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">当前状态</span>
            {statusBadge()}
          </div>
          {syncStatus?.started_at && (
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">开始时间</span>
              <span className="text-sm text-foreground">
                {new Date(syncStatus.started_at).toLocaleString("zh-CN")}
              </span>
            </div>
          )}
          {syncStatus?.finished_at && (
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">完成时间</span>
              <span className="text-sm text-foreground">
                {new Date(syncStatus.finished_at).toLocaleString("zh-CN")}
              </span>
            </div>
          )}
          {syncStatus?.error && (
            <div className="rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-400">
              {syncStatus.error}
            </div>
          )}
          <Button
            onClick={handleManualSync}
            disabled={syncStatus?.status === "running"}
            className="w-full"
          >
            {syncStatus?.status === "running" ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                同步中...
              </>
            ) : (
              <>
                <RefreshCw className="mr-2 h-4 w-4" />
                立即同步数据
              </>
            )}
          </Button>
        </CardContent>
      </Card>

      {/* Schedule Config */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Clock className="h-4 w-4" />
            定时同步配置
          </CardTitle>
          <CardDescription>
            设置每天自动从 Tushare 拉取数据的时间（北京时间）
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Primary sync */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium">主同步</Label>
              <button
                type="button"
                onClick={() => setSyncEnabled(!syncEnabled)}
                className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${
                  syncEnabled ? "bg-primary" : "bg-muted"
                }`}
              >
                <span
                  className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform ${
                    syncEnabled ? "translate-x-4" : "translate-x-0"
                  }`}
                />
              </button>
            </div>
            {syncEnabled && (
              <div className="flex items-center gap-2">
                <Label htmlFor="sync-time" className="text-sm text-muted-foreground shrink-0">
                  每日同步时间
                </Label>
                <Input
                  id="sync-time"
                  type="time"
                  value={syncTime}
                  onChange={(e) => setSyncTime(e.target.value)}
                  className="w-32"
                />
                <span className="text-xs text-muted-foreground">数据通常在收盘后 1-2 小时可用</span>
              </div>
            )}
          </div>

          <Separator />

          {/* Retry sync */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium">重试同步</Label>
              <button
                type="button"
                onClick={() => setRetryEnabled(!retryEnabled)}
                className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${
                  retryEnabled ? "bg-primary" : "bg-muted"
                }`}
              >
                <span
                  className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform ${
                    retryEnabled ? "translate-x-4" : "translate-x-0"
                  }`}
                />
              </button>
            </div>
            {retryEnabled && (
              <div className="flex items-center gap-2">
                <Label htmlFor="retry-time" className="text-sm text-muted-foreground shrink-0">
                  重试时间
                </Label>
                <Input
                  id="retry-time"
                  type="time"
                  value={retryTime}
                  onChange={(e) => setRetryTime(e.target.value)}
                  className="w-32"
                />
                <span className="text-xs text-muted-foreground">适用于主同步时数据未就绪的情况</span>
              </div>
            )}
          </div>

          <Separator />

          {/* Save */}
          {error && (
            <div className="rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-400">
              {error}
            </div>
          )}
          {saveMsg && (
            <div className="rounded-md bg-green-500/10 px-3 py-2 text-sm text-green-400">
              {saveMsg}
            </div>
          )}
          <Button onClick={handleSave} disabled={saving} className="w-full">
            {saving ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                保存中...
              </>
            ) : (
              <>
                <Save className="mr-2 h-4 w-4" />
                保存配置
              </>
            )}
          </Button>
        </CardContent>
      </Card>

      {/* Info card */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">说明</CardTitle>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground space-y-2">
          <p>
            <strong>主同步</strong>：每日定时从 Tushare 拉取全部 A 股的日 K 线、资金流向、
            北向资金、融资融券等数据。建议设置在收盘后 1-2 小时（如 21:00）。
          </p>
          <p>
            <strong>重试同步</strong>：如果主同步时 Tushare 数据尚未更新，重试会在更晚的时间
            再次尝试拉取，确保数据完整。
          </p>
          <p>
            <strong>手动同步</strong>：随时点击上方按钮或导航栏的「同步数据」按钮立即拉取最新数据。
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
