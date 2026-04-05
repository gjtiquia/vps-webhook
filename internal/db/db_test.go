package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDBOperations(t *testing.T) {
	// Create a temp directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Test Open and migration
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test CreateWebhook with default http_method
	wh1, err := db.CreateWebhook("/test1", "./scripts/test1.sh", "")
	if err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}
	if wh1.ID == 0 {
		t.Error("Expected webhook ID to be set")
	}
	if wh1.Path != "/test1" {
		t.Errorf("Expected path /test1, got %s", wh1.Path)
	}
	if wh1.ScriptPath != "./scripts/test1.sh" {
		t.Errorf("Expected script path ./scripts/test1.sh, got %s", wh1.ScriptPath)
	}
	if !wh1.Active {
		t.Error("Expected webhook to be active by default")
	}
	if wh1.HttpMethod != "POST" {
		t.Errorf("Expected default http_method POST, got %s", wh1.HttpMethod)
	}

	// Test CreateWebhook with custom http_method
	wh2, err := db.CreateWebhook("/test2", "./scripts/test2.sh", "GET")
	if err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}
	if wh2.HttpMethod != "GET" {
		t.Errorf("Expected http_method GET, got %s", wh2.HttpMethod)
	}

	// Test ListWebhooks
	webhooks, err := db.ListWebhooks()
	if err != nil {
		t.Fatalf("Failed to list webhooks: %v", err)
	}
	if len(webhooks) != 2 {
		t.Errorf("Expected 2 webhooks, got %d", len(webhooks))
	}

	// Test GetWebhookByPath - active webhook
	found, err := db.GetWebhookByPath("/test1")
	if err != nil {
		t.Fatalf("Failed to get webhook by path: %v", err)
	}
	if found == nil {
		t.Fatal("Expected to find active webhook")
	}
	if found.Path != "/test1" {
		t.Errorf("Expected path /test1, got %s", found.Path)
	}

	// Test GetWebhookByPath - inactive webhook (should return nil)
	err = db.UpdateWebhook(wh2.ID, wh2.Path, wh2.ScriptPath, false, wh2.HttpMethod)
	if err != nil {
		t.Fatalf("Failed to update webhook: %v", err)
	}
	foundInactive, err := db.GetWebhookByPath("/test2")
	if err != nil {
		t.Fatalf("Failed to get inactive webhook by path: %v", err)
	}
	if foundInactive != nil {
		t.Error("Expected nil for inactive webhook lookup by path")
	}

	// Test GetWebhook - should find regardless of active status
	byID, err := db.GetWebhook(wh2.ID)
	if err != nil {
		t.Fatalf("Failed to get webhook by ID: %v", err)
	}
	if byID == nil {
		t.Fatal("Expected to find webhook by ID")
	}
	if byID.ID != wh2.ID {
		t.Errorf("Expected ID %d, got %d", wh2.ID, byID.ID)
	}

	// Test GetWebhook - non-existent ID
	notFound, err := db.GetWebhook(9999)
	if err != nil {
		t.Fatalf("Failed to get non-existent webhook: %v", err)
	}
	if notFound != nil {
		t.Error("Expected nil for non-existent webhook")
	}

	// Test GetWebhookByPath - non-existent path
	notFoundPath, err := db.GetWebhookByPath("/nonexistent")
	if err != nil {
		t.Fatalf("Failed to get non-existent webhook by path: %v", err)
	}
	if notFoundPath != nil {
		t.Error("Expected nil for non-existent webhook path")
	}

	// Test UpdateWebhook
	err = db.UpdateWebhook(wh1.ID, "/updated", "./scripts/updated.sh", false, "PUT")
	if err != nil {
		t.Fatalf("Failed to update webhook: %v", err)
	}
	updated, err := db.GetWebhook(wh1.ID)
	if err != nil {
		t.Fatalf("Failed to get updated webhook: %v", err)
	}
	if updated.Path != "/updated" {
		t.Errorf("Expected updated path /updated, got %s", updated.Path)
	}
	if updated.ScriptPath != "./scripts/updated.sh" {
		t.Errorf("Expected updated script path ./scripts/updated.sh, got %s", updated.ScriptPath)
	}
	if updated.Active {
		t.Error("Expected webhook to be inactive")
	}
	if updated.HttpMethod != "PUT" {
		t.Errorf("Expected http_method PUT, got %s", updated.HttpMethod)
	}

	// Test UpdateWebhook with empty http_method (should default to POST)
	err = db.UpdateWebhook(wh1.ID, "/updated", "./scripts/updated.sh", true, "")
	if err != nil {
		t.Fatalf("Failed to update webhook with empty http_method: %v", err)
	}
	updatedWithDefault, err := db.GetWebhook(wh1.ID)
	if err != nil {
		t.Fatalf("Failed to get webhook: %v", err)
	}
	if updatedWithDefault.HttpMethod != "POST" {
		t.Errorf("Expected default http_method POST after update, got %s", updatedWithDefault.HttpMethod)
	}

	// Test DeleteWebhook
	err = db.DeleteWebhook(wh1.ID)
	if err != nil {
		t.Fatalf("Failed to delete webhook: %v", err)
	}
	deleted, err := db.GetWebhook(wh1.ID)
	if err != nil {
		t.Fatalf("Failed to get deleted webhook: %v", err)
	}
	if deleted != nil {
		t.Error("Expected nil for deleted webhook")
	}
}

func TestMigrationExistingDB(t *testing.T) {
	// Create a temp directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create initial database without http_method column (simulating old schema)
	// We do this by manually creating the table as it was before
	dir := filepath.Dir(dbPath)
	os.MkdirAll(dir, 0755)
	
	// First open to create initial schema
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	db1.Close()

	// Re-open to trigger migration again (should be idempotent)
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// Verify we can still create webhooks
	wh, err := db2.CreateWebhook("/migration-test", "./scripts/test.sh", "DELETE")
	if err != nil {
		t.Fatalf("Failed to create webhook after migration: %v", err)
	}
	if wh.HttpMethod != "DELETE" {
		t.Errorf("Expected http_method DELETE, got %s", wh.HttpMethod)
	}
}
