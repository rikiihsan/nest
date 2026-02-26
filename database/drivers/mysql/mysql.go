package drivers

import (
	"database/sql"
	"github.com/rikiihsan/nest/database"

	_ "github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
)

type MySQLDriver struct{}

func (d *MySQLDriver) Open(dsn string) (*sql.DB, error) {
	return sql.Open("mysql", dsn)
}

func (d *MySQLDriver) CreateBunDB(sqlDB *sql.DB) *bun.DB {
	return bun.NewDB(sqlDB, mysqldialect.New())
}

func (d *MySQLDriver) GetDriverName() string {
	return "mysql"
}

// Register MySQL driver
func init() {
	database.RegisterDriver("mysql", &MySQLDriver{})
}
