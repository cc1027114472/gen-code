import {
  ActivateThread as WailsActivateThread,
  ApproveTask as WailsApproveTask,
  AdvanceTask as WailsAdvanceTask,
  BrowserActivateTab as WailsBrowserActivateTab,
  BrowserBack as WailsBrowserBack,
  BrowserCloseTab as WailsBrowserCloseTab,
  BrowserForward as WailsBrowserForward,
  BrowserNavigate as WailsBrowserNavigate,
  BrowserOpen as WailsBrowserOpen,
  BrowserReload as WailsBrowserReload,
  BrowserState as WailsBrowserState,
  CheckBridge as WailsCheckBridge,
  CreateTask as WailsCreateTask,
  CreateThread as WailsCreateThread,
  GetAppInfo as WailsGetAppInfo,
  GetRuntimeStatus as WailsGetRuntimeStatus,
  RejectTask as WailsRejectTask,
} from "../wailsjs/go/main/App";

type ApiEnvelope<T> = {
  code: number;
  message: string;
  data: T;
};

type RuntimeApiStatus = {
  state: string;
  ready: boolean;
  message: string;
  runtimeSource: string;
  runtimeSourceDetail?: string;
  runtimeTrust?: string;
  stateStore: string;
  statePath: string;
  workspaceId: string;
  projectRoot: string;
  threadCount: number;
  activeThreadId: string;
  taskCount: number;
  eventCount: number;
};

type WorkspaceDescriptor = {
  id: string;
  projectRoot: string;
  sharedDocsRoot: string;
  createdAt: string;
  activeThreadCount: number;
};

type ThreadDescriptor = {
  id: string;
  workspaceId: string;
  name: string;
  status: string;
  activeModel: string;
  permissionMode: string;
  messageHistoryCount: number;
  toolCallCount: number;
  artifactCount: number;
  createdAt: string;
  isActive: boolean;
};

type TaskDescriptor = {
  id: string;
  threadId: string;
  title: string;
  status: string;
  kind?: string;
  inputSummary?: string;
  resultSummary?: string;
  approvalStatus?: string;
  parentTaskId?: string;
  waitingStatus?: string;
  agentStep?: number;
  agentMaxSteps?: number;
  latestChildTaskId?: string;
  agentPlanSummary?: string;
  agentPlanMode?: string;
  agentCurrentStepTitle?: string;
  agentLastReasoning?: string;
  createdAt: string;
  updatedAt?: string;
};

type ApprovalDescriptor = {
  id: string;
  threadId: string;
  taskId: string;
  toolKind: string;
  status: string;
  summary: string;
  targetPaths: string[];
  createdAt: string;
  updatedAt: string;
};

export type WriteExecutionDescriptor = {
  id: string;
  threadId: string;
  taskId: string;
  approvalId: string;
  toolKind: string;
  operation: string;
  relatedExecutionId: string;
  status: string;
  targetPaths: string[];
  patchSummary: string;
  beforeSummary: string;
  afterSummary: string;
  resultSummary: string;
  createdAt: string;
  updatedAt: string;
};

type MessageDescriptor = {
  id: string;
  threadId: string;
  role: string;
  content: string;
  createdAt: string;
};

type ToolCallDescriptor = {
  id: string;
  threadId: string;
  toolId: string;
  status: string;
  summary: string;
  createdAt: string;
};

type ArtifactDescriptor = {
  id: string;
  threadId: string;
  path: string;
  kind: string;
  createdAt: string;
};

type EventDescriptor = {
  id: string;
  threadId: string;
  type: string;
  message: string;
  createdAt: string;
};

type SkillDescriptor = {
  id: string;
  group: string;
  name?: string;
  description?: string;
  source?: string;
  verificationStatus?: string;
  localizationChecked?: boolean;
};

export type SkillGovernanceGroup = {
  group: string;
  implementedCount: number;
  verifiedCount: number;
  localizationPending: number;
};

type ToolDescriptor = {
  id: string;
  name?: string;
  description?: string;
  kind?: string;
  readOnly?: boolean;
  executable?: boolean;
  permissionMode?: string;
  source?: string;
};

type ProviderDescriptor = {
  kind: string;
  enabled: boolean;
  baseUrl?: string;
  defaultModel?: string;
  hasAuthToken: boolean;
  supportsChat: boolean;
  supportsResponses: boolean;
  preferredApiStyle?: string;
  recommended: boolean;
  recommendedReason?: string;
};

