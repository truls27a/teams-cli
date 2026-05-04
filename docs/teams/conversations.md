# Conversations

A Conversation represents a thread of messages between two or more participants. Conversations come in two shapes — one-to-one chats and group threads — distinguished by the prefix of the conversation's MRI.

## The Conversation object

### Common attributes

| Field              | Type                                                | Description                                                                                                                                                  |
| ------------------ | --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `id`               | string                                              | The conversation MRI. For a 1:1 chat this is the other participant's user MRI (`8:live:...`); for a group chat it is a thread MRI (`19:...@unq.gbl.spaces`). |
| `type`             | string                                              | Always `Conversation`.                                                                                                                                       |
| `targetLink`       | string                                              | Self-link to the conversation resource.                                                                                                                      |
| `messages`         | string                                              | Absolute URL of the conversation's message collection.                                                                                                       |
| `version`          | integer                                             | Server-side version stamp (Unix milliseconds), bumped on every server-side mutation.                                                                         |
| `lastMessage`      | [Message](./messages.md#the-message-object) \| null | The most recent message in the conversation, or `null` for empty threads.                                                                                    |
| `properties`       | [ConversationProperties](#conversationproperties)   | Per-conversation metadata.                                                                                                                                   |
| `threadProperties` | [ThreadProperties](#threadproperties) \| null       | Group-thread metadata. `null` for 1:1 chats.                                                                                                                 |

### Identifying conversations

| Prefix     | Conversation kind                           | Example                   |
| ---------- | ------------------------------------------- | ------------------------- |
| `8:live:`  | 1:1 with a personal Microsoft account.      | `8:live:exampleuser`      |
| `8:orgid:` | 1:1 with a federated work / school account. | `8:orgid:<guid>`          |
| `19:`      | Group thread (any number of participants).  | `19:abcd…@unq.gbl.spaces` |
| `48:bot:`  | 1:1 with a bot.                             | `48:bot:...`              |

For 1:1 conversations, `id` equals the other participant's user MRI; there is no separate conversation identifier.

### ConversationProperties

A string-to-string map. The fields below are the ones a client can rely on:

| Key                   | Description                                                                                                                                                           |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `consumptionhorizon`  | Read marker, formatted as `<lastReadMessageId>;<lastReadArrivalMs>;<lastSentByMeArrivalMs>`. Updated via [Mark a conversation as read](#mark-a-conversation-as-read). |
| `lastimreceivedtime`  | Unix-millisecond timestamp of the most recent incoming message.                                                                                                       |
| `isemptyconversation` | `"true"` or `"false"`.                                                                                                                                                |
| `pinnedindex`         | Zero-based pin position. Present only when the conversation is pinned.                                                                                                |
| `mute`                | JSON-encoded `{ "isMuted": boolean, "until": integer }`. Present only when notifications are muted.                                                                   |
| `userTileId`          | Opaque tile identifier for the contact card.                                                                                                                          |

### ThreadProperties

Returned only when `id` starts with `19:`.

| Key                | Description                                                       |
| ------------------ | ----------------------------------------------------------------- |
| `topic`            | Group title. May be empty.                                        |
| `createdat`        | Unix-millisecond creation timestamp.                              |
| `creator`          | MRI of the user who created the thread.                           |
| `memberCount`      | Number of participants, as a stringified integer.                 |
| `chatFilesIndexId` | Identifier used to look up files attached anywhere in the thread. |

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
      "id": "8:live:exampleuser",
      "type": "Conversation",
      "targetLink": "https://msgapi.teams.live.com/v1/users/ME/conversations/8:live:exampleuser",
      "messages": "https://msgapi.teams.live.com/v1/users/ME/conversations/8:live:exampleuser/messages",
      "version": 1715842456893,
      "lastMessage": {
        "id": "1715842456893",
        "type": "Message",
        "messagetype": "Text",
        "from": "https://msgapi.teams.live.com/v1/users/ME/contacts/8:live:other",
        "content": "ok sounds good",
        "composetime": "2026-05-04T07:04:11.213Z",
        "originalarrivaltime": "2026-05-04T07:04:11.213Z"
      },
      "properties": {
        "consumptionhorizon": "1715842456893;1715842456893;1715842401007",
        "lastimreceivedtime": "1715842456893",
        "isemptyconversation": "false"
      },
      "threadProperties": null
    }
  ],
  "_metadata": {
    "totalCount": 30,
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
GET /v1/users/ME/conversations/8:live:exampleuser?view=msnp24Equivalent HTTP/1.1
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
PUT /v1/users/ME/conversations/8:live:exampleuser/properties?name=consumptionhorizon HTTP/1.1
Host: msgapi.teams.live.com
Authentication: skypetoken=eyJhbGci...
Content-Type: application/json

{
  "consumptionhorizon": "1715842456893;1715842456893;1715842401007"
}
```
