# Authentication

The consumer Teams API uses Skype-token authentication. Every request to the chat service must include an `Authentication` header containing a valid Skype JWT.

```http
Authentication: skypetoken=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

The legacy form `X-Skypetoken: eyJ...` is also accepted.

Requests without a valid token return [`401 Unauthorized`](./errors.md) with `errorCode: 911`.

## Obtaining a token

The Skype token is issued by the Teams "consumer authz" endpoint in exchange for a Microsoft identity-platform access token. Three steps:

1. Acquire a Microsoft access token using the Microsoft identity platform.
2. Exchange that token for a Skype token.
3. Use the Skype token on every chat-service call.

### Acquire a Microsoft access token

The Teams Live web client identifies as the following first-party application. Re-use the same client identity from non-browser callers; it accepts the OAuth 2.0 device-code flow, which is the recommended grant for command-line clients.

| Setting   | Value                                                                    |
| --------- | ------------------------------------------------------------------------ |
| Client ID | `4b3e8f46-56d3-427f-b1e2-d239b2ea6bca`                                   |
| Authority | `https://login.microsoftonline.com/9188040d-6c67-4c5b-b112-36a304b66dad` |
| Scope     | `service::api.fl.spaces.skype.com::MBI_SSL offline_access`               |

The tenant `9188040d-6c67-4c5b-b112-36a304b66dad` is the well-known "MSA consumers" tenant. Both segments of `service::api.fl.spaces.skype.com::MBI_SSL` are part of a single legacy resource scope; pass them verbatim. `offline_access` requests a refresh token alongside the access token.

#### Device-code flow

```http
POST /9188040d-6c67-4c5b-b112-36a304b66dad/oauth2/v2.0/devicecode HTTP/1.1
Host: login.microsoftonline.com
Content-Type: application/x-www-form-urlencoded

client_id=4b3e8f46-56d3-427f-b1e2-d239b2ea6bca
&scope=service%3A%3Aapi.fl.spaces.skype.com%3A%3AMBI_SSL%20offline_access
```

```json
{
  "user_code": "RR5YAE2T",
  "device_code": "AAQABCEA...",
  "verification_uri": "https://www.microsoft.com/link",
  "expires_in": 900,
  "interval": 5,
  "message": "To sign in, use a web browser to open the page https://www.microsoft.com/link and enter the code RR5YAE2T to authenticate."
}
```

Display `user_code` and `verification_uri` to the user, then poll for tokens at `interval` seconds:

```http
POST /9188040d-6c67-4c5b-b112-36a304b66dad/oauth2/v2.0/token HTTP/1.1
Host: login.microsoftonline.com
Content-Type: application/x-www-form-urlencoded

client_id=4b3e8f46-56d3-427f-b1e2-d239b2ea6bca
&grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code
&device_code=AAQABCEA...
```

While the user has not finished signing in, the server responds with `400 Bad Request` and `error: authorization_pending`. Continue polling. Once the user completes sign-in, the next poll returns `200 OK`:

```json
{
  "token_type": "Bearer",
  "scope": "service::api.fl.spaces.skype.com::MBI_SSL",
  "expires_in": 86399,
  "ext_expires_in": 86399,
  "access_token": "EwA...",
  "refresh_token": "M.C5..."
}
```

Persist the `refresh_token`. The access token's lifetime is recorded in `expires_in` (~24 h).

### Exchange for a Skype token

```http
POST /api/auth/v1.0/authz/consumer HTTP/1.1
Host: teams.live.com
Authorization: Bearer EwA...
Content-Length: 0
```

The request body is empty; `Content-Length: 0` is required.

```json
{
  "skypeToken": {
    "skypetoken": "eyJhbGci...",
    "expiresIn": 86372,
    "skypeid": "live:exampleuser",
    "signinname": "user@example.com",
    "isBusinessTenant": false
  },
  "regionGtms": {
    "ams": "https://us-api.asm.skype.com",
    "chatService": "https://msgapi.teams.live.com",
    "chatServiceAfd": "https://teams.live.com/api/chatsvc/consumer",
    "middleTier": "https://teams.live.com/api/mt",
    "search": "https://msgsearch.skype.com",
    "unifiedPresence": "https://presence.teams.live.com"
  },
  "regionSettings": {
    /* region-specific feature flags */
  },
  "consumerLicenseDetails": {
    /* subscription metadata */
  }
}
```

