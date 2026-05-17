package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/core/runtime"
	"llmtrace/internal/core/skill"
)

const defaultRuntimeBaseURL = "http://127.0.0.1:10008"

var sharedFallbackService = runtime.NewDefaultServiceWithoutRecovery()

type runtimeFacade struct {
	service *runtime.Service
	client  *remoteRuntimeClient
	source  string
}

type remoteRuntimeClient struct {
	baseURL string
	client  http.Client
}

type apiEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type taskCreateInputOptions struct {
	Input       string
	InputFile   string
	PatchFile   string
	PatchPath   string
	InputSource string
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	runtimeFacade := newRuntimeFacade()

	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "doctor":
		return runDoctor(ctx, runtimeFacade)
	case "runtime":
		if len(args) < 2 || args[1] != "status" {
			return errors.New("usage: gen-code runtime status")
		}
		return printRuntimeStatus(ctx, runtimeFacade)
	case "workspace":
		if len(args) < 2 || args[1] != "show" {
			return errors.New("usage: gen-code workspace show")
		}
		return printWorkspace(ctx, runtimeFacade)
	case "threads":
		if len(args) < 2 {
			return errors.New("usage: gen-code threads <list|create|activate|messages|message-add|tool-calls|artifacts|approvals|write-executions>")
		}
		switch args[1] {
		case "list":
			return printThreads(ctx, runtimeFacade)
		case "create":
			var name, model, permission string
			for _, arg := range args[2:] {
				switch {
				case strings.HasPrefix(arg, "--name="):
					name = strings.TrimSpace(strings.TrimPrefix(arg, "--name="))
				case strings.HasPrefix(arg, "--model="):
					model = strings.TrimSpace(strings.TrimPrefix(arg, "--model="))
				case strings.HasPrefix(arg, "--permission="):
					permission = strings.TrimSpace(strings.TrimPrefix(arg, "--permission="))
				}
			}
			return createThread(ctx, runtimeFacade, name, model, permission)
		case "activate":
			var id string
			for _, arg := range args[2:] {
				if strings.HasPrefix(arg, "--id=") {
					id = strings.TrimSpace(strings.TrimPrefix(arg, "--id="))
				}
			}
			if id == "" {
				return errors.New("usage: gen-code threads activate --id=<threadId>")
			}
			return activateThread(ctx, runtimeFacade, id)
		case "messages":
			var id string
			for _, arg := range args[2:] {
				if strings.HasPrefix(arg, "--id=") {
					id = strings.TrimSpace(strings.TrimPrefix(arg, "--id="))
				}
			}
			if id == "" {
				return errors.New("usage: gen-code threads messages --id=<threadId>")
			}
			return printMessages(ctx, runtimeFacade, id)
		case "message-add":
			var id, role, content string
			for _, arg := range args[2:] {
				switch {
				case strings.HasPrefix(arg, "--id="):
					id = strings.TrimSpace(strings.TrimPrefix(arg, "--id="))
				case strings.HasPrefix(arg, "--role="):
					role = strings.TrimSpace(strings.TrimPrefix(arg, "--role="))
				case strings.HasPrefix(arg, "--content="):
					content = strings.TrimSpace(strings.TrimPrefix(arg, "--content="))
				}
			}
			if id == "" || role == "" || content == "" {
				return errors.New("usage: gen-code threads message-add --id=<threadId> --role=<role> --content=...")
			}
			return appendMessage(ctx, runtimeFacade, id, role, content)
		case "tool-calls":
			var id string
			for _, arg := range args[2:] {
				if strings.HasPrefix(arg, "--id=") {
					id = strings.TrimSpace(strings.TrimPrefix(arg, "--id="))
				}
			}
			if id == "" {
				return errors.New("usage: gen-code threads tool-calls --id=<threadId>")
			}
			return printToolCalls(ctx, runtimeFacade, id)
		case "artifacts":
			var id string
			for _, arg := range args[2:] {
				if strings.HasPrefix(arg, "--id=") {
					id = strings.TrimSpace(strings.TrimPrefix(arg, "--id="))
				}
			}
			if id == "" {
				return errors.New("usage: gen-code threads artifacts --id=<threadId>")
			}
			return printArtifacts(ctx, runtimeFacade, id)
		case "approvals":
			var id string
			for _, arg := range args[2:] {
				if strings.HasPrefix(arg, "--id=") {
					id = strings.TrimSpace(strings.TrimPrefix(arg, "--id="))
				}
			}
			if id == "" {
				return errors.New("usage: gen-code threads approvals --id=<threadId>")
			}
			return printApprovals(ctx, runtimeFacade, id)
		case "write-executions":
			var id string
			for _, arg := range args[2:] {
				if strings.HasPrefix(arg, "--id=") {
					id = strings.TrimSpace(strings.TrimPrefix(arg, "--id="))
				}
			}
			if id == "" {
				return errors.New("usage: gen-code threads write-executions --id=<threadId>")
			}
			return printWriteExecutions(ctx, runtimeFacade, id)
		default:
			return errors.New("usage: gen-code threads <list|create|activate|messages|message-add|tool-calls|artifacts|approvals|write-executions>")
		}
	case "tasks":
		if len(args) < 2 {
			return errors.New("usage: gen-code tasks <list|create|run|approve|reject|update-status>")
		}
		switch args[1] {
		case "list":
			var threadID string
			for _, arg := range args[2:] {
				if strings.HasPrefix(arg, "--thread=") {
					threadID = strings.TrimSpace(strings.TrimPrefix(arg, "--thread="))
				}
			}
			if threadID == "" {
				return errors.New("usage: gen-code tasks list --thread=<threadId>")
			}
			return printTasks(ctx, runtimeFacade, threadID)
		case "create":
			var threadID, title, kind string
			options := taskCreateInputOptions{}
			for _, arg := range args[2:] {
				switch {
				case strings.HasPrefix(arg, "--thread="):
					threadID = strings.TrimSpace(strings.TrimPrefix(arg, "--thread="))
				case strings.HasPrefix(arg, "--title="):
					title = strings.TrimSpace(strings.TrimPrefix(arg, "--title="))
				case strings.HasPrefix(arg, "--kind="):
					kind = strings.TrimSpace(strings.TrimPrefix(arg, "--kind="))
				case strings.HasPrefix(arg, "--input="):
					options.Input = strings.TrimSpace(strings.TrimPrefix(arg, "--input="))
				case strings.HasPrefix(arg, "--input-file="):
					options.InputFile = strings.TrimSpace(strings.TrimPrefix(arg, "--input-file="))
				case strings.HasPrefix(arg, "--patch-file="):
					options.PatchFile = strings.TrimSpace(strings.TrimPrefix(arg, "--patch-file="))
				case strings.HasPrefix(arg, "--path="):
					options.PatchPath = strings.TrimSpace(strings.TrimPrefix(arg, "--path="))
				}
			}
			if threadID == "" || kind == "" {
				return errors.New("usage: gen-code tasks create --thread=<threadId> --kind=<kind> [--title=...] [--input=...] [--input-file=<path>] [--patch-file=<path> --path=<workspace-relative-path>]")
			}
			return createTask(ctx, runtimeFacade, threadID, title, kind, options)
		case "run":
			var threadID, taskID string
			for _, arg := range args[2:] {
				switch {
				case strings.HasPrefix(arg, "--thread="):
					threadID = strings.TrimSpace(strings.TrimPrefix(arg, "--thread="))
				case strings.HasPrefix(arg, "--task="):
					taskID = strings.TrimSpace(strings.TrimPrefix(arg, "--task="))
				}
			}
			if threadID == "" || taskID == "" {
				return errors.New("usage: gen-code tasks run --thread=<threadId> --task=<taskId>")
			}
			return runTask(ctx, runtimeFacade, threadID, taskID)
		case "approve":
			var threadID, taskID string
			for _, arg := range args[2:] {
				switch {
				case strings.HasPrefix(arg, "--thread="):
					threadID = strings.TrimSpace(strings.TrimPrefix(arg, "--thread="))
				case strings.HasPrefix(arg, "--task="):
					taskID = strings.TrimSpace(strings.TrimPrefix(arg, "--task="))
				}
			}
			if threadID == "" || taskID == "" {
				return errors.New("usage: gen-code tasks approve --thread=<threadId> --task=<taskId>")
			}
			return approveTask(ctx, runtimeFacade, threadID, taskID)
		case "reject":
			var threadID, taskID string
			for _, arg := range args[2:] {
				switch {
				case strings.HasPrefix(arg, "--thread="):
					threadID = strings.TrimSpace(strings.TrimPrefix(arg, "--thread="))
				case strings.HasPrefix(arg, "--task="):
					taskID = strings.TrimSpace(strings.TrimPrefix(arg, "--task="))
				}
			}
			if threadID == "" || taskID == "" {
				return errors.New("usage: gen-code tasks reject --thread=<threadId> --task=<taskId>")
			}
			return rejectTask(ctx, runtimeFacade, threadID, taskID)
		case "update-status":
			var threadID, taskID, status string
			for _, arg := range args[2:] {
				switch {
				case strings.HasPrefix(arg, "--thread="):
					threadID = strings.TrimSpace(strings.TrimPrefix(arg, "--thread="))
				case strings.HasPrefix(arg, "--task="):
					taskID = strings.TrimSpace(strings.TrimPrefix(arg, "--task="))
				case strings.HasPrefix(arg, "--status="):
					status = strings.TrimSpace(strings.TrimPrefix(arg, "--status="))
				}
			}
			if threadID == "" || taskID == "" || status == "" {
				return errors.New("usage: gen-code tasks update-status --thread=<threadId> --task=<taskId> --status=<status>")
			}
			return updateTaskStatus(ctx, runtimeFacade, threadID, taskID, status)
		default:
			return errors.New("usage: gen-code tasks <list|create|run|approve|reject|update-status>")
		}
	case "agent":
		if len(args) < 2 || args[1] != "run" {
			return errors.New("usage: gen-code agent run --thread=<threadId> --goal=... [--provider=...] [--model=...] [--max-steps=...] [--max-output-tokens=...] [--title=...]")
		}
		var threadID, goal, providerKind, modelName, title string
		maxSteps := 0
		maxOutputTokens := 0
		for _, arg := range args[2:] {
			switch {
			case strings.HasPrefix(arg, "--thread="):
				threadID = strings.TrimSpace(strings.TrimPrefix(arg, "--thread="))
			case strings.HasPrefix(arg, "--goal="):
				goal = strings.TrimSpace(strings.TrimPrefix(arg, "--goal="))
			case strings.HasPrefix(arg, "--provider="):
				providerKind = strings.TrimSpace(strings.TrimPrefix(arg, "--provider="))
			case strings.HasPrefix(arg, "--model="):
				modelName = strings.TrimSpace(strings.TrimPrefix(arg, "--model="))
			case strings.HasPrefix(arg, "--max-steps="):
				value, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(arg, "--max-steps=")))
				if err != nil {
					return fmt.Errorf("invalid --max-steps: %w", err)
				}
				maxSteps = value
			case strings.HasPrefix(arg, "--max-output-tokens="):
				value, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(arg, "--max-output-tokens=")))
				if err != nil {
					return fmt.Errorf("invalid --max-output-tokens: %w", err)
				}
				maxOutputTokens = value
			case strings.HasPrefix(arg, "--title="):
				title = strings.TrimSpace(strings.TrimPrefix(arg, "--title="))
			}
		}
		if threadID == "" || goal == "" {
			return errors.New("usage: gen-code agent run --thread=<threadId> --goal=... [--provider=...] [--model=...] [--max-steps=...] [--max-output-tokens=...] [--title=...]")
		}
		return runAgentTask(ctx, runtimeFacade, threadID, title, goal, providerKind, modelName, maxSteps, maxOutputTokens)
	case "model":
		if len(args) < 2 || args[1] != "run" {
			return errors.New("usage: gen-code model run --thread=<threadId> --input=... [--provider=...] [--model=...] [--max-output-tokens=...] [--title=...]")
		}
		var threadID, input, providerKind, modelName, title string
		var maxOutputTokens int
		for _, arg := range args[2:] {
			switch {
			case strings.HasPrefix(arg, "--thread="):
				threadID = strings.TrimSpace(strings.TrimPrefix(arg, "--thread="))
			case strings.HasPrefix(arg, "--input="):
				input = strings.TrimSpace(strings.TrimPrefix(arg, "--input="))
			case strings.HasPrefix(arg, "--provider="):
				providerKind = strings.TrimSpace(strings.TrimPrefix(arg, "--provider="))
			case strings.HasPrefix(arg, "--model="):
				modelName = strings.TrimSpace(strings.TrimPrefix(arg, "--model="))
			case strings.HasPrefix(arg, "--title="):
				title = strings.TrimSpace(strings.TrimPrefix(arg, "--title="))
			case strings.HasPrefix(arg, "--max-output-tokens="):
				value := strings.TrimSpace(strings.TrimPrefix(arg, "--max-output-tokens="))
				if value != "" {
					parsed, err := strconv.Atoi(value)
					if err != nil {
						return fmt.Errorf("invalid --max-output-tokens value: %w", err)
					}
					maxOutputTokens = parsed
				}
			}
		}
		if threadID == "" || input == "" {
			return errors.New("usage: gen-code model run --thread=<threadId> --input=... [--provider=...] [--model=...] [--max-output-tokens=...] [--title=...]")
		}
		return runModelTask(ctx, runtimeFacade, threadID, title, providerKind, modelName, input, maxOutputTokens)
	case "rollback":
		if len(args) < 2 || args[1] != "latest" {
			return errors.New("usage: gen-code rollback latest --thread=<threadId>")
		}
		var threadID string
		for _, arg := range args[2:] {
			if strings.HasPrefix(arg, "--thread=") {
				threadID = strings.TrimSpace(strings.TrimPrefix(arg, "--thread="))
			}
		}
		if threadID == "" {
			return errors.New("usage: gen-code rollback latest --thread=<threadId>")
		}
		return rollbackLatest(ctx, runtimeFacade, threadID)
	case "skills":
		if len(args) < 2 || args[1] != "list" {
			return errors.New("usage: gen-code skills list [--group=<group>]")
		}
		group := ""
		for _, arg := range args[2:] {
			if strings.HasPrefix(arg, "--group=") {
				group = strings.TrimSpace(strings.TrimPrefix(arg, "--group="))
			}
		}
		return printSkills(runtimeFacade.service, group)
	case "tools":
		if len(args) < 2 || args[1] != "list" {
			return errors.New("usage: gen-code tools list")
		}
		return printTools(ctx, runtimeFacade)
	case "mcp":
		if len(args) < 2 || args[1] != "list" {
			return errors.New("usage: gen-code mcp list")
		}
		return printMCP(ctx, runtimeFacade)
	case "providers":
		if len(args) < 2 || args[1] != "list" {
			return errors.New("usage: gen-code providers list")
		}
		return printProviders(ctx, runtimeFacade)
	case "provider":
		if len(args) < 2 || args[1] != "probe" {
			return errors.New("usage: gen-code provider probe --kind=<provider>")
		}
		var kind string
		for _, arg := range args[2:] {
			if strings.HasPrefix(arg, "--kind=") {
				kind = strings.TrimSpace(strings.TrimPrefix(arg, "--kind="))
			}
		}
		if kind == "" {
			return errors.New("usage: gen-code provider probe --kind=<provider>")
		}
		return probeProvider(ctx, runtimeFacade, kind)
	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", strings.Join(args, " "))
	}
}

