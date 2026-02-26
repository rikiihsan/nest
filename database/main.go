package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bundebug"
)

// Custom errors
type DatabaseError struct {
	Message string
	Err     error
}

func (e *DatabaseError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func ErrDriverNotFound(driver string) error {
	return &DatabaseError{Message: fmt.Sprintf("driver '%s' not found", driver)}
}

func ErrSessionNotFound(name string) error {
	return &DatabaseError{Message: fmt.Sprintf("session '%s' not found", name)}
}

func ErrNoDatabaseConnection() error {
	return &DatabaseError{Message: "no database connection available"}
}

// Init initializes database connections
func Init(configs ...Config) error {
	for _, config := range configs {
		if err := Manager.createSession(config); err != nil {
			return fmt.Errorf("failed to create session '%s': %w", config.Name, err)
		}
	}
	return nil
}

// createSession creates a new database session
func (cm *ConnectionManager) createSession(config Config) error {
	// Get registered driver
	driver, exists := cm.drivers[config.Driver]
	if !exists {
		return ErrDriverNotFound(config.Driver)
	}

	// Open database connection
	sqlDB, err := driver.Open(config.Dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	if config.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	// Create Bun DB instance
	bunDB := driver.CreateBunDB(sqlDB)

	// Add debug hook if debug mode is enabled
	if config.Debug {
		bunDB.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
			bundebug.FromEnv("BUNDEBUG"),
		))
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Store session
	cm.sessions[config.Name] = &Session{
		Name:   config.Name,
		DB:     bunDB,
		SqlDB:  sqlDB,
		Config: config,
	}

	return nil
}

// InitRedis initializes Redis connection
func InitRedis(cfg RedisConfig) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		PoolTimeout:  cfg.PoolTimeout,
	})

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return nil
}

// CloseAll closes all database connections
func CloseAll() error {
	var errors []error

	// Close database sessions
	for name, session := range Manager.sessions {
		if err := session.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close session '%s': %w", name, err))
		}
	}

	// Close Redis connection
	if RedisClient != nil {
		if err := RedisClient.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close Redis: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors occurred while closing connections: %v", errors)
	}

	return nil
}

// GetRedisClient returns Redis client instance
func GetRedisClient() *redis.Client {
	return RedisClient
}

// HealthCheck performs health check on all connections
func HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error)

	// Check database connections
	for name, session := range Manager.sessions {
		if err := session.Ping(ctx); err != nil {
			results[name] = err
		} else {
			results[name] = nil
		}
	}

	// Check Redis connection
	if RedisClient != nil {
		if _, err := RedisClient.Ping(ctx).Result(); err != nil {
			results["redis"] = err
		} else {
			results["redis"] = nil
		}
	}

	return results
}

// GetConnectionStats returns connection statistics
func GetConnectionStats() map[string]interface{} {
	stats := make(map[string]interface{})

	for name, session := range Manager.sessions {
		stats[name] = session.Stats()
	}

	if RedisClient != nil {
		stats["redis"] = RedisClient.PoolStats()
	}

	return stats
}

// WithTransaction executes function within database transaction
func WithTransaction(ctx context.Context, sessionName string, fn func(tx bun.Tx) error) error {
	session, exists := Manager.sessions[sessionName]
	if !exists {
		return ErrSessionNotFound(sessionName)
	}

	return session.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(tx)
	})
}
