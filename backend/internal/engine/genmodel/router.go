// Package genmodel routes gen operations to providers with capability checks,
// circuit breakers, per-provider concurrency and a fallback (docs/GENERATION_PIPELINE.md §8).
package genmodel

import (
	"context"
	"time"

	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/breaker"
	gm "yingji/backend/internal/provider/genmodel"
)

type Router struct {
	models   map[string]gm.Model
	breakers map[string]*breaker.Breaker
	sems     map[string]chan struct{}
	def      string
}

func NewRouter(defaultName string, concurrency int, models ...gm.Model) *Router {
	if concurrency <= 0 {
		concurrency = 4
	}
	r := &Router{models: map[string]gm.Model{}, breakers: map[string]*breaker.Breaker{}, sems: map[string]chan struct{}{}, def: defaultName}
	for _, m := range models {
		r.models[m.Name()] = m
		r.breakers[m.Name()] = breaker.New(5, 30*time.Second)
		r.sems[m.Name()] = make(chan struct{}, concurrency)
	}
	return r
}

func (r *Router) Has(name string) bool { _, ok := r.models[name]; return ok }

func (r *Router) Available(name string, mode gm.Mode) bool {
	m, ok := r.models[name]
	if !ok || !m.Capabilities().Supports(mode) {
		return false
	}
	return r.breakers[name].Allow()
}

// AnyAvailable reports whether the default provider (or any provider) can serve mode now.
func (r *Router) AnyAvailable(mode gm.Mode) bool {
	if r.Available(r.def, mode) {
		return true
	}
	for name := range r.models {
		if r.Available(name, mode) {
			return true
		}
	}
	return false
}

// Resolve picks name → fallback → default, skipping open breakers and missing capabilities.
func (r *Router) Resolve(name, fallback string, mode gm.Mode) (gm.Model, error) {
	for _, n := range []string{name, fallback, r.def} {
		if n == "" {
			continue
		}
		if r.Available(n, mode) {
			return r.models[n], nil
		}
	}
	return nil, apperr.GenerationUnavailable()
}

// Run executes the request on the resolved provider, retrying once on a transient
// error and then trying the fallback once. Returns the provider actually used.
func (r *Router) Run(ctx context.Context, name, fallback string, req gm.Request) (gm.Result, string, error) {
	m, err := r.Resolve(name, fallback, req.Mode)
	if err != nil {
		return gm.Result{}, "", err
	}
	res, err := r.runOn(ctx, m, req)
	if err == nil {
		return res, m.Name(), nil
	}
	if !apperr.IsTransient(err) {
		return gm.Result{}, m.Name(), err
	}
	// retry once on the same provider
	if r.breakers[m.Name()].Allow() {
		if res, err2 := r.runOn(ctx, m, req); err2 == nil {
			return res, m.Name(), nil
		} else {
			err = err2
		}
	}
	// then a fallback provider, if any is different and available
	if fb, ferr := r.Resolve(fallback, r.def, req.Mode); ferr == nil && fb.Name() != m.Name() {
		if res, err3 := r.runOn(ctx, fb, req); err3 == nil {
			return res, fb.Name(), nil
		}
	}
	return gm.Result{}, m.Name(), err
}

func (r *Router) runOn(ctx context.Context, m gm.Model, req gm.Request) (gm.Result, error) {
	sem := r.sems[m.Name()]
	select {
	case sem <- struct{}{}:
	case <-ctx.Done():
		return gm.Result{}, ctx.Err()
	}
	defer func() { <-sem }()
	res, err := m.Run(ctx, req)
	if err != nil {
		if apperr.IsTransient(err) {
			r.breakers[m.Name()].Failure()
		}
		return gm.Result{}, err
	}
	r.breakers[m.Name()].Success()
	return res, nil
}
