package gosqlitex

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

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

// parseDSN parses and validates data source name then return the cleaned and optimized string.
//
// Dsn is expected to be absolute, relative (e.g app.db) or just :memory:.
//
// mode is the mode to use for the database connection (e.g. "ro", "rw", "rwc" & "memory". Default is "rwc" ).
//
// pragma can be string slice of:
//
// - "journal_mode(WAL)"
//
// - "busy_timeout(5000)"
//
// - "foreign_keys(ON)"
//
// - "cache_size(64)"
//
// - "temp_store(MEMORY)"
//
// - "mmap_size(268435456)"
func parseDSN(dsn, mode string, pragma []string) (string, error) {
	if dsn == "" {
		return "", ErrEmptyDSN
	}

	if strings.HasPrefix(dsn, "file:") || strings.Contains(dsn, "?") {
		return "", ErrInvalidDSN
	}

	if mode == "" {
		mode = "rwc"
	}

	query := url.Values{
		"mode":    []string{mode},
		"_pragma": pragma,
	}

	// Check if it's an in-memory configuration
	// If the user passes ":memory:" or a named variant like "file:memdb_xxx",
	// ensure they map cleanly to a unique memory block.
	if strings.HasPrefix(dsn, ":memory:") {
		query.Set("cache", "shared")

		// If they pass an exact name like ":memory:test1", strip the prefix to use as the name.
		// Fall back to value of memDBName if it's just raw ":memory:".
		memName := memDBName
		if len(dsn) > 8 {
			memName = dsn[8:]
		}

		clean := url.URL{
			Scheme:   "file",
			Opaque:   memName,
			RawQuery: query.Encode(),
		}
		return clean.String(), nil
	}

	var dsnString string
	if filepath.IsAbs(dsn) {
		clean := url.URL{
			Scheme:   "file",
			Host:     "",
			Path:     filepath.ToSlash(dsn),
			RawQuery: query.Encode(),
		}
		dsnString = clean.String()
	} else {
		encodedQuery := query.Encode()
		dsnString = fmt.Sprintf("file://%s", filepath.ToSlash(dsn))
		if encodedQuery != "" {
			dsnString = fmt.Sprintf("%s?%s", dsnString, encodedQuery)
		}
	}

	return dsnString, nil
}