package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type MountInput struct {
	ScopePath string `json:"scope_path" jsonschema:"Scope path to mount or unmount (e.g. /projects/my-app)"`
}

type MountOutput struct {
	MountedScopes []string `json:"mounted_scopes"`
}

type ListScopesInput struct{}

type ScopeInfo struct {
	Path        string `json:"path"`
	DisplayName string `json:"display_name"`
	MemoryCount int    `json:"memory_count"`
	Mounted     bool   `json:"mounted"`
}

type ListScopesOutput struct {
	Scopes []ScopeInfo `json:"scopes"`
}

func registerScopeTools(server *mcp.Server, d *deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mount_scope",
		Description: "Mount a memory scope into the current session. Only mounted scopes are searched during recall.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MountInput) (*mcp.CallToolResult, MountOutput, error) {
		session := d.vault.Sessions.GetOrCreateDefault(d.cfg.DefaultScope)

		mounted, err := d.vault.Scopes.MountScope(d.vault.Sessions, session.ID, input.ScopePath)
		if err != nil {
			return errorResult(fmt.Sprintf("mount failed: %v", err)), MountOutput{}, nil
		}

		return nil, MountOutput{MountedScopes: mounted}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "unmount_scope",
		Description: "Unmount a memory scope from the current session. Memories in this scope become invisible but are not deleted.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MountInput) (*mcp.CallToolResult, MountOutput, error) {
		session := d.vault.Sessions.GetOrCreateDefault(d.cfg.DefaultScope)

		mounted, err := d.vault.Scopes.UnmountScope(d.vault.Sessions, session.ID, input.ScopePath)
		if err != nil {
			return errorResult(fmt.Sprintf("unmount failed: %v", err)), MountOutput{}, nil
		}

		return nil, MountOutput{MountedScopes: mounted}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_scopes",
		Description: "List all available memory scopes and their mount status in the current session.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListScopesInput) (*mcp.CallToolResult, ListScopesOutput, error) {
		session := d.vault.Sessions.GetOrCreateDefault(d.cfg.DefaultScope)

		mountedSet := make(map[string]bool)
		for _, sc := range session.MountedScopes {
			mountedSet[sc] = true
		}

		allScopes := d.vault.Scopes.ListScopes()
		infos := make([]ScopeInfo, len(allScopes))
		for i, sc := range allScopes {
			infos[i] = ScopeInfo{
				Path:        sc.Path,
				DisplayName: sc.DisplayName,
				MemoryCount: sc.MemoryCount,
				Mounted:     mountedSet[sc.Path],
			}
		}

		return nil, ListScopesOutput{Scopes: infos}, nil
	})
}
