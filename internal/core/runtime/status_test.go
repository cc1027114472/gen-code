package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/core/mcp"
	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/session"
	"llmtrace/internal/core/skill"
	"llmtrace/internal/core/state"
	"llmtrace/internal/core/tool"
)

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
	})

	sessions, err := session.NewRegistryWithStore(projectRoot, store)
	require.NoError(t, err)

	service := NewService(
		"0.1.0",
		skill.Codex,
		policy.DefaultMode(),
		projectRoot,
		registry,
		skill.NewManager([]skill.Descriptor{
			{ID: "common.browser", Group: skill.Common, Name: "Browser", Description: "Reusable browser skill"},
			{ID: "codex.review", Group: skill.Codex, Name: "Review", Description: "Codex review skill"},
			{ID: "cc.swarm", Group: skill.CC, Name: "Swarm", Description: "CC swarm skill"},
		}),
		mcp.NewManager([]mcp.ServerDescriptor{{
			ID:            "server-1",
			Source:        "node_modules",
			Enabled:       true,
			ToolCount:     2,
			ResourceCount: 3,
		}}),
		sessions,
	)

	created, err := service.CreateThread(context.Background(), runtimecontract.CreateThreadRequest{
		Name:           "Thread 1",
		PermissionMode: "ask-user",
	})
	require.NoError(t, err)
	require.Equal(t, "thread-1", created.ID)

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

	tools, err := service.Tools(context.Background())
	require.NoError(t, err)
	require.Len(t, tools, 1)
	require.Equal(t, "ask-user", tools[0].Permission)
	require.Equal(t, "runtime", tools[0].Source)

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
	require.Equal(t, sessions.Workspace().ID, status.WorkspaceID)
	require.Equal(t, projectRoot, status.ProjectRoot)
	require.Equal(t, 1, status.ThreadCount)
	require.Equal(t, "thread-1", status.ActiveThreadID)
	require.Equal(t, state.StoreName, status.StateStore)
	require.Equal(t, state.PathForProject(projectRoot), status.StatePath)

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

	stream, err := service.StreamEvents(context.Background(), created.ID)
	require.NoError(t, err)
	streamed := make([]runtimecontract.EventDescriptor, 0)
	for item := range stream {
		streamed = append(streamed, item)
	}
	require.NotEmpty(t, streamed)
	require.Equal(t, created.ID, streamed[0].ThreadID)
}