func printUsage() {
	fmt.Println("gen-code commands:")
	fmt.Println("  primary workflow: agent run --thread=<threadId> --goal=... [--provider=...] [--model=...] [--max-steps=...] [--max-output-tokens=...] [--title=...]")
	fmt.Println("  inspect agent progress: tasks list --thread=<threadId>")
	fmt.Println("  doctor")
	fmt.Println("  runtime status")
	fmt.Println("  workspace show")
	fmt.Println("  threads list")
	fmt.Println("  threads create [--name=...] [--model=...] [--permission=...]")
	fmt.Println("  threads activate --id=<threadId>")
	fmt.Println("  threads messages --id=<threadId>")
	fmt.Println("  threads message-add --id=<threadId> --role=<role> --content=...")
	fmt.Println("  threads tool-calls --id=<threadId>")
	fmt.Println("  threads artifacts --id=<threadId>")
	fmt.Println("  threads approvals --id=<threadId>")
	fmt.Println("  threads write-executions --id=<threadId>")
	fmt.Println("  tasks list --thread=<threadId>")
	fmt.Println("  tasks create --thread=<threadId> --kind=<kind> [--title=...] [--input=...] [--input-file=<path>] [--patch-file=<path> --path=<workspace-relative-path>]")
	fmt.Println("  tasks run --thread=<threadId> --task=<taskId>")
	fmt.Println("  tasks approve --thread=<threadId> --task=<taskId>")
	fmt.Println("  tasks reject --thread=<threadId> --task=<taskId>")
	fmt.Println("  tasks update-status --thread=<threadId> --task=<taskId> --status=<status>")
	fmt.Println("  agent run --thread=<threadId> --goal=... [--provider=...] [--model=...] [--max-steps=...] [--max-output-tokens=...] [--title=...]")
	fmt.Println("  model run --thread=<threadId> --input=... [--provider=...] [--model=...] [--max-output-tokens=...] [--title=...]")
	fmt.Println("  rollback latest --thread=<threadId>")
	fmt.Println("  skills list [--group=<group>]")
	fmt.Println("  tools list")
	fmt.Println("  mcp list")
	fmt.Println("  providers list")
	fmt.Println("  provider probe --kind=<provider>")
}

