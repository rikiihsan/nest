package drivers

import (
	"database/sql"
	"github.com/rikiihsan/nest/database"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

type SQLiteDriver struct{}

func (d *SQLiteDriver) Open(dsn string) (*sql.DB, error) {
	return sql.Open("sqlite3", dsn)
}

func (d *SQLiteDriver) CreateBunDB(sqlDB *sql.DB) *bun.DB {
	return bun.NewDB(sqlDB, sqlitedialect.New())
}

func (d *SQLiteDriver) GetDriverName() string {
	return "sqlite"
}

// Register SQLite driver
func init() {
	database.RegisterDriver("sqlite", &SQLiteDriver{})
}
