# Microsoft Teams API

The Microsoft Teams API is the undocumented HTTP surface used by `https://teams.microsoft.com/v2/` when signed in with a work or school (Azure AD) account. It is not endorsed by Microsoft and may change without notice.

It accepts JSON-encoded request bodies, returns JSON responses, and uses standard HTTP response codes.

All requests are authenticated. See [Authentication](./authentication.md).

## Base URL

The chat-service host is region-stamped and is returned in the authz response under `regionGtms.chatService`. For the EU region:

```
https://emea.ng.msg.teams.microsoft.com
```

The chat service is also reachable through `https://teams.microsoft.com/api/chatsvc/{region}`, a Front Door proxy that shares cookies with the Teams web app. Browser clients typically use the proxy; non-browser clients should use the regional `*.ng.msg.teams.microsoft.com` host directly. The two URLs serve identical paths.

A small set of identity and discovery endpoints lives on the middle tier at `https://teams.microsoft.com/api/mt/{region}`. Where used, the full path is shown.

Do not hard-code the region. Discover the full service map at sign-in via [Authentication](./authentication.md#exchange-for-a-skype-token) and route subsequent calls through it.

## API version

This reference covers chat-service version `v1`. Prepend `/v1/` to every chat-service path shown below.

## Reference

- [Authentication](./authentication.md) — Skype token via OAuth 2.0 device code
- [Errors](./errors.md) — Status codes, error response format, rate limiting
- [Conversations](./conversations.md) — List and read conversations
- [Messages](./messages.md) — Read and send messages within a conversation