func newRuntimeFacade() *runtimeFacade {
	return &runtimeFacade{
		service: sharedFallbackService,
		client: &remoteRuntimeClient{
			baseURL: strings.TrimRight(runtimeBaseURL(), "/"),
			client: http.Client{
				Timeout: 90 * time.Second,
			},
		},
		source: "local-fallback",
	}
}

func (f *runtimeFacade) runtimeSource() string {
	return f.source
}

func (f *runtimeFacade) status(ctx context.Context) (runtimecontract.Status, error) {
	if status, err := f.client.status(); err == nil {
		f.source = "remote-app-server"
		return status, nil
	}
	f.source = "local-fallback"
	status, err := f.service.Status(ctx)
	if err == nil {
		status.RuntimeSource = f.source
	}
	return status, err
}

func (f *runtimeFacade) workspace(ctx context.Context) (runtimecontract.WorkspaceDescriptor, error) {
	if item, err := f.client.workspace(); err == nil {
		f.source = "remote-app-server"
		return item, nil
	}
	f.source = "local-fallback"
	return f.service.Workspace(ctx)
}

func (f *runtimeFacade) threads(ctx context.Context) ([]runtimecontract.ThreadDescriptor, error) {
	if items, err := f.client.threads(); err == nil {
		f.source = "remote-app-server"
		return items, nil
	}
	f.source = "local-fallback"
	return f.service.Threads(ctx)
}

func (f *runtimeFacade) createThread(ctx context.Context, request runtimecontract.CreateThreadRequest) (runtimecontract.ThreadDescriptor, error) {
	if item, err := f.client.createThread(request); err == nil {
		f.source = "remote-app-server"
		return item, nil
	}
	f.source = "local-fallback"
	return f.service.CreateThread(ctx, request)
}

func (f *runtimeFacade) activateThread(ctx context.Context, id string) (runtimecontract.ThreadDescriptor, error) {
	if item, err := f.client.activateThread(id); err == nil {
		f.source = "remote-app-server"
		return item, nil
	}
	f.source = "local-fallback"
	return f.service.ActivateThread(ctx, id)
}

func (f *runtimeFacade) tasks(ctx context.Context, threadID string) ([]runtimecontract.TaskDescriptor, error) {
	if items, err := f.client.tasks(threadID); err == nil {
		f.source = "remote-app-server"
		return items, nil
	}
	f.source = "local-fallback"
	return f.service.Tasks(ctx, threadID)
}

func (f *runtimeFacade) createTask(ctx context.Context, threadID string, request runtimecontract.CreateTaskRequest) (runtimecontract.TaskDescriptor, error) {
	if item, err := f.client.createTask(threadID, request); err == nil {
		f.source = "remote-app-server"
		return item, nil
	}
	f.source = "local-fallback"
	return f.service.CreateTask(ctx, threadID, request)
}

func (f *runtimeFacade) runTask(ctx context.Context, threadID string, taskID string, request runtimecontract.RunTaskRequest) (runtimecontract.TaskDescriptor, error) {
	if item, err := f.client.runTask(threadID, taskID, request); err == nil {
		f.source = "remote-app-server"
		return item, nil
	}
	f.source = "local-fallback"
	return f.service.RunTask(ctx, threadID, taskID, request)
}

func (f *runtimeFacade) approvals(ctx context.Context, threadID string) ([]runtimecontract.ApprovalDescriptor, error) {
	if items, err := f.client.approvals(threadID); err == nil {
		f.source = "remote-app-server"
		return items, nil
	}
	f.source = "local-fallback"
	return f.service.Approvals(ctx, threadID)
}

func (f *runtimeFacade) writeExecutions(ctx context.Context, threadID string) ([]runtimecontract.WriteExecutionDescriptor, error) {
	if items, err := f.client.writeExecutions(threadID); err == nil {
		f.source = "remote-app-server"
		return items, nil
	}
	f.source = "local-fallback"
	return f.service.WriteExecutions(ctx, threadID)
}

func (f *runtimeFacade) approveTask(ctx context.Context, threadID string, taskID string, request runtimecontract.ApproveTaskRequest) (runtimecontract.TaskDescriptor, error) {
	if item, err := f.client.approveTask(threadID, taskID, request); err == nil {
		f.source = "remote-app-server"
		return item, nil
	}
	f.source = "local-fallback"
	return f.service.ApproveTask(ctx, threadID, taskID, request)
}

func (f *runtimeFacade) rejectTask(ctx context.Context, threadID string, taskID string, request runtimecontract.RejectTaskRequest) (runtimecontract.TaskDescriptor, error) {
	if item, err := f.client.rejectTask(threadID, taskID, request); err == nil {
		f.source = "remote-app-server"
		return item, nil
	}
	f.source = "local-fallback"
	return f.service.RejectTask(ctx, threadID, taskID, request)
}

func (f *runtimeFacade) updateTaskStatus(ctx context.Context, threadID string, taskID string, request runtimecontract.UpdateTaskStatusRequest) (runtimecontract.TaskDescriptor, error) {
	if item, err := f.client.updateTaskStatus(threadID, taskID, request); err == nil {
		f.source = "remote-app-server"
		return item, nil
	}
	f.source = "local-fallback"
	return f.service.UpdateTaskStatus(ctx, threadID, taskID, request)
}

func (f *runtimeFacade) messages(ctx context.Context, threadID string) ([]runtimecontract.MessageDescriptor, error) {
	if items, err := f.client.messages(threadID); err == nil {
		f.source = "remote-app-server"
		return items, nil
	}
	f.source = "local-fallback"
	return f.service.Messages(ctx, threadID)
}

func (f *runtimeFacade) appendMessage(ctx context.Context, threadID string, request runtimecontract.CreateMessageRequest) (runtimecontract.MessageDescriptor, error) {
	if item, err := f.client.appendMessage(threadID, request); err == nil {
		f.source = "remote-app-server"
		return item, nil
	}
	f.source = "local-fallback"
	return f.service.AppendMessage(ctx, threadID, request)
}

func (f *runtimeFacade) toolCalls(ctx context.Context, threadID string) ([]runtimecontract.ToolCallDescriptor, error) {
	if items, err := f.client.toolCalls(threadID); err == nil {
		f.source = "remote-app-server"
		return items, nil
	}
	f.source = "local-fallback"
	return f.service.ToolCalls(ctx, threadID)
}

func (f *runtimeFacade) artifacts(ctx context.Context, threadID string) ([]runtimecontract.ArtifactDescriptor, error) {
	if items, err := f.client.artifacts(threadID); err == nil {
		f.source = "remote-app-server"
		return items, nil
	}
	f.source = "local-fallback"
	return f.service.Artifacts(ctx, threadID)
}

