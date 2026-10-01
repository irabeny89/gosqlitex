package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/irabeny89/gosqlitex"
)

type ParsedArgs struct {
	dir   string
	db    string
	title string
	file  string
	sep   string
	run   bool
	list  bool
}

const sep = "_"

// generateFile generates a migration file in the form <timestamp><sep><filename>.sql and place in the passed directory.
func generateFile(dir, title, sep string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	timestamp := time.Now().Format("20060102150405")
	title = strings.ReplaceAll(title, " ", sep)
	ext := ".sql"
	fileName := fmt.Sprintf("%s%s%s%s", timestamp, sep, title, ext)
	return os.Create(filepath.Join(dir, fileName))
}

// parseArgs parses the command line arguments and returns a ParsedArgs struct.
func args() (*ParsedArgs, error) {
	dir := os.Getenv("MIG_DIR")
	db := os.Getenv("DB_PATH")
	dirFlag := flag.String("dir", dir, "Path to the migration directory.")
	dbFlag := flag.String("db", db, "Path to the database file.")
	titleFlag := flag.String("title", "", "Title of the migration.")
	fileFlag := flag.String("filename", "", "Name of the migration file. This generates the sql file for you.")
	sepFlag := flag.String("sep", sep, "Separator to use when generating a filename. Default is '_'")
	runFlag := flag.Bool("run", false, "Run the migration.")

	flag.Parse()
	args := new(ParsedArgs{
		dir:   *dirFlag,
		db:    *dbFlag,
		title: *titleFlag,
		file:  *fileFlag,
		sep:   *sepFlag,
		run:   *runFlag,
	})
	return args, nil
}

// requiredArgs checks if the required fields are set in the parsed arguments.
//
// ℹ️Use this function to check if the required cli arguments are set before taking action.
func requiredArgs(args *ParsedArgs, requiredFields ...string) bool {
	if args == nil || requiredFields == nil {
		return false
	}
	var (
		ok         bool
		refV       = reflect.ValueOf(*args)
		fieldLen   = refV.NumField()
		fieldNames = make([]string, fieldLen)
	)
	for i := 0; i < fieldLen; i++ {
		fieldNames[i] = refV.Type().Field(i).Name
	}

	for i, name := range fieldNames {
		for idx, field := range requiredFields {
			if idx >= fieldLen {
				break // no more required fields to check
			}
			if name == field {
				if refV.Field(i).IsZero() {
					return false
				}
				ok = true
			}
		}
	}

	return ok
}

func getDBClient(dsn string) (*gosqlitex.DBClient, error) {
	dbClient, err := gosqlitex.NewDBClient(&gosqlitex.DBConfig{
		Dsn: dsn,
	})
	if err != nil {
		return nil, err
	}
	if err := dbClient.Ping(); err != nil {
		return nil, err
	}
	return dbClient, nil
}

func main() {
	log := func(msg string, err error) {
		if err != nil {
			wErr := fmt.Errorf("❗%s: %w", msg, err)
			fmt.Println(wErr)
		}
		fmt.Println(msg)
	}
	parsedArgs, err := args()
	if err != nil {
		log("failed to parse cli arguments", err)
		os.Exit(1)
	}
	// generate migration file
	// e.g mig8 -dir ./migrations -title "add profile table"
	if requiredArgs(parsedArgs, "dir", "title") {
		f, err := generateFile(parsedArgs.dir, parsedArgs.title, parsedArgs.sep)
		if err != nil {
			log("Failed to create migration file", err)
			os.Exit(1)
		}
		defer f.Close()
		log(fmt.Sprintf("Migration file created: %s", f.Name()), nil)
		os.Exit(0)
	}
	// run all migration files
	// e.g mig8 --db app.db --dir ./migrations --run
	if requiredArgs(parsedArgs, "db", "dir") {
		db, err := getDBClient(parsedArgs.db)
		if err != nil {
			log("failed to create db client", err)
		}
		if migErr := db.RunMigrations(parsedArgs.dir, sep); migErr != nil {
			log("Failed to run migrations", migErr)
			os.Exit(1)
		}
		log("Migrations ran successfully", nil)
		os.Exit(0)
	}
	// run a migration file
	if requiredArgs(parsedArgs, "db", "dir", "file") {
		db, err := getDBClient(parsedArgs.db)
		if err != nil {
			log("failed to create db client", err)
		}
		if migErr := db.RunOneMigration(parsedArgs.file, parsedArgs.dir, parsedArgs.sep); migErr != nil {
			log("Failed to run migration", migErr)
			os.Exit(1)
		}
		log("Migration ran successfully", nil)
		os.Exit(0)
	}

	log("♻️  Wrong usage, try again with arguments below", nil)
	flag.Usage()
	log("ℹ️  For more info: https://github.com/irabeny89/gosqlitex#gosqlitex", nil)
}
