// internal/activity/commits.go
package activity

import (
	"strings"
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

// UserCommitActivity holds commit stats per user
type UserCommitActivity struct {
	AccountID int64
	Username  string
	Count     int
	Repos     []string
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

// GetUserActivity returns commit activity grouped by user with repo details
func (r *CommitRepository) GetUserActivity(since time.Time, limit int) ([]UserCommitActivity, error) {
	rows, err := r.db.Query(`
		SELECT
			c.account_id,
			a.username,
			COUNT(*) as commit_count,
			GROUP_CONCAT(DISTINCT c.repo_name) as repos
		FROM commits c
		JOIN accounts a ON c.account_id = a.id
		WHERE c.committed_at >= ?
		GROUP BY c.account_id
		ORDER BY commit_count DESC
		LIMIT ?
	`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []UserCommitActivity
	for rows.Next() {
		var ua UserCommitActivity
		var repos string
		if err := rows.Scan(&ua.AccountID, &ua.Username, &ua.Count, &repos); err != nil {
			return nil, err
		}
		if repos != "" {
			ua.Repos = strings.Split(repos, ",")
		}
		activities = append(activities, ua)
	}

	return activities, rows.Err()
}
