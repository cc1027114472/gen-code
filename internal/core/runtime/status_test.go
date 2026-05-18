package runtime

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/core/mcp"
	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/provider"
	"llmtrace/internal/core/runner"
	"llmtrace/internal/core/session"
	"llmtrace/internal/core/skill"
	"llmtrace/internal/core/state"
	"llmtrace/internal/core/tool"
)

func TestRuntimeToolsExposeBuiltInBrowserDescriptors(t *testing.T) {
	projectRoot := t.TempDir()
	store, err := state.Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	service := newServiceFromDiscoveryWithStore(
		discoverySet{tools: append([]tool.Descriptor(nil), builtinBrowserToolDescriptors...)},
		store,
		provider.NewRegistry(""),
	)

	tools, err := service.Tools(context.Background())
	require.NoError(t, err)

	toolIDs := make([]string, 0, len(tools))
	for _, item := range tools {
		toolIDs = append(toolIDs, item.ID)
	}

	require.True(t, slices.Contains(toolIDs, "browser.open"))
	require.True(t, slices.Contains(toolIDs, "browser.navigate"))
	require.True(t, slices.Contains(toolIDs, "browser.state"))
	require.True(t, slices.Contains(toolIDs, "browser.click"))
	require.True(t, slices.Contains(toolIDs, "browser.type"))
	require.True(t, slices.Contains(toolIDs, "browser.extract"))
	require.True(t, slices.Contains(toolIDs, "browser.screenshot"))
}

