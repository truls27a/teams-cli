# Conversations

A Conversation represents a thread of messages between two or more participants. The same data model covers one-to-one chats, federated one-to-one chats, and group threads; they are distinguished by the prefix of the conversation MRI and by `threadProperties.productThreadType`.

## The Conversation object

### Common attributes

| Field                       | Type                                                | Description                                                                                                                                               |
| --------------------------- | --------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`                        | string                                              | The conversation MRI. See [Identifying conversations](#identifying-conversations).                                                                        |
| `type`                      | string                                              | Always `Conversation`.                                                                                                                                    |
| `targetLink`                | string                                              | Self-link to the conversation resource.                                                                                                                   |
| `messages`                  | string                                              | Absolute URL of the conversation's message collection.                                                                                                    |
| `version`                   | integer                                             | Server-side version stamp (Unix milliseconds), bumped on every server-side mutation.                                                                      |
| `lastUpdatedMessageId`      | string                                              | `id` of the most recently delivered or mutated message.                                                                                                   |
| `lastUpdatedMessageVersion` | integer                                             | `version` of that message.                                                                                                                                |
| `lastRcMetadataVersion`     | integer                                             | Version stamp on the conversation's roster / control metadata.                                                                                            |
| `lastMessage`               | [Message](./messages.md#the-message-object) \| null | The most recent message in the conversation, or `null` for empty threads. The embedded message also carries a `sequenceId` and a `clientmessageid` echo.  |
| `properties`                | [ConversationProperties](#conversationproperties)   | Per-conversation metadata.                                                                                                                                |
| `threadProperties`          | [ThreadProperties](#threadproperties) \| null       | Per-thread metadata. Present for every `19:` MRI, including federated one-to-one chats. `null` only for native one-to-one chats whose `id` is a user MRI. |
| `memberProperties`          | [MemberProperties](#memberproperties)               | Per-caller membership state for this thread.                                                                                                              |

### Identifying conversations

| Prefix     | Conversation kind                                                                                                                                                            | Example                   |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- |
| `8:live:`  | Native one-to-one with another personal Microsoft account. `id` is the other participant's user MRI; `threadProperties` is `null`.                                           | `8:live:exampleuser`      |
| `8:orgid:` | Native one-to-one with a federated work or school account, when both parties are on the consumer chat fabric. Rare; most federated chats are realised as `19:` threads.      | `8:orgid:<guid>`          |
| `19:`      | Thread. Used for group chats _and_ for federated one-to-one chats. Distinguish by `threadProperties.productThreadType` (`OneToOneChat` for federated 1:1, `Chat` for group). | `19:abcd…@unq.gbl.spaces` |
| `48:bot:`  | One-to-one with a bot.                                                                                                                                                       | `48:bot:...`              |

### ConversationProperties

A string-to-string map. Common entries:

| Key                   | Description                                                                                                                                                           |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `consumptionhorizon`  | Read marker, formatted as `<lastReadMessageId>;<lastReadArrivalMs>;<lastSentByMeArrivalMs>`. Updated via [Mark a conversation as read](#mark-a-conversation-as-read). |
| `lastimreceivedtime`  | Unix-millisecond timestamp of the most recent incoming message.                                                                                                       |
| `isemptyconversation` | `"true"` or `"false"`.                                                                                                                                                |
| `pinnedindex`         | Zero-based pin position. Present only when the conversation is pinned.                                                                                                |
| `mute`                | JSON-encoded `{ "isMuted": boolean, "until": integer }`. Present only when notifications are muted.                                                                   |
| `userTileId`          | Opaque tile identifier for the contact card.                                                                                                                          |

### ThreadProperties

| Key                   | Description                                                                                                  |
| --------------------- | ------------------------------------------------------------------------------------------------------------ |
| `threadType`          | Internal thread classification (e.g. `chat`, `space`).                                                       |
| `productThreadType`   | Surface classification. `OneToOneChat` for a federated 1:1; `Chat` for a group chat.                         |
| `topic`               | Group title. Empty for federated 1:1.                                                                        |
| `createdat`           | Unix-millisecond creation timestamp.                                                                         |
| `creator`             | MRI of the user who created the thread.                                                                      |
| `memberCount`         | Number of participants, as a stringified integer. Absent for federated 1:1.                                  |
| `originalThreadId`    | Predecessor thread MRI when the thread was migrated; equal to `id` otherwise.                                |
| `lastSequenceId`      | Sequence number of the last delivered message. See [`Message.sequenceId`](./messages.md#the-message-object). |
| `version`             | Thread-metadata version stamp.                                                                               |
| `rosterVersion`       | Membership-list version stamp.                                                                               |
| `isStickyThread`      | `true` when the thread is system-pinned and cannot be deleted by participants.                               |
| `isCreator`           | `true` when the caller created the thread.                                                                   |
| `gapDetectionEnabled` | `true` when the service emits sequence-gap notifications for this thread.                                    |
| `lastjoinat`          | Unix-millisecond timestamp at which the caller most recently joined.                                         |
| `chatFilesIndexId`    | Identifier used to look up files attached anywhere in the thread.                                            |

### MemberProperties

| Key                    | Description                                                                           |
| ---------------------- | ------------------------------------------------------------------------------------- |
| `role`                 | Caller's role (e.g. `User`, `Admin`).                                                 |
| `isReader`             | `true` when the caller has read-only access.                                          |
| `memberExpirationTime` | Unix-millisecond timestamp when the caller's membership expires. `0` if non-expiring. |
| `relationshipState`    | Federation state with the other party (e.g. `Allowed`).                               |

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
  "conversations": [
    /* Conversation objects */
  ],
  "_metadata": {
    "totalCount": 30,
    "syncState": "https://msgapi.teams.live.com/v1/users/ME/conversations?cursor=..."
  }
}
```

