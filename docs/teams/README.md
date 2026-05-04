# Microsoft Teams (consumer) API

The consumer Microsoft Teams API is the undocumented HTTP surface used by `https://teams.live.com/v2/` when signed in with a personal Microsoft account. It is not endorsed by Microsoft and may change without notice.

It accepts JSON-encoded request bodies, returns JSON responses, and uses standard HTTP response codes.

All requests are authenticated. See [Authentication](./authentication.md).

## Base URL

```
https://msgapi.teams.live.com
```

The chat service is also reachable through `https://teams.live.com/api/chatsvc/consumer`, a Front Door proxy that shares cookies with the Teams web app. Browser clients typically use the proxy; non-browser clients should use `msgapi.teams.live.com` directly. The two URLs serve identical paths.

A small set of identity and discovery endpoints lives on the middle tier at `https://teams.live.com/api/mt`. Where used, the full path is shown.

## API version

This reference covers chat-service version `v1`. Prepend `/v1/` to every chat-service path shown below.

## Reference

- [Authentication](./authentication.md) — Skype token
- [Errors](./errors.md) — Status codes, error response format, rate limiting
- [Conversations](./conversations.md) — List and read conversations
- [Messages](./messages.md) — Read and send messages within a conversation