| Field                         | Type    | Description                                                                                                  |
| ----------------------------- | ------- | ------------------------------------------------------------------------------------------------------------ |
| `skypeToken.skypetoken`       | string  | The Skype JWT to use as `Authentication` on every chat-service call. Note the lowercase `t` in `skypetoken`. |
| `skypeToken.expiresIn`        | integer | Lifetime in seconds (~24 h).                                                                                 |
| `skypeToken.skypeid`          | string  | The caller's user identifier without the `8:` prefix (e.g. `live:exampleuser`). Prepend `8:` to form an MRI. |
| `skypeToken.signinname`       | string  | The signed-in primary email.                                                                                 |
| `skypeToken.isBusinessTenant` | boolean | `false` for personal accounts.                                                                               |
| `regionGtms`                  | object  | Service-discovery map. Persist for the session and route subsequent calls through it.                        |
| `regionSettings`              | object  | Region-level feature flags. Opaque.                                                                          |
| `consumerLicenseDetails`      | object  | Subscription state for paid features. Opaque.                                                                |

The `regionGtms` map varies by region; do not hard-code service hostnames in client code beyond the bootstrap call to `/api/auth/v1.0/authz/consumer`.

## Use the Skype token

```http
GET /v1/users/ME/conversations HTTP/1.1
Host: msgapi.teams.live.com
Authentication: skypetoken=eyJhbGci...
BehaviorOverride: redirectAs404
```

The header value is literally `skypetoken=<jwt>`, not `Bearer`. The legacy `X-Skypetoken: <jwt>` form is also accepted.

## User MRI

Every user is identified by an MRI of the form `8:live:<id>` for personal accounts, or `8:orgid:<guid>` for federated work or school accounts. The signed-in user's MRI is the `skypeToken.skypeid` field of the authz response, prefixed with `8:`. `ME` is accepted everywhere a user MRI is expected and resolves to the caller.

## Request headers

In addition to `Authentication`, chat-service requests should include:

```http
Accept:           application/json
Content-Type:     application/json   # on requests with a body
BehaviorOverride: redirectAs404      # surface region redirects as 404
MS-CV:            <correlation vector>
User-Agent:       <client identifier>
```

`MS-CV` is a [correlation vector](https://github.com/microsoft/CorrelationVector). It is echoed in error responses and written to service logs, which makes correlation with Microsoft support possible.

## Token lifetimes and refresh

| Token       | Lifetime          | Refresh                                                                                                                                                                  |
| ----------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| MSA access  | ~24 h             | Re-call the token endpoint with `grant_type=refresh_token`.                                                                                                              |
| Skype       | ~24 h             | Re-call `POST /api/auth/v1.0/authz/consumer` with a current MSA access token.                                                                                            |
| MSA refresh | ~90 days, sliding | Rotated on every redemption. The response to a `grant_type=refresh_token` call carries a new `refresh_token` that supersedes the previous one. Persist the latest value. |

To refresh the access token:

```http
POST /9188040d-6c67-4c5b-b112-36a304b66dad/oauth2/v2.0/token HTTP/1.1
Host: login.microsoftonline.com
Content-Type: application/x-www-form-urlencoded

client_id=4b3e8f46-56d3-427f-b1e2-d239b2ea6bca
&grant_type=refresh_token
&refresh_token=M.C5...
&scope=service%3A%3Aapi.fl.spaces.skype.com%3A%3AMBI_SSL
```

The response shape is identical to the device-code token response. Replace the stored refresh token with the new one before the next refresh.

On `401 errorCode 911`, refresh the Skype token once and retry. On a second `401`, refresh the MSA access token (and the refresh token if expired) and retry. If both fail, re-run the device-code flow.
