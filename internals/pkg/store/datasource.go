package store

import (
	"errors"
	"fmt"

	"github.com/raimialiu/gocoravel/internals/pkg/store/providers"
)

// DataSource is the storage router that CoravelStore consumes. It holds a set
// of registered providers and forwards every data operation to the active one.
//
// Provider selection is interface-driven: providers self-identify through
// providers.Provider.Kind, and DataSource routes by that kind — registering a
// new backend never requires a type switch here.
type DataSource struct {
	active   providers.Provider
	registry map[providers.ProviderKind]providers.Provider
}

// Compile-time proof that DataSource exposes the full data-access surface.
var (
	_ providers.DataProvider  = (*DataSource)(nil)
	_ providers.DataConnector = (*DataSource)(nil)
)

// NewDataSource registers the given providers (keyed by their Kind) and makes
// active the one matching the requested kind. It fails if no registered
// provider reports that kind.
func NewDataSource(active providers.ProviderKind, provs ...providers.Provider) (*DataSource, error) {
	ds := &DataSource{
		registry: make(map[providers.ProviderKind]providers.Provider, len(provs)),
	}
	for _, p := range provs {
		ds.Register(p)
	}
	if err := ds.Use(active); err != nil {
		return nil, err
	}
	return ds, nil
}

// Register adds (or replaces) a provider in the registry, keyed by its Kind.
func (d *DataSource) Register(p providers.Provider) {
	if p == nil {
		return
	}
	if d.registry == nil {
		d.registry = make(map[providers.ProviderKind]providers.Provider)
	}
	d.registry[p.Kind()] = p
}

// Use switches the active provider to the one registered under kind.
func (d *DataSource) Use(kind providers.ProviderKind) error {
	p, ok := d.registry[kind]
	if !ok {
		return fmt.Errorf("store: no provider registered for kind %q", kind)
	}
	d.active = p
	return nil
}

// ActiveKind reports the kind of the currently active provider.
func (d *DataSource) ActiveKind() providers.ProviderKind {
	if d.active == nil {
		return ""
	}
	return d.active.Kind()
}

// -----------------------------------------------------------------------------
// providers.DataConnector / providers.DataProvider — routed to the active provider
// -----------------------------------------------------------------------------

// Open opens the active provider's connection.
func (d *DataSource) Open() {
	if d.active != nil {
		d.active.Open()
	}
}

func (d *DataSource) Get(key string) ([]interface{}, error) {
	p, err := d.provider()
	if err != nil {
		return nil, err
	}
	return p.Get(key)
}

func (d *DataSource) GetSingle(key string) (interface{}, error) {
	p, err := d.provider()
	if err != nil {
		return nil, err
	}
	return p.GetSingle(key)
}

func (d *DataSource) Create(key string, data interface{}) error {
	p, err := d.provider()
	if err != nil {
		return err
	}
	return p.Create(key, data)
}

func (d *DataSource) CreateMany(key string, data ...interface{}) error {
	p, err := d.provider()
	if err != nil {
		return err
	}
	return p.CreateMany(key, data...)
}

func (d *DataSource) CreateIfNotExists(key string, data interface{}) error {
	p, err := d.provider()
	if err != nil {
		return err
	}
	return p.CreateIfNotExists(key, data)
}

func (d *DataSource) Update(key string, data interface{}) error {
	p, err := d.provider()
	if err != nil {
		return err
	}
	return p.Update(key, data)
}

func (d *DataSource) UpdateMany(key string, data ...interface{}) error {
	p, err := d.provider()
	if err != nil {
		return err
	}
	return p.UpdateMany(key, data...)
}

func (d *DataSource) Delete(key string) (bool, error) {
	p, err := d.provider()
	if err != nil {
		return false, err
	}
	return p.Delete(key)
}

// provider returns the active provider, or an error if none is selected.
func (d *DataSource) provider() (providers.Provider, error) {
	if d.active == nil {
		return nil, errors.New("store: no active provider selected")
	}
	return d.active, nil
}
