---
name: beeper
description: Search and manage Beeper Desktop chats with the official CLI across Beeper-supported services such as WhatsApp, Signal, Telegram, Discord, Google Messages, Slack, Twitter/X DMs, Google Chat, LinkedIn, Instagram, Facebook Messenger, Google Voice, and Matrix. Use for finding chats or participants, listing accounts, searching or reading messages, checking contacts, focusing chats, drafting or sending replies, and managing archive or low-priority state.
---

# Beeper

Use the official `beeper-desktop-cli` on PATH.

## Supported Services

Beeper help docs currently list these addable chat networks: WhatsApp, Signal, Telegram, Discord, Google Messages, Slack, Twitter/X DMs, Google Chat, LinkedIn, Instagram, and Facebook Messenger. Some Beeper guides also list Google Voice.

## Rules

- Read/search first; mutate last.
- Never send, edit, archive, create, or upload on behalf of the user without explicit approval.
- Before any send, show the exact final message text.
- Prefer `--format raw` for piping and `--format json` when structure matters.
- Search is literal, not semantic. Use real keywords the user likely typed.
- Prefer direct shell use. Some nested subprocess wrappers can trigger CLI panics unless stdin is redirected from `/dev/null`.

## Quick Checks

```bash
beeper-desktop-cli info retrieve --format json
beeper-desktop-cli accounts list --format raw
```

## Known Quirks

- Some providers may be searchable even if their account ID does not appear in `accounts list`.
- `chats list --format json` emits one JSON object per line, not a single array. Slurp with `jq -s`.
- `messages list` does not support `--limit`; use it as-is, then paginate with `--cursor` + `--direction`.
- Some provider-specific interactive content may degrade to placeholder text and require opening the native app.
- `chats low-priority` is a local-only workaround. It edits Beeper's local `index.db` directly, not the public Desktop API.

## Fast Paths

- Fastest message search: `messages search --account-id <id> ...`
- Even tighter: `messages search --chat-id <chatID> ...`
- Fastest person lookup: `search --query <name>` first, then resolve concrete `chatID`
- For service-specific work, start from `chats search --account-id <id> ...`

## Common Tasks

### Find chats or people

```bash
beeper-desktop-cli search --query alice --format raw
beeper-desktop-cli chats search --query alice --scope participants --type single --limit 5 --format raw
beeper-desktop-cli chats search --query project --scope titles --limit 10 --format raw
beeper-desktop-cli chats search --account-id whatsapp --query alice --scope participants --type single --limit 5 --format raw
```

### Read a chat or inspect metadata

```bash
beeper-desktop-cli chats retrieve --chat-id "<chatID>" --format raw
beeper-desktop-cli messages list --chat-id "<chatID>" --format raw
```

### Search messages

```bash
beeper-desktop-cli messages search --query invoice --limit 10 --format raw
beeper-desktop-cli messages search --account-id discordgo --chat-type single --query friend --limit 10 --format raw
beeper-desktop-cli messages search --chat-id "<chatID>" --query invoice --limit 10 --format raw
beeper-desktop-cli messages search --sender me --query followup --limit 20 --format json
```

### Contacts / friend graph hints

```bash
beeper-desktop-cli accounts list --format raw
beeper-desktop-cli accounts:contacts search --account-id whatsapp --query alice --format raw
```

Use account contact search when the user is asking about the same person across networks, account IDs, or messageability.

### Enumerate chats safely

```bash
beeper-desktop-cli chats list --format json > /tmp/beeper-chats.json
jq -s 'group_by(.accountID) | map({accountID: .[0].accountID, count: length})' /tmp/beeper-chats.json
```

### Local-only low priority workaround

```bash
beeper-desktop-cli chats low-priority --chat-id "<chatID>" --low-priority=true --format raw
beeper-desktop-cli chats low-priority --chat-id "<chatID>" --low-priority=false --format raw
```

Notes:
- this mutates Beeper's local SQLite index, not the public API
- verify with `chats search --inbox low-priority ...`
- override DB path for tests with `BEEPER_DESKTOP_INDEX_DB_PATH=/path/to/index.db`
- use only when the user explicitly wants to change Beeper's native Low Priority state

### Archive only after approval

```bash
beeper-desktop-cli chats archive --chat-id "<chatID>" --archived=true
beeper-desktop-cli chats archive --chat-id "<chatID>" --archived=false
```

### Send only after approval

```bash
beeper-desktop-cli messages send --chat-id "<chatID>" --text "final approved text" --format raw
beeper-desktop-cli messages send --chat-id "<chatID>" --reply-to-message-id "<messageID>" --text "final approved text" --format raw
```

### Focus the app

```bash
beeper-desktop-cli focus --chat-id "<chatID>"
beeper-desktop-cli focus --chat-id "<chatID>" --message-id "<messageID>"
```

## Workflow

1. `search --query <name-or-keyword>` or `chats search --account-id ...`
2. Resolve concrete `chatID` and `accountID`
3. `chats retrieve --chat-id ...`
4. `messages list --chat-id ...`
5. If needed, narrow with `messages search --chat-id ...`
6. Draft reply in-thread
7. Wait for explicit approval
8. Send with `messages send`

## Notes

- The official CLI is faster than the Beeper MCP bridge in direct shell use.
- `messages search` can be extremely fast when scoped with `--account-id` or `--chat-id`.
- If the user asks for Beeper docs, use web/docs lookup; do not rely on MCP-only doc search.
