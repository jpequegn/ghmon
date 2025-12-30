// internal/activity/stars.go
package activity

import (
	"time"

	"github.com/julienpequegnot/ghmon/internal/database"
)

type Star struct {
	ID              int64
	AccountID       int64
	RepoFullName    string
	RepoDescription string
	RepoLanguage    string
	RepoStars       int
	StarredAt       time.Time
}

type StarRepository struct {
	db *database.DB
}

func NewStarRepository(db *database.DB) *StarRepository {
	return &StarRepository{db: db}
}

func (r *StarRepository) Add(accountID int64, repoFullName, description, language string, stars int, starredAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT OR IGNORE INTO stars (account_id, repo_full_name, repo_description, repo_language, repo_stars, starred_at) VALUES (?, ?, ?, ?, ?, ?)`,
		accountID, repoFullName, description, language, stars, starredAt,
	)
	return err
}

func (r *StarRepository) GetSince(since time.Time) ([]Star, error) {
	rows, err := r.db.Query(`
		SELECT id, account_id, repo_full_name, repo_description, repo_language, repo_stars, starred_at
		FROM stars
		WHERE starred_at >= ?
		ORDER BY starred_at DESC
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stars []Star
	for rows.Next() {
		var s Star
		if err := rows.Scan(&s.ID, &s.AccountID, &s.RepoFullName, &s.RepoDescription, &s.RepoLanguage, &s.RepoStars, &s.StarredAt); err != nil {
			return nil, err
		}
		stars = append(stars, s)
	}
	return stars, rows.Err()
}
