import { useEffect, useMemo, useState } from "react";
import {
  ActivateThread,
  AdvanceTask,
  CheckBridge,
  CreateTask,
  CreateThread,
  GetAppInfo,
  GetRuntimeStatus,
} from "../wailsjs/go/main/App";

type ThreadSummary = {
  id: string;
  name: string;
  status: string;
  activeModel: string;
  permissionMode: string;
  isActive: boolean;
};

type TaskSummary = {
  id: string;
  threadId: string;
  title: string;
  status: string;
  updatedAt: string;
};

type EventSummary = {
  id: string;
  threadId: string;
  type: string;
  message: string;
  createdAt: string;
};

type RuntimeStatus = {
  appName: string;
  appEnv: string;
  port: number;
  debug: boolean;
  shutdownTimeout: string;
  trustedProxies: string[];
  logLevel: string;
  httpAccessLog: boolean;
  workspaceRoot: string;
  workspaceId: string;
  projectRoot: string;
  threadCount: number;
  activeThreadId: string;
  threads: ThreadSummary[];
  tasks: TaskSummary[];
  events: EventSummary[];
  desktopReady: boolean;
  runtimeState: string;
  runtimeReady: boolean;
  runtimeMessage: string;
  runtimeSource: string;
  supportsSSE: boolean;
  sseEndpoint: string;
  lastSyncAt: string;
  skillsByGroup: Record<string, string[]>;
  toolsByGroup: Record<string, string[]>;
  mcpByGroup: Record<string, string[]>;
  missingPaths: string[];
  stateStore: string;
  statePath: string;
  usesProjectLocalStore: boolean;
  recoverySummary: string;
  updatedAt: string;
};

type BridgeCheckResult = {
  ok: boolean;
  message: string;
  checkedAt: string;
  runtimeHint: string;
};

