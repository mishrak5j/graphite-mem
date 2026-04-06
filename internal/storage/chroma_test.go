package storage

import (
	"math"
	"testing"
	"time"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

func TestQueryResultToScoredMemories_nilIDLists(t *testing.T) {
	t.Parallel()
	got := queryResultToScoredMemories(&chroma.QueryResultImpl{})
	if got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestQueryResultToScoredMemories_emptyFirstIDGroup(t *testing.T) {
	t.Parallel()
	qr := &chroma.QueryResultImpl{
		IDLists: []chroma.DocumentIDs{{}},
	}
	got := queryResultToScoredMemories(qr)
	if got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestQueryResultToScoredMemories_rows(t *testing.T) {
	t.Parallel()
	ts := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)
	tsStr := ts.Format(time.RFC3339)

	meta, err := chroma.NewDocumentMetadataFromMap(map[string]interface{}{
		"scope":      "/project/a",
		"created_at": tsStr,
	})
	if err != nil {
		t.Fatal(err)
	}

	qr := &chroma.QueryResultImpl{
		IDLists: []chroma.DocumentIDs{
			{"mem-1", "mem-2"},
		},
		DocumentsLists: []chroma.Documents{
			{
				chroma.NewTextDocument("first doc"),
				chroma.NewTextDocument("second"),
			},
		},
		DistancesLists: []embeddings.Distances{
			{0.1, 0.5},
		},
		MetadatasLists: []chroma.DocumentMetadatas{
			{meta, meta},
		},
	}

	got := queryResultToScoredMemories(qr)
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}

	if got[0].ID != "mem-1" || got[0].Text != "first doc" {
		t.Errorf("first row: %#v", got[0])
	}
	wantScore0 := 1.0 - float64(embeddings.Distance(0.1))
	if math.Abs(got[0].Score-wantScore0) > 1e-6 {
		t.Errorf("Score[0]=%v want %v", got[0].Score, wantScore0)
	}
	if got[0].Scope != "/project/a" || !got[0].CreatedAt.Equal(ts) {
		t.Errorf("first metadata: Scope=%q CreatedAt=%v", got[0].Scope, got[0].CreatedAt)
	}

	wantScore1 := 1.0 - float64(embeddings.Distance(0.5))
	if math.Abs(got[1].Score-wantScore1) > 1e-6 {
		t.Errorf("Score[1]=%v want %v", got[1].Score, wantScore1)
	}
}

func TestQueryResultToScoredMemories_invalidCreatedAtIgnored(t *testing.T) {
	t.Parallel()
	meta, err := chroma.NewDocumentMetadataFromMap(map[string]interface{}{
		"scope":      "/s",
		"created_at": "not-a-date",
	})
	if err != nil {
		t.Fatal(err)
	}
	qr := &chroma.QueryResultImpl{
		IDLists: []chroma.DocumentIDs{{"x"}},
		MetadatasLists: []chroma.DocumentMetadatas{
			{meta},
		},
	}
	got := queryResultToScoredMemories(qr)
	if len(got) != 1 || got[0].CreatedAt != (time.Time{}) {
		t.Fatalf("got %#v", got[0])
	}
}
