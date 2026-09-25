package gosqlitex

// db mode e.g. "ro", "rw", "rwc" & "memory".

const (
	pkgName        = "gosqlitex"
	dbDriver       = "sqlite"
	readDBMode     = "ro"
	writeDBMode    = "rwc"
	readDBMaxConn  = 8
	writeDBMaxConn = 1
	memDBName      = "memdb"
)

var (
	pragma = []string{
		"journal_mode(WAL)",    // WAL (Write-Ahead Logging) mode for better concurrency
		"busy_timeout(5000)",   // Wait 5 seconds for a lock to be released
		"foreign_keys(ON)",     // Enforce foreign key constraints
		"cache_size(-64000)",       // 64MB cache for caching pages in memory
		"temp_store(MEMORY)",   // Use memory for temporary tables/sorts - faster sort/joins
		"mmap_size(268435456)", // 256MB for memory mapping - read faster and less disk I/O
		"synchronous(NORMAL)",  // Normal synchronous mode - balance between performance and durability
	}
)
