package database

import (
	"os"
	"testing"
)

// TestConnectWithMissingEnvVars tests that Connect returns an error when environment variables are missing
func TestConnectWithMissingEnvVars(t *testing.T) {
	// Save the original environment variables
	origHost := os.Getenv("DB_HOST")
	origUser := os.Getenv("DB_USER")
	origPassword := os.Getenv("DB_PASSWORD")
	origDBName := os.Getenv("DB_NAME")
	origPort := os.Getenv("DB_PORT")

	// Restore the original environment variables after the test
	defer func() {
		os.Setenv("DB_HOST", origHost)
		os.Setenv("DB_USER", origUser)
		os.Setenv("DB_PASSWORD", origPassword)
		os.Setenv("DB_NAME", origDBName)
		os.Setenv("DB_PORT", origPort)
	}()

	// Clear all the database environment variables
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("DB_PORT")

	// Attempt to connect should fail but not panic
	db, err := Connect()
	if err == nil {
		t.Error("Connect() should return an error when environment variables are missing")
	}
	if db != nil {
		t.Error("Connect() should return nil DB when connection fails")
	}
}

// TestConnectWithInvalidCredentials tests that Connect returns an error with invalid credentials
func TestConnectWithInvalidCredentials(t *testing.T) {
	// Skip in CI environment or when not explicitly enabled
	if os.Getenv("RUN_DB_TESTS") != "true" {
		t.Skip("Skipping database connection test. Set RUN_DB_TESTS=true to enable.")
	}

	// Save the original environment variables
	origHost := os.Getenv("DB_HOST")
	origUser := os.Getenv("DB_USER")
	origPassword := os.Getenv("DB_PASSWORD")
	origDBName := os.Getenv("DB_NAME")
	origPort := os.Getenv("DB_PORT")

	// Restore the original environment variables after the test
	defer func() {
		os.Setenv("DB_HOST", origHost)
		os.Setenv("DB_USER", origUser)
		os.Setenv("DB_PASSWORD", origPassword)
		os.Setenv("DB_NAME", origDBName)
		os.Setenv("DB_PORT", origPort)
	}()

	// Set invalid credentials
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "nonexistentuser")
	os.Setenv("DB_PASSWORD", "wrongpassword")
	os.Setenv("DB_NAME", "nonexistentdb")
	os.Setenv("DB_PORT", "5432")

	// Attempt to connect should fail but not panic
	db, err := Connect()
	if err == nil {
		t.Error("Connect() should return an error with invalid credentials")
	}
	if db != nil {
		t.Error("Connect() should return nil DB when connection fails")
	}
}

// Example test for successful connection - only runs when explicitly enabled
// and when database is properly configured
func TestConnectSuccessful(t *testing.T) {
	// Skip unless explicitly enabled
	if os.Getenv("RUN_DB_TESTS") != "true" {
		t.Skip("Skipping database connection test. Set RUN_DB_TESTS=true to enable.")
	}

	// Check if required environment variables are set
	requiredVars := []string{"DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_PORT"}
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			t.Skipf("Skipping test because %s environment variable is not set", v)
		}
	}

	// Attempt to connect
	db, err := Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if db == nil {
		t.Fatal("Connect() returned nil DB")
	}

	// Test the connection
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get database connection: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
}
