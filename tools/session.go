package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SessionInput struct {
	Inhibit     bool     `json:"inhibit,omitempty" jsonschema:"Start in clean-slate mode (no past memories returned)"`
	MountScopes []string `json:"mount_scopes,omitempty" jsonschema:"Scope paths to mount into the new session"`
}

type SessionOutput struct {
	SessionID     string   `json:"session_id"`
	Inhibited     bool     `json:"inhibited"`
	MountedScopes []string `json:"mounted_scopes"`
}

func registerSessionTool(server *mcp.Server, d *deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "new_session",
		Description: "Create a fresh memory session. Optionally start in clean-slate mode (inhibit=true) for private-browsing-style AI interaction, or mount specific scopes.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SessionInput) (*mcp.CallToolResult, SessionOutput, error) {
		scopes := input.MountScopes
		if len(scopes) == 0 {
			scopes = []string{d.cfg.DefaultScope}
		}

		for _, sp := range scopes {
			d.vault.Scopes.CreateScope(sp, sp)
		}

		session := d.vault.Sessions.NewSession(input.Inhibit, scopes)

		return nil, SessionOutput{
			SessionID:     session.ID,
			Inhibited:     session.InhibitPastContext,
			MountedScopes: session.MountedScopes,
		}, nil
	})
}
