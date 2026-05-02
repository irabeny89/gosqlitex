package mig8

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateFile(t *testing.T) {
	tmpDir := t.TempDir()
	fileName := "add_users_table"
	sep := "_"

	filePath, err := generateFile(tmpDir, fileName, sep)
	if err != nil {
		t.Fatalf("generateFile failed: %v", err)
	}

	// check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("file was not created: %s", filePath)
	}

	// check filename format: <timestamp>_<filename>.sql
	base := filepath.Base(filePath)
	if !strings.HasSuffix(base, sep+fileName+".sql") {
		t.Errorf("unexpected filename format: %s", base)
	}

	// check if it creates the directory
	nestedDir := filepath.Join(tmpDir, "nested")
	filePath2, err := generateFile(nestedDir, fileName, sep)
	if err != nil {
		t.Fatalf("generateFile with nested dir failed: %v", err)
	}
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Errorf("nested directory was not created")
	}
	if _, err := os.Stat(filePath2); os.IsNotExist(err) {
		t.Errorf("file in nested directory was not created: %s", filePath2)
	}
}