func (f *runtimeFacade) tools(ctx context.Context) ([]runtimecontract.Tool, error) {
	if items, err := f.client.tools(); err == nil {
		f.source = "remote-app-server"
		return items, nil
	}
	f.source = "local-fallback"
	return f.service.Tools(ctx)
}

func (f *runtimeFacade) mcp(ctx context.Context) ([]runtimecontract.MCPServer, error) {
	if items, err := f.client.mcpServers(); err == nil {
		f.source = "remote-app-server"
		return items, nil
	}
	f.source = "local-fallback"
	return f.service.MCPServers(ctx)
}

func (f *runtimeFacade) providers(ctx context.Context) ([]runtimecontract.Provider, error) {
	if items, err := f.client.providers(); err == nil {
		f.source = "remote-app-server"
		return items, nil
	}
	f.source = "local-fallback"
	return f.service.Providers(ctx)
}

func (f *runtimeFacade) probeProvider(ctx context.Context, kind string) (runtimecontract.ProviderProbeResult, error) {
	if item, err := f.client.probeProvider(kind); err == nil {
		f.source = "remote-app-server"
		return item, nil
	}
	f.source = "local-fallback"
	return f.service.ProbeProvider(ctx, kind)
}

func runDoctor(ctx context.Context, facade *runtimeFacade) error {
	status, err := facade.status(ctx)
	if err != nil {
		return err
	}

	tools, err := facade.tools(ctx)
	if err != nil {
		return err
	}

	mcpServers, err := facade.mcp(ctx)
	if err != nil {
		return err
	}

	skillCounts := map[string]int{}
	totalSkills := 0
	for _, groupName := range []string{"common", "codex", "cc"} {
		group, err := skill.ParseGroup(groupName)
		if err != nil {
			return err
		}
		items := facade.service.SkillDescriptors(group)
		count := 0
		for _, item := range items {
			if group != skill.Common && item.Group == skill.Common {
				continue
			}
			count++
		}
		skillCounts[groupName] = count
		totalSkills += count
	}

	fmt.Println("gen-code doctor")
	fmt.Println()
	fmt.Printf("source: %s\n", facade.runtimeSource())

	checks := []struct {
		label  string
		ok     bool
		detail string
	}{
		{"runtime", status.Ready, fallbackText(status.Message, status.State)},
		{"skills", totalSkills > 0, fmt.Sprintf("%d discovered", totalSkills)},
		{"tools", len(tools) > 0, fmt.Sprintf("%d discovered", len(tools))},
		{"mcp", len(mcpServers) > 0, fmt.Sprintf("%d discovered", len(mcpServers))},
	}

	hasWarnings := false
	for _, check := range checks {
		state := "OK"
		if !check.ok {
			state = "WARN"
			hasWarnings = true
		}
		fmt.Printf("[%s] %s: %s\n", state, check.label, fallbackText(check.detail, "not found"))
	}

	fmt.Println()
	fmt.Printf("skill groups: common=%d codex=%d cc=%d\n", skillCounts["common"], skillCounts["codex"], skillCounts["cc"])

	if hasWarnings {
		return errors.New("doctor completed with warnings")
	}

	fmt.Println("All local checks passed.")
	return nil
}

func printRuntimeStatus(ctx context.Context, facade *runtimeFacade) error {
	status, err := facade.status(ctx)
	if err != nil {
		return err
	}

	core := facade.service.FullStatus()
	fmt.Println("runtime status")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source trust: %s\n", fallbackText(status.RuntimeTrust, runtimeSourceTrust(facade.runtimeSource())))
	fmt.Printf("  source detail: %s\n", fallbackText(status.RuntimeSourceDetail, runtimeSourceDetail(facade.runtimeSource())))
	if canonicalTarget := strings.TrimSpace(status.CanonicalRuntimeURL); canonicalTarget != "" {
		fmt.Printf("  canonical runtime target: %s\n", canonicalTarget)
	}
	fmt.Printf("  app version: %s\n", core.AppVersion)
	fmt.Printf("  app server status: %s\n", status.State)
	fmt.Printf("  runtime ready: %t\n", status.Ready)
	fmt.Printf("  runtime message: %s\n", fallbackText(status.Message, "none"))
	fmt.Printf("  desktop shell status: %s\n", core.DesktopShellStatus)
	fmt.Printf("  go bridge status: %s\n", core.GoBridgeStatus)
	fmt.Printf("  state store: %s\n", fallbackText(status.StateStore, "none"))
	fmt.Printf("  state path: %s\n", fallbackText(status.StatePath, "none"))
	fmt.Printf("  workspace id: %s\n", fallbackText(status.WorkspaceID, "none"))
	fmt.Printf("  project root: %s\n", fallbackText(status.ProjectRoot, "none"))
	fmt.Printf("  thread count: %d\n", status.ThreadCount)
	fmt.Printf("  active thread id: %s\n", fallbackText(status.ActiveThreadID, "none"))
	fmt.Printf("  active thread task count: %d\n", status.TaskCount)
	fmt.Printf("  active thread event count: %d\n", status.EventCount)
	fmt.Printf("  active skill group: %s\n", core.ActiveSkillGroup)
	fmt.Printf("  permission mode: %s\n", core.PermissionMode)
	fmt.Printf("  configured mcp server count: %d\n", core.ConfiguredMCPServers)
	fmt.Printf("  skills discovered: %d\n", countSkills(facade.service))
	fmt.Printf("  tools discovered: %d\n", len(facade.service.ToolDescriptors()))
	fmt.Printf("  mcp discovered: %d\n", len(facade.service.MCPDescriptors()))
	return nil
}

func printWorkspace(ctx context.Context, facade *runtimeFacade) error {
	item, err := facade.workspace(ctx)
	if err != nil {
		return err
	}

	fmt.Println("workspace")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  project root: %s\n", item.ProjectRoot)
	fmt.Printf("  shared docs root: %s\n", item.SharedDocsRoot)
	fmt.Printf("  created at: %s\n", item.CreatedAt)
	fmt.Printf("  active thread count: %d\n", item.ActiveThreadCount)
	return nil
}

func printThreads(ctx context.Context, facade *runtimeFacade) error {
	items, err := facade.threads(ctx)
	if err != nil {
		return err
	}

	fmt.Println("threads list")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	for _, item := range items {
		activeFlag := ""
		if item.IsActive {
			activeFlag = ", active"
		}
		fmt.Printf("  - %s (%s, %s%s)\n", item.ID, item.Name, item.PermissionMode, activeFlag)
	}
	return nil
}

func createThread(ctx context.Context, facade *runtimeFacade, name, model, permission string) error {
	item, err := facade.createThread(ctx, runtimecontract.CreateThreadRequest{
		Name:           name,
		ActiveModel:    model,
		PermissionMode: permission,
	})
	if err != nil {
		return err
	}

	fmt.Println("thread created")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  name: %s\n", item.Name)
	fmt.Printf("  active model: %s\n", fallbackText(item.ActiveModel, "none"))
	fmt.Printf("  permission mode: %s\n", item.PermissionMode)
	return nil
}

func activateThread(ctx context.Context, facade *runtimeFacade, id string) error {
	item, err := facade.activateThread(ctx, id)
	if err != nil {
		return err
	}

	fmt.Println("thread activated")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  name: %s\n", item.Name)
	fmt.Printf("  is active: %t\n", item.IsActive)
	return nil
}

func printMessages(ctx context.Context, facade *runtimeFacade, threadID string) error {
	items, err := facade.messages(ctx, threadID)
	if err != nil {
		return err
	}

	fmt.Println("thread messages")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  thread: %s\n", threadID)
	for _, item := range items {
		fmt.Printf("  - %s (%s): %s\n", item.ID, item.Role, item.Content)
	}
	return nil
}

func appendMessage(ctx context.Context, facade *runtimeFacade, threadID string, role string, content string) error {
	item, err := facade.appendMessage(ctx, threadID, runtimecontract.CreateMessageRequest{
		Role:    role,
		Content: content,
	})
	if err != nil {
		return err
	}

	fmt.Println("thread message appended")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  role: %s\n", item.Role)
	fmt.Printf("  content: %s\n", item.Content)
	return nil
}

