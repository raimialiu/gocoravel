package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound is returned by single-value reads when the key holds no value.
var ErrNotFound = errors.New("providers: key not found")

// RedisDataProvider is a Redis-backed implementation of DataProvider and
// DataConnector, built on github.com/redis/go-redis/v9.
//
// Value mapping (scaffold defaults — adjust to your needs):
//   - Single-value methods (GetSingle, Create, CreateIfNotExists, Update) treat
//     the key as a Redis string.
//   - Collection methods (Get, CreateMany, UpdateMany) treat the key as a Redis
//     list, storing one element per item.
//
// A Redis key cannot be both a string and a list at the same time, so for any
// given key use one family or the other — don't mix them.
//
// Values are serialized as JSON, so any JSON-encodable Go value round-trips.
// Reads decode into the generic shapes encoding/json produces
// (map[string]interface{}, []interface{}, float64, string, bool, nil).
type RedisDataProvider struct {
	cfg     ConnectionConfiguration
	ctx     context.Context
	rdb     *redis.Client
	openErr error
}

// Compile-time proof that the provider satisfies the store contracts.
var (
	_ Provider      = (*RedisDataProvider)(nil)
	_ DataProvider  = (*RedisDataProvider)(nil)
	_ DataConnector = (*RedisDataProvider)(nil)
)

// Kind reports this provider's identity so a DataSource can route to it through
// the Provider interface.
func (p *RedisDataProvider) Kind() ProviderKind { return Redis }

// NewRedisDataProvider builds a provider from cfg. It does not dial Redis;
// call Open before issuing any operation.
func NewRedisDataProvider(cfg ConnectionConfiguration) *RedisDataProvider {
	return &RedisDataProvider{
		cfg: cfg,
		ctx: context.Background(),
	}
}

// WithContext returns a shallow copy that uses ctx for its operations.
func (p *RedisDataProvider) WithContext(ctx context.Context) *RedisDataProvider {
	clone := *p
	clone.ctx = ctx
	return &clone
}

// -----------------------------------------------------------------------------
// DataConnector
// -----------------------------------------------------------------------------

// Open connects to Redis (from cfg.Url, or cfg.Host/Port) and verifies the
// connection with a PING.
//
// NOTE: DataConnector.Open has no error return, so a failure is recorded here
// and surfaced by the next data operation. Consider widening the interface to
// `Open() error` (see the note in data_provider.go) and returning directly.
func (p *RedisDataProvider) Open() {
	var opts *redis.Options
	if p.cfg.Url != "" {
		parsed, err := redis.ParseURL(p.cfg.Url)
		if err != nil {
			p.openErr = fmt.Errorf("redis: parse url: %w", err)
			return
		}
		opts = parsed
	} else {
		opts = &redis.Options{
			Addr:     redisAddr(p.cfg),
			Username: p.cfg.Username,
			Password: p.cfg.Password,
		}
	}
	if p.cfg.PoolSize > 0 {
		opts.PoolSize = p.cfg.PoolSize
	}

	p.rdb = redis.NewClient(opts)
	if err := p.rdb.Ping(p.ctx).Err(); err != nil {
		p.openErr = fmt.Errorf("redis: open %s: %w", opts.Addr, err)
	}
}

// Close releases the underlying connection pool. It is not part of
// DataConnector yet, but you'll want it once the interface grows a lifecycle.
func (p *RedisDataProvider) Close() error {
	if p.rdb == nil {
		return nil
	}
	return p.rdb.Close()
}

// -----------------------------------------------------------------------------
// DataProvider — single-value (string) operations
// -----------------------------------------------------------------------------

// Create stores data under key as a JSON string, overwriting any existing value.
func (p *RedisDataProvider) Create(key string, data interface{}) error {
	rdb, err := p.conn()
	if err != nil {
		return err
	}
	payload, err := encode(data)
	if err != nil {
		return err
	}
	return rdb.Set(p.ctx, key, payload, 0).Err()
}

// CreateIfNotExists stores data under key only if the key is absent (SETNX).
// If the key already exists it is a no-op and returns nil.
func (p *RedisDataProvider) CreateIfNotExists(key string, data interface{}) error {
	rdb, err := p.conn()
	if err != nil {
		return err
	}
	payload, err := encode(data)
	if err != nil {
		return err
	}
	return rdb.SetNX(p.ctx, key, payload, 0).Err()
}

// Update overwrites the value stored at key. Like Create, it upserts; it does
// not require the key to pre-exist.
func (p *RedisDataProvider) Update(key string, data interface{}) error {
	return p.Create(key, data)
}

// GetSingle returns the JSON-decoded value stored at key, or ErrNotFound if the
// key is absent.
func (p *RedisDataProvider) GetSingle(key string) (interface{}, error) {
	rdb, err := p.conn()
	if err != nil {
		return nil, err
	}
	raw, err := rdb.Get(p.ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return decode(raw)
}

// -----------------------------------------------------------------------------
// DataProvider — collection (list) operations
// -----------------------------------------------------------------------------

// CreateMany appends each item to the list stored at key (RPUSH).
func (p *RedisDataProvider) CreateMany(key string, data ...interface{}) error {
	rdb, err := p.conn()
	if err != nil {
		return err
	}
	values, err := encodeAll(data)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}
	return rdb.RPush(p.ctx, key, values...).Err()
}

// UpdateMany replaces the entire list at key with the supplied items,
// atomically (DEL + RPUSH inside a transaction).
func (p *RedisDataProvider) UpdateMany(key string, data ...interface{}) error {
	rdb, err := p.conn()
	if err != nil {
		return err
	}
	values, err := encodeAll(data)
	if err != nil {
		return err
	}
	_, err = rdb.TxPipelined(p.ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(p.ctx, key)
		if len(values) > 0 {
			pipe.RPush(p.ctx, key, values...)
		}
		return nil
	})
	return err
}

// Get returns every element of the list stored at key, JSON-decoded. An absent
// key yields an empty slice and no error.
func (p *RedisDataProvider) Get(key string) ([]interface{}, error) {
	rdb, err := p.conn()
	if err != nil {
		return nil, err
	}
	raws, err := rdb.LRange(p.ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]interface{}, 0, len(raws))
	for _, raw := range raws {
		v, err := decode(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// -----------------------------------------------------------------------------
// DataProvider — deletion
// -----------------------------------------------------------------------------

// Delete removes key. The bool reports whether a key was actually removed.
func (p *RedisDataProvider) Delete(key string) (bool, error) {
	rdb, err := p.conn()
	if err != nil {
		return false, err
	}
	n, err := rdb.Del(p.ctx, key).Result()
	return n > 0, err
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// conn returns the live client, or the reason it is unavailable.
func (p *RedisDataProvider) conn() (*redis.Client, error) {
	if p.openErr != nil {
		return nil, p.openErr
	}
	if p.rdb == nil {
		return nil, errors.New("redis: provider not opened — call Open() first")
	}
	return p.rdb, nil
}

func redisAddr(c ConnectionConfiguration) string {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := c.Port
	if port == 0 {
		port = 6379
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func encode(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("providers: encode: %w", err)
	}
	return string(b), nil
}

func encodeAll(items []interface{}) ([]interface{}, error) {
	out := make([]interface{}, 0, len(items))
	for _, it := range items {
		s, err := encode(it)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func decode(s string) (interface{}, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, fmt.Errorf("providers: decode: %w", err)
	}
	return v, nil
}
