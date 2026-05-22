package broker

import (
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
)

type routesFile struct {
	Routes []*Route `json:"routes"`
}

// Registry resolves inbound bearer tokens to routes.
type Registry struct {
	path string
	ptr  atomic.Pointer[map[string]*Route]
}

// LoadRegistry loads registry data from path.
func LoadRegistry(path string) (*Registry, error) {
	r := &Registry{path: path}
	if err := r.Reload(); err != nil {
		return nil, err
	}
	return r, nil
}

// Reload reparses the registry file and atomically swaps route state.
func (r *Registry) Reload() error {
	b, err := os.ReadFile(r.path)
	if err != nil {
		return fmt.Errorf("read routes: %w", err)
	}
	var rf routesFile
	if err := json.Unmarshal(b, &rf); err != nil {
		return fmt.Errorf("unmarshal routes: %w", err)
	}
	next := make(map[string]*Route, len(rf.Routes))
	for _, route := range rf.Routes {
		if route == nil || route.TenantKey == "" || !route.Enabled {
			continue
		}
		rc := *route
		next[rc.TenantKey] = &rc
	}
	r.ptr.Store(&next)
	return nil
}

// Resolve maps tenant key to route.
func (r *Registry) Resolve(key string) (*Route, bool) {
	m := r.ptr.Load()
	if m == nil {
		return nil, false
	}
	route, ok := (*m)[key]
	return route, ok
}

// Count returns the number of enabled routes.
func (r *Registry) Count() int {
	m := r.ptr.Load()
	if m == nil {
		return 0
	}
	return len(*m)
}