func printToolCalls(ctx context.Context, facade *runtimeFacade, threadID string) error {
	items, err := facade.toolCalls(ctx, threadID)
	if err != nil {
		return err
	}

	fmt.Println("thread tool calls")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  thread: %s\n", threadID)
	for _, item := range items {
		fmt.Printf("  - %s (%s, %s): %s\n", item.ID, item.ToolID, item.Status, item.Summary)
	}
	return nil
}

func printArtifacts(ctx context.Context, facade *runtimeFacade, threadID string) error {
	items, err := facade.artifacts(ctx, threadID)
	if err != nil {
		return err
	}

	fmt.Println("thread artifacts")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  thread: %s\n", threadID)
	for _, item := range items {
		fmt.Printf("  - %s (%s): %s\n", item.ID, item.Kind, item.Path)
	}
	return nil
}

func printApprovals(ctx context.Context, facade *runtimeFacade, threadID string) error {
	items, err := facade.approvals(ctx, threadID)
	if err != nil {
		return err
	}

	fmt.Println("thread approvals")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  thread: %s\n", threadID)
	for _, item := range items {
		fmt.Printf("  - %s (%s, task=%s, tool=%s, targets=%s): %s\n", item.ID, item.Status, item.TaskID, item.ToolKind, strings.Join(item.TargetPaths, ","), item.Summary)
	}
	return nil
}

func printWriteExecutions(ctx context.Context, facade *runtimeFacade, threadID string) error {
	items, err := facade.writeExecutions(ctx, threadID)
	if err != nil {
		return err
	}

	fmt.Println("thread write executions")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  thread: %s\n", threadID)
	for _, item := range items {
		fmt.Printf(
			"  - %s (%s, op=%s, task=%s, tool=%s, related=%s, targets=%s)\n",
			item.ID,
			item.Status,
			fallbackText(item.Operation, "apply"),
			item.TaskID,
			item.ToolKind,
			fallbackText(item.RelatedExecutionID, "none"),
			strings.Join(item.TargetPaths, ","),
		)
		fmt.Printf("    patch: %s\n", fallbackText(item.PatchSummary, "none"))
		fmt.Printf("    before: %s\n", fallbackText(item.BeforeSummary, "none"))
		fmt.Printf("    after: %s\n", fallbackText(item.AfterSummary, "none"))
		fmt.Printf("    result: %s\n", fallbackText(item.ResultSummary, "none"))
	}
	return nil
}

func printTasks(ctx context.Context, facade *runtimeFacade, threadID string) error {
	items, err := facade.tasks(ctx, threadID)
	if err != nil {
		return err
	}

	fmt.Println("tasks list")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source trust: %s\n", runtimeSourceTrust(facade.runtimeSource()))
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  thread: %s\n", threadID)
	for _, item := range items {
		role := taskWorkflowRole(item)
		waitingReason := taskWaitingReason(item)
		fmt.Printf(
			"  - %s (%s, approval=%s, waiting=%s, parent=%s, kind=%s, updated=%s, result=%s)\n",
			item.ID,
			item.Status,
			fallbackText(item.ApprovalStatus, "none"),
			fallbackText(item.WaitingStatus, "none"),
			fallbackText(item.ParentTaskID, "none"),
			fallbackText(item.Kind, "none"),
			fallbackText(item.UpdatedAt, "none"),
			fallbackText(item.ResultSummary, "none"),
		)
		if role != "" {
			fmt.Printf("    workflow: %s\n", role)
		}
		if waitingReason != "" {
			fmt.Printf("    waiting on: %s\n", waitingReason)
		}
		if item.Kind == "agent.run" {
			fmt.Printf("    plan: %s\n", fallbackText(item.AgentPlanSummary, "none"))
			fmt.Printf("    plan mode: %s\n", fallbackText(item.AgentPlanMode, "none"))
			fmt.Printf("    current step: %s\n", fallbackText(item.AgentCurrentStepTitle, "none"))
			fmt.Printf("    last reasoning: %s\n", fallbackText(item.AgentLastReasoning, "none"))
			fmt.Printf("    progress: %d/%d\n", item.AgentStep, item.AgentMaxSteps)
			fmt.Printf("    latest child: %s\n", fallbackText(item.LatestChildTaskID, "none"))
		}
	}
	return nil
}

func createTask(ctx context.Context, facade *runtimeFacade, threadID string, title string, kind string, options taskCreateInputOptions) error {
	normalizedInput, inputSource, err := resolveTaskCreateInput(kind, options)
	if err != nil {
		return err
	}
	item, err := facade.createTask(ctx, threadID, runtimecontract.CreateTaskRequest{
		Title: title,
		Kind:  kind,
		Input: normalizedInput,
	})
	if err != nil {
		return err
	}

	fmt.Println("task created")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  thread id: %s\n", item.ThreadID)
	fmt.Printf("  title: %s\n", item.Title)
	fmt.Printf("  kind: %s\n", item.Kind)
	fmt.Printf("  status: %s\n", item.Status)
	fmt.Printf("  approval: %s\n", fallbackText(item.ApprovalStatus, "none"))
	fmt.Printf("  input source: %s\n", inputSource)
	fmt.Printf("  input: %s\n", fallbackText(item.InputSummary, "none"))
	fmt.Println("  recommended input: use --input-file=<path> for JSON payloads")
	if kind == "workspace.apply_patch" {
		fmt.Println("  recommended patch input: use --patch-file=<path> --path=<workspace-relative-path>")
	} else {
		fmt.Println("  inline fallback: PowerShell JSON can use --input='{\"path\":\"go.mod\"}', --input='{\"paths\":[\"README.md\",\"go.mod\"]}', or --input='{\"query\":\"workspace\",\"path\":\"internal\",\"limit\":20}'")
	}
	return nil
}

