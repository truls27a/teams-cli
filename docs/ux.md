# UX

`gh`-style CLI. API calls them "conversations"; CLI calls them `chat`.

All commands support `--json` to output machine-readable JSON for scripting.

## `teams chat list`

Shows DMs, group chats, and channels in one table.

```
ID    TYPE      NAME
1     dm        John Doe
2     group     Design Team
3     channel   #general / My Team
```

Short numeric IDs map to real (GUID) IDs internally.

## `teams chat view <id>`

Shows last 20 messages, oldest first, flat (no thread nesting).

```
John Doe    10:32   hey, did you see the PR?
You         10:35   yeah, looking now
John Doe    10:40   sounds good
```

Flags:

- `--limit N` — change count
- `--all` — full history

Skip for now: threads, reactions, attachments, mentions.

## `teams chat send <id> "message"`

Send a message. Quotes required for multi-word.
