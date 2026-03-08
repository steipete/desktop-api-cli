package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const beeperIndexDBPathEnv = "BEEPER_DESKTOP_INDEX_DB_PATH"

type lowPriorityResult struct {
	ChatID              string `json:"chatID"`
	LowPriority         bool   `json:"lowPriority"`
	PreviousLowPriority bool   `json:"previousLowPriority"`
	DBPath              string `json:"dbPath"`
}

func (r lowPriorityResult) mustJSON() []byte {
	data, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	return data
}

func resolveBeeperIndexDBPath() string {
	if path := os.Getenv(beeperIndexDBPathEnv); path != "" {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, "Library", "Application Support", "BeeperTexts", "index.db")
}

func openBeeperIndexDB(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("could not determine Beeper index DB path; set %s", beeperIndexDBPathEnv)
	}
	return sql.Open("sqlite3", dbPath+"?_busy_timeout=5000")
}

func setChatLowPriority(ctx context.Context, dbPath, chatID string, enabled bool) (lowPriorityResult, error) {
	db, err := openBeeperIndexDB(dbPath)
	if err != nil {
		return lowPriorityResult{}, err
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return lowPriorityResult{}, err
	}
	defer tx.Rollback()

	var raw string
	err = tx.QueryRowContext(ctx, "SELECT thread FROM threads WHERE threadID = ?", chatID).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return lowPriorityResult{}, fmt.Errorf("chat not found in local Beeper index: %s", chatID)
		}
		return lowPriorityResult{}, err
	}

	previous, updated, err := mutateThreadLowPriority(raw, enabled)
	if err != nil {
		return lowPriorityResult{}, err
	}

	res, err := tx.ExecContext(ctx, "UPDATE threads SET thread = ? WHERE threadID = ?", updated, chatID)
	if err != nil {
		return lowPriorityResult{}, err
	}
	if rows, err := res.RowsAffected(); err != nil {
		return lowPriorityResult{}, err
	} else if rows != 1 {
		return lowPriorityResult{}, fmt.Errorf("expected to update 1 row for chat %s, updated %d", chatID, rows)
	}

	if err := tx.Commit(); err != nil {
		return lowPriorityResult{}, err
	}

	return lowPriorityResult{
		ChatID:              chatID,
		LowPriority:         enabled,
		PreviousLowPriority: previous,
		DBPath:              dbPath,
	}, nil
}

func mutateThreadLowPriority(raw string, enabled bool) (bool, string, error) {
	var thread map[string]any
	if err := json.Unmarshal([]byte(raw), &thread); err != nil {
		return false, "", err
	}

	previous, _ := thread["isLowPriority"].(bool)
	thread["isLowPriority"] = enabled

	extra := ensureJSONObject(thread, "extra")
	tags := ensureJSONObject(extra, "tags")
	if enabled {
		tags["m.lowpriority"] = map[string]any{"order": 0}
	} else {
		delete(tags, "m.lowpriority")
	}

	updated, err := json.Marshal(thread)
	if err != nil {
		return false, "", err
	}
	return previous, string(updated), nil
}

func ensureJSONObject(parent map[string]any, key string) map[string]any {
	if existing, ok := parent[key].(map[string]any); ok {
		return existing
	}
	child := map[string]any{}
	parent[key] = child
	return child
}
