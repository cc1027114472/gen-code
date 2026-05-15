package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/core/runtime"
	"llmtrace/internal/core/skill"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	runtimeService := runtime.NewDefaultService()

	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "doctor":
		return runDoctor(ctx, runtimeService)
	case "runtime":
		if len(args) < 2 || args[1] != "status" {
			return errors.New("usage: gen-code runtime status")
		}
		return printRuntimeStatus(ctx, runtimeService)
	case "workspace":
		if len(args) < 2 || args[1] != "show" {
			return errors.New("usage: gen-code workspace show")
		}
		return printWorkspace(ctx, runtimeService)
	case "threads":
		if len(args) < 2 {
			return errors.New("usage: gen-code threads <list|create|activate>")
		}
		switch args[1] {
		case "list":
			return printThreads(ctx, runtimeService)
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
			return createThread(ctx, runtimeService, name, model, permission)
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
			return activateThread(ctx, runtimeService, id)
		default:
			return errors.New("usage: gen-code threads <list|create|activate>")
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
		return printSkills(runtimeService, group)
	case "tools":
		if len(args) < 2 || args[1] != "list" {
			return errors.New("usage: gen-code tools list")
		}
		return printTools(ctx, runtimeService)
	case "mcp":
		if len(args) < 2 || args[1] != "list" {
			return errors.New("usage: gen-code mcp list")
		}
		return printMCP(ctx, runtimeService)
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
	fmt.Println("  skills list [--group=<group>]")
	fmt.Println("  tools list")
	fmt.Println("  mcp list")
}

func runDoctor(ctx context.Context, runtimeService *runtime.Service) error {
	status, err := runtimeService.Status(ctx)
	if err != nil {
		return err
	}

	tools, err := runtimeService.Tools(ctx)
	if err != nil {
		return err
	}

	mcpServers, err := runtimeService.MCPServers(ctx)
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
		items := runtimeService.SkillDescriptors(group)
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

func printRuntimeStatus(ctx context.Context, runtimeService *runtime.Service) error {
	status, err := runtimeService.Status(ctx)
	if err != nil {
		return err
	}

	core := runtimeService.FullStatus()
	fmt.Println("runtime status")
	fmt.Printf("  app version: %s\n", core.AppVersion)
	fmt.Printf("  app server status: %s\n", status.State)
	fmt.Printf("  runtime ready: %t\n", status.Ready)
	fmt.Printf("  runtime message: %s\n", fallbackText(status.Message, "none"))
	fmt.Printf("  desktop shell status: %s\n", core.DesktopShellStatus)
	fmt.Printf("  go bridge status: %s\n", core.GoBridgeStatus)
	fmt.Printf("  workspace id: %s\n", fallbackText(core.WorkspaceID, "none"))
	fmt.Printf("  project root: %s\n", fallbackText(core.ProjectRoot, "none"))
	fmt.Printf("  thread count: %d\n", core.ThreadCount)
	fmt.Printf("  active thread id: %s\n", fallbackText(core.ActiveThreadID, "none"))
	fmt.Printf("  active skill group: %s\n", core.ActiveSkillGroup)
	fmt.Printf("  permission mode: %s\n", core.PermissionMode)
	fmt.Printf("  configured mcp server count: %d\n", core.ConfiguredMCPServers)
	fmt.Printf("  skills discovered: %d\n", countSkills(runtimeService))
	fmt.Printf("  tools discovered: %d\n", len(runtimeService.ToolDescriptors()))
	fmt.Printf("  mcp discovered: %d\n", len(runtimeService.MCPDescriptors()))
	return nil
}

func printWorkspace(ctx context.Context, runtimeService *runtime.Service) error {
	item, err := runtimeService.Workspace(ctx)
	if err != nil {
		return err
	}

	fmt.Println("workspace")
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  project root: %s\n", item.ProjectRoot)
	fmt.Printf("  shared docs root: %s\n", item.SharedDocsRoot)
	fmt.Printf("  created at: %s\n", item.CreatedAt)
	fmt.Printf("  active thread count: %d\n", item.ActiveThreadCount)
	return nil
}

func printThreads(ctx context.Context, runtimeService *runtime.Service) error {
	items, err := runtimeService.Threads(ctx)
	if err != nil {
		return err
	}

	fmt.Println("threads list")
	for _, item := range items {
		activeFlag := ""
		if item.IsActive {
			activeFlag = ", active"
		}
		fmt.Printf("  - %s (%s, %s%s)\n", item.ID, item.Name, item.PermissionMode, activeFlag)
	}
	return nil
}

func createThread(ctx context.Context, runtimeService *runtime.Service, name, model, permission string) error {
	item, err := runtimeService.CreateThread(ctx, runtimecontract.CreateThreadRequest{
		Name:           name,
		ActiveModel:    model,
		PermissionMode: permission,
	})
	if err != nil {
		return err
	}

	fmt.Println("thread created")
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  name: %s\n", item.Name)
	fmt.Printf("  active model: %s\n", fallbackText(item.ActiveModel, "none"))
	fmt.Printf("  permission mode: %s\n", item.PermissionMode)
	return nil
}

func activateThread(ctx context.Context, runtimeService *runtime.Service, id string) error {
	item, err := runtimeService.ActivateThread(ctx, id)
	if err != nil {
		return err
	}

	fmt.Println("thread activated")
	fmt.Printf("  id: %s\n", item.ID)
	fmt.Printf("  name: %s\n", item.Name)
	fmt.Printf("  is active: %t\n", item.IsActive)
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

func printTools(ctx context.Context, runtimeService *runtime.Service) error {
	fmt.Println("tools list")
	items, err := runtimeService.Tools(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		fmt.Printf("  - %s (%s, %s)\n", item.ID, fallbackText(item.Source, "runtime"), fallbackText(item.Permission, "unknown"))
	}
	return nil
}

func printMCP(ctx context.Context, runtimeService *runtime.Service) error {
	fmt.Println("mcp list")
	items, err := runtimeService.MCPServers(ctx)
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

func fallbackText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
