package storage

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jStore struct {
	driver neo4j.DriverWithContext
}

func NewNeo4jStore(uri, user, pass string) (*Neo4jStore, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, pass, ""))
	if err != nil {
		return nil, fmt.Errorf("neo4j driver: %w", err)
	}
	if err := driver.VerifyConnectivity(context.Background()); err != nil {
		return nil, fmt.Errorf("neo4j connectivity: %w", err)
	}
	return &Neo4jStore{driver: driver}, nil
}

func (s *Neo4jStore) Close() error {
	return s.driver.Close(context.Background())
}

func (s *Neo4jStore) MergeTriples(ctx context.Context, memoryID string, scope string, triples []Triple) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	for _, t := range triples {
		cypher := `
			MERGE (sc:Scope {path: $scope})
			MERGE (s:Entity {name: $subject})
			MERGE (o:Entity {name: $object})
			MERGE (s)-[r:REL {type: $predicate}]->(o)
			SET r.memory_id = $memoryID, r.created_at = datetime()
			MERGE (s)-[:BELONGS_TO]->(sc)
			MERGE (o)-[:BELONGS_TO]->(sc)
		`
		params := map[string]any{
			"scope":     scope,
			"subject":   t.Subject,
			"object":    t.Object,
			"predicate": t.Predicate,
			"memoryID":  memoryID,
		}
		_, err := session.Run(ctx, cypher, params)
		if err != nil {
			return fmt.Errorf("merge triple (%s-%s->%s): %w", t.Subject, t.Predicate, t.Object, err)
		}
	}
	return nil
}

func (s *Neo4jStore) QueryByIntent(ctx context.Context, intent string, filter ScopeFilter, limit int) ([]RelatedMemory, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	cypher := `
		MATCH (s:Entity)-[r:REL]->(o:Entity)
		WHERE (r.type CONTAINS $intent OR s.name CONTAINS $intent OR o.name CONTAINS $intent)
	`
	params := map[string]any{
		"intent": intent,
		"limit":  limit,
	}

	if !filter.CrossScope && len(filter.Scopes) > 0 {
		cypher += `
		AND EXISTS {
			MATCH (s)-[:BELONGS_TO]->(sc:Scope)
			WHERE sc.path IN $scopes
		}
		`
		params["scopes"] = filter.Scopes
	}

	cypher += `
		RETURN s.name AS subject, r.type AS predicate, o.name AS object,
		       r.memory_id AS memory_id, r.created_at AS created_at
		LIMIT $limit
	`

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("query by intent: %w", err)
	}

	return collectRelatedMemories(ctx, result, filter)
}

func (s *Neo4jStore) QueryRelated(ctx context.Context, subject string, filter ScopeFilter, limit int) ([]RelatedMemory, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	cypher := `
		MATCH (s:Entity)-[r:REL]->(o:Entity)
		WHERE s.name CONTAINS $subject OR o.name CONTAINS $subject
	`
	params := map[string]any{
		"subject": subject,
		"limit":   limit,
	}

	if !filter.CrossScope && len(filter.Scopes) > 0 {
		cypher += `
		AND EXISTS {
			MATCH (s)-[:BELONGS_TO]->(sc:Scope)
			WHERE sc.path IN $scopes
		}
		`
		params["scopes"] = filter.Scopes
	}

	cypher += `
		RETURN s.name AS subject, r.type AS predicate, o.name AS object,
		       r.memory_id AS memory_id, r.created_at AS created_at
		LIMIT $limit
	`

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("query related: %w", err)
	}

	return collectRelatedMemories(ctx, result, filter)
}

func collectRelatedMemories(ctx context.Context, result neo4j.ResultWithContext, filter ScopeFilter) ([]RelatedMemory, error) {
	var memories []RelatedMemory
	for result.Next(ctx) {
		record := result.Record()
		subj, _ := record.Get("subject")
		pred, _ := record.Get("predicate")
		obj, _ := record.Get("object")
		memID, _ := record.Get("memory_id")

		path := fmt.Sprintf("%v -[%v]-> %v", subj, pred, obj)
		scope := ""
		if len(filter.Scopes) > 0 {
			scope = filter.Scopes[0]
		}

		memories = append(memories, RelatedMemory{
			MemoryID: fmt.Sprintf("%v", memID),
			Text:     path,
			Scope:    scope,
			Path:     path,
			Score:    1.0,
		})
	}
	return memories, result.Err()
}

func (s *Neo4jStore) Delete(ctx context.Context, memoryID string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	cypher := `
		MATCH ()-[r:REL {memory_id: $memoryID}]->()
		DELETE r
	`
	_, err := session.Run(ctx, cypher, map[string]any{"memoryID": memoryID})
	return err
}

func (s *Neo4jStore) NodeCount(ctx context.Context) (int, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "MATCH (n:Entity) RETURN count(n) AS cnt", nil)
	if err != nil {
		return 0, err
	}
	if result.Next(ctx) {
		val, _ := result.Record().Get("cnt")
		if n, ok := val.(int64); ok {
			return int(n), nil
		}
	}
	return 0, result.Err()
}

func (s *Neo4jStore) EdgeCount(ctx context.Context) (int, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "MATCH ()-[r:REL]->() RETURN count(r) AS cnt", nil)
	if err != nil {
		return 0, err
	}
	if result.Next(ctx) {
		val, _ := result.Record().Get("cnt")
		if n, ok := val.(int64); ok {
			return int(n), nil
		}
	}
	return 0, result.Err()
}
