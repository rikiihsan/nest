package drivers

import (
	"database/sql"
	"github.com/rikiihsan/nest/database"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type PostgreSQLDriver struct{}

func (d *PostgreSQLDriver) Open(dsn string) (*sql.DB, error) {
	return sql.Open("pgx", dsn)
}

func (d *PostgreSQLDriver) CreateBunDB(sqlDB *sql.DB) *bun.DB {
	return bun.NewDB(sqlDB, pgdialect.New())
}

func (d *PostgreSQLDriver) GetDriverName() string {
	return "pgx"
}

// Register PostgreSQL driver
func init() {
	database.RegisterDriver("pgx", &PostgreSQLDriver{})
}
