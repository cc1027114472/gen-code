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
	"llmtrace/internal/core/tool"
)

func TestServiceContractShapesExposeStructuredMetadata(t *testing.T) {
	registry := tool.NewRegistry()
	registry.Register(tool.Descriptor{
		ID:                 "bridge.check",
		Name:               "Bridge Check",
		Description:        "Verify the bridge",
		InputSchemaSummary: "No input",
		PermissionMode:     policy.AskUser,
		Source:             "runtime",
	})

	service := NewService(
		"0.1.0",
		skill.Codex,
		policy.DefaultMode(),
		`D:\GOWorks\gen-code-heji\gen-code`,
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
		session.NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`),
	)

	created, err := service.CreateThread(context.Background(), runtimecontract.CreateThreadRequest{
		Name:           "Thread 1",
		PermissionMode: "ask-user",
	})
	require.NoError(t, err)
	require.Equal(t, "thread-1", created.ID)

	task, err := service.CreateTask(context.Background(), created.ID, runtimecontract.CreateTaskRequest{
		Title: "Draft spec",
	})
	require.NoError(t, err)
	require.Equal(t, task.CreatedAt, task.UpdatedAt)

	updatedTask, err := service.UpdateTaskStatus(context.Background(), created.ID, task.ID, runtimecontract.UpdateTaskStatusRequest{
		Status: "running",
	})
	require.NoError(t, err)
	require.Equal(t, "running", updatedTask.Status)
	require.NotEmpty(t, updatedTask.UpdatedAt)

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
	require.Equal(t, "gen-code", status.WorkspaceID)
	require.Equal(t, `D:\GOWorks\gen-code-heji\gen-code`, status.ProjectRoot)
	require.Equal(t, 1, status.ThreadCount)
	require.Equal(t, "thread-1", status.ActiveThreadID)

	tasks, err := service.Tasks(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, "running", tasks[0].Status)
	require.NotEmpty(t, tasks[0].UpdatedAt)

	stream, err := service.StreamEvents(context.Background(), created.ID)
	require.NoError(t, err)
	streamed := make([]runtimecontract.EventDescriptor, 0)
	for item := range stream {
		streamed = append(streamed, item)
	}
	require.NotEmpty(t, streamed)
	require.Equal(t, created.ID, streamed[0].ThreadID)
}
