// internal/account/repository_test.go
package account

import (
	"path/filepath"
	"testing"

	"github.com/julienpequegnot/ghmon/internal/database"
)

func setupTestDB(t *testing.T) *database.DB {
	tmpDir := t.TempDir()
	db, err := database.New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	return db
}

func TestAddAccount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)

	acc, err := repo.Add("torvalds", "Linus Torvalds", "", "")
	if err != nil {
		t.Fatalf("failed to add account: %v", err)
	}

	if acc.Username != "torvalds" {
		t.Errorf("expected username 'torvalds', got '%s'", acc.Username)
	}
}

func TestAddDuplicateAccount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)

	_, err := repo.Add("torvalds", "Linus Torvalds", "", "")
	if err != nil {
		t.Fatalf("failed to add account: %v", err)
	}

	_, err = repo.Add("torvalds", "Linus Torvalds", "", "")
	if err == nil {
		t.Error("expected error when adding duplicate account")
	}
}

func TestListAccounts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)

	repo.Add("torvalds", "Linus", "", "")
	repo.Add("antirez", "Salvatore", "", "")

	accounts, err := repo.List()
	if err != nil {
		t.Fatalf("failed to list accounts: %v", err)
	}

	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(accounts))
	}
}

func TestRemoveAccount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)

	repo.Add("torvalds", "Linus", "", "")

	err := repo.Remove("torvalds")
	if err != nil {
		t.Fatalf("failed to remove account: %v", err)
	}

	accounts, _ := repo.List()
	if len(accounts) != 0 {
		t.Errorf("expected 0 accounts after removal, got %d", len(accounts))
	}
}