func TestServiceContractShapesExposeStructuredMetadata(t *testing.T) {
	projectRoot := t.TempDir()
	store, err := state.Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	registry := tool.NewRegistry()
	registry.Register(tool.Descriptor{
		ID:                 "bridge.check",
		Name:               "Bridge Check",
		Description:        "Verify the bridge",
		InputSchemaSummary: "No input",
		PermissionMode:     policy.AskUser,
		Source:             "runtime",
		Kind:               "bridge",
		ReadOnly:           true,
		Executable:         false,
	})

	sessions, err := session.NewRegistryWithStore(projectRoot, store)
	require.NoError(t, err)

	mcpManager := mcp.NewManager([]mcp.ServerDescriptor{{
		ID:            "server-1",
		Source:        "node_modules",
		Enabled:       true,
		ToolCount:     2,
		ResourceCount: 3,
		Status:        "enabled",
	}})

	service := NewService(
		"0.1.0",
		skill.Codex,
		policy.DefaultMode(),
		projectRoot,
		registry,
		skill.NewManager([]skill.Descriptor{
			{ID: "common.browser", Group: skill.Common, Name: "Browser", Description: "Reusable browser skill", CapabilitySummary: "builtin common skill", CapabilityVerified: false},
			{ID: "codex.review", Group: skill.Codex, Name: "Review", Description: "Codex review skill", CapabilitySummary: "capability verified", CapabilityVerified: true},
			{ID: "cc.swarm", Group: skill.CC, Name: "Swarm", Description: "CC swarm skill", CapabilitySummary: "capability verified", CapabilityVerified: true},
		}),
		mcpManager,
		provider.NewRegistry(""),
		sessions,
	)

	created, err := service.CreateThread(context.Background(), runtimecontract.CreateThreadRequest{
		Name:           "Thread 1",
		PermissionMode: "workspace-write",
	})
	require.NoError(t, err)
	require.Equal(t, "thread-1", created.ID)
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("old\n"), 0o644))

	task, err := service.CreateTask(context.Background(), created.ID, runtimecontract.CreateTaskRequest{
		Title: "Draft spec",
		Kind:  "thread.message.append",
		Input: `{"role":"user","content":"Draft the spec"}`,
	})
	require.NoError(t, err)
	require.Equal(t, task.CreatedAt, task.UpdatedAt)
	require.Equal(t, "thread.message.append", task.Kind)

	updatedTask, err := service.UpdateTaskStatus(context.Background(), created.ID, task.ID, runtimecontract.UpdateTaskStatusRequest{
		Status: "running",
	})
	require.NoError(t, err)
	require.Equal(t, "running", updatedTask.Status)
	require.NotEmpty(t, updatedTask.UpdatedAt)

	executedTask, err := service.RunTask(context.Background(), created.ID, task.ID, runtimecontract.RunTaskRequest{})
	require.NoError(t, err)
	require.Equal(t, "completed", executedTask.Status)
	require.NotEmpty(t, executedTask.ResultSummary)

	message, err := service.AppendMessage(context.Background(), created.ID, runtimecontract.CreateMessageRequest{
		Role:    "user",
		Content: "Draft the spec",
	})
	require.NoError(t, err)
	require.Equal(t, "user", message.Role)

	toolCall, err := service.AppendToolCall(context.Background(), created.ID, runtimecontract.CreateToolCallRequest{
		ToolID:  "bridge.check",
		Status:  "completed",
		Summary: "Bridge reachable",
	})
	require.NoError(t, err)
	require.Equal(t, "bridge.check", toolCall.ToolID)

	artifact, err := service.AppendArtifact(context.Background(), created.ID, runtimecontract.CreateArtifactRequest{
		Path: `D:\artifacts\spec.md`,
		Kind: "markdown",
	})
	require.NoError(t, err)
	require.Equal(t, "markdown", artifact.Kind)

	flag, err := service.SetRuntimeFlag(context.Background(), created.ID, runtimecontract.SetRuntimeFlagRequest{
		Key:   "preview",
		Value: "ready",
	})
	require.NoError(t, err)
	require.Equal(t, "ready", flag.Value)

	skills, err := service.Skills(context.Background())
	require.NoError(t, err)
	require.Len(t, skills, 2)
	require.ElementsMatch(t, []string{"common", "codex"}, []string{skills[0].Group, skills[1].Group})
	require.ElementsMatch(t, []string{"common", "codex"}, []string{skills[0].Source, skills[1].Source})
	require.ElementsMatch(t, []string{"implemented", "implemented"}, []string{skills[0].VerificationStatus, skills[1].VerificationStatus})
	require.False(t, skills[0].LocalizationChecked)
	require.False(t, skills[1].LocalizationChecked)
	require.ElementsMatch(t, []string{"shared-common", "isolated"}, []string{skills[0].IsolationStatus, skills[1].IsolationStatus})
	require.ElementsMatch(t, []bool{false, true}, []bool{skills[0].CapabilityVerified, skills[1].CapabilityVerified})
	require.Contains(t, []string{skills[0].CapabilitySummary, skills[1].CapabilitySummary}, "builtin common skill")
	require.Contains(t, []string{skills[0].CapabilitySummary, skills[1].CapabilitySummary}, "capability verified")

	tools, err := service.Tools(context.Background())
	require.NoError(t, err)
	require.Len(t, tools, 1)
	require.Equal(t, "ask-user", tools[0].Permission)
	require.Equal(t, "runtime", tools[0].Source)
	require.Equal(t, "bridge", tools[0].Kind)
	require.True(t, tools[0].ReadOnly)
	require.False(t, tools[0].Executable)

	servers, err := service.MCPServers(context.Background())
	require.NoError(t, err)
	require.Len(t, servers, 1)
	require.Equal(t, "node_modules", servers[0].Source)
	require.True(t, servers[0].Enabled)
	require.Equal(t, 2, servers[0].ToolCount)
	require.Equal(t, 3, servers[0].ResourceCount)
	require.Equal(t, "enabled", servers[0].Status)

	status, err := service.Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, "remote-app-server", status.RuntimeSource)
	require.Equal(t, "canonical", status.RuntimeTrust)
	require.Equal(t, "canonical shared runtime served by the app-server entry", status.RuntimeSourceDetail)
	require.Equal(t, "http://127.0.0.1:10008", status.CanonicalRuntimeURL)
	require.Equal(t, sessions.Workspace().ID, status.WorkspaceID)
	require.Equal(t, projectRoot, status.ProjectRoot)
	require.Equal(t, 1, status.ThreadCount)
	require.Equal(t, "thread-1", status.ActiveThreadID)
	require.Equal(t, state.StoreName, status.StateStore)
	require.Equal(t, state.PathForProject(projectRoot), status.StatePath)

	fullStatus := service.FullStatus()
	require.Equal(t, status.RuntimeSource, fullStatus.RuntimeSource)
	require.Equal(t, status.RuntimeTrust, fullStatus.RuntimeTrust)
	require.Equal(t, status.RuntimeSourceDetail, fullStatus.RuntimeSourceDetail)
	require.Equal(t, status.CanonicalRuntimeURL, fullStatus.CanonicalRuntimeURL)

	tasks, err := service.Tasks(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, "completed", tasks[0].Status)
	require.Equal(t, "thread.message.append", tasks[0].Kind)
	require.NotEmpty(t, tasks[0].UpdatedAt)

	messages, err := service.Messages(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, "Draft the spec", messages[0].Content)

	toolCalls, err := service.ToolCalls(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, toolCalls, 3)
	require.ElementsMatch(t, []string{"completed", "running", "completed"}, []string{toolCalls[0].Status, toolCalls[1].Status, toolCalls[2].Status})

	artifacts, err := service.Artifacts(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	require.Equal(t, `D:\artifacts\spec.md`, artifacts[0].Path)

	flags, err := service.RuntimeFlags(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, flags, 1)
	require.Equal(t, "preview", flags[0].Key)

	patchTask, err := service.CreateTask(context.Background(), created.ID, runtimecontract.CreateTaskRequest{
		Title: "Patch README",
		Kind:  runner.KindWorkspaceApplyPatch,
		Input: `{"path":"README.md","patch":"*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch"}`,
	})
	require.NoError(t, err)
	require.Equal(t, "direct", patchTask.ApprovalStatus)
	_, err = service.RunTask(context.Background(), created.ID, patchTask.ID, runtimecontract.RunTaskRequest{})
	require.NoError(t, err)

	writeExecutions, err := service.WriteExecutions(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, writeExecutions, 1)
	require.Equal(t, "workspace.apply_patch", writeExecutions[0].ToolKind)
	require.Equal(t, "apply", writeExecutions[0].Operation)
	require.Equal(t, []string{"README.md"}, writeExecutions[0].TargetPaths)
	require.Equal(t, "completed", writeExecutions[0].Status)
	require.Contains(t, writeExecutions[0].ResultSummary, "applied patch to README.md")

	stream, err := service.StreamEvents(context.Background(), created.ID, runtimecontract.StreamEventsRequest{})
	require.NoError(t, err)
	streamed := make([]runtimecontract.EventDescriptor, 0)
	for len(streamed) < 1 {
		item := <-stream
		streamed = append(streamed, item)
	}
	require.NotEmpty(t, streamed)
	require.Equal(t, created.ID, streamed[0].ThreadID)
}

