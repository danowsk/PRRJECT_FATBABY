package broker

// Route defines one tenant-to-upstream contract.
type Route struct {
	TenantID      string            `json:"tenant_id"`
	TenantKey     string            `json:"tenant_key"`
	UpstreamBase  string            `json:"upstream_base"`
	UpstreamKey   string            `json:"upstream_key"`
	StripHeaders  []string          `json:"strip_headers"`
	InjectHeaders map[string]string `json:"inject_headers"`
	AllowedPaths  []string          `json:"allowed_paths"`
	MaxBodyBytes  int64             `json:"max_body_bytes"`
	Enabled       bool              `json:"enabled"`
}
