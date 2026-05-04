# Errors

The consumer Teams API uses conventional HTTP status codes to indicate the success or failure of a request.

| Code                      | Meaning                                                                                                                              |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `200 OK`                  | The request succeeded.                                                                                                               |
| `201 Created`             | The resource was created. The new resource's URL is in the `Location` header.                                                        |
| `204 No Content`          | The request succeeded with no response body.                                                                                         |
| `301` / `302`             | Region redirect. Avoid by sending `BehaviorOverride: redirectAs404` and handling `404` / `errorCode 8003`.                           |
| `400 Bad Request`         | The request body was malformed or contained invalid parameters.                                                                      |
| `401 Unauthorized`        | The Skype token is missing, invalid, or expired.                                                                                     |
| `403 Forbidden`           | The caller is authenticated but not allowed to perform the action.                                                                   |
| `404 Not Found`           | The resource does not exist, is not visible to the caller, or lives in another region. The service does not distinguish these cases. |
| `409 Conflict`            | Most commonly a duplicate `clientmessageid` on send.                                                                                 |
| `412 Precondition Failed` | An `If-Match` ETag did not match the current resource version.                                                                       |
| `429 Too Many Requests`   | Rate limit exceeded. Honor `Retry-After`.                                                                                            |
| `5xx`                     | Server-side error. Retry with backoff.                                                                                               |

## Error response format

Chat-service error responses are JSON objects:

```json
{
  "errorCode": 911,
  "message": "Authentication failed.",
  "standardizedError": {
    "errorCode": 911,
    "errorSubCode": 1,
    "errorDescription": "Authentication failed."
  }
}
```

Branch on `errorCode`. `message` is a human-readable description intended for logs. `standardizedError.errorCode` is always identical to the top-level `errorCode`.

The middle tier (`/api/mt/...`) and authz endpoint (`/api/auth/...`) use a different envelope:

```json
{
  "value": { "code": "AuthFailure", "message": "Token rejected." },
  "status": 401
}
```

Branch on `value.code`.

## Chat-service error codes

| `errorCode` |  HTTP | Description                                                                                                                                                  |
| ----------: | ----: | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
|       `911` | `401` | Skype token missing, expired, or revoked. Refresh via [`POST /api/auth/v1.0/authz/consumer`](./authentication.md#exchange-for-a-skype-token) and retry once. |
|       `912` | `401` | Skype token signed for a different region. Re-issue against the correct region.                                                                              |
|      `1000` | `400` | Generic validation failure. Inspect `message`.                                                                                                               |
|      `1003` | `400` | Body too large.                                                                                                                                              |
|      `1102` | `400` | Malformed conversation MRI in the path.                                                                                                                      |
|      `5000` | `500` | Transient backend error. Retry once.                                                                                                                         |
|      `7000` | `403` | Sender blocked by recipient.                                                                                                                                 |
|      `7100` | `403` | Caller is not a participant, or a conversation policy prohibits the action.                                                                                  |
|      `8002` | `404` | Conversation not found, or not visible to the caller.                                                                                                        |
|      `8003` | `404` | Conversation lives in a different region. Re-discover the service map and retry against the new chat-service URL.                                            |
|      `8400` | `409` | Duplicate `clientmessageid`. Treat as success; the original message id is in the `Location` header.                                                          |
|     `19000` | `429` | Throttled. Honor `Retry-After`.                                                                                                                              |

## Identity-platform error codes

Returned by the Microsoft identity platform token endpoint (`/oauth2/v2.0/token`) and the device-code endpoint (`/oauth2/v2.0/devicecode`). The envelope is `{ "error": "<code>", "error_description": "<text>", "error_codes": [<int>], "timestamp": "<iso>", "trace_id": "<guid>", "correlation_id": "<guid>" }`.

| `error`                  |  HTTP | Description                                                                                                                                              |
| ------------------------ | ----: | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `authorization_pending`  | `400` | Returned by the token endpoint while polling a device-code grant. The user has not yet completed sign-in. Continue polling at `interval` seconds.        |
| `slow_down`              | `400` | Polling too aggressively. Increase the polling interval by 5 seconds.                                                                                    |
| `expired_token`          | `400` | The device code expired before the user signed in. Restart the device-code flow.                                                                         |
| `authorization_declined` | `400` | The user explicitly rejected consent. Surface to the caller; do not retry.                                                                               |
| `invalid_grant`          | `400` | The refresh token has been revoked or has expired. Restart the device-code flow.                                                                         |
| `invalid_client`         | `401` | The `client_id` is wrong.                                                                                                                                |
| `unauthorized_client`    | `400` | The client is not configured for the requested grant type. Should not occur with the Teams Live client and device code, which is verified to be allowed. |
| `invalid_scope`          | `400` | The `scope` value is malformed or unrecognised.                                                                                                          |

## Authz error codes

Returned by `POST /api/auth/v1.0/authz/consumer`.

| `value.code`           |  HTTP | Description                                                                                                                                                        |
| ---------------------- | ----: | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `AuthFailure`          | `401` | The MSA access token is invalid or expired. Refresh via the token endpoint and retry.                                                                              |
| `AuthFailure.Audience` | `401` | The MSA access token's `aud` claim is wrong for this audience. The scope on the access-token request must be exactly `service::api.fl.spaces.skype.com::MBI_SSL`.  |
| `Forbidden`            | `403` | The signed-in account is a work or school account; consumer endpoints are unavailable. `skypeToken.isBusinessTenant` would be `true` if the exchange were allowed. |
| `Throttled`            | `429` | Token-exchange rate exceeded.                                                                                                                                      |
| `InternalError`        | `500` | Transient backend error.                                                                                                                                           |

## Rate limiting

Throttled responses include:

```http
Retry-After: 4
```

Honor `Retry-After`; retrying inside the cooldown extends it.

Approximate send limits:

| Scope              | Typical limit                    |
| ------------------ | -------------------------------- |
| Per conversation   | 10 messages / 10 s, 100 / minute |
| Per caller         | 600 / minute                     |
| Per caller per day | 30 000                           |
