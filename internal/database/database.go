// internal/database/database.go
package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

func (db *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		name TEXT,
		avatar_url TEXT,
		bio TEXT,
		followers INTEGER DEFAULT 0,
		following INTEGER DEFAULT 0,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_fetched DATETIME
	);

	CREATE TABLE IF NOT EXISTS commits (
		id INTEGER PRIMARY KEY,
		account_id INTEGER NOT NULL,
		repo_name TEXT NOT NULL,
		sha TEXT NOT NULL,
		message TEXT,
		committed_at DATETIME,
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (account_id) REFERENCES accounts(id),
		UNIQUE(account_id, sha)
	);

	CREATE TABLE IF NOT EXISTS repos (
		id INTEGER PRIMARY KEY,
		account_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		full_name TEXT NOT NULL,
		description TEXT,
		language TEXT,
		stars INTEGER DEFAULT 0,
		created_at DATETIME,
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (account_id) REFERENCES accounts(id),
		UNIQUE(account_id, full_name)
	);

	CREATE TABLE IF NOT EXISTS stars (
		id INTEGER PRIMARY KEY,
		account_id INTEGER NOT NULL,
		repo_full_name TEXT NOT NULL,
		repo_description TEXT,
		repo_language TEXT,
		repo_stars INTEGER DEFAULT 0,
		starred_at DATETIME,
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (account_id) REFERENCES accounts(id),
		UNIQUE(account_id, repo_full_name)
	);

	CREATE TABLE IF NOT EXISTS digests (
		id INTEGER PRIMARY KEY,
		period_start DATETIME NOT NULL,
		period_end DATETIME NOT NULL,
		content TEXT,
		smart_analysis TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_commits_account ON commits(account_id);
	CREATE INDEX IF NOT EXISTS idx_commits_date ON commits(committed_at);
	CREATE INDEX IF NOT EXISTS idx_repos_account ON repos(account_id);
	CREATE INDEX IF NOT EXISTS idx_repos_created ON repos(created_at);
	CREATE INDEX IF NOT EXISTS idx_stars_account ON stars(account_id);
	CREATE INDEX IF NOT EXISTS idx_stars_date ON stars(starred_at);
	`

	_, err := db.conn.Exec(schema)
	return err
}
