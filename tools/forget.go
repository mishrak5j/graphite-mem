package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ForgetInput struct {
	MemoryID string `json:"memory_id" jsonschema:"required,description=ID of the memory to permanently delete"`
}

type ForgetOutput struct {
	Success bool `json:"success"`
}

func registerForgetTool(server *mcp.Server, d *deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "forget_memory",
		Description: "Permanently delete a memory from both vector and graph stores.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ForgetInput) (*mcp.CallToolResult, ForgetOutput, error) {
		if d.vector == nil {
			return errorResult("vector store unavailable"), ForgetOutput{}, nil
		}
		if d.graph == nil {
			return errorResult("graph store unavailable"), ForgetOutput{}, nil
		}

		errV := d.vector.Delete(ctx, input.MemoryID)
		errG := d.graph.Delete(ctx, input.MemoryID)

		if errV != nil || errG != nil {
			return errorResult(fmt.Sprintf("delete errors: vector=%v graph=%v", errV, errG)), ForgetOutput{}, nil
		}

		return nil, ForgetOutput{Success: true}, nil
	})
}
