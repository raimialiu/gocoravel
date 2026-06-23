package providers

import "fmt"

// ProviderFactory builds a Provider from a connection configuration.
type ProviderFactory func(ConnectionConfiguration) Provider

// registry maps each known ProviderKind to its constructor. The single
// ConnectionConfiguration passed to New is cascaded into the constructor, so
// every provider is built from the same configuration.
var registry = map[ProviderKind]ProviderFactory{
	Redis: func(cfg ConnectionConfiguration) Provider { return NewRedisDataProvider(cfg) },
}

// RegisterProvider adds or overrides the constructor for a ProviderKind, so new
// backends plug in without modifying New.
func RegisterProvider(kind ProviderKind, factory ProviderFactory) {
	registry[kind] = factory
}

// New builds the provider registered for kind, configured from cfg. The cfg is
// the cascade point: it flows into whichever provider constructor is selected.
func New(kind ProviderKind, cfg ConnectionConfiguration) (Provider, error) {
	factory, ok := registry[kind]
	if !ok {
		return nil, fmt.Errorf("providers: no provider registered for kind %q", kind)
	}
	return factory(cfg), nil
}
