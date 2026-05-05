# Conversations

A Conversation represents a thread of messages between two or more participants. The same data model covers one-to-one chats, federated one-to-one chats with external organisations or consumer Microsoft accounts, group chats, and channel threads. They are distinguished by the prefix of the conversation MRI and by `threadProperties.productThreadType`.

## The Conversation object

### Common attributes

| Field                       | Type                                                | Description                                                                                                                                                  |
| --------------------------- | --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `id`                        | string                                              | The conversation MRI. See [Identifying conversations](#identifying-conversations).                                                                           |
| `type`                      | string                                              | Always `Conversation`.                                                                                                                                       |
| `targetLink`                | string                                              | Self-link to the conversation resource.                                                                                                                      |
| `messages`                  | string                                              | Absolute URL of the conversation's message collection.                                                                                                       |
| `version`                   | integer                                             | Server-side version stamp (Unix milliseconds), bumped on every server-side mutation.                                                                         |
| `lastUpdatedMessageId`      | integer                                             | `id` of the most recently delivered or mutated message. Returned as a bare integer, not a string.                                                            |
| `lastUpdatedMessageVersion` | integer                                             | `version` of that message.                                                                                                                                   |
| `lastRcMetadataVersion`     | integer                                             | Version stamp on the conversation's roster / control metadata.                                                                                               |
| `lastMessage`               | [Message](./messages.md#the-message-object) \| null | The most recent message in the conversation, or `null` for empty threads. The embedded message also carries a `sequenceId` and a `clientmessageid` echo.     |
| `properties`                | [ConversationProperties](#conversationproperties)   | Per-conversation metadata.                                                                                                                                   |
| `threadProperties`          | [ThreadProperties](#threadproperties) \| null       | Per-thread metadata. Present for every `19:` MRI, including federated one-to-one chats.                                                                      |
| `memberProperties`          | [MemberProperties](#memberproperties)               | Per-caller membership state for this thread.                                                                                                                 |

### Identifying conversations

| Prefix     | Conversation kind                                                                                                                                                            | Example                       |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------- |
| `8:orgid:` | Native one-to-one with another user in the caller's tenant. `id` is the other participant's user MRI; `threadProperties` is `null`.                                         | `8:orgid:<guid>`              |
| `8:live:`  | Native one-to-one with a personal Microsoft account on the consumer chat fabric. Rare; most cross-fabric chats are realised as `19:` threads.                                | `8:live:<id>`                 |
| `19:`      | Thread. Used for group chats, channel threads, and federated one-to-one chats. Distinguish by `threadProperties.productThreadType` (`OneToOneChat` for federated 1:1, `Chat` for group, `TopicThread` for channel posts). | `19:abcd0123…@thread.v2`      |
| `19:meeting_` | Calendar meeting chat thread. MRI suffix is always `@thread.v2`.                                                                                                       | `19:meeting_<base64>@thread.v2` |
| `48:bot:`  | One-to-one with a bot.                                                                                                                                                       | `48:bot:...`                  |
| `48:`      | System feed. Known values: `48:notifications`, `48:mentions`, `48:notes`, `48:calllogs`, `48:drafts`.                                                                        | `48:notifications`            |

The `19:` thread MRI suffix encodes the thread fabric: `@thread.v2` for cross-tenant or modern threads, `@thread.skype` for legacy group chats, and `@thread.tacv2` for Teams channel posts.

Federated one-to-one chats between a work/school account and a personal Microsoft account use the `19:uni01_<hash>@thread.v2` shape. These appear as `productThreadType: OneToOneChat` in the conversation list. The personal-account participant's MRI is in the form `8:live:<username>` or `8:live:.cid.<hex>`. `threadProperties.creator` is `null` for these threads and `lastMessage.from` only reflects the last sender.

### ConversationProperties

A map of string keys to values. Most values are strings; some are integers (e.g. `draftVersion`). Common entries:

| Key                          | Type    | Description                                                                                                                                                          |
| ---------------------------- | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `consumptionhorizon`         | string  | Read marker, formatted as `<lastReadMessageId>;<lastReadArrivalMs>;<lastSentByMeArrivalMs>`. Updated via [Mark a conversation as read](#mark-a-conversation-as-read). |
| `consumptionHorizonBookmark` | string  | Secondary read marker in the same format.                                                                                                                            |
| `lastimreceivedtime`         | string  | ISO 8601 timestamp of the most recent incoming message.                                                                                                              |
| `isemptyconversation`        | string  | `"True"` or `"False"`.                                                                                                                                               |
| `addedBy`                    | string  | MRI of the user who added the caller to this conversation.                                                                                                           |
| `addedByTenantId`            | string  | Tenant ID of `addedBy`.                                                                                                                                              |
| `draftVersion`               | integer | Version stamp of the caller's unsent draft, if one exists.                                                                                                           |
| `meetingInfo`                | string  | JSON-encoded object with meeting metadata (e.g. `rsvpStatus`). Present on meeting threads.                                                                          |
| `alerts`                     | string  | `"true"` or `"false"`.                                                                                                                                               |
| `favorite`                   | string  | `"true"` or `"false"`.                                                                                                                                               |
| `hasImpersonation`           | string  | `"True"` or `"False"`.                                                                                                                                               |
| `pinnedindex`                | string  | Zero-based pin position. Present only when the conversation is pinned.                                                                                               |
| `mute`                       | string  | JSON-encoded `{ "isMuted": boolean, "until": integer }`. Present only when notifications are muted.                                                                  |

### ThreadProperties

| Key                   | Type    | Description                                                                                                                            |
| --------------------- | ------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `threadType`          | string  | Internal thread classification (e.g. `chat`, `space`, `streamofnotifications`).                                                       |
| `productThreadType`   | string  | Surface classification. `OneToOneChat` for a federated 1:1; `Chat` for a group chat; `TopicThread` for a channel thread; `StreamOfNotifications` for system notification feeds. |
| `topic`               | string  | Group title. Empty for one-to-one threads.                                                                                             |
| `createdat`           | string  | Unix-millisecond creation timestamp, as a string.                                                                                      |
| `creator`             | string  | MRI of the user who created the thread. `null` for federated one-to-one threads (`19:uni01_*`).                                        |
| `memberCount`         | string  | Number of participants, as a stringified integer. `null` for federated one-to-one threads.                                             |
| `originalThreadId`    | string  | Predecessor thread MRI when the thread was migrated; equal to `id` otherwise.                                                          |
| `lastSequenceId`      | string  | Sequence number of the last delivered message, as a string. See [`Message.sequenceId`](./messages.md#the-message-object).              |
| `version`             | string  | Thread-metadata version stamp (Unix milliseconds), as a string.                                                                        |
| `rosterVersion`       | integer | Membership-list version stamp. Returned as a bare integer.                                                                             |
| `lastjoinat`          | string  | Unix-millisecond timestamp at which the caller most recently joined, as a string.                                                      |
| `isStickyThread`      | string  | `"true"` or `"false"`.                                                                                                                 |
| `isCreator`           | boolean | `true` when the caller created the thread.                                                                                             |
| `gapDetectionEnabled` | string  | `"True"` or `"False"`.                                                                                                                 |
| `tenantid`            | string  | Tenant ID of the thread's home tenant.                                                                                                 |
| `hidden`              | string  | `"true"` or `"false"`.                                                                                                                 |
| `picture`             | string  | URL of the thread's avatar image.                                                                                                      |
| `chatFilesIndexId`    | string  | Identifier used to look up files attached anywhere in the thread.                                                                      |

### MemberProperties

| Key                    | Type    | Description                                                                                                          |
| ---------------------- | ------- | -------------------------------------------------------------------------------------------------------------------- |
| `role`                 | string  | Caller's role (e.g. `User`, `Admin`).                                                                                |
| `isReader`             | boolean | `true` when the caller has read-only access.                                                                         |
| `isIdentityMasked`     | boolean | `true` when the caller's identity is masked from other participants.                                                 |
| `memberExpirationTime` | integer | Unix-millisecond timestamp when the caller's membership expires. `0` if non-expiring.                                |
| `interest`             | string  | Caller's notification interest level (e.g. `Interested`).                                                           |
| `relationshipState`    | object  | Federation relationship with the other party. Contains `inQuarantine` (boolean) and `hasImpersonation` (string).    |

---

## List conversations

```
GET /v1/users/ME/conversations
```

Returns the caller's recent conversations, most recent first.

### Query parameters

| Name         | Type    | Description                                                                                                                                    |
| ------------ | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `pageSize`   | integer | Page size. Default `30`, maximum `100`.                                                                                                        |
| `view`       | string  | Field-set selector. The only stable value is `msnp24Equivalent`, which returns the legacy Skype shape used by the web client.                  |
| `targetType` | string  | Comma-separated subset of `Passport,Skype,Lync,Thread,Bot,ShortCircuit,PSTN,Agent`. Filters the result to conversations of these MRI families. |
| `startTime`  | integer | Unix-millisecond lower bound on `lastMessage.composetime`. Useful for backfill.                                                                |
| `syncState`  | string  | Continuation cursor returned as `_metadata.syncState` on the previous page.                                                                    |

### Returns

An object with a `conversations` array and a `_metadata` object containing the pagination cursor.

```json
{
  "conversations": [ /* Conversation objects */ ],
  "_metadata": {
    "totalCount": 30,
    "syncState": "https://emea.ng.msg.teams.microsoft.com/v1/users/ME/conversations?cursor=..."
  }
}
```

To page backward, re-issue the request as `GET <syncState>` unmodified until the response carries no `syncState` or returns an empty `conversations` array.

### Example request

```http
GET /v1/users/ME/conversations?pageSize=30&view=msnp24Equivalent HTTP/1.1
Host: emea.ng.msg.teams.microsoft.com
Authentication: skypetoken=eyJhbGci...
BehaviorOverride: redirectAs404
```

### Example response

```json
{
  "conversations": [
    {
      "id": "19:abcd0123…@thread.v2",
      "type": "Conversation",
      "targetLink": "https://emea.ng.msg.teams.microsoft.com/v1/users/ME/conversations/19:abcd0123…@thread.v2",
      "messages": "https://emea.ng.msg.teams.microsoft.com/v1/users/ME/conversations/19:abcd0123…@thread.v2/messages",
      "version": 1715842456893,
      "lastUpdatedMessageId": "1715842456893",
      "lastUpdatedMessageVersion": 1715842456893,
      "lastRcMetadataVersion": 1714900000000,
      "lastMessage": {
        "sequenceId": 187,
        "id": "1715842456893",
        "type": "Message",
        "messagetype": "RichText/Html",
        "contenttype": "Text",
        "from": "https://emea.ng.msg.teams.microsoft.com/v1/users/ME/contacts/8:orgid:00000000-0000-0000-0000-000000000000",
        "imdisplayname": "Other Person",
        "clientmessageid": "8472635198472635",
        "content": "<p>ok sounds good</p>",
        "composetime": "2026-05-04T07:04:11.213Z",
        "originalarrivaltime": "2026-05-04T07:04:11.213Z",
        "version": "1715842456893",
        "conversationid": "19:abcd0123…@thread.v2",
        "conversationLink": "https://emea.ng.msg.teams.microsoft.com/v1/users/ME/conversations/19:abcd0123…@thread.v2"
      },
      "properties": {
        "consumptionhorizon": "1715842456893;1715842456893;1715842401007",
        "lastimreceivedtime": "1715842456893",
        "isemptyconversation": "false"
      },
      "threadProperties": {
        "threadType": "chat",
        "productThreadType": "OneToOneChat",
        "createdat": "1714000000000",
        "creator": "8:orgid:00000000-0000-0000-0000-000000000000",
        "version": 1715842456893,
        "rosterVersion": 1714000000000,
        "lastSequenceId": 187,
        "originalThreadId": "19:abcd0123…@thread.v2",
        "isStickyThread": false,
        "isCreator": true,
        "gapDetectionEnabled": true,
        "lastjoinat": 1714000000000
      },
      "memberProperties": {
        "role": "User",
        "isReader": false,
        "memberExpirationTime": 0,
        "relationshipState": "Allowed"
      }
    }
  ],
  "_metadata": {
    "totalCount": 1,
    "syncState": "https://emea.ng.msg.teams.microsoft.com/v1/users/ME/conversations?cursor=...&pageSize=30"
  }
}
```

---

## Retrieve a conversation

```
GET /v1/users/ME/conversations/:id
```

Returns a single Conversation, in the same shape as a list element.

### Path parameters

| Name | Type   | Description                              |
| ---- | ------ | ---------------------------------------- |
| `id` | string | The URL-encoded MRI of the conversation. |

### Query parameters

| Name   | Type   | Description                             |
| ------ | ------ | --------------------------------------- |
| `view` | string | Field-set selector; `msnp24Equivalent`. |

### Returns

A [Conversation](#the-conversation-object) object.

### Example request

```http
GET /v1/users/ME/conversations/19%3Aabcd0123%E2%80%A6%40thread.v2?view=msnp24Equivalent HTTP/1.1
Host: emea.ng.msg.teams.microsoft.com
Authentication: skypetoken=eyJhbGci...
```

---

## Mark a conversation as read

```
PUT /v1/users/ME/conversations/:id/properties?name=consumptionhorizon
```

Updates the conversation's read marker. The marker is the `consumptionhorizon` entry of [ConversationProperties](#conversationproperties); there is no dedicated endpoint.

### Path parameters

| Name | Type   | Description                              |
| ---- | ------ | ---------------------------------------- |
| `id` | string | The URL-encoded MRI of the conversation. |

### Request body

| Name                 | Type   | Description                                                                                                                                                                                                                          |
| -------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `consumptionhorizon` | string | `<lastReadMessageId>;<lastReadArrivalMs>;<lastSentByMeArrivalMs>`. The third value should carry forward the previous one if the caller has not just sent a message. Backwards horizons (lower than the current value) are rejected. |

### Returns

`204 No Content` on success.

### Example request

```http
PUT /v1/users/ME/conversations/19%3Aabcd0123%E2%80%A6%40thread.v2/properties?name=consumptionhorizon HTTP/1.1
Host: emea.ng.msg.teams.microsoft.com
Authentication: skypetoken=eyJhbGci...
Content-Type: application/json

{
  "consumptionhorizon": "1715842456893;1715842456893;1715842401007"
}
```
