package broker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

type Tenant struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

type Registry struct {
	mu       sync.RWMutex
	routes   map[string]Tenant
	routesFn string
}

func NewRegistry(routesPath string) (*Registry, error) {
	r := &Registry{routesFn: routesPath}
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

func (r *Registry) Reload() error {
	b, err := os.ReadFile(r.routesFn)
	if err != nil {
		return fmt.Errorf("read routes: %w", err)
	}
	var tenants []Tenant
	if err := json.Unmarshal(b, &tenants); err != nil {
		return fmt.Errorf("decode routes: %w", err)
	}
	m := make(map[string]Tenant, len(tenants))
	for _, t := range tenants {
		if t.Key == "" || t.ID == "" {
			continue
		}
		m[t.Key] = t
	}
	r.mu.Lock()
	r.routes = m
	r.mu.Unlock()
	return nil
}

func (r *Registry) ResolveKey(key string) (Tenant, error) {
	r.mu.RLock()
	t, ok := r.routes[key]
	r.mu.RUnlock()
	if !ok {
		return Tenant{}, errors.New("unknown key")
	}
	return t, nil
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