type MCPServerStatus = "enabled" | "disabled" | "degraded" | "unreachable";

type MCPDescriptor = {
  id: string;
  source?: string;
  enabled: boolean;
  toolCount: number;
  resourceCount: number;
  status: MCPServerStatus;
};

export type BridgeCheckResult = {
  ok: boolean;
  message: string;
  checkedAt: string;
  runtimeHint: string;
};

export type BrowserTab = {
  id: string;
  title: string;
  url: string;
  status: string;
  isActive: boolean;
  canGoBack: boolean;
  canGoForward: boolean;
};

export type BrowserWorkspaceState = {
  isOpen: boolean;
  tabs: BrowserTab[];
  activeTabId: string;
};

export type RuntimeStatus = {
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
  threads: Array<{
    id: string;
    name: string;
    status: string;
    activeModel: string;
    permissionMode: string;
    isActive: boolean;
  }>;
  tasks: Array<{
    id: string;
    threadId: string;
    title: string;
    kind: string;
    input: string;
    status: string;
    resultSummary: string;
    approvalStatus: string;
    parentTaskId: string;
    waitingStatus: string;
    agentStep: number;
    agentMaxSteps: number;
    latestChildTaskId: string;
    agentPlanSummary: string;
    agentPlanMode: string;
    agentCurrentStepTitle: string;
    agentLastReasoning: string;
    createdAt: string;
    updatedAt: string;
  }>;
  executableKinds: string[];
  approvals: ApprovalDescriptor[];
  writeExecutions: WriteExecutionDescriptor[];
  messages: MessageDescriptor[];
  toolCalls: ToolCallDescriptor[];
  artifacts: ArtifactDescriptor[];
  events: EventDescriptor[];
  desktopReady: boolean;
  runtimeState: string;
  runtimeReady: boolean;
  runtimeMessage: string;
  runtimeSource: string;
  runtimeSourceDetail: string;
  runtimeTrust: string;
  supportsSSE: boolean;
  sseEndpoint: string;
  lastSyncAt: string;
  skills: SkillDescriptor[];
  skillGovernance: SkillGovernanceGroup[];
  skillsByGroup: Record<string, string[]>;
  toolsByGroup: Record<string, string[]>;
  mcpByGroup: Record<string, string[]>;
  providers: ProviderDescriptor[];
  missingPaths: string[];
  stateStore: string;
  statePath: string;
  usesProjectLocalStore: boolean;
  recoverySummary: string;
  updatedAt: string;
};

declare global {
  interface Window {
    go?: {
      main?: {
        App?: Record<string, unknown>;
      };
    };
    __GENCODE_RUNTIME_BASE_URL__?: string;
  }
}

type BrowserImportMeta = ImportMeta & {
  env?: Record<string, string | undefined>;
};

const defaultRuntimeBaseURL = "http://127.0.0.1:10008";
let fallbackBrowserState: BrowserWorkspaceState | null = null;
const defaultFetchRetries = 4;
const defaultFetchRetryDelayMs = 250;

type RuntimeStatusView = Pick<RuntimeStatus, "runtimeSource" | "runtimeSourceDetail" | "runtimeTrust" | "runtimeMessage" | "supportsSSE">;

function hasWailsBridge() {
  return typeof window !== "undefined" && !!window.go?.main?.App;
}

function runtimeBaseURL() {
  const explicit = window.__GENCODE_RUNTIME_BASE_URL__?.trim();
  if (explicit) {
    return explicit.replace(/\/$/, "");
  }

  const envBase = ((import.meta as BrowserImportMeta).env?.VITE_RUNTIME_BASE_URL || "").trim();
  if (envBase) {
    return envBase.replace(/\/$/, "");
  }

  if (window.location.port === "10008") {
    return window.location.origin.replace(/\/$/, "");
  }

  return defaultRuntimeBaseURL;
}

