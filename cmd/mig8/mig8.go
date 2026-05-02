package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/irabeny89/gosqlitex"
)

// generateFile generates a migration file in the form <timestamp><sep><filename>.sql.
func generateFile(dir, fileName, sep string) (string, error) {
	// create migration directory if it does not exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// prefix filename with timestamp
	fileName = time.Now().Format("20060102150405") + sep + fileName + ".sql"
	file, err := os.Create(filepath.Join(dir, fileName)) 
	if err != nil {
		return "", err
	}
	defer file.Close()

	return file.Name(), nil
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	dir, ok := os.LookupEnv("MIG_DIR")
	if !ok {
		log.Warn("MIG_DIR not found in environment")
	}
	db, ok := os.LookupEnv("DB_PATH")
	if !ok {
		log.Warn("DB_PATH not found in environment")
	}

	
	dirFlag := flag.String("dir", dir, "Path to the migration directory.")
	dbFlag := flag.String("db", db, "Path to the database file.")
	fileFlag := flag.String("file", "", "Name of the migration file. This generates the sql file for you.")
	sepFlag := flag.String("sep", "_", "Separator to use when generating a filename.")
	runFlag := flag.Bool("run", false, "Run the migration.")
	listFlag := flag.Bool("list", false, "List all ran migrations.")

	flag.Parse()

	if *dbFlag == "" {
		log.Error("DB_PATH not provided")
		os.Exit(1)
	}
	if *dirFlag == "" {
		log.Error("MIG_DIR not provided")
		os.Exit(1)
	}

	dbClient, err := gosqlitex.Open(&gosqlitex.Config{
		DbPath: *dbFlag,
	})
	if err != nil {
		log.Error("Failed to open database", "error", err.Error())
		os.Exit(1)
	}
	defer dbClient.Close()

	// generate migration file
	if *fileFlag != "" && !*runFlag && !*listFlag {
		filePath, err := generateFile(*dirFlag, *fileFlag, *sepFlag); 
		if err != nil {
			log.Error("Failed to generate migration file", "error", err.Error())
			os.Exit(1)
		} else {
			log.Info("Migration file generated", "file", filePath)
			os.Exit(0)
		}
	}

	// run migration files
	if *runFlag {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		err := dbClient.RunMigrationsContext(ctx, *dirFlag, *sepFlag)
		if err != nil {
			log.Error("Failed to run migrations", "error", err.Error())
			os.Exit(1)
		} else {
			log.Info("Migrations ran successfully")
			os.Exit(0)
		}
	}

	// list ran migrations
	if *listFlag {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		migrations, err := dbClient.ListMigrationsContext(ctx)
		if err != nil {
			log.Error("Failed to list migrations", "error", err.Error())
			os.Exit(1)
		} else {
			log.Info("Migrations listed successfully:\n"+strings.Join(migrations, "\n"))
			os.Exit(0)
		}
	}
}