func TestNormalizeMCPServerStatus(t *testing.T) {
	testCases := []struct {
		name string
		item mcp.ServerDescriptor
		want string
	}{
		{
			name: "prefers explicit degraded status",
			item: mcp.ServerDescriptor{Enabled: true, ToolCount: 2, ResourceCount: 1, Status: "degraded"},
			want: "degraded",
		},
		{
			name: "disabled wins when server disabled",
			item: mcp.ServerDescriptor{Enabled: false, ToolCount: 2, ResourceCount: 1},
			want: "disabled",
		},
		{
			name: "enabled with zero inventory degrades by default",
			item: mcp.ServerDescriptor{Enabled: true, ToolCount: 0, ResourceCount: 0},
			want: "degraded",
		},
		{
			name: "enabled with inventory stays enabled",
			item: mcp.ServerDescriptor{Enabled: true, ToolCount: 1, ResourceCount: 0},
			want: "enabled",
		},
		{
			name: "keeps unreachable contract value",
			item: mcp.ServerDescriptor{Enabled: true, ToolCount: 1, ResourceCount: 0, Status: "unreachable"},
			want: "unreachable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, normalizeMCPServerStatus(tc.item))
		})
	}
}

func TestServiceRunsMCPInvokeTask(t *testing.T) {
	projectRoot := t.TempDir()
	store, err := state.Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	sessions, err := session.NewRegistryWithStore(projectRoot, store)
	require.NoError(t, err)

	service := NewService(
		"0.1.0",
		skill.Codex,
		policy.DefaultMode(),
		projectRoot,
		tool.NewRegistry(),
		skill.NewManager(nil),
		mcp.NewManager([]mcp.ServerDescriptor{{
			ID:            "external-fixture",
			Source:        "fixture",
			Enabled:       true,
			ToolCount:     2,
			ResourceCount: 0,
			Status:        "enabled",
			Command:       mcpFixtureCommand(),
			Tools:         []string{"echo", "sum", "fail"},
		}}),
		provider.NewRegistry(""),
		sessions,
	)

	thread, err := service.CreateThread(context.Background(), runtimecontract.CreateThreadRequest{
		Name:           "MCP Thread",
		PermissionMode: string(policy.ReadOnly),
	})
	require.NoError(t, err)

	task, err := service.CreateTask(context.Background(), thread.ID, runtimecontract.CreateTaskRequest{
		Title: "Invoke fixture echo",
		Kind:  runner.KindMCPToolInvoke,
		Input: `{"serverId":"external-fixture","toolName":"echo","arguments":{"message":"hello"}}`,
	})
	require.NoError(t, err)

	result, err := service.RunTask(context.Background(), thread.ID, task.ID, runtimecontract.RunTaskRequest{})
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Equal(t, "mcp tool external-fixture/echo executed", result.ResultSummary)

	toolCalls, err := service.ToolCalls(context.Background(), thread.ID)
	require.NoError(t, err)
	require.Len(t, toolCalls, 2)
	require.Equal(t, "mcp.tool.invoke", toolCalls[0].ToolID)
	require.Equal(t, "running", toolCalls[0].Status)
	require.Equal(t, "completed", toolCalls[1].Status)
}

