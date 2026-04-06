package config

import (
	"testing"
)

func TestLoad_requiresNeo4jPassword(t *testing.T) {
	t.Setenv("GRAPHITE_NEO4J_PASS", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when GRAPHITE_NEO4J_PASS is empty")
	}
}

func TestLoad_acceptsNeo4jPassword(t *testing.T) {
	t.Setenv("GRAPHITE_NEO4J_PASS", "test-secret-neo4j-pass")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Neo4jPass != "test-secret-neo4j-pass" {
		t.Fatalf("Neo4jPass = %q", cfg.Neo4jPass)
	}
}
