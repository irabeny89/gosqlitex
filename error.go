package gosqlitex

import (
	"errors"
	"fmt"
)

const errTag = "error"
const errTemplate = "%s-%s: %s"
const emptyDSNMsg = "empty string not allowed as Dsn value"
const invalidDSNMsg = "use absolute or relative path for disk or :memory: for in memory db"
const dirNotAllowedMsg = "only files are allowed in migrations folder"
const separatorNotFoundMig = "migration file name separator not found"
const prefixNotNumberMig = "migration file name prefix is not a number"
const migContentChanged = "migration content changed, move the changes into a new migration file"

// Errors returned by gosqlitex
var (
	// ErrEmptyDSN is returned when the Dsn value is empty
	ErrEmptyDSN = errors.New(fmt.Sprintf(errTemplate, pkgName, errTag, emptyDSNMsg))
	// ErrInvalidDSN is returned when the Dsn value is invalid
	ErrInvalidDSN = errors.New(fmt.Sprintf(errTemplate, pkgName, errTag, invalidDSNMsg))
	// ErrDirNotAllowedMig is returned when the migrations folder contains a directory
	ErrDirNotAllowedMig = errors.New(fmt.Sprintf(errTemplate, pkgName, errTag, dirNotAllowedMsg))
	// ErrSeparatorNotFoundMig is returned when the migration file name separator is not found
	ErrSeparatorNotFoundMig = errors.New(fmt.Sprintf(errTemplate, pkgName, errTag, separatorNotFoundMig))
	// ErrPrefixNotNumberMig is returned when the migration file name prefix is not a number
	ErrPrefixNotNumberMig = errors.New(fmt.Sprintf(errTemplate, pkgName, errTag, prefixNotNumberMig))
	// ErrMigContentChanged is returned when the migration content has changed
	ErrMigContentChanged = errors.New(fmt.Sprintf(errTemplate, pkgName, errTag, migContentChanged))
)