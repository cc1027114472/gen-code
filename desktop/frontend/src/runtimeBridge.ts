import {
  ActivateThread as WailsActivateThread,
  AdvanceTask as WailsAdvanceTask,
  CheckBridge as WailsCheckBridge,
  CreateTask as WailsCreateTask,
  CreateThread as WailsCreateThread,
  GetAppInfo as WailsGetAppInfo,
  GetRuntimeStatus as WailsGetRuntimeStatus,
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
  createdAt: string;
  updatedAt?: string;
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

type MCPDescriptor = {
  id: string;
  source?: string;
  toolCount: number;
  resourceCount: number;
};

export type BridgeCheckResult = {
  ok: boolean;
  message: string;
  checkedAt: string;
  runtimeHint: string;
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
    createdAt: string;
    updatedAt: string;
  }>;
  messages: MessageDescriptor[];
  toolCalls: ToolCallDescriptor[];
  artifacts: ArtifactDescriptor[];
  events: EventDescriptor[];
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

async function buildRuntimeStatus(): Promise<RuntimeStatus> {
  const [
    status,
    workspace,
    threadPayload,
    skillPayload,
    toolPayload,
    mcpPayload,
  ] = await Promise.all([
    fetchEnvelope<RuntimeApiStatus>("/api/runtime/status"),
    fetchEnvelope<WorkspaceDescriptor>("/api/workspace"),
    fetchEnvelope<{ items: ThreadDescriptor[] }>("/api/threads"),
    fetchEnvelope<{ items: SkillDescriptor[] }>("/api/skills"),
    fetchEnvelope<{ items: ToolDescriptor[] }>("/api/tools"),
    fetchEnvelope<{ items: MCPDescriptor[] }>("/api/mcp/servers"),
  ]);

  const activeThreadID = status.activeThreadId || "";
  const [tasks, messages, toolCalls, artifacts, events] = activeThreadID
    ? await Promise.all([
        fetchEnvelope<{ items: TaskDescriptor[] }>(`/api/threads/${encodeURIComponent(activeThreadID)}/tasks`),
        fetchEnvelope<{ items: MessageDescriptor[] }>(`/api/threads/${encodeURIComponent(activeThreadID)}/messages`),
        fetchEnvelope<{ items: ToolCallDescriptor[] }>(`/api/threads/${encodeURIComponent(activeThreadID)}/tool-calls`),
        fetchEnvelope<{ items: ArtifactDescriptor[] }>(`/api/threads/${encodeURIComponent(activeThreadID)}/artifacts`),
        fetchEnvelope<{ items: EventDescriptor[] }>(`/api/threads/${encodeURIComponent(activeThreadID)}/events`),
      ])
    : [
        { items: [] as TaskDescriptor[] },
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
      createdAt: item.createdAt,
      updatedAt: item.updatedAt || item.createdAt,
    })),
    messages: messages.items,
    toolCalls: toolCalls.items,
    artifacts: artifacts.items,
    events: events.items,
    desktopReady: true,
    runtimeState: status.state,
    runtimeReady: status.ready,
    runtimeMessage: status.message,
    runtimeSource: status.runtimeSource || "runtime-http",
    supportsSSE: activeThreadID !== "",
    sseEndpoint: activeThreadID ? `${runtimeBaseURL()}/api/threads/${encodeURIComponent(activeThreadID)}/events/stream?limit=200` : "",
    lastSyncAt: new Date().toISOString(),
    skillsByGroup: groupItems(skillPayload.items),
    toolsByGroup: groupTools(toolPayload.items),
    mcpByGroup: groupMCP(mcpPayload.items),
    missingPaths: [],
    stateStore: status.stateStore || "sqlite",
    statePath: status.statePath || "",
    usesProjectLocalStore: (status.stateStore || "").toLowerCase() === "sqlite",
    recoverySummary: `Browser bridge connected. Active thread: ${activeThreadID || "none"}, tasks: ${tasks.items.length}, messages: ${messages.items.length}, tool calls: ${toolCalls.items.length}, artifacts: ${artifacts.items.length}.`,
    updatedAt: new Date().toISOString(),
  };
}

export async function GetAppInfo(): Promise<string> {
  if (hasWailsBridge()) {
    return WailsGetAppInfo();
  }
  return "gen-code browser bridge ready";
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
    runtimeHint: "runtime-http",
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
