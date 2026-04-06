package llm

import "testing"

func TestParseTriplesJSONArray(t *testing.T) {
	raw := `[{"s":"A","p":"likes","o":"B"}]`
	got, err := parseTriples(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Subject != "A" || got[0].Predicate != "likes" || got[0].Object != "B" {
		t.Fatalf("got %+v", got)
	}
}

func TestParseTriplesSingleObject(t *testing.T) {
	raw := `{"s":"Graphite-Memory MCP","p":"connected_from","o":"Cursor"}`
	got, err := parseTriples(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Subject != "Graphite-Memory MCP" {
		t.Fatalf("got %+v", got)
	}
}

func TestParseTriplesEmptyObject(t *testing.T) {
	got, err := parseTriples(`{}`)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil && len(got) != 0 {
		t.Fatalf("want empty slice, got %+v", got)
	}
}

func TestParseTriplesObjectWithBracketInString(t *testing.T) {
	// Regression: '[' inside a JSON string must not break extraction.
	raw := `{"s":"Note [bracket]","p":"has","o":"value"}`
	got, err := parseTriples(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Subject != "Note [bracket]" {
		t.Fatalf("got %+v", got)
	}
}
