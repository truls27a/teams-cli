# Authentication

The Teams API uses three credentials, each scoped to its service. The chat service accepts a Skype token. The chat-service aggregator (CSA) accepts an Azure AD bearer token whose audience is the aggregator. The middle tier (MT) accepts an Azure AD bearer token whose audience is `api.spaces.skype.com` — the same access token that is exchanged for the Skype token. All three are derived from the same Microsoft identity-platform sign-in and from the same OAuth 2.0 refresh token; only the access-token request scope differs.

```http
Authentication: skypetoken=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

```http
Authorization: Bearer eyJ0eXAi...
```

The legacy form `X-Skypetoken: eyJ...` is also accepted on the chat service.

Requests without a valid token return [`401 Unauthorized`](./errors.md) with `errorCode: 911` (chat service) or a generic `401` (CSA).

## Obtaining tokens

Four steps:

1. Acquire an Azure AD access token using the Microsoft identity platform, with scope `https://api.spaces.skype.com/.default`. Persist this token; in addition to its role in the Skype-token exchange below, it doubles as the bearer credential for the middle tier.
2. Exchange that token for a Skype token. Use the Skype token on every chat-service call.
3. Using the same refresh token, acquire a second Azure AD access token with scope `https://chatsvcagg.teams.microsoft.com/.default`. Use it directly as `Authorization: Bearer` on every CSA call.
4. Refresh each access token before its `expires_in` elapses; rotate the refresh token whenever the token endpoint returns a new one.

### Acquire an Azure AD access token

The Microsoft Teams desktop client identifies as the following first-party application. Re-use the same client identity from non-browser callers; it is registered as a public client and accepts the OAuth 2.0 device-code flow, which is the recommended grant for command-line clients.

