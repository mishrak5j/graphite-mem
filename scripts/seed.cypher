// Create default scope
MERGE (:Scope {path: "/default", display_name: "Default", created_at: datetime()});

// Create example scopes for demonstration
MERGE (:Scope {path: "/projects/architect-cli", display_name: "Architect CLI", created_at: datetime()});
MERGE (:Scope {path: "/personal/learning", display_name: "Learning", created_at: datetime()});
