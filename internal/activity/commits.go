// internal/activity/commits.go
package activity

import (
	"time"

	"github.com/julienpequegnot/ghmon/internal/database"
)

type Commit struct {
	ID          int64
	AccountID   int64
	RepoName    string
	SHA         string
	Message     string
	CommittedAt time.Time
}

type CommitRepository struct {
	db *database.DB
}

func NewCommitRepository(db *database.DB) *CommitRepository {
	return &CommitRepository{db: db}
}

func (r *CommitRepository) Add(accountID int64, repoName, sha, message string, committedAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT OR IGNORE INTO commits (account_id, repo_name, sha, message, committed_at) VALUES (?, ?, ?, ?, ?)`,
		accountID, repoName, sha, message, committedAt,
	)
	return err
}

func (r *CommitRepository) GetForAccount(accountID int64, since time.Time) ([]Commit, error) {
	rows, err := r.db.Query(`
		SELECT id, account_id, repo_name, sha, message, committed_at
		FROM commits
		WHERE account_id = ? AND committed_at >= ?
		ORDER BY committed_at DESC
	`, accountID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commits []Commit
	for rows.Next() {
		var c Commit
		if err := rows.Scan(&c.ID, &c.AccountID, &c.RepoName, &c.SHA, &c.Message, &c.CommittedAt); err != nil {
			return nil, err
		}
		commits = append(commits, c)
	}
	return commits, rows.Err()
}

func (r *CommitRepository) GetAllSince(since time.Time) ([]Commit, error) {
	rows, err := r.db.Query(`
		SELECT id, account_id, repo_name, sha, message, committed_at
		FROM commits
		WHERE committed_at >= ?
		ORDER BY committed_at DESC
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commits []Commit
	for rows.Next() {
		var c Commit
		if err := rows.Scan(&c.ID, &c.AccountID, &c.RepoName, &c.SHA, &c.Message, &c.CommittedAt); err != nil {
			return nil, err
		}
		commits = append(commits, c)
	}
	return commits, rows.Err()
}

func (r *CommitRepository) CountByAccount(since time.Time) (map[int64]int, error) {
	rows, err := r.db.Query(`
		SELECT account_id, COUNT(*) as count
		FROM commits
		WHERE committed_at >= ?
		GROUP BY account_id
		ORDER BY count DESC
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[int64]int)
	for rows.Next() {
		var accountID int64
		var count int
		if err := rows.Scan(&accountID, &count); err != nil {
			return nil, err
		}
		counts[accountID] = count
	}
	return counts, rows.Err()
}
