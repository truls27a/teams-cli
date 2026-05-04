# Authentication

The consumer Teams API uses Skype-token authentication. Every request to the chat service must include an `Authentication` header containing a valid Skype JWT.

```http
Authentication: skypetoken=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

The legacy form `X-Skypetoken: eyJ...` is also accepted.

Requests without a valid token return [`401 Unauthorized`](./errors.md) with `errorCode: 911`.

## Obtaining a token

The Skype token is issued by the Teams "consumer authz" endpoint in exchange for a Microsoft identity-platform access token. Three steps:

1. Acquire a Microsoft access token through MSAL.
2. Exchange that token for a Skype token.
3. Use the Skype token on every chat-service call.

### Acquire a Microsoft access token

| Setting   | Value                                                                    |
| --------- | ------------------------------------------------------------------------ |
| Client ID | `4b3e8f46-56d3-427f-b1e2-d239b2ea6bca`                                   |
| Authority | `https://login.microsoftonline.com/9188040d-6c67-4c5b-b112-36a304b66dad` |
| Scope     | `service::api.fl.spaces.skype.com::MBI_SSL`                              |

The tenant `9188040d-6c67-4c5b-b112-36a304b66dad` is the well-known "MSA consumers" tenant. The returned access token is opaque and only valid as input to the next step. Lifetime is approximately 60 minutes.

### Exchange for a Skype token

```http
POST /api/auth/v1.0/authz/consumer HTTP/1.1
Host: teams.live.com
Authorization: Bearer eyJ0eXAi...
Content-Length: 0
```

```json
{
  "tokens": { "skypeToken": "eyJhbGci...", "expiresIn": 86400 },
  "region": "amer",
  "regionGtms": {
    "chatService": "https://msgapi.teams.live.com",
    "chatServiceAfd": "https://teams.live.com/api/chatsvc/consumer",
    "middleTier": "https://teams.live.com/api/mt",
    "ams": "https://us-api.asm.skype.com",
    "search": "https://msgsearch.skype.com",
    "unifiedPresence": "https://presence.teams.live.com"
  }
}
```

Persist `regionGtms` and route subsequent calls through it. The map varies by region.

## User MRI

Every user is identified by an MRI of the form `8:live:<id>` for personal accounts, or `8:orgid:<guid>` for federated work or school accounts. The signed-in user's MRI is the `skypeid` claim of the Skype token (without the `8:` prefix). It can also be retrieved from the middle tier:

```http
GET /api/mt/beta/users/me/?skypeTeamsInfo=true HTTP/1.1
Host: teams.live.com
Authorization: Bearer eyJ0eXAi...
```

`ME` is accepted everywhere a user MRI is expected and resolves to the caller.

## Request headers

In addition to `Authentication`, chat-service requests should include:

```http
Accept:           application/json
Content-Type:     application/json   # on requests with a body
BehaviorOverride: redirectAs404      # surface region redirects as 404
MS-CV:            <correlation vector>
User-Agent:       <client identifier>
```

`MS-CV` (a [correlation vector](https://github.com/microsoft/CorrelationVector)) is echoed in error responses and written to service logs; including it makes correlation with Microsoft support possible.

## Token lifetime

| Token      | Lifetime | Refresh                                       |
| ---------- | -------- | --------------------------------------------- |
| MSA access | ~60 min  | MSAL silent acquisition with cached refresh.  |
| Skype      | 24 h     | Re-call `POST /api/auth/v1.0/authz/consumer`. |

Refresh proactively at 80% of lifetime. On `401 errorCode 911`, refresh once and retry; on a second `401`, re-authenticate interactively.