export default function App() {
  const [message, setMessage] = useState("等待 Go bridge 响应...");
  const [error, setError] = useState("");
  const [lastCheckedAt, setLastCheckedAt] = useState("");
  const [appInfo, setAppInfo] = useState("正在加载桌面壳...");
  const [runtimeStatus, setRuntimeStatus] = useState<RuntimeStatus | null>(null);
  const [bridgeResult, setBridgeResult] = useState<BridgeCheckResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [streamState, setStreamState] = useState("手动刷新模式");
  const [sseEnabled, setSseEnabled] = useState(false);

  const refreshStatus = async (runBridgeCheck: boolean) => {
    setLoading(true);
    setError("");

    try {
      const [info, runtime, bridge] = await Promise.all([
        GetAppInfo(),
        GetRuntimeStatus(),
        runBridgeCheck ? CheckBridge() : Promise.resolve(null),
      ]);

      const nextRuntime = runtime as RuntimeStatus;
      setAppInfo(info);
      setRuntimeStatus(nextRuntime);

      if (bridge) {
        const nextBridge = bridge as BridgeCheckResult;
        setBridgeResult(nextBridge);
        setMessage(nextBridge.message);
        setLastCheckedAt(formatTime(nextBridge.checkedAt));
      } else {
        setMessage(nextRuntime.runtimeMessage || "状态已刷新。");
        setLastCheckedAt(formatTime(nextRuntime.updatedAt));
      }

      if (!nextRuntime.supportsSSE || !nextRuntime.sseEndpoint) {
        setStreamState("未接入 SSE，保留手动刷新回退。");
        setSseEnabled(false);
      } else {
        setStreamState("检测到 SSE，准备订阅任务更新。");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "未知 bridge 错误");
      setLastCheckedAt(new Date().toLocaleTimeString("zh-CN", { hour12: false }));
      setStreamState("刷新失败，已退回手动刷新。");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void refreshStatus(true);
  }, []);

  useEffect(() => {
    if (!runtimeStatus?.supportsSSE || !runtimeStatus.sseEndpoint || sseEnabled) {
      return;
    }

    try {
      const source = new EventSource(runtimeStatus.sseEndpoint);
      setSseEnabled(true);
      setStreamState("SSE 已连接，首页会自动刷新 task 状态。");

      source.onmessage = () => {
        void refreshStatus(false);
      };

      source.onerror = () => {
        setStreamState("SSE 已断开，仍可手动刷新。");
        setSseEnabled(false);
        source.close();
      };

      return () => {
        source.close();
        setSseEnabled(false);
      };
    } catch {
      setStreamState("SSE 初始化失败，保留手动刷新。");
      setSseEnabled(false);
      return;
    }
  }, [runtimeStatus?.supportsSSE, runtimeStatus?.sseEndpoint, sseEnabled]);

  const handleCreateThread = async () => {
    setLoading(true);
    setError("");
    try {
      const nextStatus = (await CreateThread("")) as RuntimeStatus;
      setRuntimeStatus(nextStatus);
      setMessage("已创建 thread。");
      setLastCheckedAt(formatTime(nextStatus.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建 thread 失败");
    } finally {
      setLoading(false);
    }
  };

  const handleActivateThread = async (id: string) => {
    setLoading(true);
    setError("");
    try {
      const nextStatus = (await ActivateThread(id)) as RuntimeStatus;
      setRuntimeStatus(nextStatus);
      setMessage(`已切换到 thread ${id}。`);
      setLastCheckedAt(formatTime(nextStatus.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "切换 thread 失败");
    } finally {
      setLoading(false);
    }
  };

  const handleCreateTask = async () => {
    if (!runtimeStatus?.activeThreadId) {
      return;
    }

    setLoading(true);
    setError("");
    try {
      const nextStatus = (await CreateTask(runtimeStatus.activeThreadId, "")) as RuntimeStatus;
      setRuntimeStatus(nextStatus);
      setMessage("已创建 task。");
      setLastCheckedAt(formatTime(nextStatus.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建 task 失败");
    } finally {
      setLoading(false);
    }
  };

  const handleAdvanceTask = async (taskID: string) => {
    setLoading(true);
    setError("");
    try {
      const nextStatus = (await AdvanceTask(taskID)) as RuntimeStatus;
      setRuntimeStatus(nextStatus);
      setMessage("task 状态已推进。");
      setLastCheckedAt(formatTime(nextStatus.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "更新 task 状态失败");
    } finally {
      setLoading(false);
    }
  };

  const statusTone = error ? "warning" : runtimeStatus?.runtimeReady || bridgeResult?.ok ? "good" : "muted";
  const taskSummary = useMemo(() => summarizeTaskCounts(runtimeStatus?.tasks), [runtimeStatus]);
  const capabilitySummary = useMemo(() => summarizeGroups(runtimeStatus?.skillsByGroup), [runtimeStatus]);
  const runtimeSourceLabel = useMemo(() => {
    if (!runtimeStatus) {
      return "loading";
    }
    if (runtimeStatus.runtimeSource === "runtime-http") {
      return "app-server API";
    }
    if (runtimeStatus.stateStore === "sqlite") {
      return "desktop SQLite fallback";
    }
    return runtimeStatus.runtimeSource;
  }, [runtimeStatus]);

  const activityFeed = useMemo(
    () => [
      {
        label: "运行源",
        value: runtimeSourceLabel,
        tone: runtimeStatus?.runtimeSource === "runtime-http" ? "good" : "muted",
      },
      {
        label: "状态存储",
        value: runtimeStatus?.stateStore ? `${runtimeStatus.stateStore} / ${runtimeStatus.usesProjectLocalStore ? "project-local" : "shared"}` : "unknown",
        tone: runtimeStatus?.stateStore ? "good" : "muted",
      },
      {
        label: "状态文件",
        value: runtimeStatus?.statePath || "未提供",
        tone: runtimeStatus?.statePath ? "good" : "muted",
      },
      {
        label: "当前 thread",
        value: runtimeStatus?.activeThreadId || "none",
        tone: runtimeStatus?.activeThreadId ? "good" : "muted",
      },
      {
        label: "任务概览",
        value: taskSummary,
        tone: runtimeStatus?.tasks?.length ? "good" : "muted",
      },
      {
        label: "刷新模式",
        value: streamState,
        tone: sseEnabled ? "good" : "muted",
      },
      {
        label: "上次检查",
        value: lastCheckedAt || "尚未执行",
        tone: "muted",
      },
    ],
    [lastCheckedAt, runtimeSourceLabel, runtimeStatus?.activeThreadId, runtimeStatus?.statePath, runtimeStatus?.stateStore, runtimeStatus?.tasks, runtimeStatus?.usesProjectLocalStore, sseEnabled, streamState, taskSummary],
  );

  return (
    <main className="shell">
      <div className="shell__ambient shell__ambient--left" />
      <div className="shell__ambient shell__ambient--right" />

      <section className="workspace">
        <header className="topbar">
          <div className="topbar__title">
            <div className="traffic-lights" aria-hidden="true">
              <span />
              <span />
              <span />
            </div>
            <div>
              <p className="topbar__eyebrow">Gen Code / Desktop Console</p>
              <h1>桌面运行态总览</h1>
            </div>
          </div>
          <div className="topbar__meta">
            <span className="chip chip--soft">Wails Shell</span>
            <span className="chip chip--soft">{runtimeSourceLabel}</span>
            <span className={`chip chip--${statusTone}`}>{error ? "需要处理" : "状态已连接"}</span>
          </div>
        </header>

        <section className="board">
          <aside className="sidebar card">
            <div className="sidebar__section">
              <p className="section-title">导航</p>
              <button className="nav-item nav-item--active" type="button">
                <span className="nav-item__label">运行态总览</span>
                <span className="nav-item__meta">Live</span>
              </button>
              <button className="nav-item" type="button">
                <span className="nav-item__label">Threads 与 Tasks</span>
              </button>
              <button className="nav-item" type="button">
                <span className="nav-item__label">能力清单</span>
              </button>
              <button className="nav-item" type="button">
                <span className="nav-item__label">Bridge 健康</span>
              </button>
            </div>

            <div className="sidebar__section sidebar__section--muted">
              <p className="section-title">项目状态源</p>
              <p className="sidebar__note">{runtimeStatus?.statePath ?? "等待状态数据..."}</p>
              <p className="sidebar__note sidebar__note--strong">
                {runtimeStatus?.usesProjectLocalStore ? "当前展示的是 project-local state store，可用于重启恢复。" : "当前未标记 project-local store。"}
              </p>
            </div>
          </aside>

          <section className="main-panel">
            <div className="hero card">
              <div className="hero__copy">
                <p className="hero__eyebrow">Desktop runtime status</p>
                <h2>首页直接消费真实运行态、bridge 检查和可恢复的本地状态源。</h2>
                <p className="hero__lead">
                  桌面端仍然优先走 app-server API。外部服务不可用时，会切到项目内 SQLite 状态源，继续展示 thread、task、event 和恢复摘要，
                  这样重启之后也能看见上一次的工作链路，而不是退回纯内存占位。
                </p>
              </div>

              <div className="hero__actions">
                <button className="primary-action" onClick={() => void refreshStatus(true)} disabled={loading}>
                  {loading ? "刷新中..." : "立即刷新"}
                </button>
                <button className="secondary-action" onClick={handleCreateThread} disabled={loading}>
                  新建 thread
                </button>
                <button className="secondary-action" onClick={handleCreateTask} disabled={loading || !runtimeStatus?.activeThreadId}>
                  新建 task
                </button>
                <div className="hero__hint">
                  <span className="status-dot" />
                  {streamState}
                </div>
              </div>
            </div>

            <div className="stats-grid">
              <article className="card stat-card">
                <p className="section-title">桌面壳</p>
                <strong>{runtimeStatus?.desktopReady ? "Ready" : "Loading"}</strong>
                <span>{appInfo}</span>
              </article>

              <article className="card stat-card">
                <p className="section-title">运行态</p>
                <strong>{runtimeStatus ? `${runtimeStatus.appName}:${runtimeStatus.port}` : "Loading"}</strong>
                <span>{runtimeStatus?.runtimeMessage ?? "等待 bridge 响应..."}</span>
              </article>

              <article className="card stat-card">
                <p className="section-title">恢复链路</p>
                <strong>{taskSummary}</strong>
                <span>{runtimeStatus?.recoverySummary || "等待恢复摘要..."}</span>
              </article>
            </div>

            <div className="store-card card">
              <div className="thread-strip__header">
                <div>
                  <p className="section-title">状态存储</p>
                  <h3>stateStore / statePath / project-local state store</h3>
                </div>
                <span className="mini-chip">{runtimeStatus?.stateStore || "unknown"}</span>
              </div>
              <div className="store-grid">
                <div className="store-item">
                  <p>stateStore</p>
                  <strong>{runtimeStatus?.stateStore || "unknown"}</strong>
                </div>
                <div className="store-item">
                  <p>statePath</p>
                  <strong>{runtimeStatus?.statePath || "未提供"}</strong>
                </div>
                <div className="store-item">
                  <p>project-local</p>
                  <strong>{runtimeStatus?.usesProjectLocalStore ? "yes" : "no"}</strong>
                </div>
              </div>
            </div>

            <div className="thread-strip card">
              <div className="thread-strip__header">
                <div>
                  <p className="section-title">Threads</p>
                  <h3>线程列表与激活状态</h3>
                </div>
                <span className="mini-chip">{`Count ${runtimeStatus?.threadCount ?? 0}`}</span>
              </div>

              <div className="thread-list">
                {(runtimeStatus?.threads ?? []).length === 0 ? (
                  <div className="thread-empty">当前还没有 thread，可以先创建一个来演示恢复链路。</div>
                ) : (
                  runtimeStatus?.threads.map((thread) => (
                    <div className="thread-item" key={thread.id}>
                      <div>
                        <p>{thread.name}</p>
                        <strong>{`${thread.id} / ${thread.permissionMode || "default"}`}</strong>
                        <span>{thread.activeModel || "未选择 active model"}</span>
                      </div>
                      <div className="thread-item__actions">
                        {thread.isActive ? <span className="chip chip--good">Active</span> : null}
                        {!thread.isActive ? (
                          <button className="thread-action" onClick={() => handleActivateThread(thread.id)} disabled={loading}>
                            激活
                          </button>
                        ) : null}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            <div className="task-grid">
              <div className="task-panel card">
                <div className="thread-strip__header">
                  <div>
                    <p className="section-title">Tasks</p>
                    <h3>当前 thread 的任务状态</h3>
                  </div>
                  <span className="mini-chip">{`Count ${runtimeStatus?.tasks.length ?? 0}`}</span>
                </div>
                <div className="task-list">
                  {(runtimeStatus?.tasks ?? []).length === 0 ? (
                    <div className="thread-empty">当前 thread 还没有 task。</div>
                  ) : (
                    runtimeStatus?.tasks.map((task) => (
                      <div className="task-item" key={task.id}>
                        <div className="task-item__meta">
                          <div>
                            <p>{task.title}</p>
                            <strong>{`${task.id} / ${task.status}`}</strong>
                          </div>
                          <button
                            className="thread-action"
                            onClick={() => handleAdvanceTask(task.id)}
                            disabled={loading || task.status === "failed"}
                          >
                            推进状态
                          </button>
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>

              <div className="task-panel card">
                <div className="thread-strip__header">
                  <div>
                    <p className="section-title">Events</p>
                    <h3>恢复后的活动时间线</h3>
                  </div>
                  <span className="mini-chip">{`Count ${runtimeStatus?.events.length ?? 0}`}</span>
                </div>
                <div className="event-list">
                  {(runtimeStatus?.events ?? []).length === 0 ? (
                    <div className="thread-empty">暂无事件记录。</div>
                  ) : (
                    runtimeStatus?.events.map((event) => (
                      <div className="event-item" key={event.id}>
                        <p>{event.type}</p>
                        <strong>{event.message}</strong>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>

            <div className="action-strip card">
              <div>
                <p className="section-title">能力视图</p>
                <h3>bridge 状态、能力清单和恢复链路现在都汇总在一个首页里。</h3>
              </div>
              <div className="mini-actions">
                <span className="mini-chip">{`Skills ${countGroups(runtimeStatus?.skillsByGroup)}`}</span>
                <span className="mini-chip">{`Tools ${countGroups(runtimeStatus?.toolsByGroup)}`}</span>
                <span className="mini-chip">{`MCP ${countGroups(runtimeStatus?.mcpByGroup)}`}</span>
              </div>
            </div>
          </section>

          <aside className="activity card">
            <div className="activity__header">
              <div>
                <p className="section-title">Runtime activity</p>
                <h3>Bridge 与恢复观察</h3>
              </div>
              <span className="chip chip--soft">{runtimeSourceLabel}</span>
            </div>

            <div className="activity__list">
              {activityFeed.map((item) => (
                <div className="activity__item" key={item.label}>
                  <span className={`status-dot status-dot--${item.tone}`} />
                  <div>
                    <p>{item.label}</p>
                    <strong>{item.value}</strong>
                  </div>
                </div>
              ))}
            </div>

            <div className="activity__foot">
              <p className="section-title">Latest response</p>
              <div className="console-output">
                {error
                  ? error
                  : [
                      `Bridge: ${bridgeResult?.message ?? "未单独检查"}`,
                      `Runtime: ${runtimeStatus?.runtimeMessage ?? "none"}`,
                      `Recovery: ${runtimeStatus?.recoverySummary ?? "none"}`,
                      `Refresh: ${streamState}`,
                      `Capabilities: ${capabilitySummary || "none"}`,
                      `MCP: ${summarizeGroups(runtimeStatus?.mcpByGroup) || "none"}`,
                      `Missing: ${runtimeStatus?.missingPaths?.join(" | ") || "none"}`,
                    ].join("\n")}
              </div>
            </div>
          </aside>
        </section>
      </section>
    </main>
  );
}

function countGroups(groups?: Record<string, string[]>) {
  if (!groups) {
    return 0;
  }

  return Object.values(groups).reduce((sum, items) => sum + items.length, 0);
}

function summarizeGroups(groups?: Record<string, string[]>) {
  if (!groups) {
    return "";
  }

  return Object.entries(groups)
    .filter(([, items]) => items.length > 0)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([group, items]) => `${group}:${items.length}`)
    .join(" / ");
}

function summarizeTaskCounts(tasks?: TaskSummary[]) {
  if (!tasks || tasks.length === 0) {
    return "0 tasks";
  }

  const counts = tasks.reduce<Record<string, number>>((acc, task) => {
    acc[task.status] = (acc[task.status] || 0) + 1;
    return acc;
  }, {});

  return Object.entries(counts)
    .map(([status, count]) => `${status} ${count}`)
    .join(" / ");
}

function formatTime(value: string) {
  if (!value) {
    return "";
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }

  return parsed.toLocaleTimeString("zh-CN", { hour12: false });
}