func runModelTask(ctx context.Context, facade *runtimeFacade, threadID string, title string, providerKind string, modelName string, input string, maxOutputTokens int) error {
	if strings.TrimSpace(title) == "" {
		title = "Model response"
	}
	payload := map[string]any{
		"input": input,
	}
	if strings.TrimSpace(providerKind) != "" {
		payload["provider"] = providerKind
	}
	if strings.TrimSpace(modelName) != "" {
		payload["model"] = modelName
	}
	if maxOutputTokens > 0 {
		payload["maxOutputTokens"] = maxOutputTokens
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	item, err := facade.createTask(ctx, threadID, runtimecontract.CreateTaskRequest{
		Title: title,
		Kind:  "model.response.create",
		Input: string(raw),
	})
	if err != nil {
		return err
	}

	fmt.Println("model task created")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  thread id: %s\n", item.ThreadID)
	fmt.Printf("  kind: %s\n", item.Kind)
	fmt.Printf("  status: %s\n", item.Status)

	result, err := facade.runTask(ctx, threadID, item.ID, runtimecontract.RunTaskRequest{})
	if err != nil {
		return err
	}
	fmt.Println("model task executed")
	fmt.Printf("  id: %s\n", result.ID)
	fmt.Printf("  status: %s\n", result.Status)
	fmt.Printf("  result: %s\n", fallbackText(result.ResultSummary, "none"))
	return nil
}

func runAgentTask(ctx context.Context, facade *runtimeFacade, threadID string, title string, goal string, providerKind string, modelName string, maxSteps int, maxOutputTokens int) error {
	if strings.TrimSpace(title) == "" {
		title = "Agent run"
	}
	payload := map[string]any{
		"goal": goal,
	}
	if strings.TrimSpace(providerKind) != "" {
		payload["provider"] = providerKind
	}
	if strings.TrimSpace(modelName) != "" {
		payload["model"] = modelName
	}
	if maxSteps > 0 {
		payload["maxSteps"] = maxSteps
	}
	if maxOutputTokens > 0 {
		payload["maxOutputTokens"] = maxOutputTokens
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	item, err := facade.createTask(ctx, threadID, runtimecontract.CreateTaskRequest{
		Title: title,
		Kind:  "agent.run",
		Input: string(raw),
	})
	if err != nil {
		return err
	}

	fmt.Println("agent task created")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Println("  workflow: default goal-oriented workflow")
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  thread id: %s\n", item.ThreadID)
	fmt.Printf("  kind: %s\n", item.Kind)
	fmt.Printf("  status: %s\n", item.Status)

	result, err := facade.runTask(ctx, threadID, item.ID, runtimecontract.RunTaskRequest{})
	if err != nil {
		return err
	}
	fmt.Println("agent task executed")
	fmt.Printf("  id: %s\n", result.ID)
	fmt.Printf("  status: %s\n", result.Status)
	fmt.Printf("  waiting: %s\n", fallbackText(result.WaitingStatus, "none"))
	if waitingReason := taskWaitingReason(result); waitingReason != "" {
		fmt.Printf("  waiting on: %s\n", waitingReason)
	}
	fmt.Printf("  plan: %s\n", fallbackText(result.AgentPlanSummary, "none"))
	fmt.Printf("  plan mode: %s\n", fallbackText(result.AgentPlanMode, "none"))
	fmt.Printf("  current step: %s\n", fallbackText(result.AgentCurrentStepTitle, "none"))
	fmt.Printf("  last reasoning: %s\n", fallbackText(result.AgentLastReasoning, "none"))
	fmt.Printf("  progress: %d/%d\n", result.AgentStep, result.AgentMaxSteps)
	fmt.Printf("  latest child: %s\n", fallbackText(result.LatestChildTaskID, "none"))
	fmt.Printf("  result: %s\n", fallbackText(result.ResultSummary, "none"))
	if result.Status == "waiting_for_approval" || result.WaitingStatus == "waiting_for_approval" {
		fmt.Println("  next step: approve the child workspace.apply_patch task; the parent agent will auto-resume")
	}
	fmt.Printf("  inspect: gen-code tasks list --thread=%s\n", threadID)
	return nil
}

func taskWorkflowRole(item runtimecontract.TaskDescriptor) string {
	switch {
	case item.Kind == "agent.run":
		return "default goal-oriented agent parent"
	case strings.TrimSpace(item.ParentTaskID) != "":
		return fmt.Sprintf("child task of %s", item.ParentTaskID)
	default:
		return ""
	}
}

func taskWaitingReason(item runtimecontract.TaskDescriptor) string {
	switch item.WaitingStatus {
	case "waiting_for_approval":
		if strings.TrimSpace(item.LatestChildTaskID) != "" {
			return fmt.Sprintf("approval for child task %s", item.LatestChildTaskID)
		}
		return "approval for a child task"
	case "waiting_for_task":
		if strings.TrimSpace(item.LatestChildTaskID) != "" {
			return fmt.Sprintf("child task %s to finish", item.LatestChildTaskID)
		}
		return "a child task to finish"
	default:
		return ""
	}
}

func rollbackLatest(ctx context.Context, facade *runtimeFacade, threadID string) error {
	items, err := facade.writeExecutions(ctx, threadID)
	if err != nil {
		return err
	}
	candidate := latestRollbackCandidate(items)
	if candidate == nil {
		return errors.New("no rollback-eligible write execution found")
	}

	inputBytes, err := json.Marshal(map[string]string{"writeExecutionId": candidate.ID})
	if err != nil {
		return err
	}
	created, err := facade.createTask(ctx, threadID, runtimecontract.CreateTaskRequest{
		Title: "Rollback latest write execution",
		Kind:  "workspace.apply_patch.rollback",
		Input: string(inputBytes),
	})
	if err != nil {
		return err
	}

	fmt.Println("rollback task created")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  source write execution: %s\n", candidate.ID)
	fmt.Printf("  task id: %s\n", created.ID)
	fmt.Printf("  status: %s\n", created.Status)
	fmt.Printf("  approval: %s\n", fallbackText(created.ApprovalStatus, "none"))
	fmt.Printf("  result: %s\n", fallbackText(created.ResultSummary, "none"))

	if created.Status == "needs_approval" || created.ApprovalStatus == "pending" {
		fmt.Println("rollback is waiting for approval")
		return nil
	}

	executed, err := facade.runTask(ctx, threadID, created.ID, runtimecontract.RunTaskRequest{})
	if err != nil {
		return err
	}
	fmt.Println("rollback task executed")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  task id: %s\n", executed.ID)
	fmt.Printf("  status: %s\n", executed.Status)
	fmt.Printf("  approval: %s\n", fallbackText(executed.ApprovalStatus, "none"))
	fmt.Printf("  result: %s\n", fallbackText(executed.ResultSummary, "none"))
	return nil
}

func latestRollbackCandidate(items []runtimecontract.WriteExecutionDescriptor) *runtimecontract.WriteExecutionDescriptor {
	for index := range items {
		operation := strings.TrimSpace(items[index].Operation)
		if operation == "" && items[index].ToolKind == "workspace.apply_patch" {
			operation = "apply"
		}
		if operation == "apply" && items[index].Status == "completed" {
			return &items[index]
		}
	}
	return nil
}

func runTask(ctx context.Context, facade *runtimeFacade, threadID string, taskID string) error {
	item, err := facade.runTask(ctx, threadID, taskID, runtimecontract.RunTaskRequest{})
	if err != nil {
		return err
	}

	fmt.Println("task executed")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  thread id: %s\n", item.ThreadID)
	fmt.Printf("  kind: %s\n", fallbackText(item.Kind, "none"))
	fmt.Printf("  status: %s\n", item.Status)
	fmt.Printf("  approval: %s\n", fallbackText(item.ApprovalStatus, "none"))
	fmt.Printf("  result: %s\n", fallbackText(item.ResultSummary, "none"))
	return nil
}

func approveTask(ctx context.Context, facade *runtimeFacade, threadID string, taskID string) error {
	item, err := facade.approveTask(ctx, threadID, taskID, runtimecontract.ApproveTaskRequest{})
	if err != nil {
		return err
	}

	fmt.Println("task approved")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  thread id: %s\n", item.ThreadID)
	fmt.Printf("  kind: %s\n", fallbackText(item.Kind, "none"))
	fmt.Printf("  status: %s\n", item.Status)
	fmt.Printf("  approval: %s\n", fallbackText(item.ApprovalStatus, "none"))
	fmt.Printf("  result: %s\n", fallbackText(item.ResultSummary, "none"))
	return nil
}

func rejectTask(ctx context.Context, facade *runtimeFacade, threadID string, taskID string) error {
	item, err := facade.rejectTask(ctx, threadID, taskID, runtimecontract.RejectTaskRequest{})
	if err != nil {
		return err
	}

	fmt.Println("task rejected")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  thread id: %s\n", item.ThreadID)
	fmt.Printf("  kind: %s\n", fallbackText(item.Kind, "none"))
	fmt.Printf("  status: %s\n", item.Status)
	fmt.Printf("  approval: %s\n", fallbackText(item.ApprovalStatus, "none"))
	fmt.Printf("  result: %s\n", fallbackText(item.ResultSummary, "none"))
	return nil
}

func updateTaskStatus(ctx context.Context, facade *runtimeFacade, threadID string, taskID string, status string) error {
	item, err := facade.updateTaskStatus(ctx, threadID, taskID, runtimecontract.UpdateTaskStatusRequest{Status: status})
	if err != nil {
		return err
	}

	fmt.Println("task updated")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  thread id: %s\n", item.ThreadID)
	fmt.Printf("  status: %s\n", item.Status)
	fmt.Printf("  updated at: %s\n", fallbackText(item.UpdatedAt, "none"))
	return nil
}

func printSkills(runtimeService *runtime.Service, requestedGroup string) error {
	if requestedGroup != "" {
		group, err := skill.ParseGroup(requestedGroup)
		if err != nil {
			return err
		}

		fmt.Printf("skills group: %s\n", requestedGroup)
		for _, item := range runtimeService.SkillDescriptors(group) {
			if group != skill.Common && item.Group == skill.Common {
				continue
			}
			fmt.Printf("  - %s (%s)\n", item.ID, item.Group)
		}
		return nil
	}

	fmt.Println("skills list")
	for _, groupName := range []string{"common", "codex", "cc"} {
		group, err := skill.ParseGroup(groupName)
		if err != nil {
			return err
		}
		fmt.Printf("%s:\n", groupName)
		for _, item := range runtimeService.SkillDescriptors(group) {
			if groupName != "common" && item.Group == skill.Common {
				continue
			}
			fmt.Printf("  - %s (%s)\n", item.ID, item.Group)
		}
	}
	return nil
}

func printTools(ctx context.Context, facade *runtimeFacade) error {
	fmt.Println("tools list")
	items, err := facade.tools(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	for _, item := range items {
		fmt.Printf(
			"  - %s (%s, permission=%s, kind=%s, executable=%t, readOnly=%t)\n",
			item.ID,
			fallbackText(item.Source, "runtime"),
			fallbackText(item.Permission, "unknown"),
			fallbackText(item.Kind, "none"),
			item.Executable,
			item.ReadOnly,
		)
	}
	return nil
}

func printMCP(ctx context.Context, facade *runtimeFacade) error {
	fmt.Println("mcp list")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	items, err := facade.mcp(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		fmt.Printf("  - %s (%s, enabled=%t, tools=%d, resources=%d)\n", item.ID, fallbackText(item.Source, "unknown"), item.Enabled, item.ToolCount, item.ResourceCount)
	}
	return nil
}

func printProviders(ctx context.Context, facade *runtimeFacade) error {
	fmt.Println("providers list")
	items, err := facade.providers(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	for _, item := range items {
		fmt.Printf(
			"  - %s (enabled=%t, model=%s, preferredApi=%s, recommended=%t, hasAuth=%t)\n",
			item.Kind,
			item.Enabled,
			fallbackText(item.DefaultModel, "none"),
			fallbackText(item.PreferredAPIStyle, "unknown"),
			item.Recommended,
			item.HasAuthToken,
		)
	}
	return nil
}

func probeProvider(ctx context.Context, facade *runtimeFacade, kind string) error {
	item, err := facade.probeProvider(ctx, kind)
	if err != nil {
		return err
	}

	fmt.Println("provider probe")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  source detail: %s\n", runtimeSourceDetail(facade.runtimeSource()))
	fmt.Printf("  kind: %s\n", item.Kind)
	fmt.Printf("  reachable: %t\n", item.Reachable)
	fmt.Printf("  preferred api: %s\n", fallbackText(item.PreferredAPIStyle, "unknown"))
	fmt.Printf("  message: %s\n", fallbackText(item.Message, "none"))
	if len(item.Details) > 0 {
		encoded, _ := json.Marshal(item.Details)
		fmt.Printf("  details: %s\n", string(encoded))
	}
	return nil
}

func countSkills(runtimeService *runtime.Service) int {
	total := 0
	for _, groupName := range []string{"common", "codex", "cc"} {
		group, err := skill.ParseGroup(groupName)
		if err != nil {
			continue
		}
		for _, item := range runtimeService.SkillDescriptors(group) {
			if group != skill.Common && item.Group == skill.Common {
				continue
			}
			total++
		}
	}
	return total
}

func runtimeBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("GENCODE_RUNTIME_BASE_URL")); value != "" {
		return value
	}
	return defaultRuntimeBaseURL
}

func fallbackText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func runtimeSourceDetail(source string) string {
	switch source {
	case "remote-app-server":
		return "canonical shared runtime served by the app-server entry"
	case "local-fallback":
		return "project-local SQLite fallback because the canonical app-server runtime is unavailable"
	default:
		return "unknown runtime source"
	}
}

func runtimeSourceTrust(source string) string {
	switch source {
	case "remote-app-server":
		return "canonical"
	case "local-fallback":
		return "degraded"
	default:
		return "unknown"
	}
}

func resolveTaskCreateInput(kind string, options taskCreateInputOptions) (string, string, error) {
	hasInline := strings.TrimSpace(options.Input) != ""
	hasInputFile := strings.TrimSpace(options.InputFile) != ""
	hasPatchFile := strings.TrimSpace(options.PatchFile) != ""
	hasPatchPath := strings.TrimSpace(options.PatchPath) != ""

	if hasPatchFile || hasPatchPath {
		if kind != "workspace.apply_patch" {
			return "", "", errors.New("--patch-file and --path are only supported for --kind=workspace.apply_patch")
		}
		if !hasPatchFile || !hasPatchPath {
			return "", "", errors.New("usage: gen-code tasks create --thread=<threadId> --kind=workspace.apply_patch --patch-file=<path> --path=<workspace-relative-path>")
		}
		if hasInline || hasInputFile {
			return "", "", errors.New("use either --input/--input-file or --patch-file/--path, not both")
		}
		patchBytes, err := os.ReadFile(options.PatchFile)
		if err != nil {
			return "", "", fmt.Errorf("read patch file: %w", err)
		}
		payload, err := json.Marshal(map[string]string{
			"path":  options.PatchPath,
			"patch": string(patchBytes),
		})
		if err != nil {
			return "", "", fmt.Errorf("encode patch input: %w", err)
		}
		return string(payload), fmt.Sprintf("patch file %s -> %s", filepath.Base(options.PatchFile), options.PatchPath), nil
	}

	if hasInline && hasInputFile {
		return "", "", errors.New("use either --input or --input-file, not both")
	}

	if hasInputFile {
		content, err := os.ReadFile(options.InputFile)
		if err != nil {
			return "", "", fmt.Errorf("read input file: %w", err)
		}
		return normalizeTaskInput(string(content)), fmt.Sprintf("input file %s", filepath.Base(options.InputFile)), nil
	}

	if hasInline {
		return normalizeTaskInput(options.Input), "inline --input", nil
	}

	return normalizeTaskInput(options.Input), "empty", nil
}

func normalizeTaskInput(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}
	if json.Valid([]byte(trimmed)) {
		return trimmed
	}
	normalized, ok := normalizeLooseObject(trimmed)
	if ok {
		return normalized
	}
	return trimmed
}

func normalizeLooseObject(raw string) (string, bool) {
	if !strings.HasPrefix(raw, "{") || !strings.HasSuffix(raw, "}") {
		return "", false
	}
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "{"), "}"))
	if body == "" {
		return "{}", true
	}

	parts := splitLooseObject(body)
	if len(parts) == 0 {
		return "", false
	}

	values := make(map[string]any, len(parts))
	for _, part := range parts {
		key, value, ok := strings.Cut(part, ":")
		if !ok {
			return "", false
		}
		key = strings.Trim(strings.TrimSpace(key), `"'`)
		if key == "" {
			return "", false
		}
		parsedValue, ok := parseLooseValue(value)
		if !ok {
			return "", false
		}
		values[key] = parsedValue
	}

	encoded, err := json.Marshal(values)
	if err != nil {
		return "", false
	}
	return string(encoded), true
}