async function fetchEnvelope<T>(path: string, init?: RequestInit): Promise<T> {
  let lastError: unknown = null;
  for (let attempt = 0; attempt < defaultFetchRetries; attempt += 1) {
    try {
      const headers = new Headers(init?.headers);
      if (init?.body && !headers.has("Content-Type")) {
        headers.set("Content-Type", "application/json");
      }

      const response = await fetch(`${runtimeBaseURL()}${path}`, {
        headers,
        ...init,
      });
      if (!response.ok) {
        throw new Error(`request failed: ${response.status}`);
      }

      const payload = (await response.json()) as ApiEnvelope<T>;
      return payload.data;
    } catch (error) {
      lastError = error;
      if (attempt === defaultFetchRetries - 1) {
        break;
      }
      await delay(defaultFetchRetryDelayMs * (attempt + 1));
    }
  }
  throw lastError instanceof Error ? lastError : new Error("request failed");
}

async function fetchEnvelopeOptional<T>(path: string, init?: RequestInit): Promise<T | null> {
  let lastError: unknown = null;
  for (let attempt = 0; attempt < defaultFetchRetries; attempt += 1) {
    try {
      const headers = new Headers(init?.headers);
      if (init?.body && !headers.has("Content-Type")) {
        headers.set("Content-Type", "application/json");
      }

      const response = await fetch(`${runtimeBaseURL()}${path}`, {
        headers,
        ...init,
      });
      if (response.status === 404) {
        return null;
      }
      if (!response.ok) {
        throw new Error(`request failed: ${response.status}`);
      }

      const payload = (await response.json()) as ApiEnvelope<T>;
      return payload.data;
    } catch (error) {
      lastError = error;
      if (attempt === defaultFetchRetries - 1) {
        break;
      }
      await delay(defaultFetchRetryDelayMs * (attempt + 1));
    }
  }
  throw lastError instanceof Error ? lastError : new Error("request failed");
}