func TestServiceRunsSDKMCPInvokeTask(t *testing.T) {
	projectRoot := t.TempDir()
	store, err := state.Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	sessions, err := session.NewRegistryWithStore(projectRoot, store)
	require.NoError(t, err)

	service := NewService(
		"0.1.0",
		skill.Codex,
		policy.DefaultMode(),
		projectRoot,
		tool.NewRegistry(),
		skill.NewManager(nil),
		mcp.NewManager([]mcp.ServerDescriptor{{
			ID:            "sdk-external-fixture",
			Source:        "sdk",
			Enabled:       true,
			ToolCount:     2,
			ResourceCount: 0,
			Status:        "enabled",
			Command:       mcpSDKServerCommand(),
			Tools:         []string{"echo", "sum"},
			Transport:     "stdio-sdk",
		}}),
		provider.NewRegistry(""),
		sessions,
	)

	thread, err := service.CreateThread(context.Background(), runtimecontract.CreateThreadRequest{
		Name:           "MCP SDK Thread",
		PermissionMode: string(policy.ReadOnly),
	})
	require.NoError(t, err)

	task, err := service.CreateTask(context.Background(), thread.ID, runtimecontract.CreateTaskRequest{
		Title: "Invoke sdk echo",
		Kind:  runner.KindMCPToolInvoke,
		Input: `{"serverId":"sdk-external-fixture","toolName":"echo","arguments":{"message":"hello-sdk"}}`,
	})
	require.NoError(t, err)

	result, err := service.RunTask(context.Background(), thread.ID, task.ID, runtimecontract.RunTaskRequest{})
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Equal(t, "mcp tool sdk-external-fixture/echo executed", result.ResultSummary)
}