func splitLooseObject(body string) []string {
	parts := make([]string, 0)
	var current strings.Builder
	inSingle := false
	inDouble := false

	for _, r := range body {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case ',':
			if !inSingle && !inDouble {
				part := strings.TrimSpace(current.String())
				if part != "" {
					parts = append(parts, part)
				}
				current.Reset()
				continue
			}
		}
		current.WriteRune(r)
	}

	part := strings.TrimSpace(current.String())
	if part != "" {
		parts = append(parts, part)
	}
	return parts
}

func parseLooseValue(raw string) (any, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", true
	}
	if len(value) >= 2 {
		if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) || (strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
			return strings.Trim(value, `"'`), true
		}
	}

	switch strings.ToLower(value) {
	case "true":
		return true, true
	case "false":
		return false, true
	case "null":
		return nil, true
	}

	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i, true
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f, true
	}
	return value, true
}

func (c *remoteRuntimeClient) status() (runtimecontract.Status, error) {
	var item runtimecontract.Status
	err := c.fetchEnvelope("/api/runtime/status", &item)
	return item, err
}

func (c *remoteRuntimeClient) workspace() (runtimecontract.WorkspaceDescriptor, error) {
	var item runtimecontract.WorkspaceDescriptor
	err := c.fetchEnvelope("/api/workspace", &item)
	return item, err
}

func (c *remoteRuntimeClient) threads() ([]runtimecontract.ThreadDescriptor, error) {
	var payload struct {
		Items []runtimecontract.ThreadDescriptor `json:"items"`
	}
	err := c.fetchEnvelope("/api/threads", &payload)
	return payload.Items, err
}

func (c *remoteRuntimeClient) createThread(request runtimecontract.CreateThreadRequest) (runtimecontract.ThreadDescriptor, error) {
	var item runtimecontract.ThreadDescriptor
	err := c.postEnvelope("/api/threads", request, &item)
	return item, err
}

