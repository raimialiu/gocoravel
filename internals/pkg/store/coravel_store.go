package store

type (
	// CoravelStore is the high-level store facade. It consumes a DataSource,
	// which routes every operation to the active storage provider; embedding it
	// promotes the full DataProvider/DataConnector surface onto the store.
	CoravelStore struct {
		DataSource *DataSource
	}

	CoravelStoreStoreConfig func(*CoravelStore)
	SchedulerStoreConfig    func(*CoravelStore)
)

// NewCoravelStore builds a store over the given DataSource, applying any config
// options.
func NewCoravelStore(source *DataSource, configs ...SchedulerStoreConfig) *CoravelStore {
	store := &CoravelStore{DataSource: source}
	for _, config := range configs {
		config(store)
	}
	return store
}
