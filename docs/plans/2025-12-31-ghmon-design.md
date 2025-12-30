# ghmon Design Document

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:writing-plans to create implementation plans from this design.

**Goal:** Monitor GitHub accounts you follow, digest their activity (commits, new repos, stars), and surface insights about what influential developers are focusing on.

**Use Cases:**
- Stay current with developers you admire (10-30 influential devs)
- Technology radar - use developer activity as signals for emerging trends

---

## Architecture

Pipeline pattern (similar to blogmon):

```
sync → fetch → analyze → digest → export
```

| Stage | Description |
|-------|-------------|
| **sync** | Import/refresh accounts from GitHub following list |
| **fetch** | Pull recent activity (commits, repos, stars) via GitHub API |
| **analyze** | Aggregate stats, optionally generate LLM summaries |
| **digest** | Display activity summary in terminal |
| **export** | Generate markdown reports |

**Data Storage:** SQLite at `~/.ghmon/ghmon.db`

**Database Schema:**

```sql
-- GitHub accounts being monitored
CREATE TABLE accounts (
    id INTEGER PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    name TEXT,
    avatar_url TEXT,
    bio TEXT,
    followers INTEGER,
    following INTEGER,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_fetched DATETIME
);

-- Commit activity
CREATE TABLE commits (
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

-- New repositories created
CREATE TABLE repos (
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

-- Repos they've starred
CREATE TABLE stars (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL,
    repo_full_name TEXT NOT NULL,
    repo_description TEXT,
    repo_language TEXT,
    repo_stars INTEGER,
    starred_at DATETIME,
    fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES accounts(id),
    UNIQUE(account_id, repo_full_name)
);

-- Cached digests
CREATE TABLE digests (
    id INTEGER PRIMARY KEY,
    period_start DATETIME NOT NULL,
    period_end DATETIME NOT NULL,
    content TEXT,
    smart_analysis TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**Config:** `~/.ghmon/config.yaml`

```yaml
github:
  token: "ghp_xxxx"

apis:
  llm_provider: "ollama"
  llm_model: "llama3.2"

fetch:
  concurrency: 5
  timeout_seconds: 30

digest:
  default_days: 7
```

---

## Commands

| Command | Description |
|---------|-------------|
| `ghmon init` | Initialize config and database, prompt for GitHub token |
| `ghmon sync` | Import/refresh accounts from GitHub following list |
| `ghmon add <username>` | Manually add a GitHub user to monitor |
| `ghmon remove <username>` | Remove a user from monitoring |
| `ghmon fetch` | Pull recent activity for all monitored accounts |
| `ghmon accounts` | List monitored accounts with basic stats |
| `ghmon digest` | Show activity summary (default: 7 days) |
| `ghmon show <username>` | Show detailed activity for one user |
| `ghmon export` | Generate markdown report |

**Key Flags:**
- `--days N` - Override time window (default: 7)
- `--smart` - Enable LLM analysis for richer summaries
- `--limit N` - Limit results shown

**Example Workflow:**
```bash
ghmon init                    # Setup with GitHub token
ghmon sync                    # Import who you follow
ghmon fetch                   # Pull their activity
ghmon digest                  # See what's happening
ghmon digest --smart          # Get LLM insights
ghmon export > weekly.md      # Save report
```

---

## GitHub API Integration

**Authentication:** Personal Access Token (PAT) stored in config.

**Required scopes:**
- `read:user` - Read your following list
- No additional scopes needed for public activity

**API Endpoints:**

| Data | Endpoint | Notes |
|------|----------|-------|
| Following list | `GET /user/following` | Your authenticated user's following |
| User events | `GET /users/{user}/events/public` | Last 90 days, max 300 events |
| User repos | `GET /users/{user}/repos` | For new repo detection |
| Starred repos | `GET /users/{user}/starred` | What they've starred |

**Event Types Parsed:**
- `PushEvent` → commit counts and details
- `CreateEvent` (ref_type: repository) → new repos
- `WatchEvent` → stars given

**Rate Limiting:**
- Authenticated: 5000 requests/hour
- Cache data in SQLite to minimize API calls
- Show progress during fetch

---

## Digest Output

**Standard digest (`ghmon digest`):**

```
GITHUB DIGEST (Dec 24 - Dec 31, 2024)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📊 Summary: 28 accounts · 142 commits · 8 new repos · 45 stars

🔥 Most Active
  torvalds         47 commits across 2 repos
  antirez          23 commits across 1 repo
  mitchellh        18 commits across 3 repos

🆕 New Repositories
  fatih/color-v2         Go library for terminal colors
  jessfraz/bpfd          eBPF-based security daemon
  tj/n-rewrite           Node version manager rewrite

⭐ Trending Stars (most starred by your follows)
  ggerganov/llama.cpp    ★ by mass, karpathy, lexfridman
  astral-sh/ruff         ★ by gvanrossum, ambv
  ollama/ollama          ★ by antirez, mitchellh

🏷️ Languages This Week
  Rust (34%) · Go (28%) · Python (22%) · C (16%)
```

**Smart digest (`ghmon digest --smart`):**

Adds LLM-generated insights:

```
💡 Focus Areas (AI-generated)
  • Systems programming surge: torvalds and antirez both diving into
    memory-safe patterns, possibly influenced by Linux Rust adoption
  • AI tooling: 5 follows starred llama.cpp - local LLM inference gaining steam
  • Go ecosystem: fatih and mitchellh active on developer tooling
```

---

## Implementation Phases

### Phase 1 - MVP
- Project setup (Go, Cobra, SQLite)
- `init` command (config, database, token setup)
- `sync` command (import following list)
- `add` / `remove` commands
- `accounts` command (list monitored users)
- `fetch` command (pull events from GitHub API)
- `digest` command (basic aggregation)
- `show <user>` command

### Phase 2 - Intelligence
- LLM integration (Ollama) for `--smart` summaries
- Language breakdown analysis
- "Trending stars" detection (repos starred by multiple follows)
- Focus area extraction

### Phase 3 - Export & Polish
- `export` command (markdown generation)
- Daemon mode (scheduled fetching)
- Rate limit handling improvements
- Configurable digest sections

---

## Tech Stack

- **Language:** Go
- **CLI Framework:** Cobra
- **Database:** SQLite
- **API:** GitHub REST API
- **LLM:** Ollama (optional)
- **Styling:** Lipgloss (terminal output)

Same stack as blogmon for consistency.