func (c *remoteRuntimeClient) activateThread(id string) (runtimecontract.ThreadDescriptor, error) {
	var item runtimecontract.ThreadDescriptor
	err := c.postEnvelope("/api/threads/"+url.PathEscape(id)+"/activate", map[string]any{}, &item)
	return item, err
}

func (c *remoteRuntimeClient) tasks(threadID string) ([]runtimecontract.TaskDescriptor, error) {
	var payload struct {
		Items []runtimecontract.TaskDescriptor `json:"items"`
	}
	err := c.fetchEnvelope("/api/threads/"+url.PathEscape(threadID)+"/tasks", &payload)
	return payload.Items, err
}

func (c *remoteRuntimeClient) approvals(threadID string) ([]runtimecontract.ApprovalDescriptor, error) {
	var payload struct {
		Items []runtimecontract.ApprovalDescriptor `json:"items"`
	}
	err := c.fetchEnvelope("/api/threads/"+url.PathEscape(threadID)+"/approvals", &payload)
	return payload.Items, err
}

func (c *remoteRuntimeClient) writeExecutions(threadID string) ([]runtimecontract.WriteExecutionDescriptor, error) {
	var payload struct {
		Items []runtimecontract.WriteExecutionDescriptor `json:"items"`
	}
	err := c.fetchEnvelope("/api/threads/"+url.PathEscape(threadID)+"/write-executions", &payload)
	return payload.Items, err
}

func (c *remoteRuntimeClient) messages(threadID string) ([]runtimecontract.MessageDescriptor, error) {
	var payload struct {
		Items []runtimecontract.MessageDescriptor `json:"items"`
	}
	err := c.fetchEnvelope("/api/threads/"+url.PathEscape(threadID)+"/messages", &payload)
	return payload.Items, err
}

func (c *remoteRuntimeClient) appendMessage(threadID string, request runtimecontract.CreateMessageRequest) (runtimecontract.MessageDescriptor, error) {
	var item runtimecontract.MessageDescriptor
	err := c.postEnvelope("/api/threads/"+url.PathEscape(threadID)+"/messages", request, &item)
	return item, err
}

func (c *remoteRuntimeClient) toolCalls(threadID string) ([]runtimecontract.ToolCallDescriptor, error) {
	var payload struct {
		Items []runtimecontract.ToolCallDescriptor `json:"items"`
	}
	err := c.fetchEnvelope("/api/threads/"+url.PathEscape(threadID)+"/tool-calls", &payload)
	return payload.Items, err
}

func (c *remoteRuntimeClient) artifacts(threadID string) ([]runtimecontract.ArtifactDescriptor, error) {
	var payload struct {
		Items []runtimecontract.ArtifactDescriptor `json:"items"`
	}
	err := c.fetchEnvelope("/api/threads/"+url.PathEscape(threadID)+"/artifacts", &payload)
	return payload.Items, err
}

func (c *remoteRuntimeClient) createTask(threadID string, request runtimecontract.CreateTaskRequest) (runtimecontract.TaskDescriptor, error) {
	var item runtimecontract.TaskDescriptor
	err := c.postEnvelope("/api/threads/"+url.PathEscape(threadID)+"/tasks", request, &item)
	return item, err
}

func (c *remoteRuntimeClient) runTask(threadID string, taskID string, request runtimecontract.RunTaskRequest) (runtimecontract.TaskDescriptor, error) {
	var item runtimecontract.TaskDescriptor
	err := c.postEnvelope("/api/threads/"+url.PathEscape(threadID)+"/tasks/"+url.PathEscape(taskID)+"/run", request, &item)
	return item, err
}

func (c *remoteRuntimeClient) approveTask(threadID string, taskID string, request runtimecontract.ApproveTaskRequest) (runtimecontract.TaskDescriptor, error) {
	var item runtimecontract.TaskDescriptor
	err := c.postEnvelope("/api/threads/"+url.PathEscape(threadID)+"/tasks/"+url.PathEscape(taskID)+"/approve", request, &item)
	return item, err
}

func (c *remoteRuntimeClient) rejectTask(threadID string, taskID string, request runtimecontract.RejectTaskRequest) (runtimecontract.TaskDescriptor, error) {
	var item runtimecontract.TaskDescriptor
	err := c.postEnvelope("/api/threads/"+url.PathEscape(threadID)+"/tasks/"+url.PathEscape(taskID)+"/reject", request, &item)
	return item, err
}

func (c *remoteRuntimeClient) updateTaskStatus(threadID string, taskID string, request runtimecontract.UpdateTaskStatusRequest) (runtimecontract.TaskDescriptor, error) {
	var item runtimecontract.TaskDescriptor
	err := c.postEnvelope("/api/threads/"+url.PathEscape(threadID)+"/tasks/"+url.PathEscape(taskID)+"/status", request, &item)
	return item, err
}

func (c *remoteRuntimeClient) tools() ([]runtimecontract.Tool, error) {
	var payload struct {
		Items []runtimecontract.Tool `json:"items"`
	}
	err := c.fetchEnvelope("/api/tools", &payload)
	return payload.Items, err
}

func (c *remoteRuntimeClient) mcpServers() ([]runtimecontract.MCPServer, error) {
	var payload struct {
		Items []runtimecontract.MCPServer `json:"items"`
	}
	err := c.fetchEnvelope("/api/mcp/servers", &payload)
	return payload.Items, err
}

func (c *remoteRuntimeClient) providers() ([]runtimecontract.Provider, error) {
	var payload struct {
		Items []runtimecontract.Provider `json:"items"`
	}
	err := c.fetchEnvelope("/api/providers", &payload)
	return payload.Items, err
}

func (c *remoteRuntimeClient) probeProvider(kind string) (runtimecontract.ProviderProbeResult, error) {
	var item runtimecontract.ProviderProbeResult
	err := c.postEnvelope("/api/providers/"+url.PathEscape(kind)+"/probe", map[string]any{}, &item)
	return item, err
}

func (c *remoteRuntimeClient) fetchEnvelope(path string, target any) error {
	response, err := c.client.Get(c.baseURL + path)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return decodeEnvelope(response, target)
}

func (c *remoteRuntimeClient) postEnvelope(path string, body any, target any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return decodeEnvelope(response, target)
}

func decodeEnvelope(response *http.Response, target any) error {
	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		if len(body) == 0 {
			return fmt.Errorf("request failed: %s", response.Status)
		}
		return fmt.Errorf("request failed: %s %s", response.Status, strings.TrimSpace(string(body)))
	}

	switch typed := target.(type) {
	case *runtimecontract.Status:
		var envelope apiEnvelope[runtimecontract.Status]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *runtimecontract.WorkspaceDescriptor:
		var envelope apiEnvelope[runtimecontract.WorkspaceDescriptor]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *runtimecontract.ThreadDescriptor:
		var envelope apiEnvelope[runtimecontract.ThreadDescriptor]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *runtimecontract.TaskDescriptor:
		var envelope apiEnvelope[runtimecontract.TaskDescriptor]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *runtimecontract.MessageDescriptor:
		var envelope apiEnvelope[runtimecontract.MessageDescriptor]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []runtimecontract.ThreadDescriptor `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []runtimecontract.ThreadDescriptor `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []runtimecontract.TaskDescriptor `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []runtimecontract.TaskDescriptor `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []runtimecontract.MessageDescriptor `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []runtimecontract.MessageDescriptor `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []runtimecontract.ToolCallDescriptor `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []runtimecontract.ToolCallDescriptor `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []runtimecontract.ArtifactDescriptor `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []runtimecontract.ArtifactDescriptor `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []runtimecontract.ApprovalDescriptor `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []runtimecontract.ApprovalDescriptor `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []runtimecontract.WriteExecutionDescriptor `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []runtimecontract.WriteExecutionDescriptor `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []runtimecontract.Tool `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []runtimecontract.Tool `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []runtimecontract.MCPServer `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []runtimecontract.MCPServer `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *runtimecontract.ProviderProbeResult:
		var envelope apiEnvelope[runtimecontract.ProviderProbeResult]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []runtimecontract.Provider `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []runtimecontract.Provider `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	default:
		return fmt.Errorf("unsupported envelope target")
	}

	return nil
}
