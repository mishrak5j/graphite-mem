package tools

import (
	"context"
	"fmt"

	"github.com/mishrak5j/graphite-mem/internal/storage"
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
		vector := d.ingestor // need access to stores -- use a helper
		_ = vector

		vs, ok := getVectorStore(d)
		if !ok {
			return errorResult("vector store unavailable"), ForgetOutput{}, nil
		}
		gs, ok2 := getGraphStore(d)
		if !ok2 {
			return errorResult("graph store unavailable"), ForgetOutput{}, nil
		}

		errV := vs.Delete(ctx, input.MemoryID)
		errG := gs.Delete(ctx, input.MemoryID)

		if errV != nil || errG != nil {
			return errorResult(fmt.Sprintf("delete errors: vector=%v graph=%v", errV, errG)), ForgetOutput{}, nil
		}

		return nil, ForgetOutput{Success: true}, nil
	})
}

var (
	vectorStoreRef storage.VectorStore
	graphStoreRef  storage.GraphStore
)

func SetStoreRefs(vs storage.VectorStore, gs storage.GraphStore) {
	vectorStoreRef = vs
	graphStoreRef = gs
}

func getVectorStore(d *deps) (storage.VectorStore, bool) {
	if vectorStoreRef != nil {
		return vectorStoreRef, true
	}
	return nil, false
}

func getGraphStore(d *deps) (storage.GraphStore, bool) {
	if graphStoreRef != nil {
		return graphStoreRef, true
	}
	return nil, false
}
