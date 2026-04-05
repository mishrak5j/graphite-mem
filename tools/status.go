package tools

import (
	"context"

	"github.com/mishrak5j/graphite-mem/internal/vault"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type StatusInput struct{}

type SuppressionInfo struct {
	Term         string  `json:"term"`
	RemainingTTL int     `json:"remaining_ttl"`
	Weight       float64 `json:"weight"`
}

type StatusOutput struct {
	SessionID          string            `json:"session_id"`
	Inhibited          bool              `json:"inhibited"`
	TurnCounter        int               `json:"turn_counter"`
	MountedScopes      []string          `json:"mounted_scopes"`
	ActiveSuppressions []SuppressionInfo `json:"active_suppressions"`
	TotalMemories      int               `json:"total_memories"`
	GraphNodes         int               `json:"graph_nodes"`
	GraphEdges         int               `json:"graph_edges"`
}

func registerStatusTool(server *mcp.Server, d *deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "memory_status",
		Description: "Get full diagnostic view of the current memory session: inhibit state, mounted scopes, active suppressions, and store statistics.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, StatusOutput, error) {
		session := d.vault.Sessions.GetOrCreateDefault(d.cfg.DefaultScope)

		suppressions := vault.GetActiveSuppressions(d.vault.Sessions, session.ID)
		supInfos := make([]SuppressionInfo, len(suppressions))
		for i, s := range suppressions {
			supInfos[i] = SuppressionInfo{
				Term:         s.Term,
				RemainingTTL: s.RemainingTTL,
				Weight:       s.Weight,
			}
		}

		var totalMem, nodes, edges int
		if d.vector != nil {
			totalMem, _ = d.vector.Count(ctx)
		}
		if d.graph != nil {
			nodes, _ = d.graph.NodeCount(ctx)
			edges, _ = d.graph.EdgeCount(ctx)
		}

		return nil, StatusOutput{
			SessionID:          session.ID,
			Inhibited:          session.InhibitPastContext,
			TurnCounter:        session.TurnCounter,
			MountedScopes:      session.MountedScopes,
			ActiveSuppressions: supInfos,
			TotalMemories:      totalMem,
			GraphNodes:         nodes,
			GraphEdges:         edges,
		}, nil
	})
}