| Setting   | Value                                                                                |
| --------- | ------------------------------------------------------------------------------------ |
| Client ID | `1fec8e78-bce4-4aaf-ab1b-5451cc387264`                                               |
| Authority | `https://login.microsoftonline.com/<tenant>` (the user's tenant ID, or `organizations`) |
| Scope     | `https://api.spaces.skype.com/.default offline_access`                               |

`<tenant>` is the user's Azure AD tenant ID. Use the literal value `organizations` to let the Microsoft identity platform pick the correct work or school tenant from the user's sign-in. `offline_access` requests a refresh token alongside the access token.

#### Device-code flow

```http
POST /<tenant>/oauth2/v2.0/devicecode HTTP/1.1
Host: login.microsoftonline.com
Content-Type: application/x-www-form-urlencoded

client_id=1fec8e78-bce4-4aaf-ab1b-5451cc387264
&scope=https%3A%2F%2Fapi.spaces.skype.com%2F.default%20offline_access
```

```json
{
  "user_code": "ABCD1234",
  "device_code": "AAQABCEA...",
  "verification_uri": "https://login.microsoft.com/device",
  "expires_in": 900,
  "interval": 5,
  "message": "To sign in, use a web browser to open the page https://login.microsoft.com/device and enter the code ABCD1234 to authenticate."
}
```

Display `user_code` and `verification_uri` to the user, then poll for tokens at `interval` seconds:

```http
POST /<tenant>/oauth2/v2.0/token HTTP/1.1
Host: login.microsoftonline.com
Content-Type: application/x-www-form-urlencoded

client_id=1fec8e78-bce4-4aaf-ab1b-5451cc387264
&grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code
&device_code=AAQABCEA...
```

While the user has not finished signing in, the server responds with `400 Bad Request` and `error: authorization_pending`. Continue polling. Once the user completes sign-in, the next poll returns `200 OK`:

```json
{
  "token_type": "Bearer",
  "scope": "https://api.spaces.skype.com/Authorization.ReadWrite https://api.spaces.skype.com/.default ...",
  "expires_in": 8326,
  "ext_expires_in": 8326,
  "access_token": "eyJ0eXAi...",
  "refresh_token": "0.AY...."
}
```

Persist the `refresh_token`. The access token's lifetime is recorded in `expires_in` (typically ~2 hours).

### Exchange for a Skype token

```http
POST /api/authsvc/v1.0/authz HTTP/1.1
Host: teams.microsoft.com
Authorization: Bearer eyJ0eXAi...
Content-Length: 0
```

The request body is empty; `Content-Length: 0` is required.

```json
{
  "tokens": {
    "skypeToken": "eyJhbGci...",
    "expiresIn": 8063,
    "tokenType": "SkypeToken"
  },
  "region": "emea",
  "partition": "emea01",
  "regionGtms": {
    "ams": "https://eu-api.asm.skype.com",
    "chatService": "https://emea.ng.msg.teams.microsoft.com",
    "chatServiceAfd": "https://teams.microsoft.com/api/chatsvc/emea",
    "chatServiceAggregator": "https://chatsvcagg.teams.microsoft.com",
    "middleTier": "https://teams.microsoft.com/api/mt/emea",
    "unifiedPresence": "...",
    "search": "..."
  },
  "regionSettings": { /* region-level feature flags */ },
  "licenseDetails": { /* subscription metadata */ },
  "teamsDataBoundary": "eudb",
  "ocdiRedirect": "...",
  "isMultiGeo": false
}
```

| Field                    | Type    | Description                                                                                                  |
| ------------------------ | ------- | ------------------------------------------------------------------------------------------------------------ |
| `tokens.skypeToken`      | string  | The Skype JWT to use as `Authentication` on every chat-service call.                                         |
| `tokens.expiresIn`       | integer | Lifetime in seconds (~2 hours).                                                                              |
| `tokens.tokenType`       | string  | Always `SkypeToken`.                                                                                         |
| `region`                 | string  | Geo affinity for the account (e.g. `emea`, `amer`, `apac`).                                                  |
| `partition`              | string  | Sub-region partition (e.g. `emea01`).                                                                        |
| `regionGtms`             | object  | Service-discovery map. Persist for the session and route subsequent calls through it.                         |
| `regionSettings`         | object  | Region-level feature flags. Opaque.                                                                          |
| `licenseDetails`         | object  | Subscription state for paid features. Opaque.                                                                |
| `teamsDataBoundary`      | string  | Data-residency boundary the account is bound to (e.g. `eudb` for the EU Data Boundary).                      |
| `isMultiGeo`             | boolean | Whether the tenant has multi-geo storage enabled.                                                            |

The `regionGtms` map varies by region; do not hard-code service hostnames in client code beyond the bootstrap call to `/api/authsvc/v1.0/authz`. The aggregator entry (`chatServiceAggregator`) is global and the same value for every account; its primary front, `https://teams.microsoft.com/api/csa/api`, is also stable and may be hard-coded.

## Aggregator bearer token

The CSA aggregator does not consume Skype tokens. It accepts an Azure AD access token whose audience is the aggregator service. Acquire one by re-using the refresh token returned by the device-code flow with a different scope:

```http
POST /<tenant>/oauth2/v2.0/token HTTP/1.1
Host: login.microsoftonline.com
Content-Type: application/x-www-form-urlencoded

client_id=1fec8e78-bce4-4aaf-ab1b-5451cc387264
&grant_type=refresh_token
&refresh_token=0.AY...
&scope=https%3A%2F%2Fchatsvcagg.teams.microsoft.com%2F.default
```

```json
{
  "token_type": "Bearer",
  "scope": "https://chatsvcagg.teams.microsoft.com/.default",
  "expires_in": 4499,
  "ext_expires_in": 4499,
  "access_token": "eyJ0eXAi...",
  "refresh_token": "0.AY...."
}
```

The same refresh token mints both the spaces-audience token (used for the Skype-token exchange) and the aggregator-audience token; multi-resource refresh is the default behaviour for public clients on the Microsoft identity platform. Replace the stored refresh token with the rotated value if one is returned.

Requests issued with a token whose audience is `api.spaces.skype.com` against CSA, or vice-versa, are rejected with `401 Unauthorized`. The middle tier is the exception: it consumes the same `api.spaces.skype.com` access token that the Skype-token exchange uses.

## Use the Skype token

```http
GET /v1/users/ME/conversations HTTP/1.1
Host: emea.ng.msg.teams.microsoft.com
Authentication: skypetoken=eyJhbGci...
BehaviorOverride: redirectAs404
```

The header value is literally `skypetoken=<jwt>`, not `Bearer`. The legacy `X-Skypetoken: <jwt>` form is also accepted.

## Use the aggregator bearer token

```http
GET /v1/teams/users/me?isPrefetch=false&enableMembershipSummary=true HTTP/1.1
Host: teams.microsoft.com
Authorization: Bearer eyJ0eXAi...
```

The aggregator does not require `BehaviorOverride` and does not accept `Authentication: skypetoken=`.

## Use the spaces-audience bearer token

The middle tier accepts the same access token that is exchanged for the Skype token — its audience is `api.spaces.skype.com`. Re-use it directly:

```http
POST /emea/beta/users/fetchShortProfile HTTP/1.1
Host: teams.microsoft.com
Authorization: Bearer eyJ0eXAi...
Content-Type: application/json
```

The middle tier rejects the aggregator-audience bearer with `401 Unauthorized`. Token audiences are not interchangeable across services.

## User MRI

Every Azure AD user is identified by an MRI of the form `8:orgid:<object-id>`, where `<object-id>` is the user's AAD object ID (a GUID). The signed-in user's MRI is the `oid` claim of the access token's `id_token`, prepended with `8:orgid:`. `ME` is accepted everywhere a user MRI is expected and resolves to the caller.

Federated chats with consumer Microsoft accounts surface the consumer participant as `8:live:<id>` rather than as an `orgid` MRI.

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

| Token                   | Lifetime          | Used by                                                                | Refresh                                                                                                                                                                  |
| ----------------------- | ----------------- | ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| AAD access (spaces aud) | ~2 hours          | Skype-token exchange; middle tier (`Authorization: Bearer`).           | Re-call the token endpoint with `grant_type=refresh_token` and `scope=https://api.spaces.skype.com/.default`.                                                            |
| AAD access (CSA aud)    | ~75 minutes       | Chat-service aggregator (`Authorization: Bearer`).                     | Re-call the token endpoint with `grant_type=refresh_token` and `scope=https://chatsvcagg.teams.microsoft.com/.default`. Used directly; no exchange step.                  |
| Skype                   | ~2 hours          | Chat service (`Authentication: skypetoken=`).                          | Re-call `POST /api/authsvc/v1.0/authz` with a current spaces-audience AAD access token.                                                                                  |
| AAD refresh             | ~90 days, sliding | Mints both AAD access tokens above.                                    | Rotated on every redemption against either scope. The response carries a new `refresh_token` that supersedes the previous one. Persist the latest value.                |

To refresh the access token:

```http
POST /<tenant>/oauth2/v2.0/token HTTP/1.1
Host: login.microsoftonline.com
Content-Type: application/x-www-form-urlencoded

client_id=1fec8e78-bce4-4aaf-ab1b-5451cc387264
&grant_type=refresh_token
&refresh_token=0.AY...
&scope=https%3A%2F%2Fapi.spaces.skype.com%2F.default
```

The response shape is identical to the device-code token response. Replace the stored refresh token with the new one before the next refresh.

On chat-service `401 errorCode 911`, refresh the Skype token once and retry. On a second `401`, refresh the spaces-audience AAD access token (and the refresh token if expired) and retry. On CSA `401`, refresh the CSA-audience AAD access token and retry. On middle-tier `401`, refresh the spaces-audience AAD access token and retry. If refresh fails on any path, re-run the device-code flow.

## Conditional Access

Tenants may enforce Conditional Access policies that require additional signals before issuing or renewing tokens — for example, multi-factor authentication, device-compliance attestation, or restrictions on third-party clients. When triggered, the token endpoint responds with `400 Bad Request` and an `error_codes` array containing the specific failure (e.g. `50076` for MFA required, `53000` for device-compliance). The user must complete the required step in a browser session before the device-code flow will succeed; there is no programmatic workaround.
