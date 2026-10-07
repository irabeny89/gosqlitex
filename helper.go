// helper package contains utility functions for the gosqlitex package.

package gosqlitex

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// validateFilename checks that the migration filename is valid.
// It must be in the format <timestamp><separator><name>.sql
func validateFilename(name, sep string) error {
	// split the filename on the first separator to get the timestamp.
	v, _, ok := strings.Cut(name, sep)
	if !ok {
		return ErrSeparatorNotFoundMig
	}
	// check if the timestamp is a valid integer
	if _, err := strconv.Atoi(v); err != nil {
		return ErrPrefixNotNumberMig
	}
	// ensure .sql extension
	if !strings.HasSuffix(name, ".sql") {
		return ErrInvalidFileExtensionMig
	}
	return nil
}

// CreateDSN constructs a DSN (Data Source Name) for the database connection.
// 
// You only pass the file path or named memory e.g file:memdb for each pool instance. The pragma and other configurations are set automatically.
func CreateDSN(path string, isRead, isMemory bool) string {
	var query url.Values
	if isRead {
		readPragma := slices.DeleteFunc(pragma, func(p string) bool {
			return p == "journal_mode" || p == "synchronous" || strings.HasPrefix(p, "cache(")
		})
		query = url.Values{
			"_pragma": readPragma,
		}
		if isMemory {
			query.Add("_pragma", "query_only(ON)")
			query.Set("cache", "shared")
			query.Set("mode", "memory")
		} else {
			query.Set("mode", readDBMode)
		}
	} else {
		query = url.Values{
			"_pragma": pragma,
		}
		if isMemory {
			query.Set("mode", "memory")
			query.Set("cache", "shared")
		} else {
			query.Set("mode", writeDBMode)
		}
	}
	return fmt.Sprintf("%s?%s", path, query.Encode())
}

// setupPools initializes the read and write pools for the database client.
// 
// You do not need to ping the pools because they are pinged and confirmed ready.
func setupPools(rDSN, wDSN string) (*DBClient, error) {
	// NOTE: Open the WRITE pool FIRST so it physically creates the database file
	wPool, err := DBPool(wDSN, writeDBMaxConn)
	if err != nil {
		return nil, err
	}
	// Ping the write pool immediately to force file creation and execute WAL activation
	if err := wPool.Ping(); err != nil {
		wPool.Close()
		return nil, fmt.Errorf("failed to initialize write pool: %w", err)
	}
	rPool, err := DBPool(rDSN, readDBMaxConn)
	if err != nil {
		wPool.Close()
		return nil, err
	}
	if err := rPool.Ping(); err != nil {
		rPool.Close()
		wPool.Close()
		return nil, fmt.Errorf("failed to initialize read pool: %w", err)
	}
	return &DBClient{ReadPool: rPool, WritePool: wPool}, nil
}
