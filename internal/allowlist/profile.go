package allowlist

// Profile represents a named set of allowed terms for a specific
// editorial context.
type Profile struct {
	ID          string   `yaml:"profile_id"`
	Description string   `yaml:"description"`
	Terms       []string `yaml:"terms"`
}
