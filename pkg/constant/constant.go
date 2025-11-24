package constant

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

// Context keys
const (
	CtxKeyUserExtID ContextKey = "user_ext_id"
	CtxKeyUserRole  ContextKey = "user_role"
)