function delay(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

function groupItems<T extends { group?: string; source?: string; id: string }>(items: T[]) {
  return items.reduce<Record<string, string[]>>((acc, item) => {
    const group = item.group || item.source || "common";
    acc[group] = acc[group] || [];
    acc[group].push(item.id);
    acc[group].sort();
    return acc;
  }, {});
}

function groupTools(items: ToolDescriptor[]) {
  return items.reduce<Record<string, string[]>>((acc, item) => {
    const group = item.source || "runtime";
    acc[group] = acc[group] || [];
    const parts: string[] = [];
    if (item.kind) {
      parts.push(item.kind);
    }
    if (item.permissionMode) {
      parts.push(item.permissionMode);
    }
    parts.push(item.executable ? "executable" : "descriptor");
    if (item.readOnly) {
      parts.push("read-only");
    }
    acc[group].push(parts.length > 0 ? `${item.id} (${parts.join(", ")})` : item.id);
    acc[group].sort();
    return acc;
  }, {});
}

function groupMCP(items: MCPDescriptor[]) {
  return items.reduce<Record<string, string[]>>((acc, item) => {
    const group = item.source || "runtime";
    acc[group] = acc[group] || [];
    acc[group].push(item.id);
    acc[group].sort();
    return acc;
  }, {});
}

function summarizeSkillGovernance(items: SkillDescriptor[]): SkillGovernanceGroup[] {
  const summaries = new Map<string, SkillGovernanceGroup>([
    ["common", { group: "common", implementedCount: 0, verifiedCount: 0, localizationPending: 0 }],
    ["codex", { group: "codex", implementedCount: 0, verifiedCount: 0, localizationPending: 0 }],
    ["cc", { group: "cc", implementedCount: 0, verifiedCount: 0, localizationPending: 0 }],
  ]);

  for (const item of items) {
    const group = item.group?.trim() || "common";
    const current = summaries.get(group) ?? { group, implementedCount: 0, verifiedCount: 0, localizationPending: 0 };
    current.implementedCount += 1;
    if ((item.verificationStatus || "").trim().toLowerCase() === "verified") {
      current.verifiedCount += 1;
    }
    if (!item.localizationChecked) {
      current.localizationPending += 1;
    }
    summaries.set(group, current);
  }

  const preferredOrder = ["common", "codex", "cc"];
  const ordered: SkillGovernanceGroup[] = [];
  for (const group of preferredOrder) {
    const summary = summaries.get(group);
    if (summary) {
      ordered.push(summary);
      summaries.delete(group);
    }
  }
  return [...ordered, ...[...summaries.values()].sort((left, right) => left.group.localeCompare(right.group))];
}

function runtimeSourceOrDefault(status?: Partial<RuntimeStatusView>) {
  return status?.runtimeSource || "remote-app-server";
}

function runtimeTrustOrDefault(status?: Partial<RuntimeStatusView>) {
  const trust = status?.runtimeTrust?.trim();
  if (trust) {
    return trust;
  }
  return runtimeSourceOrDefault(status) === "local-fallback" ? "degraded" : "canonical";
}

function runtimeMessageOrDefault(status?: Partial<RuntimeStatusView>) {
  const message = status?.runtimeMessage?.trim();
  if (message) {
    return message;
  }
  if (runtimeSourceOrDefault(status) === "local-fallback") {
    return "desktop local-fallback ?????????????";
  }
  return "remote app-server runtime ???";
}

export function formatRuntimeLaneLabel(status?: Partial<RuntimeStatusView>) {
  switch (runtimeSourceOrDefault(status)) {
    case "remote-app-server":
      return "????? / remote-app-server";
    case "local-fallback":
      return "???? / local-fallback";
    default:
      return status?.runtimeSource || "?????";
  }
}

export function formatRuntimeLaneDetail(status?: Partial<RuntimeStatusView>) {
  const source = runtimeSourceOrDefault(status);
  const detail = status?.runtimeSourceDetail?.trim();
  if (detail) {
    return detail;
  }
  if (source === "local-fallback") {
    return "canonical app-server ???????????? SQLite fallback?";
  }
  return "?????? canonical app-server shared runtime?";
}

export function formatRuntimeTrustLabel(status?: Partial<RuntimeStatusView>) {
  switch (runtimeTrustOrDefault(status)) {
    case "canonical":
      return "???? / canonical";
    case "degraded":
      return "???? / degraded";
    default:
      return `???? / ${runtimeTrustOrDefault(status)}`;
  }
}

export function formatRefreshMode(status?: Partial<RuntimeStatusView>, sseConnected?: boolean) {
  if (!status?.supportsSSE) {
    return "??????";
  }
  if (sseConnected) {
    return "SSE ????";
  }
  return "SSE ???";
}

export function formatRefreshModeDetail(status?: Partial<RuntimeStatusView>, sseConnected?: boolean) {
  if (!status?.supportsSSE) {
    if (runtimeSourceOrDefault(status) === "local-fallback") {
      return "?? fallback ??? SSE?????????????????????";
    }
    return "?????????? SSE?????????????";
  }
  if (sseConnected) {
    return "SSE ?????????????????????";
  }
  return "??????? SSE?????????????????????";
}

export function formatFallbackNote(status?: Partial<RuntimeStatusView>) {
  if (runtimeSourceOrDefault(status) !== "local-fallback") {
    return "???????????????????? app-server?";
  }
  return "??? desktop local-fallback??????????????????????????????? canonical ??????";
}

async function buildRuntimeStatus(): Promise<RuntimeStatus> {
  const [
    status,
    workspace,
    threadPayload,
    skillPayload,
    toolPayload,
    providerPayload,
    mcpPayload,
  ] = await Promise.all([
    fetchEnvelope<RuntimeApiStatus>("/api/runtime/status"),
    fetchEnvelope<WorkspaceDescriptor>("/api/workspace"),
    fetchEnvelope<{ items: ThreadDescriptor[] }>("/api/threads"),
    fetchEnvelope<{ items: SkillDescriptor[] }>("/api/skills"),
    fetchEnvelope<{ items: ToolDescriptor[] }>("/api/tools"),
    fetchEnvelope<{ items: ProviderDescriptor[] }>("/api/providers"),
    fetchEnvelope<{ items: MCPDescriptor[] }>("/api/mcp/servers"),
  ]);

  const activeThreadID = status.activeThreadId || "";
  const [tasks, approvals, writeExecutions, messages, toolCalls, artifacts, events] = activeThreadID
    ? await Promise.all([
        fetchEnvelope<{ items: TaskDescriptor[] }>(`/api/threads/${encodeURIComponent(activeThreadID)}/tasks`),
        fetchEnvelope<{ items: ApprovalDescriptor[] }>(`/api/threads/${encodeURIComponent(activeThreadID)}/approvals`),
        fetchEnvelopeOptional<{ items: WriteExecutionDescriptor[] }>(`/api/threads/${encodeURIComponent(activeThreadID)}/write-executions`),
        fetchEnvelope<{ items: MessageDescriptor[] }>(`/api/threads/${encodeURIComponent(activeThreadID)}/messages`),
        fetchEnvelope<{ items: ToolCallDescriptor[] }>(`/api/threads/${encodeURIComponent(activeThreadID)}/tool-calls`),
        fetchEnvelope<{ items: ArtifactDescriptor[] }>(`/api/threads/${encodeURIComponent(activeThreadID)}/artifacts`),
        fetchEnvelope<{ items: EventDescriptor[] }>(`/api/threads/${encodeURIComponent(activeThreadID)}/events`),
      ])
    : [
        { items: [] as TaskDescriptor[] },
        { items: [] as ApprovalDescriptor[] },
        { items: [] as WriteExecutionDescriptor[] },
        { items: [] as MessageDescriptor[] },
        { items: [] as ToolCallDescriptor[] },
        { items: [] as ArtifactDescriptor[] },
        { items: [] as EventDescriptor[] },
      ];

  return {
    appName: "gen-code",
    appEnv: "browser",
    port: Number(new URL(runtimeBaseURL()).port || "80"),
    debug: false,
    shutdownTimeout: "10s",
    trustedProxies: ["127.0.0.1"],
    logLevel: "info",
    httpAccessLog: true,
    workspaceRoot: workspace.projectRoot,
    workspaceId: workspace.id,
    projectRoot: workspace.projectRoot,
    threadCount: status.threadCount || threadPayload.items.length,
    activeThreadId: activeThreadID,
    threads: threadPayload.items.map((item) => ({
      id: item.id,
      name: item.name,
      status: item.status,
      activeModel: item.activeModel || "",
      permissionMode: item.permissionMode,
      isActive: item.isActive,
    })),
    tasks: tasks.items.map((item) => ({
      id: item.id,
      threadId: item.threadId,
      title: item.title,
      kind: item.kind || "prompt",
      input: item.inputSummary || "",
      status: item.status,
      resultSummary: item.resultSummary || "",
      approvalStatus: item.approvalStatus || "",
      parentTaskId: item.parentTaskId || "",
      waitingStatus: item.waitingStatus || "",
      agentStep: item.agentStep || 0,
      agentMaxSteps: item.agentMaxSteps || 0,
      latestChildTaskId: item.latestChildTaskId || "",
      agentPlanSummary: item.agentPlanSummary || "",
      agentPlanMode: item.agentPlanMode || "",
      agentCurrentStepTitle: item.agentCurrentStepTitle || "",
      agentLastReasoning: item.agentLastReasoning || "",
      createdAt: item.createdAt,
      updatedAt: item.updatedAt || item.createdAt,
    })),
    executableKinds: toolPayload.items
      .filter((item) => item.executable && item.kind)
      .map((item) => item.kind as string)
      .sort((left, right) => left.localeCompare(right)),
    approvals: approvals.items,
    writeExecutions: writeExecutions?.items ?? [],
    messages: messages.items,
    toolCalls: toolCalls.items,
    artifacts: artifacts.items,
    events: events.items,
    desktopReady: true,
    runtimeState: status.state,
    runtimeReady: status.ready,
    runtimeMessage: runtimeMessageOrDefault(status),
    runtimeSource: status.runtimeSource || "remote-app-server",
    runtimeSourceDetail: status.runtimeSourceDetail || formatRuntimeLaneDetail(status),
    runtimeTrust: status.runtimeTrust || runtimeTrustOrDefault(status),
    supportsSSE: activeThreadID !== "",
    sseEndpoint: activeThreadID ? `${runtimeBaseURL()}/api/threads/${encodeURIComponent(activeThreadID)}/events/stream?limit=200` : "",
    lastSyncAt: new Date().toISOString(),
    skills: [...skillPayload.items].sort((left, right) => {
      if ((left.group || "") === (right.group || "")) {
        return left.id.localeCompare(right.id);
      }
      return (left.group || "").localeCompare(right.group || "");
    }),
    skillGovernance: summarizeSkillGovernance(skillPayload.items),
    skillsByGroup: groupItems(skillPayload.items),
    toolsByGroup: groupTools(toolPayload.items),
    mcpByGroup: groupMCP(mcpPayload.items),
    providers: [...providerPayload.items].sort((left, right) => {
      if (left.recommended !== right.recommended) {
        return left.recommended ? -1 : 1;
      }
      return left.kind.localeCompare(right.kind);
    }),
    missingPaths: [],
    stateStore: status.stateStore || "sqlite",
    statePath: status.statePath || "",
    usesProjectLocalStore: (status.stateStore || "").toLowerCase() === "sqlite",
    recoverySummary: `浏览器桥接已连接。活动线程：${activeThreadID || "无"}，任务：${tasks.items.length}，消息：${messages.items.length}，工具调用：${toolCalls.items.length}，产物：${artifacts.items.length}。`,
    updatedAt: new Date().toISOString(),
  };
}

export async function GetAppInfo(): Promise<string> {
  if (hasWailsBridge()) {
    return WailsGetAppInfo();
  }
  return "gen-code 浏览器桥接已就绪";
}

export async function GetRuntimeStatus(): Promise<RuntimeStatus> {
  if (hasWailsBridge()) {
    return WailsGetRuntimeStatus();
  }
  return buildRuntimeStatus();
}

export async function CheckBridge(): Promise<BridgeCheckResult> {
  if (hasWailsBridge()) {
    return WailsCheckBridge();
  }

  const result = await fetchEnvelope<{ ok: boolean; message: string }>(
    "/api/bridge/check",
    {
      method: "POST",
      body: JSON.stringify({ source: "browser-bridge" }),
    },
  );
  return {
    ok: result.ok,
    message: result.message,
    checkedAt: new Date().toISOString(),
    runtimeHint: "remote-app-server",
  };
}

export async function CreateThread(name: string): Promise<RuntimeStatus> {
  if (hasWailsBridge()) {
    return WailsCreateThread(name);
  }

  await fetchEnvelope("/api/threads", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
  return buildRuntimeStatus();
}

export async function ActivateThread(id: string): Promise<RuntimeStatus> {
  if (hasWailsBridge()) {
    return WailsActivateThread(id);
  }

  await fetchEnvelope(`/api/threads/${encodeURIComponent(id)}/activate`, {
    method: "POST",
    body: JSON.stringify({}),
  });
  return buildRuntimeStatus();
}

export async function CreateTask(threadID: string, payload: string): Promise<RuntimeStatus> {
  if (hasWailsBridge()) {
    return WailsCreateTask(threadID, payload);
  }

  await fetchEnvelope(`/api/threads/${encodeURIComponent(threadID)}/tasks`, {
    method: "POST",
    body: payload,
  });
  return buildRuntimeStatus();
}

export async function AdvanceTask(taskID: string): Promise<RuntimeStatus> {
  if (hasWailsBridge()) {
    return WailsAdvanceTask(taskID);
  }

  const runtime = await buildRuntimeStatus();
  if (!runtime.activeThreadId) {
    throw new Error("no active thread");
  }

  await fetchEnvelope(`/api/threads/${encodeURIComponent(runtime.activeThreadId)}/tasks/${encodeURIComponent(taskID)}/run`, {
    method: "POST",
    body: JSON.stringify({}),
  });
  return buildRuntimeStatus();
}

export async function ApproveTask(threadID: string, taskID: string): Promise<RuntimeStatus> {
  if (hasWailsBridge()) {
    return WailsApproveTask(threadID, taskID);
  }

  await fetchEnvelope(`/api/threads/${encodeURIComponent(threadID)}/tasks/${encodeURIComponent(taskID)}/approve`, {
    method: "POST",
    body: JSON.stringify({}),
  });
  return buildRuntimeStatus();
}

export async function RejectTask(threadID: string, taskID: string): Promise<RuntimeStatus> {
  if (hasWailsBridge()) {
    return WailsRejectTask(threadID, taskID);
  }

  await fetchEnvelope(`/api/threads/${encodeURIComponent(threadID)}/tasks/${encodeURIComponent(taskID)}/reject`, {
    method: "POST",
    body: JSON.stringify({}),
  });
  return buildRuntimeStatus();
}

function defaultBrowserState(): BrowserWorkspaceState {
  return {
    isOpen: true,
    activeTabId: "browser-tab-local",
    tabs: [
      {
        id: "browser-tab-local",
        title: "127.0.0.1:5174",
        url: "http://127.0.0.1:5174/",
        status: "ready",
        isActive: true,
        canGoBack: false,
        canGoForward: false,
      },
    ],
  };
}

function cloneBrowserState(state: BrowserWorkspaceState): BrowserWorkspaceState {
  return {
    isOpen: state.isOpen,
    activeTabId: state.activeTabId,
    tabs: state.tabs.map((tab) => ({ ...tab })),
  };
}

function ensureFallbackBrowserState(): BrowserWorkspaceState {
  if (!fallbackBrowserState) {
    fallbackBrowserState = defaultBrowserState();
  }
  return fallbackBrowserState;
}

export async function GetBrowserState(): Promise<BrowserWorkspaceState> {
  if (hasWailsBridge()) {
    return WailsBrowserState();
  }
  return cloneBrowserState(ensureFallbackBrowserState());
}

export async function BrowserOpen(url?: string): Promise<BrowserWorkspaceState> {
  if (hasWailsBridge()) {
    return WailsBrowserOpen(url || "");
  }
  const current = ensureFallbackBrowserState();
  const nextID = `browser-tab-local-${current.tabs.length + 1}`;
  current.tabs = current.tabs.map((item: BrowserTab) => ({ ...item, isActive: false }));
  current.tabs.push({
    id: nextID,
    title: url || "本地预览",
    url: url || "http://127.0.0.1:5174/",
    status: "ready",
    isActive: true,
    canGoBack: false,
    canGoForward: false,
  });
  current.activeTabId = nextID;
  current.isOpen = true;
  return cloneBrowserState(current);
}

export async function BrowserNavigate(tabId: string, url: string): Promise<BrowserWorkspaceState> {
  if (hasWailsBridge()) {
    return WailsBrowserNavigate(tabId, url);
  }
  const current = ensureFallbackBrowserState();
  const targetID = tabId || current.activeTabId || current.tabs[0]?.id || "browser-tab-local";
  const hasTarget = current.tabs.some((item: BrowserTab) => item.id === targetID);
  if (hasTarget) {
    current.tabs = current.tabs.map((item: BrowserTab) =>
      item.id === targetID
        ? { ...item, title: url, url, status: "ready", isActive: true, canGoBack: false, canGoForward: false }
        : { ...item, isActive: false },
    );
  } else {
    current.tabs = [
      ...current.tabs.map((item: BrowserTab) => ({ ...item, isActive: false })),
      {
        id: targetID,
        title: url,
        url,
        status: "ready",
        isActive: true,
        canGoBack: false,
        canGoForward: false,
      },
    ];
  }
  current.activeTabId = targetID;
  current.isOpen = true;
  return cloneBrowserState(current);
}

export async function BrowserBack(tabId: string): Promise<BrowserWorkspaceState> {
  if (hasWailsBridge()) {
    return WailsBrowserBack(tabId);
  }
  return cloneBrowserState(ensureFallbackBrowserState());
}

export async function BrowserForward(tabId: string): Promise<BrowserWorkspaceState> {
  if (hasWailsBridge()) {
    return WailsBrowserForward(tabId);
  }
  return cloneBrowserState(ensureFallbackBrowserState());
}

export async function BrowserReload(tabId: string): Promise<BrowserWorkspaceState> {
  if (hasWailsBridge()) {
    return WailsBrowserReload(tabId);
  }
  return cloneBrowserState(ensureFallbackBrowserState());
}

export async function BrowserCloseTab(tabId: string): Promise<BrowserWorkspaceState> {
  if (hasWailsBridge()) {
    return WailsBrowserCloseTab(tabId);
  }
  const current = ensureFallbackBrowserState();
  current.tabs = current.tabs.filter((item: BrowserTab) => item.id !== tabId);
  if (current.tabs.length === 0) {
    fallbackBrowserState = defaultBrowserState();
    return cloneBrowserState(fallbackBrowserState);
  }
  if (!current.tabs.some((item: BrowserTab) => item.id === current.activeTabId)) {
    current.activeTabId = current.tabs[0].id;
  }
  current.tabs = current.tabs.map((item: BrowserTab) => ({ ...item, isActive: item.id === current.activeTabId }));
  return cloneBrowserState(current);
}

export async function BrowserActivateTab(tabId: string): Promise<BrowserWorkspaceState> {
  if (hasWailsBridge()) {
    return WailsBrowserActivateTab(tabId);
  }
  const current = ensureFallbackBrowserState();
  current.activeTabId = tabId;
  current.tabs = current.tabs.map((item: BrowserTab) => ({ ...item, isActive: item.id === tabId }));
  current.isOpen = true;
  return cloneBrowserState(current);
}
