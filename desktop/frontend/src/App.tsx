import { useEffect, useMemo, useState } from "react";
import { ActivateThread, CheckBridge, CreateTask, CreateThread, GetAppInfo, GetRuntimeStatus } from "../wailsjs/go/main/App";

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
};

type EventSummary = {
  id: string;
  threadId: string;
  type: string;
  message: string;
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
  skillsByGroup: Record<string, string[]>;
  toolsByGroup: Record<string, string[]>;
  mcpByGroup: Record<string, string[]>;
  missingPaths: string[];
  updatedAt: string;
};

type BridgeCheckResult = {
  ok: boolean;
  message: string;
  checkedAt: string;
  runtimeHint: string;
};

export default function App() {
  const [message, setMessage] = useState("Waiting for Go bridge...");
  const [error, setError] = useState("");
  const [lastCheckedAt, setLastCheckedAt] = useState("");
  const [appInfo, setAppInfo] = useState("Loading desktop shell...");
  const [runtimeStatus, setRuntimeStatus] = useState<RuntimeStatus | null>(null);
  const [bridgeResult, setBridgeResult] = useState<BridgeCheckResult | null>(null);
  const [loading, setLoading] = useState(true);

  const loadDashboard = async (runBridgeCheck: boolean) => {
    setLoading(true);
    setError("");

    try {
      const [info, runtime, bridge] = await Promise.all([
        GetAppInfo(),
        GetRuntimeStatus(),
        runBridgeCheck ? CheckBridge() : Promise.resolve(null),
      ]);

      setAppInfo(info);
      setRuntimeStatus(runtime as RuntimeStatus);

      if (bridge) {
        const nextBridge = bridge as BridgeCheckResult;
        setBridgeResult(nextBridge);
        setMessage(nextBridge.message);
        setLastCheckedAt(formatTime(nextBridge.checkedAt));
      } else {
        setBridgeResult(null);
        setMessage("Runtime status loaded");
        setLastCheckedAt(formatTime((runtime as RuntimeStatus).updatedAt));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown bridge error");
      setLastCheckedAt(new Date().toLocaleTimeString("zh-CN", { hour12: false }));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadDashboard(true);
  }, []);

  const handleCheck = async () => {
    await loadDashboard(true);
  };

  const handleCreateThread = async () => {
    setLoading(true);
    setError("");
    try {
      const nextStatus = (await CreateThread("")) as RuntimeStatus;
      setRuntimeStatus(nextStatus);
      setMessage("Thread created");
      setLastCheckedAt(formatTime(nextStatus.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create thread");
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
      setMessage(`Active thread switched to ${id}`);
      setLastCheckedAt(formatTime(nextStatus.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to activate thread");
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
      setMessage("Task queued");
      setLastCheckedAt(formatTime(nextStatus.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create task");
    } finally {
      setLoading(false);
    }
  };

  const statusTone = error ? "warning" : bridgeResult?.ok || runtimeStatus?.runtimeReady ? "good" : "muted";
  const skillGroups = useMemo(() => summarizeGroups(runtimeStatus?.skillsByGroup), [runtimeStatus]);
  const toolGroups = useMemo(() => summarizeGroups(runtimeStatus?.toolsByGroup), [runtimeStatus]);
  const mcpGroups = useMemo(() => summarizeGroups(runtimeStatus?.mcpByGroup), [runtimeStatus]);

  const activityFeed = useMemo(
    () => [
      {
        label: "Workspace",
        value: runtimeStatus?.workspaceId ? `${runtimeStatus.workspaceId} / threads ${runtimeStatus.threadCount}` : "Workspace unavailable",
        tone: runtimeStatus?.workspaceId ? "good" : "muted",
      },
      {
        label: "Active thread",
        value: runtimeStatus?.activeThreadId || "No active thread",
        tone: runtimeStatus?.activeThreadId ? "good" : "muted",
      },
      {
        label: "Tasks",
        value: runtimeStatus?.tasks?.length ? `${runtimeStatus.tasks.length} queued` : "No tasks yet",
        tone: runtimeStatus?.tasks?.length ? "good" : "muted",
      },
      {
        label: "Go bridge",
        value: error ? "Invocation failed" : bridgeResult?.message ?? runtimeStatus?.runtimeMessage ?? "Runtime status loaded",
        tone: error ? "warning" : "good",
      },
      {
        label: "Last check",
        value: lastCheckedAt || "Not run yet",
        tone: "muted",
      },
      {
        label: "Skills",
        value: skillGroups || "No skills discovered",
        tone: skillGroups ? "good" : "muted",
      },
    ],
    [bridgeResult?.message, error, lastCheckedAt, runtimeStatus?.activeThreadId, runtimeStatus?.runtimeMessage, runtimeStatus?.tasks?.length, runtimeStatus?.threadCount, runtimeStatus?.workspaceId, skillGroups],
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
              <p className="topbar__eyebrow">Gen Code / Local Desktop Console</p>
              <h1>启动总控台</h1>
            </div>
          </div>
          <div className="topbar__meta">
            <span className="chip chip--soft">Wails Shell</span>
            <span className="chip chip--soft">{runtimeStatus?.workspaceId ?? "Workspace"}</span>
            <span className={`chip chip--${statusTone}`}>{error ? "Bridge Warning" : "Bridge Live"}</span>
          </div>
        </header>

        <section className="board">
          <aside className="sidebar card">
            <div className="sidebar__section">
              <p className="section-title">Workspace</p>
              <button className="nav-item nav-item--active" type="button">
                <span className="nav-item__label">总控首页</span>
                <span className="nav-item__meta">Live</span>
              </button>
              <button className="nav-item" type="button">
                <span className="nav-item__label">线程与任务</span>
              </button>
              <button className="nav-item" type="button">
                <span className="nav-item__label">技能与工具</span>
              </button>
              <button className="nav-item" type="button">
                <span className="nav-item__label">桌面集成</span>
              </button>
            </div>

            <div className="sidebar__section sidebar__section--muted">
              <p className="section-title">Project root</p>
              <p className="sidebar__note">{runtimeStatus?.projectRoot ?? "Waiting for runtime data."}</p>
            </div>
          </aside>

          <section className="main-panel">
            <div className="hero card">
              <div className="hero__copy">
                <p className="hero__eyebrow">Workspace shell status</p>
                <h2>workspace 之上已经能创建线程、在线程内排队任务，并把关键事件记录成活动日志。</h2>
                <p className="hero__lead">
                  这一步先提供最小任务编排入口和日志面板能力，不引入复杂执行器，让 desktop、app-server、runtime 对线程任务状态保持同一视图。
                </p>
              </div>

              <div className="hero__actions">
                <button className="primary-action" onClick={handleCheck} disabled={loading}>
                  {loading ? "刷新中..." : "检查 Go 桥接"}
                </button>
                <button className="secondary-action" onClick={handleCreateThread} disabled={loading}>
                  新建线程
                </button>
                <button className="secondary-action" onClick={handleCreateTask} disabled={loading || !runtimeStatus?.activeThreadId}>
                  新建任务
                </button>
                <div className="hero__hint">
                  <span className="status-dot" />
                  当前版本的任务是线程内元数据与事件编排入口，还不包含真实执行器。
                </div>
              </div>
            </div>

            <div className="stats-grid">
              <article className="card stat-card">
                <p className="section-title">Workspace</p>
                <strong>{runtimeStatus?.workspaceId ?? "Loading"}</strong>
                <span>{runtimeStatus?.projectRoot ?? appInfo}</span>
              </article>

              <article className="card stat-card">
                <p className="section-title">Active Thread</p>
                <strong>{runtimeStatus?.activeThreadId ?? "None"}</strong>
                <span>{runtimeStatus?.threadCount ? `Total threads ${runtimeStatus.threadCount}` : "Create a thread to begin."}</span>
              </article>

              <article className="card stat-card">
                <p className="section-title">Runtime Status</p>
                <strong>{runtimeStatus ? `${runtimeStatus.appName}:${runtimeStatus.port}` : "Loading"}</strong>
                <span>
                  {runtimeStatus
                    ? `Env=${runtimeStatus.appEnv}, state=${runtimeStatus.runtimeState}, tasks=${runtimeStatus.tasks.length}`
                    : "Waiting for runtime data."}
                </span>
              </article>
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
                  <div className="thread-empty">还没有 thread。先创建一个独立工作会话。</div>
                ) : (
                  runtimeStatus?.threads.map((thread) => (
                    <div className="thread-item" key={thread.id}>
                      <div>
                        <p>{thread.name}</p>
                        <strong>{`${thread.id} / ${thread.permissionMode}`}</strong>
                        <span>{thread.activeModel || "No model selected"}</span>
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
                    <h3>当前线程任务</h3>
                  </div>
                  <span className="mini-chip">{`Count ${runtimeStatus?.tasks.length ?? 0}`}</span>
                </div>
                <div className="task-list">
                  {(runtimeStatus?.tasks ?? []).length === 0 ? (
                    <div className="thread-empty">当前线程还没有任务。</div>
                  ) : (
                    runtimeStatus?.tasks.map((task) => (
                      <div className="task-item" key={task.id}>
                        <p>{task.title}</p>
                        <strong>{`${task.id} / ${task.status}`}</strong>
                      </div>
                    ))
                  )}
                </div>
              </div>

              <div className="task-panel card">
                <div className="thread-strip__header">
                  <div>
                    <p className="section-title">Events</p>
                    <h3>当前线程事件日志</h3>
                  </div>
                  <span className="mini-chip">{`Count ${runtimeStatus?.events.length ?? 0}`}</span>
                </div>
                <div className="event-list">
                  {(runtimeStatus?.events ?? []).length === 0 ? (
                    <div className="thread-empty">当前线程还没有事件。</div>
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
                <p className="section-title">Capability view</p>
                <h3>workspace / thread / tasks / events 已接入首页，同时保留 runtime、skills、tools、MCP 总览。</h3>
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
                <h3>事件与状态</h3>
              </div>
              <span className="chip chip--soft">Local</span>
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
                      `Bridge: ${bridgeResult?.message ?? runtimeStatus?.runtimeMessage ?? "Not checked"}`,
                      `Workspace: ${runtimeStatus?.workspaceId ?? "none"}`,
                      `Active thread: ${runtimeStatus?.activeThreadId ?? "none"}`,
                      `Tasks: ${runtimeStatus?.tasks.length ?? 0}`,
                      `Latest event: ${runtimeStatus?.events[0]?.message ?? "none"}`,
                      `Skills: ${skillGroups || "none"}`,
                      `Tools: ${summarizeGroups(runtimeStatus?.toolsByGroup) || "none"}`,
                      `MCP: ${summarizeGroups(runtimeStatus?.mcpByGroup) || "none"}`,
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
    .map(([group, items]) => {
      const summary = items.slice(0, 2).join(", ");
      const suffix = items.length > 2 ? ` +${items.length - 2}` : "";
      return `${group}:${items.length}${summary ? ` [${summary}${suffix}]` : ""}`;
    })
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
