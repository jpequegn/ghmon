# ghmon

Monitor GitHub accounts you follow and get digests of their activity.

## Features

- Import accounts from your GitHub following list
- Track commits, new repositories, and stars given
- Generate activity digests with trending insights
- Optional LLM-powered analysis of focus areas
- Export reports to markdown

## Installation

```bash
go install github.com/julienpequegnot/ghmon@latest
```

Or build from source:

```bash
git clone https://github.com/julienpequegnot/ghmon
cd ghmon
go build -o ghmon .
```

## Quick Start

```bash
# Initialize with your GitHub token
ghmon init

# Import accounts you follow
ghmon sync

# Fetch recent activity
ghmon fetch

# View digest
ghmon digest

# Get AI-powered insights
ghmon digest --smart

# Export to markdown
ghmon export > weekly.md
```

## Commands

| Command | Description |
|---------|-------------|
| `ghmon init` | Initialize config and database |
| `ghmon sync` | Import accounts from GitHub following |
| `ghmon add <user>` | Add a user to monitor |
| `ghmon remove <user>` | Remove a user |
| `ghmon fetch` | Pull recent activity |
| `ghmon accounts` | List monitored accounts |
| `ghmon digest` | Show activity summary |
| `ghmon show <user>` | Show user details |
| `ghmon export` | Generate markdown report |

## Configuration

Config is stored in `~/.ghmon/config.yaml`

```yaml
github:
  token: "ghp_xxxx"

apis:
  llm_provider: "ollama"
  llm_model: "llama3.2"

digest:
  default_days: 7
```

## Development Status

### Phase 1 (MVP) - Complete
- [x] Project setup
- [x] init command
- [x] sync command
- [x] add/remove commands
- [x] accounts command
- [x] fetch command
- [x] digest command (--days flag)
- [x] show command (--days flag)

### Phase 2 (Intelligence)
- [ ] LLM integration
- [ ] Language analysis
- [ ] Trending stars detection
- [ ] Focus area extraction

### Phase 3 (Export & Polish)
- [ ] export command
- [ ] Daemon mode
- [ ] Rate limit improvements
