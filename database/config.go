package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

// DatabaseDriver interface for dynamic driver loading
type DatabaseDriver interface {
	Open(dsn string) (*sql.DB, error)
	CreateBunDB(sqlDB *sql.DB) *bun.DB
	GetDriverName() string
}

// Config represents database configuration
type Config struct {
	Name            string
	Dsn             string
	Driver          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	Debug           bool
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	MaxRetries   int
	PoolSize     int
	MinIdleConns int
	PoolTimeout  time.Duration
	IdleTimeout  time.Duration
}

// Session holds database connection info
type Session struct {
	Name   string
	DB     *bun.DB
	SqlDB  *sql.DB
	Config Config
}

// ConnectionManager manages all database connections
type ConnectionManager struct {
	sessions map[string]*Session
	drivers  map[string]DatabaseDriver
}

// Global instances
var (
	Manager     *ConnectionManager
	RedisClient *redis.Client
)

// Initialize connection manager
func init() {
	Manager = &ConnectionManager{
		sessions: make(map[string]*Session),
		drivers:  make(map[string]DatabaseDriver),
	}
}

// RegisterDriver registers a database driver
func RegisterDriver(name string, driver DatabaseDriver) {
	Manager.drivers[name] = driver
}

// GetSession returns database session by name
func GetSession(name string) (*Session, bool) {
	session, exists := Manager.sessions[name]
	return session, exists
}

// GetDB returns bun.DB instance by name
func GetDB(name string) (*bun.DB, error) {
	session, exists := Manager.sessions[name]
	if !exists {
		return nil, ErrSessionNotFound(name)
	}
	return session.DB, nil
}

// GetAllSessions returns all active sessions
func GetAllSessions() map[string]*Session {
	return Manager.sessions
}

// Close closes specific database connection
func (s *Session) Close() error {
	if s.SqlDB != nil {
		return s.SqlDB.Close()
	}
	return nil
}

// Ping tests database connectivity
func (s *Session) Ping(ctx context.Context) error {
	if s.SqlDB != nil {
		return s.SqlDB.PingContext(ctx)
	}
	return ErrNoDatabaseConnection()
}

// Stats returns database statistics
func (s *Session) Stats() sql.DBStats {
	if s.SqlDB != nil {
		return s.SqlDB.Stats()
	}
	return sql.DBStats{}
}
