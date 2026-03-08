package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMutateThreadLowPriority(t *testing.T) {
	raw := `{"id":"!chat:beeper.local","isLowPriority":false,"extra":{"tags":{},"markedUnreadUpdatedAt":123}}`

	previous, updated, err := mutateThreadLowPriority(raw, true)
	require.NoError(t, err)
	assert.False(t, previous)

	var thread map[string]any
	require.NoError(t, json.Unmarshal([]byte(updated), &thread))
	assert.Equal(t, true, thread["isLowPriority"])
	tags := thread["extra"].(map[string]any)["tags"].(map[string]any)
	assert.Equal(t, map[string]any{"order": float64(0)}, tags["m.lowpriority"])

	previous, updated, err = mutateThreadLowPriority(updated, false)
	require.NoError(t, err)
	assert.True(t, previous)
	require.NoError(t, json.Unmarshal([]byte(updated), &thread))
	assert.Equal(t, false, thread["isLowPriority"])
	tags = thread["extra"].(map[string]any)["tags"].(map[string]any)
	_, exists := tags["m.lowpriority"]
	assert.False(t, exists)
}

func TestSetChatLowPriority(t *testing.T) {
	dbPath := createTempIndexDB(t)
	insertThreadRow(t, dbPath, "!chat:beeper.local", `{"id":"!chat:beeper.local","isLowPriority":false,"extra":{"tags":{}}}`)

	res, err := setChatLowPriority(context.Background(), dbPath, "!chat:beeper.local", true)
	require.NoError(t, err)
	assert.False(t, res.PreviousLowPriority)
	assert.True(t, res.LowPriority)

	raw := readThreadRow(t, dbPath, "!chat:beeper.local")
	assert.True(t, decodeThread(t, raw)["isLowPriority"].(bool))

	res, err = setChatLowPriority(context.Background(), dbPath, "!chat:beeper.local", false)
	require.NoError(t, err)
	assert.True(t, res.PreviousLowPriority)
	assert.False(t, res.LowPriority)
	raw = readThreadRow(t, dbPath, "!chat:beeper.local")
	thread := decodeThread(t, raw)
	assert.False(t, thread["isLowPriority"].(bool))
	tags := thread["extra"].(map[string]any)["tags"].(map[string]any)
	_, exists := tags["m.lowpriority"]
	assert.False(t, exists)
}

func TestChatsLowPriorityCommand(t *testing.T) {
	dbPath := createTempIndexDB(t)
	insertThreadRow(t, dbPath, "!chat:beeper.local", `{"id":"!chat:beeper.local","isLowPriority":false,"extra":{"tags":{}}}`)

	repoRoot := filepath.Join("..", "..")

	cmd := exec.Command("go", "run", "./cmd/beeper-desktop-cli", "--format", "raw", "chats", "low-priority", "--chat-id", "!chat:beeper.local", "--low-priority=true")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), beeperIndexDBPathEnv+"="+dbPath)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Contains(t, string(out), `"lowPriority":true`)

	raw := readThreadRow(t, dbPath, "!chat:beeper.local")
	assert.True(t, decodeThread(t, raw)["isLowPriority"].(bool))
}

func createTempIndexDB(t *testing.T) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "index.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE threads (
		threadID VARCHAR NOT NULL PRIMARY KEY,
		accountID VARCHAR,
		thread JSON NOT NULL,
		timestamp INTEGER DEFAULT 0
	)`)
	require.NoError(t, err)

	return dbPath
}

func insertThreadRow(t *testing.T, dbPath, chatID, threadJSON string) {
	t.Helper()

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`INSERT INTO threads(threadID, accountID, thread, timestamp) VALUES(?, ?, ?, 0)`, chatID, "whatsapp", threadJSON)
	require.NoError(t, err)
}

func readThreadRow(t *testing.T, dbPath, chatID string) string {
	t.Helper()

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	var raw string
	require.NoError(t, db.QueryRow(`SELECT thread FROM threads WHERE threadID = ?`, chatID).Scan(&raw))
	return raw
}

func decodeThread(t *testing.T, raw string) map[string]any {
	t.Helper()

	var thread map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &thread))
	return thread
}
