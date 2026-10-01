package gosqlitex

import (
	"errors"
	"fmt"
)
const (
	errTag = "error"
	errTemplate = "%s-%s: %s"
	emptyDSNMsg = "empty string not allowed as Dsn value"
	invalidDSNMsg = "use absolute or relative path for disk or :memory: for in memory db"
	dirNotAllowedMsg = "only files are allowed in migrations folder"
	separatorNotFoundMig = "migration file name separator not found"
	prefixNotNumberMig = "migration file name prefix is not a number"
	migContentChanged = "migration content changed, move the changes into a new migration file"
	invalidFileExtensionMig = "migration file has an invalid extension"
)
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
	// ErrInvalidFileExtensionMig is returned when the migration file has an invalid extension
	ErrInvalidFileExtensionMig = errors.New(fmt.Sprintf(errTemplate, pkgName, errTag, invalidFileExtensionMig))
)