func TestServiceCreateTaskNormalizesPlainModelInput(t *testing.T) {
	projectRoot := t.TempDir()
	store, err := state.Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	sessions, err := session.NewRegistryWithStore(projectRoot, store)
	require.NoError(t, err)

	service := NewService(
		"0.1.0",
		skill.Codex,
		policy.DefaultMode(),
		projectRoot,
		tool.NewRegistry(),
		skill.NewManager(nil),
		mcp.NewManager(nil),
		provider.NewRegistry(""),
		sessions,
	)

	thread, err := service.CreateThread(context.Background(), runtimecontract.CreateThreadRequest{
		Name:           "Model Plain Input",
		PermissionMode: "workspace-write",
	})
	require.NoError(t, err)

	task, err := service.CreateTask(context.Background(), thread.ID, runtimecontract.CreateTaskRequest{
		Title: "Plain model input",
		Kind:  runner.KindModelResponse,
		Input: "Reply with exactly: plain text works.",
	})
	require.NoError(t, err)
	require.Equal(t, runner.KindModelResponse, task.Kind)
	require.JSONEq(t, `{"input":"Reply with exactly: plain text works."}`, task.InputSummary)
}

func TestServiceSetCanonicalRuntimeURLOverridesSnapshot(t *testing.T) {
	projectRoot := t.TempDir()
	store, err := state.Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	sessions, err := session.NewRegistryWithStore(projectRoot, store)
	require.NoError(t, err)

	service := NewService(
		"0.1.0",
		skill.Codex,
		policy.DefaultMode(),
		projectRoot,
		tool.NewRegistry(),
		skill.NewManager(nil),
		mcp.NewManager(nil),
		provider.NewRegistry(""),
		sessions,
	)

	service.SetCanonicalRuntimeURL("http://127.0.0.1:10018/")

	status, err := service.Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:10018", status.CanonicalRuntimeURL)

	fullStatus := service.FullStatus()
	require.Equal(t, "http://127.0.0.1:10018", fullStatus.CanonicalRuntimeURL)
}

func TestServiceCreateRollbackTaskRequiresApprovalForAskUser(t *testing.T) {
	projectRoot := t.TempDir()

	store, err := state.Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	sessions, err := session.NewRegistryWithStore(projectRoot, store)
	require.NoError(t, err)

	service := NewService(
		"0.1.0",
		skill.Codex,
		policy.DefaultMode(),
		projectRoot,
		tool.NewRegistry(),
		skill.NewManager(nil),
		mcp.NewManager(nil),
		provider.NewRegistry(""),
		sessions,
	)

	thread, err := service.CreateThread(context.Background(), runtimecontract.CreateThreadRequest{
		Name:           "Rollback Approval",
		PermissionMode: "ask-user",
	})
	require.NoError(t, err)

	applyTask, ok := sessions.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Patch README",
		Kind:           runner.KindWorkspaceApplyPatch,
		Input:          `{"path":"README.md","patch":"*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch"}`,
		Status:         "completed",
		ResultSummary:  "applied patch to README.md: updated 2 line(s)",
		ApprovalStatus: "direct",
	})
	require.True(t, ok)
	_, err = sessions.CreateWriteExecution(thread.ID, session.CreateWriteExecutionInput{
		TaskID:                applyTask.ID,
		ToolKind:              runner.KindWorkspaceApplyPatch,
		Operation:             "apply",
		Status:                "completed",
		TargetPaths:           []string{"README.md"},
		PatchHash:             "abc123",
		PatchSummary:          "2 patch line(s)",
		BeforeSnapshotSummary: "exists, 1 line(s), 4 byte(s), sha256:oldhash123456",
		AfterSnapshotSummary:  "exists, 1 line(s), 4 byte(s), sha256:newhash123456",
		RollbackPayload: []session.WriteExecutionFileSnapshot{{
			Path:          "README.md",
			BeforeExists:  true,
			BeforeContent: "old\n",
			BeforeHash:    "oldhash123456",
			AfterExists:   true,
			AfterHash:     "newhash123456",
		}},
		ResultSummary: "applied patch to README.md: updated 2 line(s)",
	})
	require.NoError(t, err)

	writeExecutions, err := service.WriteExecutions(context.Background(), thread.ID)
	require.NoError(t, err)
	require.Len(t, writeExecutions, 1)

	rollbackTask, err := service.CreateTask(context.Background(), thread.ID, runtimecontract.CreateTaskRequest{
		Title: "Rollback latest",
		Kind:  runner.KindWorkspaceApplyPatchRollback,
		Input: `{"writeExecutionId":"` + writeExecutions[0].ID + `"}`,
	})
	require.NoError(t, err)
	require.Equal(t, "needs_approval", rollbackTask.Status)
	require.Equal(t, "pending", rollbackTask.ApprovalStatus)
	require.Contains(t, rollbackTask.ResultSummary, "approval required for rollback of README.md")
}

