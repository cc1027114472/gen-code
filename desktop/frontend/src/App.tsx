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
  const [streamState, setStreamState] = useState("Manual refresh mode");
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
        setMessage(nextRuntime.runtimeMessage || "Runtime status refreshed.");
        setLastCheckedAt(formatTime(nextRuntime.updatedAt));
      }

      if (!nextRuntime.supportsSSE) {
        setStreamState("SSE unavailable, manual refresh remains active.");
        setSseEnabled(false);
      } else {
        setStreamState("SSE detected, waiting to subscribe.");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown bridge error");
      setLastCheckedAt(new Date().toLocaleTimeString("zh-CN", { hour12: false }));
      setStreamState("Refresh failed, staying on manual mode.");
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
      setStreamState("SSE connected. The dashboard will auto-refresh.");

      source.onmessage = () => {
        void refreshStatus(false);
      };

      source.onerror = () => {
        setStreamState("SSE disconnected. Manual refresh is still available.");
        setSseEnabled(false);
        source.close();
      };

      return () => {
        source.close();
        setSseEnabled(false);
      };
    } catch {
      setStreamState("SSE could not be initialized. Manual refresh remains active.");
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
      setMessage("Thread created.");
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
      setMessage(`Active thread switched to ${id}.`);
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
      setMessage("Task queued.");
      setLastCheckedAt(formatTime(nextStatus.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create task");
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
      setMessage("Task status updated.");
      setLastCheckedAt(formatTime(nextStatus.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update task status");
    } finally {
      setLoading(false);
    }
  };

  const statusTone = error ? "warning" : runtimeStatus?.runtimeReady || bridgeResult?.ok ? "good" : "muted";
  const taskSummary = useMemo(() => summarizeTaskCounts(runtimeStatus?.tasks), [runtimeStatus]);
  const capabilitySummary = useMemo(() => summarizeGroups(runtimeStatus?.skillsByGroup), [runtimeStatus]);

  const activityFeed = useMemo(
    () => [
      {
        label: "Runtime source",
        value: runtimeStatus?.runtimeSource || "unknown",
        tone: runtimeStatus?.runtimeSource === "runtime-http" ? "good" : "muted",
      },
      {
        label: "Workspace",
        value: runtimeStatus?.workspaceId ? `${runtimeStatus.workspaceId} / threads ${runtimeStatus.threadCount}` : "not ready",
        tone: runtimeStatus?.workspaceId ? "good" : "muted",
      },
      {
        label: "Active thread",
        value: runtimeStatus?.activeThreadId || "none",
        tone: runtimeStatus?.activeThreadId ? "good" : "muted",
      },
      {
        label: "Tasks",
        value: taskSummary,
        tone: runtimeStatus?.tasks?.length ? "good" : "muted",
      },
      {
        label: "Refresh mode",
        value: streamState,
        tone: sseEnabled ? "good" : "muted",
      },
      {
        label: "Last check",
        value: lastCheckedAt || "not run",
        tone: "muted",
      },
    ],
    [lastCheckedAt, runtimeStatus?.activeThreadId, runtimeStatus?.runtimeSource, runtimeStatus?.tasks, runtimeStatus?.threadCount, runtimeStatus?.workspaceId, sseEnabled, streamState, taskSummary],
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
              <h1>Runtime Control Room</h1>
            </div>
          </div>
          <div className="topbar__meta">
            <span className="chip chip--soft">Wails Shell</span>
            <span className="chip chip--soft">{runtimeStatus?.runtimeSource ?? "loading"}</span>
            <span className={`chip chip--${statusTone}`}>{error ? "Needs attention" : "Status connected"}</span>
          </div>
        </header>

        <section className="board">
          <aside className="sidebar card">
            <div className="sidebar__section">
              <p className="section-title">Workspace</p>
              <button className="nav-item nav-item--active" type="button">
                <span className="nav-item__label">Runtime overview</span>
                <span className="nav-item__meta">Live</span>
              </button>
              <button className="nav-item" type="button">
                <span className="nav-item__label">Threads and tasks</span>
              </button>
              <button className="nav-item" type="button">
                <span className="nav-item__label">Capabilities</span>
              </button>
              <button className="nav-item" type="button">
                <span className="nav-item__label">Desktop bridge</span>
              </button>
            </div>

            <div className="sidebar__section sidebar__section--muted">
              <p className="section-title">Project root</p>
              <p className="sidebar__note">{runtimeStatus?.projectRoot ?? "Waiting for runtime data..."}</p>
            </div>
          </aside>

          <section className="main-panel">
            <div className="hero card">
              <div className="hero__copy">
                <p className="hero__eyebrow">Desktop runtime status</p>
                <h2>The homepage now shows threads, tasks, events, and bridge health from one consistent runtime view.</h2>
                <p className="hero__lead">
                  The desktop shell prefers live runtime data first. If the external app-server is unavailable, it falls back to a stable
                  desktop-local state so we can still demonstrate thread creation, task queuing, and lifecycle progress without blocking on
                  server availability.
                </p>
              </div>

              <div className="hero__actions">
                <button className="primary-action" onClick={() => void refreshStatus(true)} disabled={loading}>
                  {loading ? "Refreshing..." : "Refresh now"}
                </button>
                <button className="secondary-action" onClick={handleCreateThread} disabled={loading}>
                  New thread
                </button>
                <button className="secondary-action" onClick={handleCreateTask} disabled={loading || !runtimeStatus?.activeThreadId}>
                  New task
                </button>
                <div className="hero__hint">
                  <span className="status-dot" />
                  {streamState}
                </div>
              </div>
            </div>

            <div className="stats-grid">
              <article className="card stat-card">
                <p className="section-title">Desktop Shell</p>
                <strong>{runtimeStatus?.desktopReady ? "Ready" : "Loading"}</strong>
                <span>{appInfo}</span>
              </article>

              <article className="card stat-card">
                <p className="section-title">Runtime</p>
                <strong>{runtimeStatus ? `${runtimeStatus.appName}:${runtimeStatus.port}` : "Loading"}</strong>
                <span>{runtimeStatus?.runtimeMessage ?? "Waiting for bridge response..."}</span>
              </article>

              <article className="card stat-card">
                <p className="section-title">Tasks</p>
                <strong>{taskSummary}</strong>
                <span>{runtimeStatus?.activeThreadId ? `Active thread ${runtimeStatus.activeThreadId}` : "Create a thread to begin."}</span>
              </article>
            </div>

            <div className="thread-strip card">
              <div className="thread-strip__header">
                <div>
                  <p className="section-title">Threads</p>
                  <h3>Thread list and activation state</h3>
                </div>
                <span className="mini-chip">{`Count ${runtimeStatus?.threadCount ?? 0}`}</span>
              </div>

              <div className="thread-list">
                {(runtimeStatus?.threads ?? []).length === 0 ? (
                  <div className="thread-empty">No threads yet. Create one to start a separate working session.</div>
                ) : (
                  runtimeStatus?.threads.map((thread) => (
                    <div className="thread-item" key={thread.id}>
                      <div>
                        <p>{thread.name}</p>
                        <strong>{`${thread.id} / ${thread.permissionMode || "default"}`}</strong>
                        <span>{thread.activeModel || "No active model selected"}</span>
                      </div>
                      <div className="thread-item__actions">
                        {thread.isActive ? <span className="chip chip--good">Active</span> : null}
                        {!thread.isActive ? (
                          <button className="thread-action" onClick={() => handleActivateThread(thread.id)} disabled={loading}>
                            Activate
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
                    <h3>Tasks for the active thread</h3>
                  </div>
                  <span className="mini-chip">{`Count ${runtimeStatus?.tasks.length ?? 0}`}</span>
                </div>
                <div className="task-list">
                  {(runtimeStatus?.tasks ?? []).length === 0 ? (
                    <div className="thread-empty">No tasks in the active thread yet.</div>
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
                            Advance status
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
                    <h3>Thread activity log</h3>
                  </div>
                  <span className="mini-chip">{`Count ${runtimeStatus?.events.length ?? 0}`}</span>
                </div>
                <div className="event-list">
                  {(runtimeStatus?.events ?? []).length === 0 ? (
                    <div className="thread-empty">No events recorded yet.</div>
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
                <h3>Threads, tasks, bridge status, and runtime discovery now share one homepage instead of separate placeholders.</h3>
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
                <h3>Events and health</h3>
              </div>
              <span className="chip chip--soft">{runtimeStatus?.runtimeSource ?? "Local"}</span>
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
                      `Bridge: ${bridgeResult?.message ?? "not checked separately"}`,
                      `Runtime: ${runtimeStatus?.runtimeMessage ?? "none"}`,
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
