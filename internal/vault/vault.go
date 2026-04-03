package vault

type Vault struct {
	Sessions *SessionManager
	Scopes   *ScopeRegistry
	DefaultScope string
}

func New(defaultScope string) *Vault {
	return &Vault{
		Sessions:     NewSessionManager(),
		Scopes:       NewScopeRegistry(defaultScope),
		DefaultScope: defaultScope,
	}
}
