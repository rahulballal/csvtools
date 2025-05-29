package main

import (
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// sanitizeName cleans a string to be a valid SQL identifier (table or column name).
// It replaces non-alphanumeric characters with underscores and ensures it starts with a letter or underscore.
func sanitizeName(name string) string {
	// Replace non-alphanumeric characters (except underscore) with underscore
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	sanitized := reg.ReplaceAllString(name, "_")

	// Ensure it doesn't start with a number
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "_" + sanitized
	}

	// Remove leading/trailing underscores if multiple
	sanitized = strings.Trim(sanitized, "_")

	// If after sanitization it's empty, provide a default
	if sanitized == "" {
		return "unnamed_column"
	}
	return sanitized
}

// processCSVFile reads a CSV file, creates a table in the database, and inserts its data.
func processCSVFile(db *sql.DB, filePath string) error {
	fmt.Printf("Processing file: %s\n", filePath)

	// Open the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file %s: %w", filePath, err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Read the header row
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header from %s: %w", filePath, err)
	}

	// Sanitize header names for column names
	sanitizedHeaders := make([]string, len(header))
	for i, h := range header {
		sanitizedHeaders[i] = sanitizeName(h)
	}

	// Determine table name from file name
	fileName := filepath.Base(filePath)
	tableName := sanitizeName(strings.TrimSuffix(fileName, filepath.Ext(fileName)))
	if tableName == "" {
		tableName = "default_table" // Fallback if file name is empty or un-sanitizable
	}

	// Construct CREATE TABLE SQL
	var columns []string
	for _, h := range sanitizedHeaders {
		columns = append(columns, fmt.Sprintf("%s TEXT", h))
	}
	createTableSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(columns, ", "))

	// Execute CREATE TABLE
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}
	fmt.Printf("Table '%s' created or already exists.\n", tableName)

	// Prepare INSERT statement
	placeholders := make([]string, len(sanitizedHeaders))
	for i := range sanitizedHeaders {
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(sanitizedHeaders, ", "),
		strings.Join(placeholders, ", "),
	)

	// Read and insert data rows
	tx, err := db.Begin() // Start a transaction for faster inserts
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			err := tx.Rollback()
			if err != nil {
				return
			}
			panic(r) // Re-throw panic after rollback
		} else if err != nil {
			err := tx.Rollback()
			if err != nil {
				return
			} // Rollback on error
		} else {
			err = tx.Commit() // Commit on success
		}
	}()

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement for %s: %w", tableName, err)
	}
	defer func(stmt *sql.Stmt) {
		_ = stmt.Close()
	}(stmt)

	insertedRows := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			return fmt.Errorf("failed to read record from %s: %w", filePath, err)
		}

		// Ensure the record has enough values for the columns
		if len(record) < len(sanitizedHeaders) {
			// Pad with empty strings if record has fewer columns than header
			for i := len(record); i < len(sanitizedHeaders); i++ {
				record = append(record, "")
			}
		} else if len(record) > len(sanitizedHeaders) {
			// Truncate if record has more columns than header
			record = record[:len(sanitizedHeaders)]
		}

		// Convert []string to []interface{} for stmt.Exec
		args := make([]interface{}, len(record))
		for i, v := range record {
			args[i] = v
		}

		_, err = stmt.Exec(args...)
		if err != nil {
			return fmt.Errorf("failed to insert row into %s: %w", tableName, err)
		}
		insertedRows++
	}

	fmt.Printf("Successfully inserted %d rows into table '%s'.\n", insertedRows, tableName)
	return nil
}

func main() {
	// Get source and destination directories from the flags passed
	var sourceDir string
	var destDir string
	flag.StringVar(&sourceDir, "src", "", "Directory containing CSV files")
	flag.StringVar(&destDir, "dest", "", "Directory containing SQLite db")
	flag.Parse()

	if sourceDir == "" || destDir == "" {
		fmt.Println("sourceDir and destDir are required")
		os.Exit(1)
	}

	databaseFilePath := fmt.Sprintf("%s/%s.db", destDir, "combined.db")

	// Open (or create) the SQLite database
	db, err := sql.Open("sqlite3", databaseFilePath)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer func(db *sql.DB) {
		_ = db.Close()
	}(db)

	// Ping the database to ensure connection is established
	if err = db.Ping(); err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		return
	}
	fmt.Printf("Successfully connected to SQLite database: %s\n", databaseFilePath)

	// Read all CSV files in the specified directory
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		fmt.Printf("Error reading CSV directory: %v\n", err)
		return
	}

	for _, fileInfo := range files {
		if !fileInfo.IsDir() && strings.HasSuffix(fileInfo.Name(), ".csv") {
			filePath := filepath.Join(sourceDir, fileInfo.Name())
			err := processCSVFile(db, filePath)
			if err != nil {
				fmt.Printf("Error processing %s: %v\n", filePath, err)
			}
		}
	}

	fmt.Println("\nAll CSV files processed. You can now inspect the database.")
}
