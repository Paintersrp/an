package note

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/Paintersrp/an/internal/config"
	"github.com/Paintersrp/an/internal/pathutil"
)

type hookContext struct {
	File     string
	Vault    string
	Relative string
	Filename string
}

func RunPreOpenHooks(path string) error {
	hooks, err := loadWorkspaceHooks()
	if err != nil {
		return err
	}
	return executeHookCommands("pre_open", hooks.PreOpen, path)
}

func RunPostOpenHooks(path string) error {
	hooks, err := loadWorkspaceHooks()
	if err != nil {
		return err
	}
	return executeHookCommands("post_open", hooks.PostOpen, path)
}

func RunPostCreateHooks(path string) error {
	hooks, err := loadWorkspaceHooks()
	if err != nil {
		return err
	}
	return executeHookCommands("post_create", hooks.PostCreate, path)
}

func loadWorkspaceHooks() (config.HookConfig, error) {
	var hooks config.HookConfig
	if err := viper.UnmarshalKey("workspace_hooks", &hooks); err != nil {
		return config.HookConfig{}, fmt.Errorf("failed to load workspace hooks: %w", err)
	}
	return hooks, nil
}

func executeHookCommands(phase string, commands []config.CommandTemplate, path string) error {
	if len(commands) == 0 {
		return nil
	}

	ctx := newHookContext(path)
	for _, command := range commands {
		cmd, wait, name, err := buildHookCommand(command, ctx)
		if err != nil {
			return err
		}
		if cmd == nil {
			continue
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("%s hook %q failed to start: %w", phase, name, err)
		}

		if wait {
			if err := cmd.Wait(); err != nil {
				return fmt.Errorf("%s hook %q failed: %w", phase, name, err)
			}
			continue
		}

		if err := cmd.Process.Release(); err != nil {
			return fmt.Errorf("%s hook %q release failed: %w", phase, name, err)
		}
	}

	return nil
}

func buildHookCommand(template config.CommandTemplate, ctx hookContext) (*exec.Cmd, bool, string, error) {
	execName := strings.TrimSpace(applyHookPlaceholders(template.Exec, ctx))
	if execName == "" {
		return nil, false, "", nil
	}

	args := expandHookArgs(template.Args, ctx)
	cmd := exec.Command(execName, args...)

	if template.Silence != nil && *template.Silence {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	wait := true
	if template.Wait != nil {
		wait = *template.Wait
	}

	return cmd, wait, execName, nil
}

func newHookContext(path string) hookContext {
	vault := viper.GetString("vaultdir")
	relative, err := pathutil.VaultRelative(vault, path)
	if err != nil {
		relative = path
	}

	return hookContext{
		File:     path,
		Vault:    vault,
		Relative: relative,
		Filename: filepath.Base(path),
	}
}

func expandHookArgs(args []string, ctx hookContext) []string {
	if len(args) == 0 {
		return nil
	}

	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		expanded = append(expanded, applyHookPlaceholders(arg, ctx))
	}

	return expanded
}

func applyHookPlaceholders(value string, ctx hookContext) string {
	replacements := map[string]string{
		"{file}":     ctx.File,
		"{vault}":    ctx.Vault,
		"{relative}": ctx.Relative,
		"{filename}": ctx.Filename,
	}

	result := value
	for placeholder, replacement := range replacements {
		result = strings.ReplaceAll(result, placeholder, replacement)
	}

	return result
}
