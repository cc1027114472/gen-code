import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  ActivateThread,
  ApproveTask,
  AdvanceTask,
  BrowserActivateTab,
  BrowserBack,
  BrowserCloseTab,
  BrowserClick,
  BrowserExtract,
  BrowserForward,
  BrowserNavigate,
  BrowserOpen,
  BrowserReload,
  BrowserScreenshot,
  BrowserType,
  CheckBridge,
  CreateTask,
  CreateThread,
  formatFallbackNote,
  formatRefreshMode,
  formatRefreshModeDetail,
  formatRuntimeLaneDetail,
  formatRuntimeLaneLabel,
  formatRuntimeTrustLabel,
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
  className?: string;
  badge: string;
  title: string;
  body: ReactNode;
  meta: string;
  timestamp: number;
  actions?: ReactNode;
};

type RuntimeTask = RuntimeStatus["tasks"][number];
type RuntimeApproval = RuntimeStatus["approvals"][number];
type RuntimeWriteExecution = RuntimeStatus["writeExecutions"][number];
type RuntimeThread = RuntimeStatus["threads"][number];

type WorkspaceWorkflowSummary = {
  id: string;
  root: string;
  projectRoot: string;
  activeThreadId: string;
  activeThreadName: string;
  threadCount: number;
  taskCount: number;
  waitingTaskCount: number;
  approvalRequiredCount: number;
  pendingApprovalCount: number;
  completedTaskCount: number;
  failedTaskCount: number;
  writeExecutionCount: number;
  summary: string;
};

type ThreadWorkflowSummary = {
  id: string;
  name: string;
  status: string;
  permissionMode: string;
  activeModel: string;
  taskCount: number;
  waitingTaskCount: number;
  waitingForTaskCount: number;
  waitingForApprovalCount: number;
  approvalRequiredCount: number;
  pendingApprovalCount: number;
  completedTaskCount: number;
  failedTaskCount: number;
  childTaskCount: number;
  writeExecutionCount: number;
  latestTaskId: string;
  latestApprovalTaskId: string;
  latestWriteTaskId: string;
  summary: string;
};

type ExtendedRuntimeTask = RuntimeTask & {
  waitingTaskId?: string;
  waitingSummary?: string;
  workflowLabel?: string;
  childTaskIds?: string[];
  latestChildTaskId?: string;
  approvalId?: string;
  approvalSummary?: string;
  writeExecutionId?: string;
  writeExecutionSummary?: string;
};

type ExtendedRuntimeStatus = RuntimeStatus & {
  workspaceSummary?: WorkspaceWorkflowSummary;
  activeThreadSummary?: ThreadWorkflowSummary;
  tasks: ExtendedRuntimeTask[];
};

type TaskGroup = {
  title: string;
  tone: "neutral" | "good" | "warning";
  tasks: ExtendedRuntimeTask[];
};

type TaskStateSignal = {
  tone: FlowItem["tone"];
  summary: string;
  consumesResultSummary?: boolean;
};

type PatchStats = {
  added: number;
  removed: number;
  hasPreview: boolean;
};

type ApprovalViewModel = {
  id: string;
  taskId: string;
  title: string;
  toolKind: string;
  taskKind: string;
  taskStatus: string;
  taskStatusLabel: string;
  approvalStatus: string;
  approvalStatusLabel: string;
  summary: string;
  targetPaths: string[];
  patchPreview: string;
  patchStats: PatchStats;
  patchStatsLabel: string;
  updatedAt: string;
};

type WriteExecutionViewModel = {
  id: string;
  taskId: string;
  approvalId: string;
  title: string;
  toolKind: string;
  operation: string;
  relatedExecutionId: string;
  status: string;
  statusLabel: string;
  targetPaths: string[];
  patchSummary: string;
  beforeSummary: string;
  afterSummary: string;
  resultSummary: string;
  updatedAt: string;
};

type SkillGovernanceViewModel = {
  group: string;
  implementedCount: number;
  verifiedCount: number;
  localizationPending: number;
  skillIDs: string[];
  isolationSummary: string;
  summary: string;
};

type SkillGovernanceRollup = {
  implementedCount: number;
  verifiedCount: number;
  localizationPending: number;
  summary: string;
};

const defaultDraft: Draft = {
  title: "",
  kind: "agent.run",
  input: "",
};

const defaultPreviewURL = "http://127.0.0.1:5174/";
const embeddedPreviewParam = "gcPreview";
const showPreviewDebug = import.meta.env.DEV;
const runtimeEventTypes = [
  "thread.created",
  "thread.activated",
  "task.created",
  "task.started",
  "task.completed",
  "task.failed",
  "task.approval_required",
  "task.approved",
  "task.rejected",
  "task.recovered_as_failed",
  "task.rollback_required",
  "toolcall.started",
  "toolcall.completed",
  "toolcall.failed",
  "toolcall.approved",
  "toolcall.rejected",
] as const;

