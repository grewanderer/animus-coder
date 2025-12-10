package llm

import "fmt"

// ModelRoute binds a logical model to a provider and physical model name.
type ModelRoute struct {
	Name        string
	Provider    string
	Model       string
	Temperature float64
	MaxTokens   int
}

// Registry resolves models to providers and tracks metadata (expensive flags).
type Registry struct {
	providers    map[string]Provider
	models       map[string]ModelRoute
	defaultModel string
	expensive    map[string]bool
}

// MarkExpensive marks a model as expensive for strategy accounting.
func (r *Registry) MarkExpensive(modelID string, expensive bool) {
	if r.expensive == nil {
		r.expensive = make(map[string]bool)
	}
	r.expensive[modelID] = expensive
}

// IsExpensive reports whether a model is marked expensive.
func (r *Registry) IsExpensive(modelID string) bool {
	return r.expensive[modelID]
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		models:    make(map[string]ModelRoute),
		expensive: make(map[string]bool),
	}
}

// RegisterProvider adds a provider implementation.
func (r *Registry) RegisterProvider(name string, p Provider) {
	r.providers[name] = p
}

// RegisterModel adds a model route.
func (r *Registry) RegisterModel(name string, route ModelRoute, isDefault bool) {
	route.Name = name
	r.models[name] = route
	if isDefault || r.defaultModel == "" {
		r.defaultModel = name
	}
}

// Resolve returns the provider and route for a given model name (default if empty).
func (r *Registry) Resolve(modelName string) (Provider, ModelRoute, error) {
	if modelName == "" {
		modelName = r.defaultModel
	}

	route, ok := r.models[modelName]
	if !ok {
		return nil, ModelRoute{}, fmt.Errorf("model %q not registered", modelName)
	}

	p, ok := r.providers[route.Provider]
	if !ok {
		return nil, ModelRoute{}, fmt.Errorf("provider %q not registered for model %q", route.Provider, modelName)
	}

	return p, route, nil
}
