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
}
