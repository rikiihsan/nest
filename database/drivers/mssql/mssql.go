package mssql

import (
	"database/sql"

	"github.com/rikiihsan/nest/database"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mssqldialect"
)

type MSSQLDriver struct{}

func (d *MSSQLDriver) Open(dsn string) (*sql.DB, error) {
	return sql.Open("sqlserver", dsn)
}

func (d *MSSQLDriver) CreateBunDB(sqlDB *sql.DB) *bun.DB {
	return bun.NewDB(sqlDB, mssqldialect.New())
}

func (d *MSSQLDriver) GetDriverName() string {
	return "sqlserver"
}

// Register MSSQL driver
func init() {
	database.RegisterDriver("sqlserver", &MSSQLDriver{})
}