To page backward, re-issue the request as `GET <syncState>` unmodified until the response carries no `syncState` or returns an empty `conversations` array.

### Example request

```http
GET /v1/users/ME/conversations?pageSize=30&view=msnp24Equivalent HTTP/1.1
Host: msgapi.teams.live.com
Authentication: skypetoken=eyJhbGci...
BehaviorOverride: redirectAs404
```

### Example response

```json
{
  "conversations": [
    {
      "id": "19:abcd0123…@unq.gbl.spaces",
      "type": "Conversation",
      "targetLink": "https://msgapi.teams.live.com/v1/users/ME/conversations/19:abcd0123…@unq.gbl.spaces",
      "messages": "https://msgapi.teams.live.com/v1/users/ME/conversations/19:abcd0123…@unq.gbl.spaces/messages",
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
        "from": "https://msgapi.teams.live.com/v1/users/ME/contacts/8:orgid:0000aaaa-…",
        "imdisplayname": "Other Person",
        "clientmessageid": "8472635198472635",
        "content": "<p>ok sounds good</p>",
        "composetime": "2026-05-04T07:04:11.213Z",
        "originalarrivaltime": "2026-05-04T07:04:11.213Z",
        "version": "1715842456893",
        "conversationid": "19:abcd0123…@unq.gbl.spaces",
        "conversationLink": "https://msgapi.teams.live.com/v1/users/ME/conversations/19:abcd0123…@unq.gbl.spaces"
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
        "creator": "8:live:exampleuser",
        "version": 1715842456893,
        "rosterVersion": 1714000000000,
        "lastSequenceId": 187,
        "originalThreadId": "19:abcd0123…@unq.gbl.spaces",
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
    "syncState": "https://msgapi.teams.live.com/v1/users/ME/conversations?cursor=...&pageSize=30"
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
GET /v1/users/ME/conversations/19%3Aabcd0123%E2%80%A6%40unq.gbl.spaces?view=msnp24Equivalent HTTP/1.1
Host: msgapi.teams.live.com
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

| Name                 | Type   | Description                                                                                                                                                                                                                         |
| -------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `consumptionhorizon` | string | `<lastReadMessageId>;<lastReadArrivalMs>;<lastSentByMeArrivalMs>`. The third value should carry forward the previous one if the caller has not just sent a message. Backwards horizons (lower than the current value) are rejected. |

### Returns

`204 No Content` on success.

### Example request

```http
PUT /v1/users/ME/conversations/19%3Aabcd0123%E2%80%A6%40unq.gbl.spaces/properties?name=consumptionhorizon HTTP/1.1
Host: msgapi.teams.live.com
Authentication: skypetoken=eyJhbGci...
Content-Type: application/json

{
  "consumptionhorizon": "1715842456893;1715842456893;1715842401007"
}
```