func TestServiceTaskDescriptorExposesAgentPlanMetadata(t *testing.T) {
	projectRoot := t.TempDir()
	store, err := state.Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	sessions, err := session.NewRegistryWithStore(projectRoot, store)
	require.NoError(t, err)

	service := NewService(
		"0.1.0",
		skill.Codex,
		policy.DefaultMode(),
		projectRoot,
		tool.NewRegistry(),
		skill.NewManager(nil),
		mcp.NewManager(nil),
		provider.NewRegistry(""),
		sessions,
	)

	thread, err := service.CreateThread(context.Background(), runtimecontract.CreateThreadRequest{
		Name:           "Agent Descriptor",
		PermissionMode: "workspace-write",
	})
	require.NoError(t, err)

	agentState := `{"taskId":"task-1","threadId":"thread-1","stepIndex":2,"maxSteps":4,"waitingChildTaskId":"task-2","lastAction":{"type":"read_files_batch","reasoningSummary":"Read selected files"},"status":"running","goal":"Inspect files","plan":{"summary":"Filter matching files first, then read the selected files, then answer.","mode":"filter_then_read","steps":[{"title":"Filter matching files","expectedActionTypes":["list_files_filtered"]},{"title":"Read the selected files","expectedActionTypes":["read_files_batch","read_file"]},{"title":"Answer with the findings","expectedActionTypes":["respond"]}],"requiredSequence":["list_files_filtered","read_files_batch|read_file","respond"]},"currentStepTitle":"Answer with the findings","lastReasoning":"Read selected files","completedActions":["list_files_filtered","read_files_batch"]}`
		_, ok := sessions.CreateTask(thread.ID, session.CreateTaskInput{
		Title:      "Agent run",
		Kind:       runner.KindAgentRun,
		Input:      `{"goal":"Inspect files","maxSteps":4}`,
		Status:     "running",
		AgentState: agentState,
	})
	require.True(t, ok)

	tasks, err := service.Tasks(context.Background(), thread.ID)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, "Filter matching files first, then read the selected files, then answer.", tasks[0].AgentPlanSummary)
	require.Equal(t, "filter_then_read", tasks[0].AgentPlanMode)
	require.Equal(t, "Answer with the findings", tasks[0].AgentCurrentStepTitle)
	require.Equal(t, "Read selected files", tasks[0].AgentLastReasoning)
	require.Equal(t, "task-2", tasks[0].LatestChildTaskID)
}

func TestServiceCreateAgentTaskSeedsPlanMode(t *testing.T) {
	projectRoot := t.TempDir()
	store, err := state.Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	sessions, err := session.NewRegistryWithStore(projectRoot, store)
	require.NoError(t, err)

	service := NewService(
		"0.1.0",
		skill.Codex,
		policy.DefaultMode(),
		projectRoot,
		tool.NewRegistry(),
		skill.NewManager(nil),
		mcp.NewManager(nil),
		provider.NewRegistry(""),
		sessions,
	)

	thread, err := service.CreateThread(context.Background(), runtimecontract.CreateThreadRequest{
		Name:           "Agent Seeded Plan",
		PermissionMode: "workspace-write",
	})
	require.NoError(t, err)

	task, err := service.CreateTask(context.Background(), thread.ID, runtimecontract.CreateTaskRequest{
		Title: "Agent run",
		Kind:  runner.KindAgentRun,
		Input: `{"goal":"先确认 README.md 是否存在和 metadata，再读取内容并回答","maxSteps":5}`,
	})
	require.NoError(t, err)
	require.Equal(t, "stat_then_read", task.AgentPlanMode)
	require.Equal(t, "Check file status first, then read content if needed, then answer.", task.AgentPlanSummary)
	require.Equal(t, "Check file status", task.AgentCurrentStepTitle)
}

