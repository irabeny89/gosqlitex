package gosqlitex

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// DBPool creates a db pool for sqlite.
//
// ℹ️ NOTE: sqlite WAL mode requires -shm and -wal files.
//
// Initialize the writer connection first (1 connection) if creating multiple db pools(read and writer).
// This is because WAL mode requires the "writer" connection (rwc) to create the -shm
// and -wal files and the "reader" connection (ro) cannot create them.
// 
// 🆒 Better to use the DBclient to get an optimized db read and write pools.
func DBPool(dsn string, maxConn int) (*sql.DB, error) {
	if dsn == "" {
		return nil, ErrInvalidDSN
	}
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxConn)
	db.SetMaxIdleConns(maxConn)
	// no max lifetime - db will be open until the application closes it
	db.SetConnMaxLifetime(0)
	return db, nil
}
