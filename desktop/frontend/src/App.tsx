import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  ActivateThread,
  ApproveTask,
  AdvanceTask,
  BrowserActivateTab,
  BrowserBack,
  BrowserCloseTab,
  BrowserForward,
  BrowserNavigate,
  BrowserOpen,
  BrowserReload,
  CheckBridge,
  CreateTask,
  CreateThread,
  GetAppInfo,
  GetBrowserState,
  GetRuntimeStatus,
  RejectTask,
  type BridgeCheckResult,
  type BrowserWorkspaceState,
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
  kind: "model.response.create",
  input: "",
};

const defaultPreviewURL = "http://127.0.0.1:5174/";
const embeddedPreviewParam = "gcPreview";
const showPreviewDebug = import.meta.env.DEV;

const executableKinds = [
  "model.response.create",
  "workspace.read_file",
  "workspace.list_files",
  "workspace.search_text",
  "thread.message.append",
];

export default function App() {
  const embeddedPreview = useMemo(() => getEmbeddedPreviewState(), []);
  if (embeddedPreview) {
    return <EmbeddedPreviewPage pane={embeddedPreview.pane} threadID={embeddedPreview.threadID} threadName={embeddedPreview.threadName} />;
  }

  const [runtimeStatus, setRuntimeStatus] = useState<RuntimeStatus | null>(null);
  const [browserState, setBrowserState] = useState<BrowserWorkspaceState | null>(null);
  const [bridgeResult, setBridgeResult] = useState<BridgeCheckResult | null>(null);
  const [statusMessage, setStatusMessage] = useState("正在加载桌面状态...");
  const [streamState, setStreamState] = useState("手动刷新模式");
  const [appInfo, setAppInfo] = useState("正在连接桌面外壳...");
  const [lastCheckedAt, setLastCheckedAt] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [sseEnabled, setSseEnabled] = useState(false);
  const [draft, setDraft] = useState<Draft>(defaultDraft);
  const [browserOpen, setBrowserOpen] = useState(true);
  const [addressDraft, setAddressDraft] = useState(defaultPreviewURL);
  const [lastSubmittedPreviewURL, setLastSubmittedPreviewURL] = useState("");
  const addressInputRef = useRef<HTMLInputElement | null>(null);
  const addressDraftRef = useRef(defaultPreviewURL);
  const lastSyncedPreviewURL = useRef(defaultPreviewURL);
  const threadPreviewMemory = useRef<Record<string, string>>({});
  const previewOwnerThreadID = useRef("");

  const refreshStatus = async (runBridgeCheck: boolean) => {
    setLoading(true);
    setError("");
    try {
      const [info, runtime, browser, bridge] = await Promise.all([
        GetAppInfo(),
        GetRuntimeStatus(),
        GetBrowserState(),
        runBridgeCheck ? CheckBridge() : Promise.resolve(null),
      ]);

      setAppInfo(info);
      setRuntimeStatus(runtime);
      setBrowserState(browser);
      setBrowserOpen(browser.isOpen);

      const activeTab = browser.tabs.find((item) => item.id === browser.activeTabId) ?? browser.tabs[0];
      if (activeTab?.url && (addressDraftRef.current === lastSyncedPreviewURL.current || !addressDraftRef.current.trim())) {
        setAddressDraft(activeTab.url);
        addressDraftRef.current = activeTab.url;
        lastSyncedPreviewURL.current = activeTab.url;
      }

      if (bridge) {
        setBridgeResult(bridge);
        setStatusMessage(bridge.message || "桌面桥接已连接");
        setLastCheckedAt(formatTime(bridge.checkedAt));
      } else {
        setStatusMessage(runtime.runtimeMessage || "运行状态已刷新");
        setLastCheckedAt(formatTime(runtime.updatedAt));
      }

      if (!runtime.supportsSSE || !runtime.sseEndpoint) {
        setStreamState("当前未接入 SSE，使用手动刷新");
        setSseEnabled(false);
      } else {
        setStreamState("SSE 已连接，工作台会自动刷新");
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
      setStreamState("SSE 已断开，已回退到手动刷新");
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
  const activeBrowserTab = useMemo(
    () => browserState?.tabs.find((tab) => tab.id === browserState.activeTabId) ?? browserState?.tabs[0] ?? null,
    [browserState],
  );

  const tasks = runtimeStatus?.tasks ?? [];
  const approvals = runtimeStatus?.approvals ?? [];
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
  const pendingApprovals = useMemo(() => approvals.filter((item) => item.status === "pending"), [approvals]);

  const flowItems = useMemo(() => {
    const items: FlowItem[] = [];

    for (const task of tasks.slice(0, 8)) {
      items.push({
        id: `task-${task.id}`,
        tone: task.status === "completed" ? "good" : task.status === "failed" ? "warning" : "neutral",
        badge: `task / ${task.status}`,
        title: `${task.title} / ${task.kind || "prompt"}`,
        body: task.resultSummary || task.input || "等待任务输入",
        meta: formatTime(task.updatedAt || task.createdAt),
        timestamp: toTimestamp(task.updatedAt || task.createdAt),
        actions:
          task.status === "needs_approval"
            ? (
                <div className="flow-item__actions">
                  <button className="thread-action" onClick={() => void handleApproveTask(task.id)} disabled={loading}>
                    Approve
                  </button>
                  <button className="thread-action thread-action--danger" onClick={() => void handleRejectTask(task.id)} disabled={loading}>
                    Reject
                  </button>
                </div>
              )
            : (
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

  const runtimeSourceLabel =
    runtimeStatus?.runtimeSource === "runtime-http"
      ? "远端 App Server"
      : runtimeStatus?.runtimeSource === "desktop-local"
        ? "本地 fallback"
        : runtimeStatus?.runtimeSource || "未连接";
  const statusTone = error ? "warning" : runtimeStatus?.runtimeReady || bridgeResult?.ok ? "good" : "muted";
  const projectName = runtimeStatus?.projectRoot?.replace(/\\/g, "/").split("/").filter(Boolean).pop() || "gen-code";
  const contextSummary = `messages ${messages.length} / tool calls ${toolCalls.length} / artifacts ${artifacts.length}`;
  const activeThreadRememberedURL = activeThread ? threadPreviewMemory.current[activeThread.id] || "" : "";
  const preferredProvider = runtimeStatus?.providers.find((item) => item.recommended) ?? runtimeStatus?.providers[0] ?? null;
  const providerSummary = preferredProvider
    ? `${preferredProvider.kind} / ${preferredProvider.preferredApiStyle || "unknown"} / ${preferredProvider.defaultModel || "no-model"}`
    : "暂无 provider";

  const threadPreviewSeed = (threadID: string, rawURL?: string) => {
    const candidate = rawURL?.trim();
    if (candidate) {
      return candidate;
    }
    return threadPreviewMemory.current[threadID] || defaultPreviewURL;
  };

  const navigatePreviewForThread = async (thread: RuntimeStatus["threads"][number], rawURL?: string) => {
    if (!browserOpen || !thread) {
      return;
    }

    const seedURL = threadPreviewSeed(thread.id, rawURL);
    if (!isThreadPreviewURL(seedURL)) {
      return;
    }

    const targetURL = withThreadContext(seedURL, thread.id, thread.name);
    const currentURL = activeBrowserTab?.url || "";
    if (targetURL === currentURL && previewOwnerThreadID.current === thread.id) {
      return;
    }

    setLastSubmittedPreviewURL(targetURL);
    const next = await BrowserNavigate(activeBrowserTab?.id || "", targetURL);
    previewOwnerThreadID.current = thread.id;
    threadPreviewMemory.current[thread.id] = targetURL;
    setBrowserState(next);
    setBrowserOpen(next.isOpen);
    setAddressDraft(targetURL);
    addressDraftRef.current = targetURL;
    lastSyncedPreviewURL.current = targetURL;
  };

  const handleCreateThread = async () => {
    setLoading(true);
    setError("");
    try {
      const previousIDs = new Set((runtimeStatus?.threads ?? []).map((thread) => thread.id));
      const next = (await CreateThread("")) as RuntimeStatus;
      const createdThread =
        next.threads.find((thread) => !previousIDs.has(thread.id)) ??
        next.threads[next.threads.length - 1] ??
        null;

      if (createdThread && !createdThread.isActive) {
        const activated = (await ActivateThread(createdThread.id)) as RuntimeStatus;
        setRuntimeStatus(activated);
        const targetThread = activated.threads.find((thread) => thread.id === createdThread.id) ?? createdThread;
        await navigatePreviewForThread(targetThread);
        setLastCheckedAt(formatTime(activated.updatedAt));
      } else {
        setRuntimeStatus(next);
        if (createdThread) {
          await navigatePreviewForThread(createdThread);
        }
        setLastCheckedAt(formatTime(next.updatedAt));
      }

      setStatusMessage("已创建新 thread");
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
      const targetThread = next.threads.find((thread) => thread.id === id);
      if (targetThread) {
        await navigatePreviewForThread(targetThread);
      }
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
      const payload = JSON.stringify({
        title: draft.title,
        kind: draft.kind,
        input: normalizeTaskInput(draft.kind, draft.input),
      });
      const next = (await CreateTask(runtimeStatus.activeThreadId, payload)) as RuntimeStatus;
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

  const handleRunTask = async (taskID: string) => {
    if (!runtimeStatus?.activeThreadId) return;

    setLoading(true);
    setError("");
    try {
      const next = (await AdvanceTask(taskID)) as RuntimeStatus;
      setRuntimeStatus(next);
      setStatusMessage("task 已触发执行");
      setLastCheckedAt(formatTime(next.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "执行 task 失败");
    } finally {
      setLoading(false);
    }
  };

  const handleApproveTask = async (taskID: string) => {
    if (!runtimeStatus?.activeThreadId) return;

    setLoading(true);
    setError("");
    try {
      const next = (await ApproveTask(runtimeStatus.activeThreadId, taskID)) as RuntimeStatus;
      setRuntimeStatus(next);
      setStatusMessage("写任务已批准，正在执行补丁");
      setLastCheckedAt(formatTime(next.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "批准任务失败");
    } finally {
      setLoading(false);
    }
  };

  const handleRejectTask = async (taskID: string) => {
    if (!runtimeStatus?.activeThreadId) return;

    setLoading(true);
    setError("");
    try {
      const next = (await RejectTask(runtimeStatus.activeThreadId, taskID)) as RuntimeStatus;
      setRuntimeStatus(next);
      setStatusMessage("写任务已拒绝，未修改项目文件");
      setLastCheckedAt(formatTime(next.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "拒绝任务失败");
    } finally {
      setLoading(false);
    }
  };

  const handleOpenPreview = async () => {
    const draftValue = addressInputRef.current?.value || addressDraftRef.current;
    const openURL =
      activeThread && isThreadPreviewURL(draftValue) ? withThreadContext(draftValue, activeThread.id, activeThread.name) : draftValue;
    const next = await BrowserOpen(openURL);
    if (activeThread && isThreadPreviewURL(openURL)) {
      previewOwnerThreadID.current = activeThread.id;
    }
    setBrowserState(next);
    setBrowserOpen(next.isOpen);
    const nextActiveTab = next.tabs.find((tab) => tab.id === next.activeTabId) ?? next.tabs[0];
    if (activeThread && nextActiveTab?.url && isThreadPreviewURL(nextActiveTab.url)) {
      threadPreviewMemory.current[activeThread.id] = nextActiveTab.url;
      setAddressDraft(nextActiveTab.url);
      addressDraftRef.current = nextActiveTab.url;
      lastSyncedPreviewURL.current = nextActiveTab.url;
    }
  };

  const handleNavigatePreview = async () => {
    const draftValue = addressInputRef.current?.value || addressDraftRef.current;
    if (activeThread) {
      await navigatePreviewForThread(activeThread, draftValue);
      return;
    }

    const next = await BrowserNavigate(activeBrowserTab?.id || "", draftValue);
    setLastSubmittedPreviewURL(draftValue);
    setBrowserState(next);
    setBrowserOpen(next.isOpen);
    setAddressDraft(draftValue);
    addressDraftRef.current = draftValue;
    lastSyncedPreviewURL.current = draftValue;
  };

  const handleToggleBrowser = () => {
    setBrowserOpen((current) => !current);
  };

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
              <p className="topbar__eyebrow">Gen Code / Codex-style Desktop</p>
              <h1>Thread 工作台</h1>
            </div>
          </div>

          <div className="workspace-topbar__meta">
            <span className="chip chip--soft">Wails Shell</span>
            <span className="chip chip--soft">{runtimeSourceLabel}</span>
            <span className="chip chip--soft">{appInfo}</span>
            {lastCheckedAt ? <span className="chip chip--soft">{`更新于 ${lastCheckedAt}`}</span> : null}
            <button className="secondary-action browser-toggle" onClick={handleToggleBrowser} type="button">
              {browserOpen ? "收起预览" : "展开预览"}
            </button>
            <span className={`chip chip--${statusTone}`}>{error ? "需要处理" : "状态正常"}</span>
          </div>
        </header>

        <section className={`workbench ${browserOpen ? "workbench--with-browser" : "workbench--focused"}`}>
          <section className="workbench-main">
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
                        data-testid={`thread-card-${thread.id}`}
                        data-thread-id={thread.id}
                        data-active={thread.isActive ? "true" : "false"}
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
                <p className="sidebar-note">右侧预览区首轮只服务本地联调，不扩成公网通用浏览器。</p>
                <p className="sidebar-note">当前 active thread 会驱动本地预览 URL 自动携带 thread 参数，并记住各自的预览地址。</p>
                <p className="sidebar-note sidebar-note--strong">可执行 kind: {executableKinds.join(" / ")}</p>
              </section>
            </aside>

            <section className="center-stage">
              <section className="stage-header card">
                <div>
                  <p className="section-title">当前线程</p>
                  <h2>{activeThread?.name || "等待 active thread"}</h2>
                  <p className="stage-header__lead">
                    {activeThread
                      ? "这里展示当前 thread 的消息流、任务进展、工具调用和实时事件。"
                      : "请先创建或激活一个 thread。"}
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
                    <h3>给 active thread 追加一个真实任务</h3>
                  </div>
                  <span className="mini-chip">{runtimeStatus?.activeThreadId ? "active thread ready" : "no active thread"}</span>
                </div>

                <div className="composer-grid">
                  <label className="field">
                    <span className="field__label">标题</span>
                    <input
                      data-testid="task-title-input"
                      value={draft.title}
                      onChange={(event) => setDraft((current) => ({ ...current, title: event.target.value }))}
                      placeholder="例如：让模型总结 README"
                    />
                  </label>

                  <label className="field field--compact">
                    <span className="field__label">kind</span>
                    <select
                      data-testid="task-kind-select"
                      value={draft.kind}
                      onChange={(event) => setDraft((current) => ({ ...current, kind: event.target.value }))}
                    >
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
                    data-testid="task-input-textarea"
                    value={draft.input}
                    onChange={(event) => setDraft((current) => ({ ...current, input: event.target.value }))}
                    placeholder='模型任务可直接输入 prompt，也可使用 JSON，例如 {"provider":"anthropic","model":"gpt-5.4-A","input":"请总结 README"}'
                    rows={4}
                  />
                </label>

                <div className="composer-actions">
                  <button className="primary-action" data-testid="create-task-button" onClick={handleCreateTask} disabled={loading || !runtimeStatus?.activeThreadId}>
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

                {pendingApprovals.length > 0 ? (
                  <div className="approval-panel" data-testid="approval-panel">
                    {pendingApprovals.map((approval) => (
                      <article className="flow-item flow-item--warning" key={approval.id}>
                        <div className="flow-item__header">
                          <span className="mini-chip">approval / pending</span>
                          <span className="flow-item__meta">{formatTime(approval.updatedAt)}</span>
                        </div>
                        <h4>{approval.toolKind}</h4>
                        <p>{approval.summary}</p>
                        <p>{approval.targetPaths.join(", ")}</p>
                        <div className="flow-item__actions">
                          <button className="thread-action" onClick={() => void handleApproveTask(approval.taskId)} disabled={loading}>
                            Approve
                          </button>
                          <button className="thread-action thread-action--danger" onClick={() => void handleRejectTask(approval.taskId)} disabled={loading}>
                            Reject
                          </button>
                        </div>
                      </article>
                    ))}
                  </div>
                ) : null}

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

              <section className="context-strip card">
                <div className="section-header">
                  <div>
                    <p className="section-title">结果抽屉</p>
                    <h3>当前线程的关键信息与运行摘要</h3>
                  </div>
                  <span className="mini-chip">{activeThread?.id || "no-thread"}</span>
                </div>

                <div className="context-grid">
                  <ResultCard label="Latest Task" title={latestTask ? `${latestTask.title} / ${latestTask.status}` : "暂无任务"} body={latestTask?.resultSummary || latestTask?.input || "暂无结果"} />
                  <ResultCard label="Latest Tool Call" title={latestToolCall ? `${latestToolCall.toolId} / ${latestToolCall.status}` : "暂无 tool call"} body={latestToolCall?.summary || "暂无摘要"} />
                  <ResultCard label="Latest Message" title={latestMessage ? latestMessage.role : "暂无 message"} body={latestMessage?.content || "暂无 message"} />
                  <ResultCard label="Latest Event" title={latestEvent ? latestEvent.type : "暂无 event"} body={latestEvent?.message || "暂无 event"} />
                  <ResultCard label="Latest Artifact" title={latestArtifact ? latestArtifact.kind : "暂无 artifact"} body={latestArtifact?.path || "暂无 artifact"} />
                  <InfoCard label="Bridge / SSE" value={sseEnabled ? "Connected" : "Manual"} detail={`${bridgeResult?.message || "桥接待检查"} / ${streamState}`} />
                  <InfoCard label="Approvals" value={String(pendingApprovals.length)} detail={pendingApprovals[0]?.summary || "当前没有待审批写任务"} />
                  <InfoCard label="Provider" value={preferredProvider?.kind || "暂无 provider"} detail={providerSummary} />
                  <InfoCard label="State Store" value={runtimeStatus?.stateStore || "sqlite"} detail={runtimeStatus?.statePath || "project-local state store"} />
                </div>
              </section>
            </section>
          </section>

          {browserOpen ? (
            <aside className="browser-rail card">
              <section className="browser-shell">
                <div className="browser-tabs">
                  {(browserState?.tabs ?? []).map((tab) => (
                    <div
                      key={tab.id}
                      className={`browser-tab ${tab.isActive ? "browser-tab--active" : ""}`}
                      data-testid={`browser-tab-${tab.id}`}
                      data-tab-id={tab.id}
                      data-active={tab.isActive ? "true" : "false"}
                    >
                      <button className="browser-tab__label" onClick={() => void BrowserActivateTab(tab.id).then(setBrowserState)} type="button">
                        <span>{tab.title}</span>
                      </button>
                      <button
                        className="browser-tab__close"
                        onClick={(event) => {
                          event.stopPropagation();
                          void BrowserCloseTab(tab.id).then(setBrowserState);
                        }}
                        type="button"
                      >
                        ×
                      </button>
                    </div>
                  ))}
                  <button className="browser-add-tab" onClick={() => void handleOpenPreview()} type="button">
                    +
                  </button>
                </div>

                <div className="browser-toolbar">
                  <div className="browser-toolbar__actions">
                    <button className="browser-nav" onClick={() => activeBrowserTab && void BrowserBack(activeBrowserTab.id).then(setBrowserState)} type="button">
                      ←
                    </button>
                    <button className="browser-nav" onClick={() => activeBrowserTab && void BrowserForward(activeBrowserTab.id).then(setBrowserState)} type="button">
                      →
                    </button>
                    <button className="browser-nav" onClick={() => activeBrowserTab && void BrowserReload(activeBrowserTab.id).then(setBrowserState)} type="button">
                      刷新
                    </button>
                  </div>

                  <form
                    className="browser-address"
                    onSubmit={(event) => {
                      event.preventDefault();
                      void handleNavigatePreview();
                    }}
                  >
                    <input
                      data-testid="browser-address-input"
                      ref={addressInputRef}
                      value={addressDraft}
                      onKeyDown={(event) => {
                        if (event.key === "Enter") {
                          event.preventDefault();
                          void handleNavigatePreview();
                        }
                      }}
                      onChange={(event) => {
                        setAddressDraft(event.target.value);
                        addressDraftRef.current = event.target.value;
                      }}
                      placeholder="输入本地预览地址，例如 http://127.0.0.1:5174/"
                    />
                    <button className="browser-nav" data-testid="browser-navigate-button" type="submit">
                      前往
                    </button>
                  </form>
                </div>

                <div className="browser-statusbar">
                  <span className="browser-statusbar__title" data-testid="browser-status-title">
                    {activeBrowserTab?.title || "本地预览"}
                  </span>
                  <span className="mini-chip">{activeBrowserTab?.status || "ready"}</span>
                </div>

                {showPreviewDebug ? (
                  <div className="left-panel left-panel--muted" data-testid="browser-debug-panel">
                    <p className="section-title">Preview Debug</p>
                    <p className="sidebar-note">{`submitted: ${lastSubmittedPreviewURL || "none"}`}</p>
                    <p className="sidebar-note">{`tab url: ${activeBrowserTab?.url || "none"}`}</p>
                    <p className="sidebar-note">{`owner: ${previewOwnerThreadID.current || "none"}`}</p>
                    <p className="sidebar-note">{`memory: ${activeThreadRememberedURL || "none"}`}</p>
                  </div>
                ) : null}

                <div className="browser-surface">
                  {activeBrowserTab ? (
                    <iframe
                      className="browser-frame"
                      data-testid="browser-preview-frame"
                      src={activeBrowserTab.url}
                      title={activeBrowserTab.title}
                    />
                  ) : (
                    <div className="browser-empty">暂无可用标签，点击右上角 + 新建一个本地预览标签。</div>
                  )}
                </div>
              </section>
            </aside>
          ) : null}
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

function summarizeText(value: string, maxLength: number) {
  const compact = value.replace(/\s+/g, " ").trim();
  if (!compact) return "空内容";
  if (compact.length <= maxLength) return compact;
  return `${compact.slice(0, maxLength)}...`;
}

function normalizeTaskInput(kind: string, rawInput: string) {
  const trimmed = rawInput.trim();
  if (kind !== "model.response.create") {
    return trimmed;
  }

  if (!trimmed) {
    return JSON.stringify({ input: "" });
  }

  if (trimmed.startsWith("{") && trimmed.endsWith("}")) {
    try {
      const parsed = JSON.parse(trimmed) as Record<string, unknown>;
      return JSON.stringify({
        provider: typeof parsed.provider === "string" ? parsed.provider : undefined,
        model: typeof parsed.model === "string" ? parsed.model : undefined,
        input: typeof parsed.input === "string" ? parsed.input : trimmed,
        maxOutputTokens: typeof parsed.maxOutputTokens === "number" ? parsed.maxOutputTokens : undefined,
      });
    } catch {
      return JSON.stringify({ input: trimmed });
    }
  }

  return JSON.stringify({ input: trimmed });
}

function isThreadPreviewURL(value: string) {
  try {
    const parsed = new URL(value);
    return parsed.protocol === "file:" || parsed.hostname === "127.0.0.1" || parsed.hostname === "localhost";
  } catch {
    return false;
  }
}

function withThreadContext(baseURL: string, threadID: string, threadName: string) {
  try {
    const parsed = new URL(baseURL);
    if (shouldUseEmbeddedPreview(parsed)) {
      parsed.searchParams.set(embeddedPreviewParam, "1");
    }
    parsed.searchParams.set("threadId", threadID);
    if (threadName) {
      parsed.searchParams.set("threadName", threadName);
    }
    return parsed.toString();
  } catch {
    return baseURL;
  }
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

function shouldUseEmbeddedPreview(parsed: URL) {
  return (
    (parsed.hostname === "127.0.0.1" || parsed.hostname === "localhost") &&
    (parsed.port === "5174" || (!parsed.port && parsed.protocol === "file:"))
  );
}

function getEmbeddedPreviewState() {
  if (typeof window === "undefined") {
    return null;
  }

  const params = new URLSearchParams(window.location.search);
  if (params.get(embeddedPreviewParam) !== "1") {
    return null;
  }

  return {
    pane: params.get("pane") || "default",
    threadID: params.get("threadId") || "",
    threadName: params.get("threadName") || "",
  };
}

function EmbeddedPreviewPage({
  pane,
  threadID,
  threadName,
}: {
  pane: string;
  threadID: string;
  threadName: string;
}) {
  const paneTitle =
    pane === "thread-one" ? "Thread One Preview" : pane === "thread-two" ? "Thread Two Preview" : "Local Preview";

  return (
    <main className="shell shell--embedded-preview" data-testid="embedded-preview-root">
      <section className="workspace-shell workspace-shell--embedded-preview">
        <header className="workspace-topbar workspace-topbar--embedded-preview">
          <div className="workspace-topbar__title">
            <div>
              <p className="topbar__eyebrow">Gen Code / Local Preview</p>
              <h1>{paneTitle}</h1>
            </div>
          </div>
          <div className="workspace-topbar__meta">
            <span className="chip chip--soft">{threadID || "no thread"}</span>
            <span className="chip chip--soft">{threadName || "preview mode"}</span>
          </div>
        </header>

        <section className="workbench workbench--focused">
          <section className="center-stage">
            <section className="stage-header card">
              <div>
                <p className="section-title">本地预览</p>
                <h2>{threadName || "未命名 thread"}</h2>
                <p className="stage-header__lead">右侧浏览器已进入轻量嵌入态，避免把整套工作台递归渲染进 iframe。</p>
              </div>
              <div className="stage-header__meta">
                <span className="chip chip--soft">{`pane ${pane}`}</span>
                <span className="chip chip--soft">{threadID || "no-thread"}</span>
              </div>
            </section>

            <section className="flow-panel card">
              <div className="section-header">
                <div>
                  <p className="section-title">预览上下文</p>
                  <h3>当前 thread 的本地预览参数</h3>
                </div>
              </div>
              <div className="flow-list">
                <article className="flow-item flow-item--good">
                  <div className="flow-item__header">
                    <span className="mini-chip">preview</span>
                    <span className="flow-item__meta">embedded</span>
                  </div>
                  <h4>{paneTitle}</h4>
                  <p>{`threadId=${threadID || "none"} / threadName=${threadName || "none"} / pane=${pane}`}</p>
                </article>
              </div>
            </section>
          </section>
        </section>
      </section>
    </main>
  );
}
