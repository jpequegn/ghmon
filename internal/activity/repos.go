// internal/activity/repos.go
package activity

import (
	"time"

	"github.com/julienpequegnot/ghmon/internal/database"
)

type Repo struct {
	ID          int64
	AccountID   int64
	Name        string
	FullName    string
	Description string
	Language    string
	Stars       int
	CreatedAt   time.Time
}

type RepoRepository struct {
	db *database.DB
}

func NewRepoRepository(db *database.DB) *RepoRepository {
	return &RepoRepository{db: db}
}

func (r *RepoRepository) Add(accountID int64, name, fullName, description, language string, stars int, createdAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT OR IGNORE INTO repos (account_id, name, full_name, description, language, stars, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		accountID, name, fullName, description, language, stars, createdAt,
	)
	return err
}

func (r *RepoRepository) GetNewSince(since time.Time) ([]Repo, error) {
	rows, err := r.db.Query(`
		SELECT id, account_id, name, full_name, description, language, stars, created_at
		FROM repos
		WHERE created_at >= ?
		ORDER BY created_at DESC
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repo
	for rows.Next() {
		var repo Repo
		if err := rows.Scan(&repo.ID, &repo.AccountID, &repo.Name, &repo.FullName, &repo.Description, &repo.Language, &repo.Stars, &repo.CreatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}
	return repos, rows.Err()
}
