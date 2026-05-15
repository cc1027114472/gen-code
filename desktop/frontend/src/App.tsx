import { useEffect, useMemo, useState, type ReactNode } from "react";
import {
  ActivateThread,
  AdvanceTask,
  CheckBridge,
  CreateTask,
  CreateThread,
  GetAppInfo,
  GetRuntimeStatus,
  type BridgeCheckResult,
  type RuntimeStatus,
} from "./runtimeBridge";

type Draft = {
  title: string;
  kind: string;
  input: string;
};

type FlowItem = {
  id: string;
  tone: "neutral" | "good" | "warning";
  badge: string;
  title: string;
  body: string;
  meta: string;
  timestamp: number;
  actions?: ReactNode;
};

const defaultDraft: Draft = {
  title: "",
  kind: "workspace.read_file",
  input: "",
};

const executableKinds = ["workspace.read_file", "workspace.list_files", "workspace.search_text", "thread.message.append"];

export default function App() {
  const [runtimeStatus, setRuntimeStatus] = useState<RuntimeStatus | null>(null);
  const [bridgeResult, setBridgeResult] = useState<BridgeCheckResult | null>(null);
  const [statusMessage, setStatusMessage] = useState("正在加载桌面状态...");
  const [streamState, setStreamState] = useState("手动刷新模式");
  const [appInfo, setAppInfo] = useState("正在加载桌面壳...");
  const [lastCheckedAt, setLastCheckedAt] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [sseEnabled, setSseEnabled] = useState(false);
  const [draft, setDraft] = useState<Draft>(defaultDraft);

  const refreshStatus = async (runBridgeCheck: boolean) => {
    setLoading(true);
    setError("");
    try {
      const [info, runtime, bridge] = await Promise.all([
        GetAppInfo(),
        GetRuntimeStatus(),
        runBridgeCheck ? CheckBridge() : Promise.resolve(null),
      ]);

      setAppInfo(info);
      setRuntimeStatus(runtime);

      if (bridge) {
        setBridgeResult(bridge);
        setStatusMessage(bridge.message);
        setLastCheckedAt(formatTime(bridge.checkedAt));
      } else {
        setStatusMessage(runtime.runtimeMessage || "状态已刷新");
        setLastCheckedAt(formatTime(runtime.updatedAt));
      }

      if (!runtime.supportsSSE || !runtime.sseEndpoint) {
        setStreamState("当前未接入 SSE，使用手动刷新");
        setSseEnabled(false);
      } else {
        setStreamState("SSE 已就绪，事件会自动刷新");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "刷新失败");
      setStreamState("刷新失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void refreshStatus(true);
  }, []);

  useEffect(() => {
    if (!runtimeStatus?.supportsSSE || !runtimeStatus.sseEndpoint) {
      setSseEnabled(false);
      return;
    }

    const source = new EventSource(runtimeStatus.sseEndpoint);
    setSseEnabled(true);
    setStreamState("SSE 已连接，工作台正在实时刷新");

    const refresh = () => void refreshStatus(false);
    source.onmessage = refresh;
    source.onerror = () => {
      setSseEnabled(false);
      setStreamState("SSE 已断开，回退到手动刷新");
      source.close();
    };

    return () => {
      source.close();
      setSseEnabled(false);
    };
  }, [runtimeStatus?.supportsSSE, runtimeStatus?.sseEndpoint]);

  const activeThread = useMemo(
    () => runtimeStatus?.threads.find((thread) => thread.id === runtimeStatus.activeThreadId) ?? null,
    [runtimeStatus],
  );

  const tasks = runtimeStatus?.tasks ?? [];
  const messages = runtimeStatus?.messages ?? [];
  const toolCalls = runtimeStatus?.toolCalls ?? [];
  const artifacts = runtimeStatus?.artifacts ?? [];
  const events = runtimeStatus?.events ?? [];

  const latestTask = useMemo(() => newestBy(tasks, (item) => item.updatedAt || item.createdAt), [tasks]);
  const latestToolCall = useMemo(
    () =>
      newestBy(toolCalls, (item) => item.createdAt, (left, right) => {
        const leftWeight = toolCallStatusWeight(left.status);
        const rightWeight = toolCallStatusWeight(right.status);
        return rightWeight - leftWeight;
      }),
    [toolCalls],
  );
  const latestMessage = useMemo(() => newestBy(messages, (item) => item.createdAt), [messages]);
  const latestArtifact = useMemo(() => newestBy(artifacts, (item) => item.createdAt), [artifacts]);
  const latestEvent = useMemo(() => newestBy(events, (item) => item.createdAt), [events]);

  const flowItems = useMemo(() => {
    const items: FlowItem[] = [];

    for (const task of tasks.slice(0, 8)) {
      items.push({
        id: `task-${task.id}`,
        tone: task.status === "completed" ? "good" : task.status === "failed" ? "warning" : "neutral",
        badge: `task / ${task.status}`,
        title: `${task.title} · ${task.kind || "prompt"}`,
        body: task.resultSummary || task.input || "等待执行内容",
        meta: formatTime(task.updatedAt || task.createdAt),
        timestamp: toTimestamp(task.updatedAt || task.createdAt),
        actions: (
          <button className="thread-action" onClick={() => void handleRunTask(task.id)} disabled={loading}>
            Run Task
          </button>
        ),
      });
    }

    for (const toolCall of toolCalls.slice(0, 8)) {
      items.push({
        id: `tool-${toolCall.id}`,
        tone: toolCall.status === "completed" ? "good" : toolCall.status === "failed" ? "warning" : "neutral",
        badge: `tool call / ${toolCall.status}`,
        title: toolCall.toolId,
        body: toolCall.summary,
        meta: formatTime(toolCall.createdAt),
        timestamp: toTimestamp(toolCall.createdAt),
      });
    }

    for (const message of messages.slice(0, 8)) {
      items.push({
        id: `message-${message.id}`,
        tone: message.role === "assistant" ? "good" : "neutral",
        badge: `message / ${message.role}`,
        title: summarizeText(message.content, 48),
        body: message.content,
        meta: formatTime(message.createdAt),
        timestamp: toTimestamp(message.createdAt),
      });
    }

    for (const event of events.slice(0, 10)) {
      items.push({
        id: `event-${event.id}`,
        tone: event.type.includes("failed") ? "warning" : event.type.includes("completed") ? "good" : "neutral",
        badge: `event / ${event.type}`,
        title: event.type,
        body: event.message,
        meta: formatTime(event.createdAt),
        timestamp: toTimestamp(event.createdAt),
      });
    }

    return items.sort((left, right) => right.timestamp - left.timestamp).slice(0, 16);
  }, [events, loading, messages, tasks, toolCalls]);

  const runtimeSourceLabel = runtimeStatus?.runtimeSource === "runtime-http" ? "远端 API" : runtimeStatus?.stateStore === "sqlite" ? "本地 fallback" : runtimeStatus?.runtimeSource || "未连接";
  const statusTone = error ? "warning" : runtimeStatus?.runtimeReady || bridgeResult?.ok ? "good" : "muted";
  const projectName = runtimeStatus?.projectRoot?.replace(/\\/g, "/").split("/").filter(Boolean).pop() || "gen-code";
  const contextSummary = `messages ${messages.length} / tool calls ${toolCalls.length} / artifacts ${artifacts.length}`;

  const handleCreateThread = async () => {
    setLoading(true);
    setError("");
    try {
      const next = (await CreateThread("")) as RuntimeStatus;
      setRuntimeStatus(next);
      setStatusMessage("已创建新 thread");
      setLastCheckedAt(formatTime(next.updatedAt));
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
      const next = (await ActivateThread(id)) as RuntimeStatus;
      setRuntimeStatus(next);
      setStatusMessage(`已切换到 thread ${id}`);
      setLastCheckedAt(formatTime(next.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "切换 thread 失败");
    } finally {
      setLoading(false);
    }
  };

  const handleCreateTask = async () => {
    if (!runtimeStatus?.activeThreadId) return;

    setLoading(true);
    setError("");
    try {
      const next = (await CreateTask(runtimeStatus.activeThreadId, JSON.stringify(draft))) as RuntimeStatus;
      setRuntimeStatus(next);
      setStatusMessage("已创建 task");
      setLastCheckedAt(formatTime(next.updatedAt));
      setDraft((current) => ({ ...current, title: "", input: "" }));
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建 task 失败");
    } finally {
      setLoading(false);
    }
  };

  async function handleRunTask(taskID: string) {
    if (!runtimeStatus?.activeThreadId) return;

    setLoading(true);
    setError("");
    try {
      const next = (await AdvanceTask(taskID)) as RuntimeStatus;
      setRuntimeStatus(next);
      setStatusMessage("task 已执行");
      setLastCheckedAt(formatTime(next.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "执行 task 失败");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="shell">
      <div className="shell__ambient shell__ambient--left" />
      <div className="shell__ambient shell__ambient--right" />

      <section className="workspace-shell">
        <header className="workspace-topbar">
          <div className="workspace-topbar__title">
            <div className="traffic-lights" aria-hidden="true">
              <span />
              <span />
              <span />
            </div>
            <div>
              <p className="topbar__eyebrow">Gen Code / CC-style Desktop</p>
              <h1>Thread 工作台</h1>
            </div>
          </div>

          <div className="workspace-topbar__meta">
            <span className="chip chip--soft">Wails Shell</span>
            <span className="chip chip--soft">{runtimeSourceLabel}</span>
            <span className={`chip chip--${statusTone}`}>{error ? "需要处理" : "状态正常"}</span>
          </div>
        </header>

        <section className="workbench">
          <aside className="rail rail--left card">
            <section className="project-card">
              <p className="section-title">项目</p>
              <h2>{projectName}</h2>
              <p className="project-card__lead">{runtimeStatus?.projectRoot || "正在加载项目路径..."}</p>
              <div className="project-card__meta">
                <span className="mini-chip">{`workspace ${runtimeStatus?.workspaceId || "loading"}`}</span>
                <span className="mini-chip">{`threads ${runtimeStatus?.threadCount || 0}`}</span>
              </div>
            </section>

            <section className="left-panel">
              <div className="section-header">
                <div>
                  <p className="section-title">线程导航</p>
                  <h3>Workspace Threads</h3>
                </div>
                <button className="secondary-action" onClick={handleCreateThread} disabled={loading}>
                  新建 thread
                </button>
              </div>

              <div className="thread-stack">
                {(runtimeStatus?.threads ?? []).length === 0 ? (
                  <div className="thread-empty">当前还没有 thread</div>
                ) : (
                  runtimeStatus?.threads.map((thread) => (
                    <button
                      className={`thread-card ${thread.isActive ? "thread-card--active" : ""}`}
                      key={thread.id}
                      onClick={() => {
                        if (!thread.isActive) {
                          void handleActivateThread(thread.id);
                        }
                      }}
                      type="button"
                    >
                      <div className="thread-card__head">
                        <span className={`chip chip--${thread.isActive ? "good" : "muted"}`}>{thread.isActive ? "Active" : "Idle"}</span>
                        <span className="thread-card__status">{thread.status}</span>
                      </div>
                      <strong>{thread.name}</strong>
                      <span>{thread.id}</span>
                      <p>{thread.permissionMode || "ask-user"}</p>
                    </button>
                  ))
                )}
              </div>
            </section>

            <section className="left-panel left-panel--muted">
              <p className="section-title">能力摘要</p>
              <p className="sidebar-note">只读工具已接入，写能力与审批流留待下一阶段。</p>
              <p className="sidebar-note sidebar-note--strong">可执行 kind: {executableKinds.join(" / ")}</p>
            </section>
          </aside>

          <section className="center-stage">
            <section className="stage-header card">
              <div>
                <p className="section-title">当前线程</p>
                <h2>{activeThread?.name || "等待 active thread"}</h2>
                <p className="stage-header__lead">
                  {activeThread ? "这里展示当前 thread 的消息流、任务进展、工具调用和实时事件。" : "请先创建或激活一个 thread。"}
                </p>
              </div>
              <div className="stage-header__meta">
                <span className="chip chip--soft">{activeThread?.permissionMode || "ask-user"}</span>
                <span className="chip chip--soft">{streamState}</span>
              </div>
            </section>

            <section className="composer card">
              <div className="section-header">
                <div>
                  <p className="section-title">任务输入</p>
                  <h3>给 active thread 追加一个真实工具任务</h3>
                </div>
                <span className="mini-chip">{runtimeStatus?.activeThreadId ? "active thread ready" : "no active thread"}</span>
              </div>

              <div className="composer-grid">
                <label className="field">
                  <span className="field__label">标题</span>
                  <input value={draft.title} onChange={(event) => setDraft((current) => ({ ...current, title: event.target.value }))} placeholder="例如：读取 README" />
                </label>

                <label className="field field--compact">
                  <span className="field__label">kind</span>
                  <select value={draft.kind} onChange={(event) => setDraft((current) => ({ ...current, kind: event.target.value }))}>
                    {executableKinds.map((kind) => (
                      <option key={kind} value={kind}>
                        {kind}
                      </option>
                    ))}
                  </select>
                </label>
              </div>

              <label className="field">
                <span className="field__label">input</span>
                <textarea
                  value={draft.input}
                  onChange={(event) => setDraft((current) => ({ ...current, input: event.target.value }))}
                  placeholder='例如：{"path":"README.md"}'
                  rows={4}
                />
              </label>

              <div className="composer-actions">
                <button className="primary-action" onClick={handleCreateTask} disabled={loading || !runtimeStatus?.activeThreadId}>
                  创建 task
                </button>
                <span className="composer-actions__hint">{statusMessage}</span>
              </div>
            </section>

            <section className="flow-panel card">
              <div className="section-header">
                <div>
                  <p className="section-title">消息流</p>
                  <h3>任务、工具调用、消息与事件按时间线展示</h3>
                </div>
                <span className="mini-chip">{contextSummary}</span>
              </div>

              <div className="flow-list">
                {flowItems.length === 0 ? (
                  <div className="thread-empty">当前还没有可展示的消息流内容</div>
                ) : (
                  flowItems.map((item) => (
                    <article className={`flow-item flow-item--${item.tone}`} key={item.id}>
                      <div className="flow-item__header">
                        <span className="mini-chip">{item.badge}</span>
                        <span className="flow-item__meta">{item.meta}</span>
                      </div>
                      <h4>{item.title}</h4>
                      <p>{item.body}</p>
                      {item.actions ? <div className="flow-item__actions">{item.actions}</div> : null}
                    </article>
                  ))
                )}
              </div>
            </section>
          </section>

          <aside className="rail rail--right card">
            <section className="right-section">
              <div className="section-header">
                <div>
                  <p className="section-title">结果抽屉</p>
                  <h3>最新结果与运行状态</h3>
                </div>
                <span className="mini-chip">{activeThread?.id || "no-thread"}</span>
              </div>

              <div className="result-stack">
                <ResultCard label="Latest Task" title={latestTask ? `${latestTask.title} / ${latestTask.status}` : "暂无任务"} body={latestTask?.resultSummary || latestTask?.input || "暂无结果"} />
                <ResultCard label="Latest Tool Call" title={latestToolCall ? `${latestToolCall.toolId} / ${latestToolCall.status}` : "暂无 tool call"} body={latestToolCall?.summary || "暂无摘要"} />
                <ResultCard label="Latest Message" title={latestMessage ? latestMessage.role : "暂无 message"} body={latestMessage?.content || "暂无 message"} />
                <ResultCard label="Latest Artifact" title={latestArtifact ? latestArtifact.kind : "暂无 artifact"} body={latestArtifact?.path || "暂无 artifact"} />
                <ResultCard label="Latest Event" title={latestEvent ? latestEvent.type : "暂无 event"} body={latestEvent?.message || "暂无 event"} />
              </div>
            </section>

            <section className="right-section">
              <div className="section-header">
                <div>
                  <p className="section-title">运行状态</p>
                  <h3>Bridge / Runtime / SSE</h3>
                </div>
                <span className={`chip chip--${statusTone}`}>{runtimeStatus?.runtimeReady ? "ready" : "degraded"}</span>
              </div>

              <div className="info-grid">
                <InfoCard label="桌面壳" value={runtimeStatus?.desktopReady ? "Ready" : "Loading"} detail={appInfo} />
                <InfoCard label="运行态" value={runtimeStatus ? `${runtimeStatus.appName}:${runtimeStatus.port}` : "Loading"} detail={runtimeStatus?.runtimeMessage || "等待 bridge 响应"} />
                <InfoCard label="SSE" value={sseEnabled ? "Connected" : "Manual"} detail={streamState} />
                <InfoCard label="最近检查" value={lastCheckedAt || "尚未执行"} detail={bridgeResult?.message || "未单独检查"} />
              </div>
            </section>

            <section className="right-section">
              <div className="section-header">
                <div>
                  <p className="section-title">能力摘要</p>
                  <h3>Skills / Tools / MCP</h3>
                </div>
                <span className="mini-chip">{runtimeSourceLabel}</span>
              </div>

              <div className="summary-list">
                <SummaryRow label="运行来源" value={runtimeSourceLabel} />
                <SummaryRow label="状态存储" value={runtimeStatus?.stateStore ? `${runtimeStatus.stateStore} / ${runtimeStatus.usesProjectLocalStore ? "project-local" : "shared"}` : "unknown"} />
                <SummaryRow label="当前 thread" value={runtimeStatus?.activeThreadId || "none"} />
                <SummaryRow label="上下文" value={contextSummary} />
                <SummaryRow label="可执行 kind" value={executableKinds.join(" / ")} />
                <SummaryRow label="Skills" value={summarizeGroups(runtimeStatus?.skillsByGroup) || "none"} />
                <SummaryRow label="Tools" value={summarizeGroups(runtimeStatus?.toolsByGroup) || "none"} />
                <SummaryRow label="MCP" value={summarizeGroups(runtimeStatus?.mcpByGroup) || "none"} />
              </div>
            </section>

            <section className="right-section right-section--console">
              <p className="section-title">Latest Response</p>
              <div className="console-output">
                {error
                  ? error
                  : [
                      `Bridge: ${bridgeResult?.message || "未单独检查"}`,
                      `Runtime: ${runtimeStatus?.runtimeMessage || "none"}`,
                      `Recovery: ${runtimeStatus?.recoverySummary || "none"}`,
                      `Active Thread: ${activeThread?.name || "none"}`,
                      `Task: ${latestTask ? `${latestTask.title} / ${latestTask.status} / ${latestTask.resultSummary || latestTask.input}` : "none"}`,
                      `Tool Call: ${latestToolCall ? `${latestToolCall.toolId} / ${latestToolCall.status} / ${latestToolCall.summary}` : "none"}`,
                      `Refresh: ${streamState}`,
                      `Capabilities: ${summarizeGroups(runtimeStatus?.skillsByGroup) || "none"}`,
                      `MCP: ${summarizeGroups(runtimeStatus?.mcpByGroup) || "none"}`,
                    ].join("\n")}
              </div>
            </section>
          </aside>
        </section>
      </section>
    </main>
  );
}

function newestBy<T>(items: T[], getTime: (item: T) => string, tieBreaker?: (left: T, right: T) => number) {
  if (items.length === 0) {
    return null;
  }

  return (
    [...items].sort((left, right) => {
      const delta = toTimestamp(getTime(right)) - toTimestamp(getTime(left));
      if (delta !== 0) {
        return delta;
      }
      if (tieBreaker) {
        return tieBreaker(left, right);
      }
      return 0;
    })[0] ?? null
  );
}

function toolCallStatusWeight(status: string) {
  switch (status) {
    case "completed":
      return 3;
    case "failed":
      return 2;
    case "running":
      return 1;
    default:
      return 0;
  }
}

function ResultCard({ label, title, body }: { label: string; title: string; body: string }) {
  return (
    <article className="result-card">
      <p>{label}</p>
      <strong>{title}</strong>
      <span>{body}</span>
    </article>
  );
}

function InfoCard({ label, value, detail }: { label: string; value: string; detail: string }) {
  return (
    <article className="info-card">
      <p>{label}</p>
      <strong>{value}</strong>
      <span>{detail}</span>
    </article>
  );
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="summary-row">
      <p>{label}</p>
      <strong>{value}</strong>
    </div>
  );
}

function summarizeGroups(groups?: Record<string, string[]>) {
  if (!groups) return "";
  return Object.entries(groups)
    .filter(([, items]) => items.length > 0)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([group, items]) => `${group}:${items.length}`)
    .join(" / ");
}

function summarizeText(value: string, maxLength: number) {
  const compact = value.replace(/\s+/g, " ").trim();
  if (!compact) return "空内容";
  if (compact.length <= maxLength) return compact;
  return `${compact.slice(0, maxLength)}...`;
}

function toTimestamp(value: string) {
  const timestamp = new Date(value).getTime();
  return Number.isNaN(timestamp) ? 0 : timestamp;
}

function formatTime(value: string) {
  if (!value) return "";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleTimeString("zh-CN", { hour12: false });
}
