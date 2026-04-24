package gosqlitex

import (
	"database/sql"
	"net/url"

	_ "modernc.org/sqlite"
)

// MARK: - Structs

// DbClient is a client that is used for reading and writing to the database.
// It manages a read pool for concurrent reads and a single-connection write pool
// to ensure write safety and optimal performance with SQLite WAL mode.
type DbClient struct {
	// Read pool (multi connections). E.g. SELECT.
	readPool *sql.DB
	// Write pool (single connection). E.g. INSERT, UPDATE, DELETE.
	writePool *sql.DB
}

type sqliteConfig struct {
	// dbPath is the path to the database file E.g "app.db".
	dbPath string
	// driver is the driver to use for the database connection. E.g "sqlite".
	driver string
	// mode is the mode to use for the database connection (e.g. "ro", "rw", "rwc" & "memory". Default is "rwc" ).
	mode string
	// pragmas are the pragmas to use for the database connection. E.g []string{"journal_mode(WAL)", "busy_timeout(5000)", "foreign_keys(ON)"}
	pragma []string
}

// MARK: - Interfaces

// MARK: - Const & Var

// MARK: - Methods

func (cnf *sqliteConfig) newPool(maxConn int) (*sql.DB, error) {
	const scheme = "file"
	query := url.Values{
		"mode":    []string{cnf.mode},
		"_pragma": cnf.pragma,
	}
	dsn := "file:" + cnf.dbPath + "?" + query.Encode()
	// fmt.Println("DSN:", dsn)
	db, err := sql.Open(cnf.driver, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxConn)
	db.SetMaxIdleConns(maxConn)
	// no max lifetime - db will be open until the application closes it
	db.SetConnMaxLifetime(0)
	return db, nil
}

// Ping checks if the database connection is alive.
func (c *DbClient) Ping() error {
	err := c.readPool.Ping()
	if err != nil {
		return err
	}
	err = c.writePool.Ping()
	if err != nil {
		return err
	}
	return nil
}

// Query executes a query that returns rows, using the read pool. E.g SELECT * FROM users
func (c *DbClient) Query(query string, args ...any) (*sql.Rows, error) {
	return c.readPool.Query(query, args...)
}

// QueryRow executes a query that returns a single row, using the read pool. E.g SELECT * FROM users WHERE id = 1
func (c *DbClient) QueryRow(query string, args ...any) *sql.Row {
	return c.readPool.QueryRow(query, args...)
}

// Exec executes a query that returns a result, using the write pool. E.g INSERT, UPDATE, DELETE, CREATE, DROP, etc
func (c *DbClient) Exec(query string, args ...any) (sql.Result, error) {
	return c.writePool.Exec(query, args...)
}

// MARK: - Private Func

// MARK: - Public Func

// Config holds the configuration for opening a new database connection pool.
type Config struct {
	// DbPath is the path to the database file (e.g., "app.db").
	DbPath string
	// Driver is the name of the database driver to use (e.g., "sqlite").
	Driver string
}

// Open opens a new database connection with optimized SQLite settings.
// It returns a DbClient which manages separate read and write pools.
// The default configuration uses WAL mode and optimized pragmas for performance.
func Open(cnf *Config) (*DbClient, error) {
	// sqlite performance default
	var (
		rMaxConn = 8
		wMaxConn = 1
		dbPath   = "app.db"
		driver   = "sqlite"
		pragma   = []string{
			"journal_mode(WAL)",    // WAL (Write-Ahead Logging) mode for better concurrency
			"busy_timeout(5000)",   // Wait 5 seconds for a lock to be released
			"foreign_keys(ON)",     // Enforce foreign key constraints
			"cache_size(64)",       // 64MB cache for caching pages in memory
			"temp_store(MEMORY)",   // Use memory for temporary tables/sorts - faster sort/joins
			"mmap_size(268435456)", // 256MB for memory mapping - read faster and less disk I/O
		}
	)
	if cnf.DbPath != "" {
		dbPath = cnf.DbPath
	}
	if cnf.Driver != "" {
		driver = cnf.Driver
	}
	readConfig := &sqliteConfig{
		dbPath: dbPath,
		driver: driver,
		mode:   "ro",
		pragma: pragma,
	}
	writeConfig := &sqliteConfig{
		dbPath: dbPath,
		driver: driver,
		mode:   "rwc",
		pragma: append(pragma, "synchronous(NORMAL)"),
	}

	//! NOTE: sqlite WAL mode requires -shm and -wal files.
	//! Initialize the writer connection first (1 connection).
	//! This is because WAL mode requires the "writer" connection (rwc) to create the -shm and -wal files,
	//! and the "reader" connection (ro) cannot create them.

	w, err := writeConfig.newPool(wMaxConn)
	if err != nil {
		return nil, err
	}
	// Ping the writer to ensure the file is created and WAL mode is initialized.
	if err := w.Ping(); err != nil {
		return nil, err
	}

	r, err := readConfig.newPool(rMaxConn)
	if err != nil {
		return nil, err
	}

	client := &DbClient{
		readPool:  r,
		writePool: w,
	}

	return client, nil
}
