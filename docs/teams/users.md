# Users

A User is a profile record returned by the middle tier (MT). Profile lookups resolve a member MRI or AAD object ID to a display name and contact details — used to fill in names that the chat-service aggregator omits from `chats[].members[].friendlyName`.

## The User object

| Field               | Type    | Description                                                                                                  |
| ------------------- | ------- | ------------------------------------------------------------------------------------------------------------ |
| `mri`               | string  | Member MRI. `8:orgid:<guid>` for AAD users; bots and apps appear as `28:<id>`.                               |
| `objectId`          | string  | AAD object ID, equal to the GUID portion of `mri` for `8:orgid:` users.                                      |
| `displayName`       | string  | Full display name as configured in the user's tenant.                                                        |
| `givenName`         | string  | First name.                                                                                                  |
| `surname`           | string  | Last name.                                                                                                   |
| `email`             | string  | Primary email address.                                                                                       |
| `userPrincipalName` | string  | AAD UPN.                                                                                                     |
| `userType`          | string  | Tenant relationship. Known values: `Member`, `Guest`.                                                        |
| `tenantName`        | string  | Display name of the user's home tenant.                                                                      |
| `isShortProfile`    | boolean | `true` when the record is the abbreviated form returned by `fetchShortProfile`. Always `true` on this route. |
| `type`              | string  | Profile classification. Treat as opaque.                                                                     |

---

## Fetch short profiles

```
POST /{region}/beta/users/fetchShortProfile
```

Resolves a batch of MRIs or AAD object IDs to user profiles. Used to enrich chat rosters when [`Chat.members[].friendlyName`](./chats.md#chatmember) is absent.

### Authentication

Bearer-authenticated. Use the same Azure AD access token whose audience is `api.spaces.skype.com` that is exchanged for the Skype token. See [Authentication](./authentication.md). The middle tier does not consume Skype tokens.

### Path parameters

| Name     | Type   | Description                                                                                                          |
| -------- | ------ | -------------------------------------------------------------------------------------------------------------------- |
| `region` | string | The user's geo affinity (e.g. `emea`, `amer`, `apac`). The same value as `region` in the [authz](./authentication.md#exchange-for-a-skype-token) response. |

### Request body

A JSON array of identifiers. Each element is one of:

- `8:orgid:<guid>` — the standard MRI form.
- `<guid>` — a bare AAD object ID. Equivalent to `8:orgid:<guid>`.
- `<upn>` — an AAD User Principal Name. Set `?isMailAddress=true` if the value is an email/UPN.

```json
["8:orgid:00000000-0000-0000-0000-000000000000", "8:orgid:11111111-1111-1111-1111-111111111111"]
```

`8:live:<id>` MRIs (consumer Microsoft accounts seen in `19:uni01_*` federated chats) are rejected as `InvalidUserId` when sent alone, and silently dropped when batched with valid identifiers. There is no MT fallback for consumer-account peers.

### Query parameters

All optional. The endpoint returns the same shape regardless of these flags; they are documented here for parity with the official Teams web client request.

| Name                   | Type    | Default |
| ---------------------- | ------- | ------- |
| `isMailAddress`        | boolean | `false` |
| `enableGuest`          | boolean | `true`  |
| `includeIBBarredUsers` | boolean | `false` |
| `skypeTeamsInfo`       | boolean | `true`  |

### Returns

```json
{
  "type": "...",
  "value": [ /* User objects */ ]
}
```

`value` contains one [User](#the-user-object) per resolvable identifier. Identifiers that cannot be resolved are silently omitted; map the response back to the request by `mri` or `objectId`.

If *every* identifier in the request is invalid (e.g. only `8:live:` MRIs), the response is `400 Bad Request`:

```json
{ "errorCode": "InvalidUserId", "message": "UserId should be Skype Mri or ADObjectId or UPN." }
```

### Example request

```http
POST /emea/beta/users/fetchShortProfile?isMailAddress=false&enableGuest=true&includeIBBarredUsers=false&skypeTeamsInfo=true HTTP/1.1
Host: teams.microsoft.com
Authorization: Bearer eyJ0eXAi...
Content-Type: application/json

["8:orgid:00000000-0000-0000-0000-000000000000"]
```

### Example response

```json
{
  "type": "...",
  "value": [
    {
      "mri": "8:orgid:00000000-0000-0000-0000-000000000000",
      "objectId": "00000000-0000-0000-0000-000000000000",
      "displayName": "Alice Example",
      "givenName": "Alice",
      "surname": "Example",
      "email": "alice@example.com",
      "userPrincipalName": "alice@example.com",
      "userType": "Member",
      "tenantName": "Example Corp",
      "isShortProfile": true,
      "type": "..."
    }
  ]
}
```
