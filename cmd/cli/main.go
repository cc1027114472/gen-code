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
	"strings"
	"time"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/core/runtime"
	"llmtrace/internal/core/skill"
)

const defaultRuntimeBaseURL = "http://127.0.0.1:10008"

var sharedFallbackService = runtime.NewDefaultService()

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
			return errors.New("usage: gen-code threads <list|create|activate>")
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
		default:
			return errors.New("usage: gen-code threads <list|create|activate>")
		}
	case "tasks":
		if len(args) < 2 {
			return errors.New("usage: gen-code tasks <list|create|update-status>")
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
			var threadID, title string
			for _, arg := range args[2:] {
				switch {
				case strings.HasPrefix(arg, "--thread="):
					threadID = strings.TrimSpace(strings.TrimPrefix(arg, "--thread="))
				case strings.HasPrefix(arg, "--title="):
					title = strings.TrimSpace(strings.TrimPrefix(arg, "--title="))
				}
			}
			if threadID == "" {
				return errors.New("usage: gen-code tasks create --thread=<threadId> [--title=...]")
			}
			return createTask(ctx, runtimeFacade, threadID, title)
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
			return errors.New("usage: gen-code tasks <list|create|update-status>")
		}
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
	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", strings.Join(args, " "))
	}
}

func printUsage() {
	fmt.Println("gen-code commands:")
	fmt.Println("  doctor")
	fmt.Println("  runtime status")
	fmt.Println("  workspace show")
	fmt.Println("  threads list")
	fmt.Println("  threads create [--name=...] [--model=...] [--permission=...]")
	fmt.Println("  threads activate --id=<threadId>")
	fmt.Println("  tasks list --thread=<threadId>")
	fmt.Println("  tasks create --thread=<threadId> [--title=...]")
	fmt.Println("  tasks update-status --thread=<threadId> --task=<taskId> --status=<status>")
	fmt.Println("  skills list [--group=<group>]")
	fmt.Println("  tools list")
	fmt.Println("  mcp list")
}

func newRuntimeFacade() *runtimeFacade {
	return &runtimeFacade{
		service: sharedFallbackService,
		client: &remoteRuntimeClient{
			baseURL: strings.TrimRight(runtimeBaseURL(), "/"),
			client: http.Client{
				Timeout: 1500 * time.Millisecond,
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

func (f *runtimeFacade) updateTaskStatus(ctx context.Context, threadID string, taskID string, request runtimecontract.UpdateTaskStatusRequest) (runtimecontract.TaskDescriptor, error) {
	if item, err := f.client.updateTaskStatus(threadID, taskID, request); err == nil {
		f.source = "remote-app-server"
		return item, nil
	}
	f.source = "local-fallback"
	return f.service.UpdateTaskStatus(ctx, threadID, taskID, request)
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
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  name: %s\n", item.Name)
	fmt.Printf("  is active: %t\n", item.IsActive)
	return nil
}

func printTasks(ctx context.Context, facade *runtimeFacade, threadID string) error {
	items, err := facade.tasks(ctx, threadID)
	if err != nil {
		return err
	}

	fmt.Println("tasks list")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  thread: %s\n", threadID)
	for _, item := range items {
		fmt.Printf("  - %s (%s, updated=%s)\n", item.ID, item.Status, fallbackText(item.UpdatedAt, "none"))
	}
	return nil
}

func createTask(ctx context.Context, facade *runtimeFacade, threadID string, title string) error {
	item, err := facade.createTask(ctx, threadID, runtimecontract.CreateTaskRequest{Title: title})
	if err != nil {
		return err
	}

	fmt.Println("task created")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  thread id: %s\n", item.ThreadID)
	fmt.Printf("  title: %s\n", item.Title)
	fmt.Printf("  status: %s\n", item.Status)
	return nil
}

func updateTaskStatus(ctx context.Context, facade *runtimeFacade, threadID string, taskID string, status string) error {
	item, err := facade.updateTaskStatus(ctx, threadID, taskID, runtimecontract.UpdateTaskStatusRequest{Status: status})
	if err != nil {
		return err
	}

	fmt.Println("task updated")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
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
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	items, err := facade.tools(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		fmt.Printf("  - %s (%s, %s)\n", item.ID, fallbackText(item.Source, "runtime"), fallbackText(item.Permission, "unknown"))
	}
	return nil
}

func printMCP(ctx context.Context, facade *runtimeFacade) error {
	fmt.Println("mcp list")
	fmt.Printf("  source: %s\n", facade.runtimeSource())
	items, err := facade.mcp(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		fmt.Printf("  - %s (%s, enabled=%t, tools=%d, resources=%d)\n", item.ID, fallbackText(item.Source, "unknown"), item.Enabled, item.ToolCount, item.ResourceCount)
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

func (c *remoteRuntimeClient) createTask(threadID string, request runtimecontract.CreateTaskRequest) (runtimecontract.TaskDescriptor, error) {
	var item runtimecontract.TaskDescriptor
	err := c.postEnvelope("/api/threads/"+url.PathEscape(threadID)+"/tasks", request, &item)
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
	default:
		return fmt.Errorf("unsupported envelope target")
	}

	return nil
}
