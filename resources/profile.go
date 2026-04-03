package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/mishrak5j/graphite-mem/internal/storage"
	"github.com/mishrak5j/graphite-mem/internal/vault"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterAll(server *mcp.Server, graph storage.GraphStore, v *vault.Vault) {
	server.AddResource(&mcp.Resource{
		URI:         "memory://profile",
		Name:        "User Memory Profile",
		Description: "Synthesized user profile based on knowledge graph traversal, scoped to currently mounted scopes.",
		MIMEType:    "text/plain",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		session := v.Sessions.GetOrCreateDefault(v.DefaultScope)
		scopeFilter := v.Scopes.ResolveScopeFilter(session, false)

		related, err := graph.QueryRelated(ctx, "", scopeFilter, 50)
		if err != nil {
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:      "memory://profile",
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("Error building profile: %v", err),
				}},
			}, nil
		}

		var sb strings.Builder
		sb.WriteString("# User Memory Profile\n\n")
		sb.WriteString(fmt.Sprintf("Session: %s\n", session.ID))
		sb.WriteString(fmt.Sprintf("Mounted Scopes: %s\n\n", strings.Join(session.MountedScopes, ", ")))

		if len(related) == 0 {
			sb.WriteString("No memories found in mounted scopes.\n")
		} else {
			sb.WriteString("## Knowledge Graph Relationships\n\n")
			for _, r := range related {
				sb.WriteString(fmt.Sprintf("- %s (scope: %s)\n", r.Path, r.Scope))
			}
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      "memory://profile",
				MIMEType: "text/plain",
				Text:     sb.String(),
			}},
		}, nil
	})
}
