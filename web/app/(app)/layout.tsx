"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { syncApi } from "@/lib/api-client";
import { Activity, Trophy, Target, BarChart3, Filter, LogOut, Database, Settings, CheckCircle2, XCircle, Briefcase } from "lucide-react";
import { cn } from "@/lib/utils";

const NAV_ITEMS = [
  { href: "/ranking", label: "综合排名", icon: Trophy },
  { href: "/analyze", label: "个股分析", icon: BarChart3 },
  { href: "/alpha300", label: "Alpha300", icon: Target },
  { href: "/screener", label: "筛选器", icon: Filter },
  { href: "/portfolio", label: "持仓", icon: Briefcase },
];

type SyncState = "idle" | "running" | "success" | "error";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const pathname = usePathname();
  const router = useRouter();
  const [syncState, setSyncState] = useState<SyncState>("idle");
  const [syncError, setSyncError] = useState<string>("");
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Cleanup polling on unmount
  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  const pollSyncStatus = useCallback(() => {
    pollRef.current = setInterval(async () => {
      try {
        const res = await syncApi.status();
        if (res.status === "completed") {
          setSyncState("success");
          if (pollRef.current) clearInterval(pollRef.current);
          // Auto-reset after 3s
          setTimeout(() => setSyncState("idle"), 3000);
          // Refresh current page data
          router.refresh();
        } else if (res.status === "failed") {
          setSyncState("error");
          setSyncError(res.error || "同步失败");
          if (pollRef.current) clearInterval(pollRef.current);
          setTimeout(() => setSyncState("idle"), 5000);
        }
      } catch {
        // polling error — ignore
      }
    }, 2000);
  }, [router]);

  const handleSync = useCallback(async () => {
    if (syncState === "running") return;
    setSyncState("running");
    setSyncError("");
    try {
      const res = await syncApi.trigger();
      if (res.status === "running") {
        pollSyncStatus();
      }
    } catch (err) {
      setSyncState("error");
      setSyncError(err instanceof Error ? err.message : "触发同步失败");
      setTimeout(() => setSyncState("idle"), 3000);
    }
  }, [syncState, pollSyncStatus]);

  if (loading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  if (!user) return null;

  const syncIcon = () => {
    switch (syncState) {
      case "running":
        return <Database className={cn("h-4 w-4 animate-spin")} />;
      case "success":
        return <CheckCircle2 className="h-4 w-4 text-green-500" />;
      case "error":
        return <XCircle className="h-4 w-4 text-red-500" />;
      default:
        return <Database className="h-4 w-4" />;
    }
  };

  return (
    <div className="flex flex-1 flex-col min-h-screen">
      {/* Top nav */}
      <header className="sticky top-0 z-50 border-b border-border bg-background/80 backdrop-blur-md">
        <div className="mx-auto flex h-14 max-w-7xl items-center gap-6 px-4">
          {/* Logo */}
          <Link href="/ranking" className="flex items-center gap-2 shrink-0">
            <Activity className="h-6 w-6 text-primary" />
            <span className="text-lg font-bold text-foreground">
              AlphaPulse
            </span>
          </Link>

          {/* Nav links */}
          <nav className="flex items-center gap-1">
            {NAV_ITEMS.map((item) => {
              const active = pathname.startsWith(item.href);
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={cn(
                    "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
                    active
                      ? "bg-primary/10 text-primary"
                      : "text-muted-foreground hover:text-foreground hover:bg-accent"
                  )}
                >
                  <item.icon className="h-4 w-4" />
                  {item.label}
                </Link>
              );
            })}
          </nav>

          {/* Spacer */}
          <div className="flex-1" />

          {/* Sync + Settings + User */}
          <div className="flex items-center gap-2">
            {/* Data sync button */}
            <button
              onClick={handleSync}
              disabled={syncState === "running"}
              title={
                syncState === "running"
                  ? "正在同步数据..."
                  : syncState === "success"
                    ? "同步完成"
                    : syncState === "error"
                      ? syncError
                      : "手动同步数据"
              }
              className={cn(
                "flex items-center gap-1.5 rounded-md px-2.5 py-1 text-sm transition-colors",
                syncState === "running"
                  ? "text-primary bg-primary/10 cursor-wait"
                  : syncState === "success"
                    ? "text-green-500 bg-green-500/10"
                    : syncState === "error"
                      ? "text-red-500 bg-red-500/10"
                      : "text-muted-foreground hover:text-foreground hover:bg-accent"
              )}
            >
              {syncIcon()}
              {syncState === "running" ? "同步中" : "同步数据"}
            </button>

            {/* Settings link */}
            <Link
              href="/settings"
              className={cn(
                "flex items-center rounded-md p-1.5 text-sm transition-colors",
                pathname === "/settings"
                  ? "text-primary bg-primary/10"
                  : "text-muted-foreground hover:text-foreground hover:bg-accent"
              )}
              title="设置"
            >
              <Settings className="h-4 w-4" />
            </Link>

            {/* User info */}
            <span className="text-sm text-muted-foreground">{user.username}</span>
            <button
              onClick={logout}
              className="flex items-center gap-1 rounded-md px-2 py-1 text-sm text-muted-foreground transition-colors hover:text-foreground hover:bg-accent"
            >
              <LogOut className="h-4 w-4" />
              退出
            </button>
          </div>
        </div>
      </header>

      {/* Page content */}
      <main className="flex-1">{children}</main>
    </div>
  );
}