func TestTaskDescriptorAgentWorkflowSummary(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)

	t.Run("waiting for approval prefers explicit runtime summary", func(t *testing.T) {
		descriptor := toTaskDescriptor(session.Task{
			ID:             "task-agent",
			ThreadID:       "thread-1",
			Title:          "Agent run",
			Status:         "waiting_for_approval",
			Kind:           runner.KindAgentRun,
			ResultSummary:  "agent step 2/4: waiting for approval for child task task-child",
			WaitingStatus:  "waiting_for_approval",
			CreatedAt:      now,
			UpdatedAt:      now,
			AgentState:     runner.MarshalAgentRunStateForRuntime(runner.AgentRunState{TaskID: "task-agent", ThreadID: "thread-1", StepIndex: 2, MaxSteps: 4, WaitingChildTaskID: "task-child", Status: "waiting_for_approval"}),
		})

		require.Equal(t, "task-child", descriptor.LatestChildTaskID)
		require.Equal(t, "waiting_for_approval", descriptor.WaitingStatus)
		require.Equal(t, 2, descriptor.AgentStep)
		require.Equal(t, 4, descriptor.AgentMaxSteps)
		require.Equal(t, "agent step 2/4: waiting for approval for child task task-child", descriptor.ResultSummary)
	})

	t.Run("waiting for child task derives explicit summary when empty", func(t *testing.T) {
		descriptor := toTaskDescriptor(session.Task{
			ID:            "task-agent",
			ThreadID:      "thread-1",
			Title:         "Agent run",
			Status:        "waiting_for_task",
			Kind:          runner.KindAgentRun,
			WaitingStatus: "waiting_for_task",
			CreatedAt:     now,
			UpdatedAt:     now,
			AgentState: runner.MarshalAgentRunStateForRuntime(runner.AgentRunState{
				TaskID:             "task-agent",
				ThreadID:           "thread-1",
				StepIndex:          1,
				MaxSteps:           4,
				WaitingChildTaskID: "task-child",
				Status:             "waiting_for_task",
			}),
		})

		require.Equal(t, "task-child", descriptor.LatestChildTaskID)
		require.Equal(t, 1, descriptor.AgentStep)
		require.Equal(t, 4, descriptor.AgentMaxSteps)
		require.Equal(t, "agent step 1/4: waiting for child task task-child", descriptor.ResultSummary)
	})

	t.Run("completed parent keeps final result summary", func(t *testing.T) {
		descriptor := toTaskDescriptor(session.Task{
			ID:            "task-agent",
			ThreadID:      "thread-1",
			Title:         "Agent run",
			Status:        "completed",
			Kind:          runner.KindAgentRun,
			ResultSummary: "agent completed: updated README and summarized the change",
			CreatedAt:     now,
			UpdatedAt:     now,
			AgentState: runner.MarshalAgentRunStateForRuntime(runner.AgentRunState{
				TaskID:    "task-agent",
				ThreadID:  "thread-1",
				StepIndex: 3,
				MaxSteps:  4,
				Status:    "completed",
			}),
		})

		require.Equal(t, "agent completed: updated README and summarized the change", descriptor.ResultSummary)
		require.Equal(t, 3, descriptor.AgentStep)
		require.Equal(t, 4, descriptor.AgentMaxSteps)
	})

	t.Run("queued parent falls back to explicit queued summary", func(t *testing.T) {
		descriptor := toTaskDescriptor(session.Task{
			ID:        "task-agent",
			ThreadID:  "thread-1",
			Title:     "Agent run",
			Status:    "queued",
			Kind:      runner.KindAgentRun,
			CreatedAt: now,
			UpdatedAt: now,
			AgentState: runner.MarshalAgentRunStateForRuntime(runner.AgentRunState{
				TaskID:    "task-agent",
				ThreadID:  "thread-1",
				StepIndex: 0,
				MaxSteps:  4,
				Status:    "queued",
			}),
		})

		require.Equal(t, "agent queued", descriptor.ResultSummary)
		require.Equal(t, 0, descriptor.AgentStep)
		require.Equal(t, 4, descriptor.AgentMaxSteps)
	})
}
