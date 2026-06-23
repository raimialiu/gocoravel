package providers

type (
	// ConnectionConfiguration holds the settings used to open a connection to a
	// backing store.
	ConnectionConfiguration struct {
		Url      string
		Host     string
		Port     int
		Username string
		Password string
		PoolSize int
	}

	// ProviderKind identifies a concrete storage backend. The DataSource router
	// uses it (reported via Provider.Kind) to select a provider through the
	// interface alone, with no concrete type switch.
	ProviderKind string

	// DataProvider is the backend-agnostic CRUD contract every storage backend
	// implements. Keys are strings; values are arbitrary (JSON-encodable) data.
	DataProvider interface {
		Get(key string) ([]interface{}, error)
		GetSingle(key string) (interface{}, error)
		Create(key string, data interface{}) error
		CreateMany(key string, data ...interface{}) error
		CreateIfNotExists(key string, data interface{}) error
		Update(key string, data interface{}) error
		UpdateMany(key string, data ...interface{}) error
		Delete(key string) (bool, error)
	}

	// DataConnector establishes the backing connection.
	DataConnector interface {
		Open()
	}

	// Provider is a fully-featured, self-identifying storage backend: it can
	// connect, serve data, and report its Kind, so a DataSource can route to it
	// purely through this interface.
	Provider interface {
		DataProvider
		DataConnector
		Kind() ProviderKind
	}
)

const (
	// Redis is the kind reported by RedisDataProvider.
	Redis ProviderKind = "redis"
)