export default function App() {
  const embeddedPreview = useMemo(() => getEmbeddedPreviewState(), []);
  if (embeddedPreview) {
    return <EmbeddedPreviewPage pane={embeddedPreview.pane} threadID={embeddedPreview.threadID} threadName={embeddedPreview.threadName} />;
  }

  const [runtimeStatus, setRuntimeStatus] = useState<ExtendedRuntimeStatus | null>(null);
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
  const [browserSelectorDraft, setBrowserSelectorDraft] = useState("[data-testid='browser-address-input']");
  const [browserTextDraft, setBrowserTextDraft] = useState("browser demo text");
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
      setRuntimeStatus(runtime as ExtendedRuntimeStatus);
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
    for (const eventType of runtimeEventTypes) {
      source.addEventListener(eventType, refresh);
    }
    source.onerror = () => {
      setSseEnabled(false);
      setStreamState("SSE 已断开，已回退到手动刷新");
      source.close();
    };

    return () => {
      for (const eventType of runtimeEventTypes) {
        source.removeEventListener(eventType, refresh);
      }
      source.close();
      setSseEnabled(false);
    };
  }, [runtimeStatus?.supportsSSE, runtimeStatus?.sseEndpoint]);

  const activeThread = useMemo(
    () => runtimeStatus?.threads.find((thread) => thread.id === runtimeStatus.activeThreadId) ?? null,
    [runtimeStatus],
  );
  const extendedRuntime = runtimeStatus as ExtendedRuntimeStatus | null;
  const workspaceSummary = extendedRuntime?.workspaceSummary;
  const activeThreadWorkflow = extendedRuntime?.activeThreadSummary;
  const activeBrowserTab = useMemo(
    () => browserState?.tabs.find((tab) => tab.id === browserState.activeTabId) ?? browserState?.tabs[0] ?? null,
    [browserState],
  );
  const browserLatestSummary = browserState?.latestActionSummary || "browser workspace ready";
  const browserLatestError = browserState?.latestActionError || "";
  const browserLatestExtract = browserState?.latestExtractText || "";
  const browserLatestArtifactPath = browserState?.latestArtifactPath || "";

  const tasks: ExtendedRuntimeTask[] = extendedRuntime?.tasks ?? [];
  const taskMap = useMemo(() => new Map(tasks.map((task) => [task.id, task])), [tasks]);
  const executableKinds = useMemo(() => {
    const defaults = [
      "agent.run",
      "model.response.create",
      "thread.message.append",
      "workspace.list_files",
      "workspace.read_file",
      "workspace.search_text",
      "workspace.stat_file",
      "workspace.read_files_batch",
      "workspace.list_files_filtered",
      "workspace.search_text_detailed",
    ];
    return [...new Set([...(runtimeStatus?.executableKinds ?? []), ...defaults])].sort((left, right) => left.localeCompare(right));
  }, [runtimeStatus?.executableKinds]);
  const approvals = runtimeStatus?.approvals ?? [];
  const writeExecutions = runtimeStatus?.writeExecutions ?? [];
  const messages = runtimeStatus?.messages ?? [];
  const toolCalls = runtimeStatus?.toolCalls ?? [];
  const artifacts = runtimeStatus?.artifacts ?? [];
  const events = runtimeStatus?.events ?? [];
  const skills = runtimeStatus?.skills ?? [];
  const skillGovernance = runtimeStatus?.skillGovernance ?? [];

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
  const latestWriteExecution = useMemo(() => newestBy(writeExecutions, (item) => item.updatedAt || item.createdAt), [writeExecutions]);
  const latestAgentTask = useMemo(
    () => newestBy(tasks.filter((task) => isAgentParentTask(task)), (item) => item.updatedAt || item.createdAt),
    [tasks],
  );

  const approvalViewModels = useMemo(() => {
    const taskMap = new Map(tasks.map((task) => [task.id, task]));
    return approvals
      .map((approval) => createApprovalViewModel(approval, taskMap.get(approval.taskId)))
      .sort((left, right) => toTimestamp(right.updatedAt) - toTimestamp(left.updatedAt));
  }, [approvals, tasks]);
  const latestApproval = useMemo(() => newestBy(approvalViewModels, (item) => item.updatedAt), [approvalViewModels]);

  const writeExecutionViewModels = useMemo(() => {
    const taskMap = new Map(tasks.map((task) => [task.id, task]));
    const approvalMap = new Map(approvals.map((approval) => [approval.taskId, approval]));
    return writeExecutions
      .map((execution) => createWriteExecutionViewModel(execution, taskMap.get(execution.taskId), approvalMap.get(execution.taskId)))
      .sort((left, right) => toTimestamp(right.updatedAt) - toTimestamp(left.updatedAt));
  }, [approvals, tasks, writeExecutions]);

  const approvalByTaskID = useMemo(() => new Map(approvalViewModels.map((item) => [item.taskId, item])), [approvalViewModels]);
  const writeExecutionByTaskID = useMemo(() => new Map(writeExecutionViewModels.map((item) => [item.taskId, item])), [writeExecutionViewModels]);
  const pendingApprovals = useMemo(() => approvalViewModels.filter((item) => item.approvalStatus === "pending"), [approvalViewModels]);
  const recentWriteExecutions = useMemo(() => writeExecutionViewModels.slice(0, 3), [writeExecutionViewModels]);
  const latestRollbackCandidate = useMemo(
    () => writeExecutionViewModels.find((item) => item.operation === "apply" && item.status === "completed") ?? null,
    [writeExecutionViewModels],
  );
  const skillGovernanceViewModels = useMemo<SkillGovernanceViewModel[]>(() => {
    const skillIDsByGroup = new Map<string, string[]>();
    for (const skill of skills) {
      const group = (skill.group || "common").trim() || "common";
      const current = skillIDsByGroup.get(group) ?? [];
      current.push(skill.id);
      current.sort((left, right) => left.localeCompare(right));
      skillIDsByGroup.set(group, current);
    }
    return skillGovernance.map((item) => {
      const skillIDs = skillIDsByGroup.get(item.group) ?? [];
      const isolationStates = skills
        .filter((skill) => ((skill.group || "common").trim() || "common") === item.group)
        .map((skill) => (skill.isolationStatus || "unknown").trim() || "unknown");
      const uniqueIsolationStates = [...new Set(isolationStates)].sort((left, right) => left.localeCompare(right));
      const isolationSummary = uniqueIsolationStates.length > 0 ? uniqueIsolationStates.join(", ") : "unknown";
      return {
        group: item.group,
        implementedCount: item.implementedCount,
        verifiedCount: item.verifiedCount,
        localizationPending: item.localizationPending,
        skillIDs,
        isolationSummary,
        summary: `${item.group}: implemented=${item.implementedCount} verified=${item.verifiedCount} localization-pending=${item.localizationPending} isolation=${isolationSummary}`,
      };
    });
  }, [skillGovernance, skills]);
  const skillGovernanceRollup = useMemo<SkillGovernanceRollup>(() => {
    const rollup = skillGovernanceViewModels.reduce(
      (acc, item) => {
        acc.implementedCount += item.implementedCount;
        acc.verifiedCount += item.verifiedCount;
        acc.localizationPending += item.localizationPending;
        return acc;
      },
      { implementedCount: 0, verifiedCount: 0, localizationPending: 0, summary: "" },
    );
    rollup.summary =
      rollup.implementedCount > 0
        ? `implemented=${rollup.implementedCount} verified=${rollup.verifiedCount} localization-pending=${rollup.localizationPending}`
        : "当前未发现可汇总的 skill 治理数据";
    return rollup;
  }, [skillGovernanceViewModels]);

  const workflowHighlights = useMemo(
    () => [
      { label: "等待中", value: String(activeThreadWorkflow?.waitingTaskCount ?? workspaceSummary?.waitingTaskCount ?? 0), detail: "因子任务或审批而阻塞的任务数" },
      { label: "待审批", value: String(activeThreadWorkflow?.pendingApprovalCount ?? workspaceSummary?.pendingApprovalCount ?? 0), detail: "等待用户决策的审批项数" },
      { label: "写执行", value: String(activeThreadWorkflow?.writeExecutionCount ?? workspaceSummary?.writeExecutionCount ?? 0), detail: "当前视图内已完成的写执行数" },
      { label: "失败", value: String(activeThreadWorkflow?.failedTaskCount ?? workspaceSummary?.failedTaskCount ?? 0), detail: "需要关注或重跑的任务数" },
    ],
    [activeThreadWorkflow, workspaceSummary],
  );
  const waitingTasks = useMemo<ExtendedRuntimeTask[]>(
    () => tasks.filter((task) => task.waitingStatus || task.status === "waiting_for_task" || task.status === "waiting_for_approval"),
    [tasks],
  );
  const childTasks = useMemo<ExtendedRuntimeTask[]>(() => tasks.filter((task) => task.parentTaskId), [tasks]);
  const latestChildTask = useMemo(() => newestBy(childTasks, (item) => item.updatedAt || item.createdAt), [childTasks]);
  const approvalRequiredTasks = useMemo<ExtendedRuntimeTask[]>(
    () => tasks.filter((task) => task.status === "needs_approval" || task.approvalStatus === "pending"),
    [tasks],
  );

  const flowItems = useMemo(() => {
    const items: FlowItem[] = [];

    for (const task of tasks.slice(0, 8)) {
      const approval = approvalByTaskID.get(task.id);
      const writeExecution = writeExecutionByTaskID.get(task.id);
      const taskStatusLabel = formatTaskStatus(task.status);
      const parentTask = task.parentTaskId ? taskMap.get(task.parentTaskId) : undefined;
      const agentMeta = task.kind === "agent.run" ? formatAgentMeta(task) : "";
      const roleLabel = formatTaskRoleLabel(task);
      const waitingLabel = formatTaskWaitingStatusLabel(task);
      const taskTone = getTaskTone(task);
      const toneClass = isAgentParentTask(task)
        ? "flow-item--agent-parent"
        : task.parentTaskId
          ? "flow-item--agent-child"
          : "";
      items.push({
        id: `task-${task.id}`,
        className: toneClass,
        tone: taskTone,
        badge: `${roleLabel} / ${taskStatusLabel}${waitingLabel ? ` / ${waitingLabel}` : ""}`,
        title: `${formatTaskHeadline(task, parentTask)}${agentMeta ? ` / ${agentMeta}` : ""}`,
        body: writeExecution ? (
          <WriteExecutionDetails execution={writeExecution} />
        ) : approval ? (
          <ApprovalDetails approval={approval} />
        ) : (
          <FlowBodyText text={formatTaskDisplaySummary(task, parentTask)} />
        ),
        meta: formatTime(task.updatedAt || task.createdAt),
        timestamp: toTimestamp(task.updatedAt || task.createdAt),
        actions:
          task.status === "needs_approval"
            ? (
                <ApprovalActions taskId={task.id} loading={loading} onApprove={handleApproveTask} onReject={handleRejectTask} />
              )
            : (
                <button className="thread-action" onClick={() => void handleRunTask(task.id)} disabled={loading} type="button">
                  运行任务
                </button>
              ),
      });
    }

    for (const toolCall of toolCalls.slice(0, 8)) {
      items.push({
        id: `tool-${toolCall.id}`,
        tone: toolCall.status === "completed" ? "good" : toolCall.status === "failed" ? "warning" : "neutral",
        badge: `工具调用 / ${formatToolCallStatus(toolCall.status)}`,
        title: toolCall.toolId,
        body: <FlowBodyText text={toolCall.summary} />,
        meta: formatTime(toolCall.createdAt),
        timestamp: toTimestamp(toolCall.createdAt),
      });
    }

    for (const message of messages.slice(0, 8)) {
      items.push({
        id: `message-${message.id}`,
        tone: message.role === "assistant" ? "good" : "neutral",
        badge: `消息 / ${formatMessageRole(message.role)}`,
        title: summarizeText(message.content, 48),
        body: <FlowBodyText text={message.content} />,
        meta: formatTime(message.createdAt),
        timestamp: toTimestamp(message.createdAt),
      });
    }

    for (const event of events.slice(0, 10)) {
      items.push({
        id: `event-${event.id}`,
        tone: event.type.includes("failed") ? "warning" : event.type.includes("completed") ? "good" : "neutral",
        badge: `事件 / ${formatEventType(event.type)}`,
        title: event.type,
        body: <FlowBodyText text={event.message} />,
        meta: formatTime(event.createdAt),
        timestamp: toTimestamp(event.createdAt),
      });
    }

    return items.sort((left, right) => right.timestamp - left.timestamp).slice(0, 16);
  }, [approvalByTaskID, events, loading, messages, tasks, toolCalls, writeExecutionByTaskID]);

  const runtimeLaneLabel = formatRuntimeLaneLabel(runtimeStatus ?? undefined);
  const runtimeLaneDetail = formatRuntimeLaneDetail(runtimeStatus ?? undefined);
  const runtimeTrustLabel = formatRuntimeTrustLabel(runtimeStatus ?? undefined);
  const refreshModeLabel = formatRefreshMode(runtimeStatus ?? undefined, sseEnabled);
  const refreshModeDetail = formatRefreshModeDetail(runtimeStatus ?? undefined, sseEnabled);
  const fallbackNote = formatFallbackNote(runtimeStatus ?? undefined);
  const statusTone = error ? "warning" : runtimeStatus?.runtimeReady || bridgeResult?.ok ? "good" : "muted";
  const projectName = runtimeStatus?.projectRoot?.replace(/\\/g, "/").split("/").filter(Boolean).pop() || "gen-code";
  const contextSummary = `消息 ${messages.length} / 工具调用 ${toolCalls.length} / 写执行 ${writeExecutions.length}`;
  const activeThreadRememberedURL = activeThread ? threadPreviewMemory.current[activeThread.id] || "" : "";
  const preferredProvider = runtimeStatus?.providers.find((item) => item.recommended) ?? runtimeStatus?.providers[0] ?? null;
  const providerSummary = preferredProvider
    ? `${preferredProvider.kind} / ${preferredProvider.preferredApiStyle || "unknown"} / ${preferredProvider.defaultModel || "no-model"}`
    : "暂无 Provider";

  const threadPreviewSeed = (threadID: string, rawURL?: string) => {
    const candidate = rawURL?.trim();
    if (candidate) {
      return candidate;
    }
    return threadPreviewMemory.current[threadID] || defaultPreviewURL;
  };

  const applyBrowserState = (next: BrowserWorkspaceState) => {
    setBrowserState(next);
    setBrowserOpen(next.isOpen);
    const nextActiveTab = next.tabs.find((tab) => tab.id === next.activeTabId) ?? next.tabs[0] ?? null;
    if (nextActiveTab?.url && (addressDraftRef.current === lastSyncedPreviewURL.current || !addressDraftRef.current.trim())) {
      setAddressDraft(nextActiveTab.url);
      addressDraftRef.current = nextActiveTab.url;
      lastSyncedPreviewURL.current = nextActiveTab.url;
    }
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
      const name = `Thread ${new Date().toLocaleTimeString("zh-CN", { hour12: false })}`;
      const next = (await CreateThread(name)) as RuntimeStatus;
      const createdThread = next.threads.find((thread) => thread.isActive) ?? next.threads[0];
      setRuntimeStatus(next as ExtendedRuntimeStatus);
      if (createdThread) {
        await navigatePreviewForThread(createdThread);
      }
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
      setRuntimeStatus(next as ExtendedRuntimeStatus);
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
      setRuntimeStatus(next as ExtendedRuntimeStatus);
      setStatusMessage(draft.kind === "agent.run" ? "已创建 agent.run，请点击“运行任务”开始默认工作流" : "已创建 task");
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
      const targetTask = (runtimeStatus.tasks ?? []).find((task) => task.id === taskID);
      const next = (await AdvanceTask(taskID)) as RuntimeStatus;
      setRuntimeStatus(next as ExtendedRuntimeStatus);
      setStatusMessage(targetTask?.kind === "agent.run" ? "agent.run 已触发执行，正在派生子任务" : "task 已触发执行");
      setLastCheckedAt(formatTime(next.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "执行 task 失败");
    } finally {
      setLoading(false);
    }
  }

  async function handleApproveTask(taskID: string) {
    if (!runtimeStatus?.activeThreadId) return;

    setLoading(true);
    setError("");
    try {
      const next = (await ApproveTask(runtimeStatus.activeThreadId, taskID)) as RuntimeStatus;
      setRuntimeStatus(next as ExtendedRuntimeStatus);
      setStatusMessage("写任务已批准，正在执行补丁");
      setLastCheckedAt(formatTime(next.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "批准任务失败");
    } finally {
      setLoading(false);
    }
  }

  async function handleRejectTask(taskID: string) {
    if (!runtimeStatus?.activeThreadId) return;

    setLoading(true);
    setError("");
    try {
      const next = (await RejectTask(runtimeStatus.activeThreadId, taskID)) as RuntimeStatus;
      setRuntimeStatus(next as ExtendedRuntimeStatus);
      setStatusMessage("写任务已拒绝，未修改项目文件");
      setLastCheckedAt(formatTime(next.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "拒绝任务失败");
    } finally {
      setLoading(false);
    }
  }

  async function handleRollbackLatest(writeExecutionId: string) {
    if (!runtimeStatus?.activeThreadId) return;

    setLoading(true);
    setError("");
    try {
      const existingTaskIDs = new Set((runtimeStatus.tasks ?? []).map((task) => task.id));
      const payload = JSON.stringify({
        title: "Rollback latest write execution",
        kind: "workspace.apply_patch.rollback",
        input: JSON.stringify({ writeExecutionId }),
      });
      const created = (await CreateTask(runtimeStatus.activeThreadId, payload)) as RuntimeStatus;
      const createdTask =
        created.tasks.find((task) => !existingTaskIDs.has(task.id)) ??
        created.tasks.find((task) => task.kind === "workspace.apply_patch.rollback" && task.input.includes(writeExecutionId)) ??
        null;

      if (!createdTask) {
        setRuntimeStatus(created);
        setStatusMessage("已创建 rollback task，请检查当前线程状态");
        setLastCheckedAt(formatTime(created.updatedAt));
        return;
      }

      if (createdTask.status === "needs_approval" || createdTask.approvalStatus === "pending") {
        setRuntimeStatus(created);
        setStatusMessage("rollback 已进入待审批状态");
        setLastCheckedAt(formatTime(created.updatedAt));
        return;
      }

      const next = (await AdvanceTask(createdTask.id)) as RuntimeStatus;
      setRuntimeStatus(next);
      setStatusMessage("rollback 已触发执行");
      setLastCheckedAt(formatTime(next.updatedAt));
    } catch (err) {
      setError(err instanceof Error ? err.message : "发起 rollback 失败");
    } finally {
      setLoading(false);
    }
  }

  const handleOpenPreview = async () => {
    const draftValue = addressInputRef.current?.value || addressDraftRef.current;
    const openURL =
      activeThread && isThreadPreviewURL(draftValue) ? withThreadContext(draftValue, activeThread.id, activeThread.name) : draftValue;
    const next = await BrowserOpen(openURL);
    if (activeThread && isThreadPreviewURL(openURL)) {
      previewOwnerThreadID.current = activeThread.id;
    }
    const nextActiveTab = next.tabs.find((tab) => tab.id === next.activeTabId) ?? next.tabs[0];
    if (activeThread && nextActiveTab?.url && isThreadPreviewURL(nextActiveTab.url)) {
      threadPreviewMemory.current[activeThread.id] = nextActiveTab.url;
      setAddressDraft(nextActiveTab.url);
      addressDraftRef.current = nextActiveTab.url;
      lastSyncedPreviewURL.current = nextActiveTab.url;
    }
    applyBrowserState(next);
  };

  const handleNavigatePreview = async () => {
    const draftValue = addressInputRef.current?.value || addressDraftRef.current;
    if (activeThread) {
      await navigatePreviewForThread(activeThread, draftValue);
      return;
    }

    const next = await BrowserNavigate(activeBrowserTab?.id || "", draftValue);
    setLastSubmittedPreviewURL(draftValue);
    applyBrowserState(next);
    setAddressDraft(draftValue);
    addressDraftRef.current = draftValue;
    lastSyncedPreviewURL.current = draftValue;
  };

  const handleBrowserClick = async () => {
    if (!activeBrowserTab) {
      return;
    }
    applyBrowserState(await BrowserClick(activeBrowserTab.id, browserSelectorDraft));
  };

  const handleBrowserType = async () => {
    if (!activeBrowserTab) {
      return;
    }
    applyBrowserState(await BrowserType(activeBrowserTab.id, browserSelectorDraft, browserTextDraft));
  };

  const handleBrowserExtract = async () => {
    if (!activeBrowserTab) {
      return;
    }
    applyBrowserState(await BrowserExtract(activeBrowserTab.id, browserSelectorDraft));
  };

  const handleBrowserScreenshot = async () => {
    if (!activeBrowserTab) {
      return;
    }
    applyBrowserState(await BrowserScreenshot(activeBrowserTab.id));
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
              <h1>线程工作台</h1>
            </div>
          </div>

          <div className="workspace-topbar__meta">
            <span className="chip chip--soft">Wails Shell</span>
            <span className={`chip ${runtimeStatus?.runtimeSource === "local-fallback" ? "chip--warning" : "chip--soft"}`}>{runtimeLaneLabel}</span>
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
                  <span className="mini-chip">{`工作区 ${runtimeStatus?.workspaceId || "加载中"}`}</span>
                  <span className="mini-chip">{`线程 ${runtimeStatus?.threadCount || 0}`}</span>
                </div>
                {workspaceSummary?.summary ? <p className="sidebar-note sidebar-note--strong">{workspaceSummary.summary}</p> : null}
              </section>

              <section className="left-panel">
                <div className="section-header">
                  <div>
                    <p className="section-title">线程导航</p>
                    <h3>工作区线程</h3>
                  </div>
                  <button className="secondary-action" onClick={handleCreateThread} disabled={loading} type="button">
                    新建线程
                  </button>
                </div>

                <div className="thread-stack">
                  {(runtimeStatus?.threads ?? []).length === 0 ? (
                    <div className="thread-empty">当前还没有线程</div>
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
                          <span className={`chip chip--${thread.isActive ? "good" : "muted"}`}>{thread.isActive ? "当前" : "空闲"}</span>
                          <span className="thread-card__status">{thread.status}</span>
                        </div>
                        <strong>{thread.name}</strong>
                        <span>{thread.id}</span>
                        <p>{thread.permissionMode || "ask-user"}</p>
                      </button>
                    ))
                  )}
                </div>
                <div className="summary-list summary-list--runtime">
                  <article className="summary-row">
                    <p>最新审批快照</p>
                    <strong>{approvalViewModels[0] ? summarizeApprovalViewModel(approvalViewModels[0]) : "当前还没有审批快照。"}</strong>
                  </article>
                  <article className="summary-row">
                    <p>最新写执行快照</p>
                    <strong>{latestWriteExecution ? summarizeWriteExecution(latestWriteExecution) : "当前还没有写执行快照。"}</strong>
                  </article>
                </div>
              </section>

              <section className="left-panel left-panel--muted">
                <p className="section-title">能力摘要</p>
                <p className="sidebar-note">右侧预览区首轮只服务本地联调，不扩成公网通用浏览器。</p>
                <p className="sidebar-note">当前活动线程会驱动本地预览 URL 自动携带 thread 参数，并记住各自的预览地址。</p>
                <p className="sidebar-note sidebar-note--strong">可执行类型：{executableKinds.join(" / ")}</p>
              </section>
            </aside>

            <section className="center-stage">
              <section className="stage-header card">
                <div>
                  <p className="section-title">当前线程</p>
                  <h2>{activeThread?.name || "等待活动线程"}</h2>
                  <p className="stage-header__lead">
                    {activeThreadWorkflow?.summary || (activeThread ? "这里展示当前线程的消息流、任务进度、工具调用和实时事件。" : "请先创建或激活一个线程。")}
                  </p>
                </div>
                <div className="stage-header__meta">
                  <span className="chip chip--soft">{activeThread?.permissionMode || "ask-user"}</span>
                  <span className={`chip ${runtimeStatus?.runtimeSource === "local-fallback" ? "chip--warning" : "chip--soft"}`}>{runtimeLaneLabel}</span>
                  <span className={`chip ${runtimeStatus?.supportsSSE ? "chip--good" : "chip--warning"}`}>{refreshModeLabel}</span>
                  {activeThreadWorkflow?.latestTaskId ? <span className="chip chip--soft">{`最新任务 ${activeThreadWorkflow.latestTaskId}`}</span> : null}
                </div>
              </section>

              <section className="card">
                <div className="section-header">
                  <div>
                    <p className="section-title">工作流概览</p>
                    <h3>工作区与活动线程的显式运行摘要</h3>
                  </div>
                </div>
                <div className="results-grid">
                  {workflowHighlights.map((item) => (
                    <InfoCard key={item.label} label={item.label} value={item.value} detail={item.detail} />
                  ))}
                </div>
                <div className="flow-list">
                  <article className="flow-item flow-item--neutral">
                    <div className="flow-item__header">
                      <span className="mini-chip">工作区</span>
                      <span className="flow-item__meta">{workspaceSummary?.activeThreadName || "暂无活动线程"}</span>
                    </div>
                    <h4>{workspaceSummary?.id || runtimeStatus?.workspaceId || "工作区"}</h4>
                    <p>{workspaceSummary?.summary || runtimeStatus?.projectRoot || "暂无工作区摘要。"}</p>
                  </article>
                  <article className="flow-item flow-item--neutral">
                    <div className="flow-item__header">
                      <span className="mini-chip">线程</span>
                      <span className="flow-item__meta">{activeThreadWorkflow?.status || activeThread?.status || "空闲"}</span>
                    </div>
                    <h4>{activeThreadWorkflow?.name || activeThread?.name || "暂无活动线程"}</h4>
                    <p>{activeThreadWorkflow?.summary || "激活一个线程后，可以查看任务、审批和写执行关系。"}</p>
                  </article>
                  <article className={`flow-item flow-item--${latestAgentTask ? getTaskTone(latestAgentTask) : "neutral"} ${latestAgentTask ? "flow-item--agent-parent" : ""}`} data-testid="latest-agent-overview-card">
                    <div className="flow-item__header">
                      <span className="mini-chip">Agent 闭环</span>
                      <span className="flow-item__meta">{latestAgentTask ? formatTaskStatus(latestAgentTask.status) : "就绪"}</span>
                    </div>
                    <h4>{latestAgentTask ? formatTaskHeadline(latestAgentTask) : "默认目标工作流"}</h4>
                    <p>{formatAgentWorkflowOverview(latestAgentTask, taskMap)}</p>
                  </article>
                </div>
              </section>

              <section className="composer card">
                <div className="section-header">
                  <div>
                    <p className="section-title">任务输入</p>
                  <h3>给活动线程追加一条真实任务</h3>
                  </div>
                  <span className="mini-chip">{runtimeStatus?.activeThreadId ? "活动线程已就绪" : "暂无活动线程"}</span>
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
                    <span className="field__label">类型</span>
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
                  <span className="field__label">输入</span>
                    <textarea
                      data-testid="task-input-textarea"
                      value={draft.input}
                      onChange={(event) => setDraft((current) => ({ ...current, input: event.target.value }))}
                      placeholder='默认推荐直接输入 agent.run 目标，例如“更新 README 并回复结果”；也支持 JSON，如 {"goal":"请先筛出 *.go，再查 TODO 行号"}、{"path":"go.mod"}、{"paths":["README.md","go.mod"]}'
                      rows={4}
                    />
                  </label>

                <div className="composer-actions">
                  <button className="primary-action" data-testid="create-task-button" onClick={handleCreateTask} disabled={loading || !runtimeStatus?.activeThreadId} type="button">
                    创建 task
                  </button>
                  <span className="composer-actions__hint">{error || statusMessage}</span>
                </div>
              </section>

              <section className="flow-panel card">
                <div className="section-header">
                  <div>
                    <p className="section-title">消息流</p>
                    <h3>任务、工具调用、消息、事件和写执行审计按时间线展示</h3>
                  </div>
                  <span className="mini-chip">{contextSummary}</span>
                </div>

                {pendingApprovals.length > 0 ? (
                  <div className="approval-panel" data-testid="approval-panel">
                    {pendingApprovals.map((approval) => (
                      <article className="flow-item flow-item--warning" key={approval.id}>
                        <div className="flow-item__header">
                          <span className="mini-chip">{`审批 / ${approval.approvalStatusLabel}`}</span>
                          <span className="flow-item__meta">{formatTime(approval.updatedAt)}</span>
                        </div>
                        <h4>{approval.title}</h4>
                        <ApprovalDetails approval={approval} />
                        <ApprovalActions taskId={approval.taskId} loading={loading} onApprove={handleApproveTask} onReject={handleRejectTask} />
                      </article>
                    ))}
                  </div>
                ) : null}

                {waitingTasks.length > 0 ? (
                  <div className="approval-panel" data-testid="waiting-task-panel">
                    {waitingTasks.slice(0, 4).map((task) => (
                      <article className={`flow-item flow-item--${getTaskTone(task)} ${isAgentParentTask(task) ? "flow-item--agent-parent" : task.parentTaskId ? "flow-item--agent-child" : ""}`} key={task.id}>
                        <div className="flow-item__header">
                          <span className="mini-chip">{formatTaskWaitingStatusLabel(task)}</span>
                          <span className="flow-item__meta">{formatTime(task.updatedAt || task.createdAt)}</span>
                        </div>
                        <h4>{formatTaskHeadline(task, task.parentTaskId ? taskMap.get(task.parentTaskId) : undefined)}</h4>
                        <p>{formatTaskDisplaySummary(task, task.parentTaskId ? taskMap.get(task.parentTaskId) : undefined)}</p>
                        <p>
                          {formatTaskLinkSummary(task)}
                        </p>
                      </article>
                    ))}
                  </div>
                ) : null}

                {childTasks.length > 0 ? (
                  <div className="approval-panel" data-testid="child-task-panel">
                    {childTasks.slice(0, 4).map((task) => (
                      <article className={`flow-item flow-item--${getTaskTone(task)} flow-item--agent-child`} key={task.id}>
                        <div className="flow-item__header">
                          <span className="mini-chip">{formatTaskRoleLabel(task)}</span>
                          <span className="flow-item__meta">{formatTaskStatus(task.status)}</span>
                        </div>
                        <h4>{formatTaskHeadline(task, task.parentTaskId ? taskMap.get(task.parentTaskId) : undefined)}</h4>
                        <p>{formatTaskDisplaySummary(task, task.parentTaskId ? taskMap.get(task.parentTaskId) : undefined)}</p>
                      </article>
                    ))}
                  </div>
                ) : null}

                {recentWriteExecutions.length > 0 ? (
                  <div className="write-execution-panel" data-testid="write-execution-panel">
                    {recentWriteExecutions.map((execution) => (
                      <article className="flow-item flow-item--good" key={execution.id}>
                        <div className="flow-item__header">
                          <span className="mini-chip">{`写执行 / ${execution.statusLabel}`}</span>
                          <span className="flow-item__meta">{formatTime(execution.updatedAt)}</span>
                        </div>
                        <h4>{execution.title}</h4>
                        <WriteExecutionDetails execution={execution} />
                        {latestRollbackCandidate?.id === execution.id ? (
                          <div className="flow-item__actions">
                            <button
                              className="thread-action"
                              data-testid={`rollback-latest-${execution.id}`}
                              onClick={() => void handleRollbackLatest(execution.id)}
                              disabled={loading}
                              type="button"
                            >
                              回退最近一次
                            </button>
                          </div>
                        ) : null}
                      </article>
                    ))}
                  </div>
                ) : null}

                {approvalRequiredTasks.length > 0 ? (
                  <div className="approval-panel" data-testid="approval-required-task-panel">
                    {approvalRequiredTasks.slice(0, 4).map((task) => (
                      <article className="flow-item flow-item--warning" key={task.id}>
                        <div className="flow-item__header">
                          <span className="mini-chip">任务关系</span>
                          <span className="flow-item__meta">{task.kind}</span>
                        </div>
                        <h4>{task.title}</h4>
                        <p>{task.approvalSummary || task.workflowLabel || "已检测到审批关系。"}</p>
                        <p>{task.writeExecutionSummary || (task.writeExecutionId ? `写执行 ${task.writeExecutionId}` : "暂无写执行记录")}</p>
                      </article>
                    ))}
                  </div>
                ) : null}

                <div className="flow-list">
                  {flowItems.length === 0 ? (
                    <div className="thread-empty">当前还没有可展示的消息流内容</div>
                  ) : (
                    flowItems.map((item) => (
                      <article className={`flow-item flow-item--${item.tone}${item.className ? ` ${item.className}` : ""}`} key={item.id}>
                        <div className="flow-item__header">
                          <span className="mini-chip">{item.badge}</span>
                          <span className="flow-item__meta">{item.meta}</span>
                        </div>
                        <h4>{item.title}</h4>
                        {item.body}
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
                    <h3>当前线程的关键结果、审批状态与写执行摘要</h3>
                  </div>
                  <span className="mini-chip">{activeThread?.id || "暂无线程"}</span>
                </div>

                <div className={`fallback-note ${runtimeStatus?.runtimeSource === "local-fallback" ? "fallback-note--warning" : ""}`}>
                  <strong>{runtimeTrustLabel}</strong>
                  <span>{fallbackNote}</span>
                </div>

                <div className="context-grid">
                  <ResultCard
                    label="最新任务"
                    title={latestTask ? formatLatestTaskTitle(latestTask, latestTask.parentTaskId ? taskMap.get(latestTask.parentTaskId) : undefined) : "暂无任务"}
                    body={latestTask ? formatLatestTaskBody(latestTask, latestTask.parentTaskId ? taskMap.get(latestTask.parentTaskId) : undefined) : "暂无结果"}
                    className={latestTask ? getTaskCardClassName(latestTask) : undefined}
                    testId="latest-task-card"
                  />
                  <ResultCard
                    label="最新 Agent"
                    title={latestAgentTask ? formatLatestTaskTitle(latestAgentTask) : "暂无 Agent 父任务"}
                    body={latestAgentTask ? formatLatestTaskBody(latestAgentTask) : "当前线程还没有 agent.run 父任务"}
                    className={latestAgentTask ? getTaskCardClassName(latestAgentTask) : undefined}
                    testId="latest-agent-card"
                  />
                  <ResultCard
                    label="最新子任务"
                    title={latestChildTask ? formatLatestTaskTitle(latestChildTask, latestChildTask.parentTaskId ? taskMap.get(latestChildTask.parentTaskId) : undefined) : "暂无 Agent 子任务"}
                    body={latestChildTask ? formatLatestTaskBody(latestChildTask, latestChildTask.parentTaskId ? taskMap.get(latestChildTask.parentTaskId) : undefined) : "当前线程还没有派生子任务"}
                    className={latestChildTask ? getTaskCardClassName(latestChildTask) : undefined}
                    testId="latest-child-task-card"
                  />
                  <ResultCard
                    label="最新工具调用"
                    title={latestToolCall ? `${latestToolCall.toolId} / ${formatToolCallStatus(latestToolCall.status)}` : "暂无工具调用"}
                    body={latestToolCall?.summary || "暂无摘要"}
                    testId="latest-toolcall-card"
                  />
                  <ResultCard
                    label="最新审批"
                    title={latestApproval ? `${latestApproval.toolKind} / ${latestApproval.approvalStatusLabel}` : "暂无审批"}
                    body={latestApproval ? `${latestApproval.summary} / ${latestApproval.patchStatsLabel}` : "当前线程还没有最近审批摘要"}
                    testId="latest-approval-card"
                  />
                  <ResultCard label="最新消息" title={latestMessage ? formatMessageRole(latestMessage.role) : "暂无消息"} body={latestMessage?.content || "暂无消息"} testId="latest-message-card" />
                  <ResultCard label="最新事件" title={latestEvent ? formatEventType(latestEvent.type) : "暂无事件"} body={latestEvent?.message || "暂无事件"} testId="latest-event-card" />
                  <ResultCard
                    label="最近写执行"
                    title={latestWriteExecution ? `${latestWriteExecution.toolKind} / ${formatWriteExecutionStatus(latestWriteExecution.status)}` : "暂无写执行"}
                    body={latestWriteExecution ? summarizeWriteExecution(latestWriteExecution) : "尚未记录 patch 写执行审计"}
                    testId="latest-write-execution-card"
                  />
                  <InfoCard label="最新产物" value={latestArtifact ? latestArtifact.kind : "暂无产物"} detail={latestArtifact?.path || "暂无产物"} />
                  <InfoCard
                    label="运行链路"
                    value={runtimeLaneLabel}
                    detail={`${runtimeLaneDetail} / ${bridgeResult?.message || runtimeStatus?.runtimeMessage || "桥接状态待检查"}`}
                  />
                  <InfoCard label="刷新方式" value={refreshModeLabel} detail={`${refreshModeDetail} / ${streamState}`} />
                  <InfoCard
                    label="审批状态"
                    value={pendingApprovals.length > 0 ? `${pendingApprovals.length} 条待审批` : "无待审批"}
                    detail={pendingApprovals[0] ? `${pendingApprovals[0].summary} / ${pendingApprovals[0].patchStatsLabel}` : "当前没有待审批写任务"}
                  />
                  <InfoCard
                    label="写执行历史"
                    value={writeExecutionViewModels.length > 0 ? `${writeExecutionViewModels.length} 条记录` : "暂无记录"}
                    detail={latestWriteExecution ? `${latestWriteExecution.patchSummary} / ${latestWriteExecution.afterSummary}` : "通过审批并完成 patch 后，会在这里显示最近一次审计摘要"}
                  />
                  <InfoCard
                    label="Skill 治理"
                    value={skillGovernanceRollup.implementedCount > 0 ? `${skillGovernanceRollup.implementedCount} 个已发现` : "暂无技能清单"}
                    detail={skillGovernanceRollup.summary}
                  />
                  <ResultCard
                    label="Skill 分组"
                    title={skillGovernanceViewModels[0] ? `${skillGovernanceViewModels[0].group} / ${skillGovernanceViewModels.length} 组` : "暂无分组"}
                    body={
                      skillGovernanceViewModels.length > 0
                        ? skillGovernanceViewModels
                            .map((item) => `${item.summary}${item.skillIDs.length > 0 ? ` / ${item.skillIDs.slice(0, 3).join(", ")}` : ""}`)
                            .join(" | ")
                        : "当前运行态还没有可展示的 Skill 治理摘要"
                    }
                  />
                  <InfoCard label="Provider" value={preferredProvider?.kind || "暂无 Provider"} detail={providerSummary} />
                  <InfoCard label="状态存储" value={runtimeStatus?.stateStore || "sqlite"} detail={runtimeStatus?.statePath || "项目本地状态存储"} />
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
                      <button className="browser-tab__label" onClick={() => void BrowserActivateTab(tab.id).then(applyBrowserState)} type="button">
                        <span>{tab.title}</span>
                      </button>
                      <button
                        className="browser-tab__close"
                        onClick={(event) => {
                          event.stopPropagation();
                          void BrowserCloseTab(tab.id).then(applyBrowserState);
                        }}
                        type="button"
                      >
                        关
                      </button>
                    </div>
                  ))}
                  <button className="browser-add-tab" onClick={() => void handleOpenPreview()} type="button">
                    +
                  </button>
                </div>

                <div className="browser-toolbar">
                  <div className="browser-toolbar__actions">
                    <button className="browser-nav" onClick={() => activeBrowserTab && void BrowserBack(activeBrowserTab.id).then(applyBrowserState)} type="button">
                      后退
                    </button>
                    <button className="browser-nav" onClick={() => activeBrowserTab && void BrowserForward(activeBrowserTab.id).then(applyBrowserState)} type="button">
                      前进
                    </button>
                    <button className="browser-nav" onClick={() => activeBrowserTab && void BrowserReload(activeBrowserTab.id).then(applyBrowserState)} type="button">
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
                  <span className="mini-chip">{activeBrowserTab?.status || "就绪"}</span>
                </div>

                <div className="left-panel left-panel--muted">
                  <p className="section-title">Browser Actions</p>
                  <div className="browser-toolbar__actions">
                    <input
                      className="browser-address"
                      value={browserSelectorDraft}
                      onChange={(event) => setBrowserSelectorDraft(event.target.value)}
                      placeholder="[data-testid='target'] / #id / .class"
                    />
                    <input
                      className="browser-address"
                      value={browserTextDraft}
                      onChange={(event) => setBrowserTextDraft(event.target.value)}
                      placeholder="type text"
                    />
                  </div>
                  <div className="browser-toolbar__actions">
                    <button className="browser-nav" onClick={() => void handleBrowserClick()} type="button">
                      Click
                    </button>
                    <button className="browser-nav" onClick={() => void handleBrowserType()} type="button">
                      Type
                    </button>
                    <button className="browser-nav" onClick={() => void handleBrowserExtract()} type="button">
                      Extract
                    </button>
                    <button className="browser-nav" onClick={() => void handleBrowserScreenshot()} type="button">
                      Shot
                    </button>
                  </div>
                  <p className="sidebar-note">{`Latest: ${browserLatestSummary}`}</p>
                  {browserLatestError ? <p className="sidebar-note">{`Error: ${browserLatestError}`}</p> : null}
                  {browserLatestExtract ? <p className="sidebar-note">{`Extract: ${browserLatestExtract}`}</p> : null}
                  {browserLatestArtifactPath ? <p className="sidebar-note">{`Screenshot: ${browserLatestArtifactPath}`}</p> : null}
                </div>

                {showPreviewDebug ? (
                  <div className="left-panel left-panel--muted" data-testid="browser-debug-panel">
                    <p className="section-title">预览调试</p>
                    <p className="sidebar-note">{`提交地址：${lastSubmittedPreviewURL || "无"}`}</p>
                    <p className="sidebar-note">{`标签地址：${activeBrowserTab?.url || "无"}`}</p>
                    <p className="sidebar-note">{`归属线程：${previewOwnerThreadID.current || "无"}`}</p>
                    <p className="sidebar-note">{`线程记忆：${activeThreadRememberedURL || "无"}`}</p>
                  </div>
                ) : null}

                <div className="browser-surface">
                  {activeBrowserTab ? (
                    <iframe className="browser-frame" data-testid="browser-preview-frame" src={activeBrowserTab.url} title={activeBrowserTab.title} />
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

function groupTasksForDisplay(tasks: ExtendedRuntimeTask[]): TaskGroup[] {
  const waiting = tasks.filter((task) => task.waitingStatus || task.status === "waiting_for_task" || task.status === "waiting_for_approval");
  const approvals = tasks.filter((task) => task.status === "needs_approval" || task.approvalStatus === "pending");
  const children = tasks.filter((task) => task.parentTaskId);
  const finished = tasks.filter((task) => task.status === "completed" || task.status === "failed");

  const groups: TaskGroup[] = [
    { title: "Waiting", tone: "warning", tasks: waiting },
    { title: "Approval", tone: "warning", tasks: approvals },
    { title: "Child", tone: "neutral", tasks: children },
    { title: "Finished", tone: "good", tasks: finished },
  ];

  return groups.filter((group) => group.tasks.length > 0);
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

function createApprovalViewModel(approval: RuntimeApproval, task?: RuntimeTask): ApprovalViewModel {
  const parsedPatch = parsePatchPreview(task?.input || "");
  return {
    id: approval.id,
    taskId: approval.taskId,
    title: task?.title || approval.toolKind,
    toolKind: approval.toolKind,
    taskKind: task?.kind || approval.toolKind,
    taskStatus: task?.status || "",
    taskStatusLabel: formatTaskStatus(task?.status || ""),
    approvalStatus: task?.approvalStatus || approval.status,
    approvalStatusLabel: formatApprovalStatus(task?.approvalStatus || approval.status),
    summary: approval.summary || task?.resultSummary || "等待审批",
    targetPaths: approval.targetPaths || [],
    patchPreview: parsedPatch.preview,
    patchStats: parsedPatch.stats,
    patchStatsLabel: formatPatchStats(parsedPatch.stats),
    updatedAt: approval.updatedAt || task?.updatedAt || approval.createdAt,
  };
}

function createWriteExecutionViewModel(
  execution: RuntimeWriteExecution,
  task?: RuntimeTask,
  approval?: RuntimeApproval,
): WriteExecutionViewModel {
  return {
    id: execution.id,
    taskId: execution.taskId,
    approvalId: execution.approvalId || approval?.id || "",
    title: task?.title || execution.toolKind,
    toolKind: execution.toolKind,
    operation: execution.operation || "apply",
    relatedExecutionId: execution.relatedExecutionId || "",
    status: execution.status,
    statusLabel: formatWriteExecutionStatus(execution.status),
    targetPaths: execution.targetPaths || approval?.targetPaths || [],
    patchSummary: execution.patchSummary || task?.resultSummary || "暂无补丁摘要",
    beforeSummary: execution.beforeSummary || "暂无执行前摘要",
    afterSummary: execution.afterSummary || "暂无执行后摘要",
    resultSummary: execution.resultSummary || task?.resultSummary || "暂无执行结果",
    updatedAt: execution.updatedAt || task?.updatedAt || execution.createdAt,
  };
}

function parsePatchPreview(rawInput: string) {
  const trimmed = rawInput.trim();
  const fallback = {
    preview: trimmed || "当前任务没有可展示的补丁预览。",
    stats: createPatchStats(trimmed),
  };

  if (!trimmed) {
    return {
      preview: "当前任务没有可展示的补丁预览。",
      stats: { added: 0, removed: 0, hasPreview: false },
    };
  }

  try {
    const parsed = JSON.parse(trimmed) as { patch?: unknown; input?: unknown };
    if (typeof parsed.patch === "string" && parsed.patch.trim()) {
      return {
        preview: parsed.patch.trim(),
        stats: createPatchStats(parsed.patch),
      };
    }
    if (typeof parsed.input === "string" && parsed.input.trim()) {
      return {
        preview: parsed.input.trim(),
        stats: createPatchStats(parsed.input),
      };
    }
  } catch {
    return fallback;
  }

  return fallback;
}

function createPatchStats(patchPreview: string): PatchStats {
  if (!patchPreview.trim()) {
    return { added: 0, removed: 0, hasPreview: false };
  }

  let added = 0;
  let removed = 0;
  for (const line of patchPreview.split(/\r?\n/)) {
    if (line.startsWith("+++") || line.startsWith("---")) {
      continue;
    }
    if (line.startsWith("+")) {
      added += 1;
      continue;
    }
    if (line.startsWith("-")) {
      removed += 1;
    }
  }

  return { added, removed, hasPreview: true };
}

function formatPatchStats(stats: PatchStats) {
  if (!stats.hasPreview) {
    return "无补丁预览";
  }
  return `+${stats.added} / -${stats.removed}`;
}

function isAgentParentTask(task: ExtendedRuntimeTask) {
  return task.kind === "agent.run" && !task.parentTaskId;
}

function summarizeStateDetail(value: string, fallback: string, maxLength = 160) {
  const compact = value.replace(/\s+/g, " ").trim();
  if (!compact) {
    return fallback;
  }
  if (compact.length <= maxLength) {
    return compact;
  }
  return `${compact.slice(0, maxLength)}...`;
}

function getTaskStateSignal(task: ExtendedRuntimeTask): TaskStateSignal | null {
  const resultSummary = (task.resultSummary || "").trim();
  const waitingStatus = (task.waitingStatus || task.status || "").trim();
  const isAgentParent = isAgentParentTask(task);
  const isChildTask = Boolean(task.parentTaskId);

  if (resultSummary.startsWith("agent recovery failed:") || resultSummary.includes("agent recovery failed")) {
    const detail = resultSummary.startsWith("agent recovery failed:")
      ? resultSummary.slice("agent recovery failed:".length).trim()
      : resultSummary;
    return {
      tone: "warning",
      summary: `状态：task.recovered_as_failed / ${summarizeStateDetail(detail, "子任务状态恢复失败")}`,
      consumesResultSummary: true,
    };
  }
  if (resultSummary.includes("interrupted by runtime restart")) {
    return {
      tone: "warning",
      summary: "状态：task.recovered_as_failed / 运行时重启中断了当前 Agent",
      consumesResultSummary: true,
    };
  }
  if (resultSummary.startsWith("agent failed: child approval rejected:")) {
    return {
      tone: "warning",
      summary: `状态：子任务审批已拒绝 / ${summarizeStateDetail(resultSummary.slice("agent failed: child approval rejected:".length).trim(), "某个子任务审批被拒绝")}`,
      consumesResultSummary: true,
    };
  }
  if (resultSummary.startsWith("agent child task failed:")) {
    return {
      tone: "warning",
      summary: `状态：子任务失败 / ${summarizeStateDetail(resultSummary.slice("agent child task failed:".length).trim(), "最新子任务执行失败")}`,
      consumesResultSummary: true,
    };
  }
  if (waitingStatus === "waiting_for_approval") {
    const detail =
      task.approvalStatus === "pending"
        ? "approvalStatus=pending，等待批准后自动续跑"
        : task.approvalId
          ? `审批 ${task.approvalId} 正在阻塞续跑`
          : isAgentParent
            ? "Agent 会在批准后自动续跑"
            : isChildTask
              ? "子任务会在批准后继续执行"
              : "任务会在批准后继续执行";
    return {
      tone: "warning",
      summary: `状态：waiting_for_approval / ${detail}`,
    };
  }
  if (waitingStatus === "waiting_for_task") {
    const detail = task.waitingTaskId
      ? `等待子任务 ${task.waitingTaskId} 完成`
      : task.latestChildTaskId
        ? `等待最新子任务 ${task.latestChildTaskId} 完成`
        : isAgentParent
          ? "Agent 会在子任务完成后自动续跑"
          : "任务会在关联子任务完成后继续";
    return {
      tone: "warning",
      summary: `状态：waiting_for_task / ${detail}`,
    };
  }
  if (isChildTask && task.approvalStatus === "rejected") {
    return {
      tone: "warning",
      summary: `状态：子任务审批已拒绝 / ${summarizeStateDetail(resultSummary, "approvalStatus=rejected")}`,
      consumesResultSummary: Boolean(resultSummary),
    };
  }
  if (isChildTask && task.status === "failed") {
    return {
      tone: "warning",
      summary: `状态：子任务失败 / ${summarizeStateDetail(resultSummary, "子任务执行失败")}`,
      consumesResultSummary: Boolean(resultSummary),
    };
  }
  if (task.status === "failed") {
    return {
      tone: "warning",
      summary: `状态：failed / ${summarizeStateDetail(resultSummary, "任务执行失败")}`,
      consumesResultSummary: Boolean(resultSummary),
    };
  }
  return null;
}

function getTaskTone(task: ExtendedRuntimeTask): FlowItem["tone"] {
  const signal = getTaskStateSignal(task);
  if (signal) {
    return signal.tone;
  }
  if (task.status === "completed") {
    return "good";
  }
  if (task.status === "needs_approval") {
    return "warning";
  }
  return "neutral";
}

function formatTaskRoleLabel(task: ExtendedRuntimeTask) {
  if (isAgentParentTask(task)) {
    return "Agent 父任务";
  }
  if (task.parentTaskId) {
    return "Agent 子任务";
  }
  return "任务";
}

function formatTaskWaitingStatusLabel(task: ExtendedRuntimeTask) {
  if (task.waitingStatus) {
    return formatWaitingStatus(task.waitingStatus);
  }
  if (task.status === "waiting_for_task" || task.status === "waiting_for_approval") {
    return formatWaitingStatus(task.status);
  }
  return "";
}

function formatTaskHeadline(task: ExtendedRuntimeTask, parentTask?: ExtendedRuntimeTask) {
  const base = task.title || task.kind || "未命名任务";
  if (isAgentParentTask(task)) {
    return `${base} / agent.run`;
  }
  if (parentTask) {
    return `${base} / ${task.kind || "任务"} / 父任务 ${parentTask.title || parentTask.id}`;
  }
  return `${base} / ${task.kind || "任务"}`;
}

function formatTaskLinkSummary(task: ExtendedRuntimeTask) {
  const parts: string[] = [`任务 ${task.id}`];
  if (task.parentTaskId) {
    parts.push(`父任务 ${task.parentTaskId}`);
  }
  if (task.waitingTaskId) {
    parts.push(`关联子任务 ${task.waitingTaskId}`);
  }
  if (task.approvalId) {
    parts.push(`审批 ${task.approvalId}`);
  }
  if (task.latestChildTaskId) {
    parts.push(`最新子任务 ${task.latestChildTaskId}`);
  }
  return parts.join(" / ") || "当前没有额外关联信息。";
}

function formatTaskStatus(status: string) {
  switch (status) {
    case "queued":
      return "排队中";
    case "running":
      return "执行中";
    case "waiting_for_task":
      return "等待子任务";
    case "waiting_for_approval":
      return "等待审批后续跑";
    case "completed":
      return "已完成";
    case "failed":
      return "已失败";
    case "needs_approval":
      return "待审批";
    default:
      return status || "未知状态";
  }
}

function formatWaitingStatus(status: string) {
  switch (status) {
    case "waiting_for_task":
      return "等待子任务完成";
    case "waiting_for_approval":
      return "等待审批后续跑";
    default:
      return status || "等待中";
  }
}

function formatAgentMeta(task: RuntimeTask) {
  const parts: string[] = [];
  if (task.agentStep > 0 && task.agentMaxSteps > 0) {
    parts.push(`步骤 ${task.agentStep}/${task.agentMaxSteps}`);
  }
  if (task.agentPlanMode) {
    parts.push(`模式 ${task.agentPlanMode}`);
  }
  if (task.latestChildTaskId) {
    parts.push(`最新子任务 ${task.latestChildTaskId}`);
  }
  if (task.waitingStatus) {
    parts.push(formatWaitingStatus(task.waitingStatus));
  }
  return parts.join(" / ");
}

function formatAgentStateSummary(task: RuntimeTask) {
  const signal = getTaskStateSignal(task as ExtendedRuntimeTask);
  if (signal) {
    return signal.summary;
  }
  const waitingStatus = task.waitingStatus || task.status;
  if (waitingStatus === "waiting_for_approval") {
    return "状态：等待审批通过后自动续跑";
  }
  if (waitingStatus === "waiting_for_task") {
    return "状态：等待子任务完成后自动续跑";
  }
  if (task.status === "completed") {
    return "状态：Agent 已完成本轮闭环";
  }

  const resultSummary = (task.resultSummary || "").trim();
  if (resultSummary.startsWith("agent recovery failed:")) {
    return `状态：恢复失败（task.recovered_as_failed） / ${resultSummary.slice("agent recovery failed:".length).trim() || "子任务状态不一致"}`;
  }
  if (resultSummary.startsWith("agent failed: child approval rejected:")) {
    return `状态：审批已拒绝并中断续跑 / ${resultSummary.slice("agent failed: child approval rejected:".length).trim() || "写入审批被拒绝"}`;
  }
  if (resultSummary.includes("interrupted by runtime restart")) {
    return "状态：运行时重启中断，已按 task.recovered_as_failed 收口";
  }
  if (resultSummary.startsWith("agent child task failed:")) {
    return `状态：子任务失败 / ${resultSummary.slice("agent child task failed:".length).trim() || "请检查最近子任务结果"}`;
  }
  if (resultSummary.startsWith("agent failed: exceeded maxSteps=")) {
    return `状态：超过步数上限 / ${resultSummary.slice("agent failed:".length).trim()}`;
  }
  if (resultSummary.startsWith("agent action parse error:")) {
    return `状态：动作解析失败 / ${resultSummary.slice("agent action parse error:".length).trim() || "模型输出不符合最小 JSON 协议"}`;
  }
  if (resultSummary.startsWith("provider error:")) {
    return `状态：Provider 调用失败 / ${resultSummary.slice("provider error:".length).trim() || "请检查模型或网络状态"}`;
  }
  if (resultSummary.startsWith("agent action \"") && resultSummary.includes("\" is not supported")) {
    return `状态：动作不受支持 / ${resultSummary}`;
  }
  if (task.status === "failed") {
    return "状态：Agent 已失败";
  }
  if (task.status === "running") {
    return "状态：Agent 正在执行";
  }
  if (task.status === "queued") {
    return "状态：Agent 已排队，等待开始";
  }
  return "";
}

function formatAgentDetails(task: RuntimeTask) {
  const parts: string[] = [];
  const signal = getTaskStateSignal(task as ExtendedRuntimeTask);
  const stateSummary = signal?.summary || formatAgentStateSummary(task);
  if (stateSummary) {
    parts.push(stateSummary);
  }
  if (task.agentPlanSummary) {
    parts.push(`计划：${task.agentPlanSummary}`);
  }
  if (task.agentPlanMode) {
    parts.push(`模式：${task.agentPlanMode}`);
  }
  if (task.agentCurrentStepTitle) {
    parts.push(`当前步骤：${task.agentCurrentStepTitle}`);
  }
  if (task.agentLastReasoning) {
    parts.push(`思路摘要：${task.agentLastReasoning}`);
  }
  if (task.resultSummary && !signal?.consumesResultSummary) {
    parts.push(`结果：${task.resultSummary}`);
  }
  return parts.join("\n");
}

function formatLatestTaskBody(task: ExtendedRuntimeTask, parentTask?: ExtendedRuntimeTask) {
  if (task.kind === "agent.run") {
    return formatAgentDetails(task) || task.resultSummary || task.input || "暂无结果";
  }
  return formatTaskDisplaySummary(task, parentTask) || task.resultSummary || task.input || "暂无结果";
}

function formatApprovalStatus(status: string) {
  switch (status) {
    case "pending":
      return "待审批";
    case "approved":
      return "已批准";
    case "rejected":
      return "已拒绝";
    case "executed":
      return "已执行";
    case "direct":
      return "直接执行";
    default:
      return status || "未知审批状态";
  }
}

function formatWriteExecutionStatus(status: string) {
  switch (status) {
    case "completed":
      return "已执行";
    case "failed":
      return "执行失败";
    case "running":
      return "执行中";
    default:
      return status || "未知写执行状态";
  }
}

function formatToolCallStatus(status: string) {
  switch (status) {
    case "queued":
      return "排队中";
    case "running":
      return "执行中";
    case "completed":
      return "已完成";
    case "failed":
      return "已失败";
    default:
      return status || "未知状态";
  }
}

function formatMessageRole(role: string) {
  switch (role) {
    case "assistant":
      return "助手";
    case "user":
      return "用户";
    case "system":
      return "系统";
    default:
      return role || "消息";
  }
}

function formatEventType(type: string) {
  return type || "事件";
}

function FlowBodyText({ text }: { text: string }) {
  return <p>{text || "暂无内容"}</p>;
}

function formatTaskDisplaySummary(task: ExtendedRuntimeTask, parentTask?: ExtendedRuntimeTask) {
  if (isAgentParentTask(task)) {
    return formatAgentDetails(task) || task.waitingSummary || task.resultSummary || task.input || "等待 Agent 执行";
  }

  const browserSummary = formatBrowserTaskSummary(task);
  if (browserSummary) {
    const parts = [
      browserSummary,
      task.workflowLabel,
      formatTaskLinkSummary(task),
      task.waitingSummary,
    ].filter(Boolean) as string[];
    return parts.join("\n");
  }

  const signal = getTaskStateSignal(task);
  const parts = [
    signal?.summary,
    task.workflowLabel,
    formatTaskLinkSummary(task),
    task.waitingSummary,
    signal?.consumesResultSummary ? "" : task.resultSummary,
  ].filter(Boolean) as string[];
  if (parts.length > 0) {
    return parts.join("\n");
  }
  if (parentTask) {
    return `${task.kind || "task"} 正在为父任务 ${parentTask.title || parentTask.id} 提供执行结果。`;
  }
  return task.input || "等待任务输入";
}

function formatLatestTaskTitle(task: ExtendedRuntimeTask, parentTask?: ExtendedRuntimeTask) {
  return `${formatTaskHeadline(task, parentTask)} / ${formatTaskStatus(task.status)}`;
}

function getTaskCardClassName(task: ExtendedRuntimeTask) {
  if (isAgentParentTask(task)) {
    return "result-card--agent-parent";
  }
  if (task.parentTaskId) {
    return "result-card--agent-child";
  }
  return "";
}

function formatAgentWorkflowOverview(task: ExtendedRuntimeTask | null, taskMap?: Map<string, ExtendedRuntimeTask>) {
  if (!task) {
    return "默认入口是 agent.run：围绕目标自动派生读取、写入、审批和最终回复。";
  }

  const parts: string[] = [];
  const stateSummary = formatAgentStateSummary(task);
  if (stateSummary) {
    parts.push(`${stateSummary}。`);
  }
  if (task.agentStep > 0 && task.agentMaxSteps > 0) {
    parts.push(`当前执行到第 ${task.agentStep}/${task.agentMaxSteps} 步。`);
  }
  const waitingLabel = formatTaskWaitingStatusLabel(task);
  if (waitingLabel) {
    parts.push(`当前状态：${waitingLabel}。`);
  }
  if (task.latestChildTaskId) {
    parts.push(`最新子任务：${task.latestChildTaskId}。`);
    const latestChildBrowserSummary = formatBrowserTaskSummary(taskMap?.get(task.latestChildTaskId));
    if (latestChildBrowserSummary) {
      parts.push(`${latestChildBrowserSummary}。`);
    }
  }
  if (task.resultSummary) {
    parts.push(task.resultSummary);
  } else if (task.agentCurrentStepTitle) {
    parts.push(`当前步骤：${task.agentCurrentStepTitle}。`);
  }
  return parts.join(" ") || "默认入口是 agent.run：围绕目标自动派生读取、写入、审批和最终回复。";
}

function formatBrowserTaskSummary(task?: ExtendedRuntimeTask | null) {
  if (!task || !task.kind.startsWith("browser.")) {
    return "";
  }
  const resultSummary = (task.resultSummary || "").trim();
  if (!resultSummary) {
    return task.kind;
  }
  if (task.kind === "browser.extract") {
    return `browser.extract / 提取结果：${resultSummary}`;
  }
  if (task.kind === "browser.screenshot") {
    const artifactPath = resultSummary.replace("browser screenshot captured:", "").trim();
    return artifactPath ? `browser.screenshot / 截图产物：${artifactPath}` : `browser.screenshot / ${resultSummary}`;
  }
  return `${task.kind} / ${resultSummary}`;
}

function ApprovalActions({
  taskId,
  loading,
  onApprove,
  onReject,
}: {
  taskId: string;
  loading: boolean;
  onApprove: (taskId: string) => Promise<void>;
  onReject: (taskId: string) => Promise<void>;
}) {
  return (
    <div className="flow-item__actions">
      <button className="thread-action" onClick={() => void onApprove(taskId)} disabled={loading} type="button">
        批准执行
      </button>
      <button className="thread-action thread-action--danger" onClick={() => void onReject(taskId)} disabled={loading} type="button">
        拒绝
      </button>
    </div>
  );
}

function ApprovalDetails({ approval }: { approval: ApprovalViewModel }) {
  return (
    <div className="approval-details">
      <div className="approval-details__meta">
        <span className="mini-chip">{approval.toolKind}</span>
        <span className="mini-chip">{approval.taskStatusLabel}</span>
        <span className="mini-chip">{approval.approvalStatusLabel}</span>
        <span className="mini-chip">{approval.patchStatsLabel}</span>
      </div>
      <p>{approval.summary}</p>
      <div className="approval-details__section">
        <span className="approval-details__label">目标路径</span>
        <p>{approval.targetPaths.length > 0 ? approval.targetPaths.join(", ") : "未提供目标路径"}</p>
      </div>
      <div className="approval-details__section">
        <span className="approval-details__label">补丁预览</span>
        <pre className="approval-details__patch">{approval.patchPreview}</pre>
      </div>
    </div>
  );
}

function WriteExecutionDetails({ execution }: { execution: WriteExecutionViewModel }) {
  return (
    <div className="approval-details">
      <div className="approval-details__meta">
        <span className="mini-chip">{execution.toolKind}</span>
        <span className="mini-chip">{execution.operation === "rollback" ? "rollback" : "apply"}</span>
        <span className="mini-chip">{execution.statusLabel}</span>
        <span className="mini-chip">{execution.approvalId ? "来自审批任务" : "直接执行"}</span>
      </div>
      {execution.relatedExecutionId ? (
        <div className="approval-details__section">
          <span className="approval-details__label">来源写执行</span>
          <p>{execution.relatedExecutionId}</p>
        </div>
      ) : null}
      <p>{execution.resultSummary}</p>
      <div className="approval-details__section">
        <span className="approval-details__label">目标路径</span>
        <p>{execution.targetPaths.length > 0 ? execution.targetPaths.join(", ") : "未提供目标路径"}</p>
      </div>
      <div className="approval-details__section">
        <span className="approval-details__label">补丁摘要</span>
        <p>{execution.patchSummary}</p>
      </div>
      <div className="approval-details__section">
        <span className="approval-details__label">执行前摘要</span>
        <p>{execution.beforeSummary}</p>
      </div>
      <div className="approval-details__section">
        <span className="approval-details__label">执行后摘要</span>
        <p>{execution.afterSummary}</p>
      </div>
    </div>
  );
}

function ResultCard({ label, title, body, className, testId }: { label: string; title: string; body: string; className?: string; testId?: string }) {
  return (
    <article className={`result-card${className ? ` ${className}` : ""}`} data-testid={testId}>
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

function summarizeWriteExecution(execution: RuntimeWriteExecution) {
  const parts = [execution.resultSummary, execution.afterSummary].filter(Boolean);
  return summarizeText(parts.join(" / "), 180);
}

function summarizeApproval(approval: RuntimeApproval) {
  const parts = [approval.summary, approval.targetPaths.join(", "), formatApprovalStatus(approval.status)].filter(Boolean);
  return summarizeText(parts.join(" / "), 180);
}

function summarizeApprovalViewModel(approval: ApprovalViewModel) {
  const parts = [approval.summary, approval.targetPaths.join(", "), approval.approvalStatusLabel, approval.patchStatsLabel].filter(Boolean);
  return summarizeText(parts.join(" / "), 180);
}

function summarizeText(value: string, maxLength: number) {
  const compact = value.replace(/\s+/g, " ").trim();
  if (!compact) return "空内容";
  if (compact.length <= maxLength) return compact;
  return `${compact.slice(0, maxLength)}...`;
}

function normalizeTaskInput(kind: string, rawInput: string) {
  const trimmed = rawInput.trim();
  if (kind === "agent.run") {
    if (!trimmed) {
      return JSON.stringify({ goal: "" });
    }
    if (trimmed.startsWith("{") && trimmed.endsWith("}")) {
      try {
        const parsed = JSON.parse(trimmed) as Record<string, unknown>;
        return JSON.stringify({
          goal: typeof parsed.goal === "string" ? parsed.goal : trimmed,
          provider: typeof parsed.provider === "string" ? parsed.provider : undefined,
          model: typeof parsed.model === "string" ? parsed.model : undefined,
          maxSteps: typeof parsed.maxSteps === "number" ? parsed.maxSteps : undefined,
          maxOutputTokens: typeof parsed.maxOutputTokens === "number" ? parsed.maxOutputTokens : undefined,
        });
      } catch {
        return JSON.stringify({ goal: trimmed });
      }
    }
    return JSON.stringify({ goal: trimmed });
  }

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
  const [controlledInput, setControlledInput] = useState("browser demo text");
  const [controlledResult, setControlledResult] = useState("等待受控浏览器动作");
  const readCookieValue = (name: string) => {
    if (typeof document === "undefined") {
      return "";
    }
    const cookie = document.cookie
      .split(";")
      .map((item) => item.trim())
      .find((item) => item.startsWith(`${name}=`));
    if (!cookie) {
      return "";
    }
    return decodeURIComponent(cookie.slice(name.length + 1));
  };
  const readAuthenticatedFixtureState = () => ({
    session: readCookieValue("gc_auth"),
    profile: readCookieValue("gc_auth_profile"),
    role: readCookieValue("gc_auth_role"),
    scope: readCookieValue("gc_auth_scope"),
    transport: readCookieValue("gc_auth_transport"),
  });
  const [authenticatedFixtureState, setAuthenticatedFixtureState] = useState(() => readAuthenticatedFixtureState());
  useEffect(() => {
    if (typeof window === "undefined") {
      return undefined;
    }
    const syncCookie = () => setAuthenticatedFixtureState(readAuthenticatedFixtureState());
    syncCookie();
    const timer = window.setInterval(syncCookie, 500);
    return () => window.clearInterval(timer);
  }, []);
  const authenticatedSessionActive = authenticatedFixtureState.session === "acceptance-session";
  const authenticatedSessionLabel = authenticatedSessionActive
    ? "session=acceptance-session"
    : "session=missing";
  const authenticatedProfileLabel = authenticatedFixtureState.profile
    ? `profile=${authenticatedFixtureState.profile}`
    : "profile=missing";
  const authenticatedRoleLabel = authenticatedFixtureState.role
    ? `role=${authenticatedFixtureState.role}`
    : "role=missing";
  const authenticatedScopeLabel = authenticatedFixtureState.scope
    ? `scope=${authenticatedFixtureState.scope}`
    : "scope=missing";
  const authenticatedTransportLabel = authenticatedFixtureState.transport
    ? `transport=${authenticatedFixtureState.transport}`
    : "transport=missing";
  const authenticatedResult = authenticatedSessionActive
    ? `identity=authenticated-browser;session=acceptance-session;profile=${authenticatedFixtureState.profile || "missing"};role=${authenticatedFixtureState.role || "missing"};scope=${authenticatedFixtureState.scope || "missing"};transport=${authenticatedFixtureState.transport || "missing"}`
    : `identity=authenticated-browser;session=missing;profile=${authenticatedFixtureState.profile || "missing"};role=${authenticatedFixtureState.role || "missing"};scope=${authenticatedFixtureState.scope || "missing"};transport=${authenticatedFixtureState.transport || "missing"}`;
  const paneTitle =
    pane === "thread-one" ? "线程一预览" : pane === "thread-two" ? "线程二预览" : "本地预览";

  return (
    <main className="shell shell--embedded-preview" data-testid="embedded-preview-root">
      <section className="workspace-shell workspace-shell--embedded-preview">
        <header className="workspace-topbar workspace-topbar--embedded-preview">
          <div className="workspace-topbar__title">
            <div>
              <p className="topbar__eyebrow">Gen Code / 本地预览</p>
              <h1>{paneTitle}</h1>
            </div>
          </div>
          <div className="workspace-topbar__meta">
            <span className="chip chip--soft">{threadID || "暂无线程"}</span>
            <span className="chip chip--soft">{threadName || "嵌入预览模式"}</span>
          </div>
        </header>

        <section className="workbench workbench--focused">
          <section className="center-stage">
            <section className="stage-header card">
              <div>
                <p className="section-title">本地预览</p>
                <h2>{threadName || "未命名线程"}</h2>
                <p className="stage-header__lead">右侧浏览器已进入轻量嵌入态，避免把整套工作台递归渲染进 iframe。</p>
              </div>
              <div className="stage-header__meta">
                <span className="chip chip--soft">{`预览面板 ${pane}`}</span>
                <span className="chip chip--soft">{threadID || "暂无线程"}</span>
              </div>
            </section>

            <section className="flow-panel card">
              <div className="section-header">
                <div>
                  <p className="section-title">预览上下文</p>
                  <h3>当前线程的本地预览参数</h3>
                </div>
              </div>
              <div className="flow-list">
                <article className="flow-item flow-item--good">
                  <div className="flow-item__header">
                    <span className="mini-chip">预览</span>
                    <span className="flow-item__meta">嵌入态</span>
                  </div>
                  <h4>{paneTitle}</h4>
                  <p>{`线程 ID=${threadID || "无"} / 线程名=${threadName || "无"} / 预览面板=${pane}`}</p>
                </article>
              </div>
            </section>

            <section className="flow-panel card">
              <div className="section-header">
                <div>
                  <p className="section-title">受控浏览器 Fixture</p>
                  <h3>用于 canonical full lane 的最小本地交互链路</h3>
                </div>
              </div>
              <div className="flow-list">
                <article className="flow-item flow-item--neutral">
                  <div className="flow-item__header">
                    <span className="mini-chip">controlled-browser</span>
                    <span className="flow-item__meta">local fixture</span>
                  </div>
                  <h4 data-testid="controlled-browser-heading">受控浏览器验收面板</h4>
                  <p>这个面板只暴露稳定的本地选择器，供 runtime browser tools 和 full acceptance 使用。</p>
                  <div className="browser-toolbar__actions">
                    <input
                      className="browser-address"
                      data-testid="controlled-browser-input"
                      value={controlledInput}
                      onChange={(event) => setControlledInput(event.target.value)}
                      placeholder="输入受控浏览器测试文本"
                    />
                    <button
                      className="browser-nav"
                      data-testid="controlled-browser-apply"
                      onClick={() => setControlledResult(`Controlled browser result: ${controlledInput || "empty"}`)}
                      type="button"
                    >
                      应用输入
                    </button>
                  </div>
                  <p data-testid="controlled-browser-result">{controlledResult}</p>
                </article>
                <article className={`flow-item ${authenticatedSessionActive ? "flow-item--good" : "flow-item--warning"}`}>
                  <div className="flow-item__header">
                    <span className="mini-chip">authenticated-browser</span>
                    <span className="flow-item__meta">cookie-gated fixture</span>
                  </div>
                  <h4 data-testid="authenticated-browser-heading">Authenticated browser acceptance panel</h4>
                  <p>这个面板通过稳定 cookie 展示 authenticated session baseline，供 browser policy 与 canonical acceptance 验证。</p>
                  <p data-testid="authenticated-browser-session">{authenticatedSessionLabel}</p>
                  <p data-testid="authenticated-browser-profile">{authenticatedProfileLabel}</p>
                  <p data-testid="authenticated-browser-role">{authenticatedRoleLabel}</p>
                  <p data-testid="authenticated-browser-scope">{authenticatedScopeLabel}</p>
                  <p data-testid="authenticated-browser-transport">{authenticatedTransportLabel}</p>
                  <p data-testid="authenticated-browser-result">{authenticatedResult}</p>
                </article>
              </div>
            </section>
          </section>
        </section>
      </section>
    </main>
  );
}
