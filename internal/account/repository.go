// internal/account/repository.go
package account

import (
	"fmt"
	"time"

	"github.com/julienpequegnot/ghmon/internal/database"
)

type Account struct {
	ID          int64
	Username    string
	Name        string
	AvatarURL   string
	Bio         string
	Followers   int
	Following   int
	AddedAt     time.Time
	LastFetched *time.Time
}

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Add(username, name, avatarURL, bio string) (*Account, error) {
	result, err := r.db.Exec(
		`INSERT INTO accounts (username, name, avatar_url, bio) VALUES (?, ?, ?, ?)`,
		username, name, avatarURL, bio,
	)
	if err != nil {
		return nil, fmt.Errorf("account already exists or error: %w", err)
	}

	id, _ := result.LastInsertId()
	return &Account{
		ID:       id,
		Username: username,
		Name:     name,
	}, nil
}

func (r *Repository) Remove(username string) error {
	var id int64
	err := r.db.QueryRow("SELECT id FROM accounts WHERE username = ?", username).Scan(&id)
	if err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	r.db.Exec("DELETE FROM commits WHERE account_id = ?", id)
	r.db.Exec("DELETE FROM repos WHERE account_id = ?", id)
	r.db.Exec("DELETE FROM stars WHERE account_id = ?", id)

	_, err = r.db.Exec("DELETE FROM accounts WHERE id = ?", id)
	return err
}

func (r *Repository) List() ([]Account, error) {
	rows, err := r.db.Query(`
		SELECT id, username, name, avatar_url, bio, followers, following, added_at, last_fetched
		FROM accounts ORDER BY username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		var lastFetched *time.Time
		if err := rows.Scan(&a.ID, &a.Username, &a.Name, &a.AvatarURL, &a.Bio, &a.Followers, &a.Following, &a.AddedAt, &lastFetched); err != nil {
			return nil, err
		}
		a.LastFetched = lastFetched
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (r *Repository) Get(username string) (*Account, error) {
	var a Account
	var lastFetched *time.Time
	err := r.db.QueryRow(`
		SELECT id, username, name, avatar_url, bio, followers, following, added_at, last_fetched
		FROM accounts WHERE username = ?
	`, username).Scan(&a.ID, &a.Username, &a.Name, &a.AvatarURL, &a.Bio, &a.Followers, &a.Following, &a.AddedAt, &lastFetched)
	if err != nil {
		return nil, err
	}
	a.LastFetched = lastFetched
	return &a, nil
}

func (r *Repository) GetByID(id int64) (*Account, error) {
	var a Account
	var lastFetched *time.Time
	err := r.db.QueryRow(`
		SELECT id, username, name, avatar_url, bio, followers, following, added_at, last_fetched
		FROM accounts WHERE id = ?
	`, id).Scan(&a.ID, &a.Username, &a.Name, &a.AvatarURL, &a.Bio, &a.Followers, &a.Following, &a.AddedAt, &lastFetched)
	if err != nil {
		return nil, err
	}
	a.LastFetched = lastFetched
	return &a, nil
}

func (r *Repository) UpdateLastFetched(id int64) error {
	_, err := r.db.Exec("UPDATE accounts SET last_fetched = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (r *Repository) Exists(username string) bool {
	var count int
	r.db.QueryRow("SELECT COUNT(*) FROM accounts WHERE username = ?", username).Scan(&count)
	return count > 0
}

func (r *Repository) Count() int {
	var count int
	r.db.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&count)
	return count
}
