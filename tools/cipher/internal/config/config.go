package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	GitHubToken      string
	RepoOwner        string
	RepoName         string
	BaseBranch       string
	BotUsername      string
	BotLabel         string
	DiscordToken     string
	DiscordChannelID string
	AnthropicAPIKey  string
	ClaudeCmd        string
	RepoPath         string
	WorktreeBase     string
	StateFile        string
	LogDir           string
	PromptsDir       string
	UseDocker        bool
	DockerImage      string
	MaxOpenPRs       int
	MaxCIAttempts    int
	WatcherInterval  int
}

func Load() (*Config, error) {
	_ = godotenv.Load("/home/cipher/cipher-bot/.env")

	cfg := &Config{
		GitHubToken:      mustEnv("GITHUB_TOKEN"),
		RepoOwner:        env("REPO_OWNER", "verana-labs"),
		RepoName:         env("REPO_NAME", "verana"),
		BaseBranch:       env("BASE_BRANCH", "main"),
		BotUsername:      env("BOT_USERNAME", "cipher-bot"),
		BotLabel:         env("BOT_LABEL", "cipher"),
		DiscordToken:     mustEnv("DISCORD_TOKEN"),
		DiscordChannelID: mustEnv("DISCORD_CHANNEL_ID"),
		AnthropicAPIKey:  mustEnv("ANTHROPIC_API_KEY"),
		ClaudeCmd:        env("CLAUDE_CMD", "claude"),
		RepoPath:         env("REPO_PATH", "/home/cipher/verana"),
		WorktreeBase:     env("WORKTREE_BASE", "/home/cipher/worktrees"),
		StateFile:        env("STATE_FILE", "/home/cipher/cipher-bot/state.json"),
		LogDir:           env("LOG_DIR", "/home/cipher/logs"),
		PromptsDir:       env("PROMPTS_DIR", "/home/cipher/cipher-bot/prompts"),
		UseDocker:        env("USE_DOCKER", "true") == "true",
		DockerImage:      env("DOCKER_IMAGE", "cipher-claude:latest"),
		MaxOpenPRs:       envInt("MAX_OPEN_PRS", 5),
		MaxCIAttempts:    envInt("MAX_CI_ATTEMPTS", 5),
		WatcherInterval:  envInt("WATCHER_INTERVAL", 300),
	}

	for _, d := range []string{cfg.WorktreeBase, cfg.LogDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	if err := os.MkdirAll(cfg.StateFile[:len(cfg.StateFile)-len("/state.json")], 0o755); err != nil {
		_ = err // best effort
	}
	return cfg, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %q not set", key))
	}
	return v
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
