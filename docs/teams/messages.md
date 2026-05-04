# Messages

A Message is a single entry in a [Conversation](./conversations.md). The same collection supports listing existing messages and posting new ones.

## The Message object

### Common attributes

| Field                 | Type                                    | Description                                                                                                                                       |
| --------------------- | --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`                  | string                                  | Server-assigned message identifier. Numeric, monotonically increasing per conversation. Treat as opaque.                                          |
| `type`                | string                                  | Always `Message`.                                                                                                                                 |
| `messagetype`         | string                                  | Body format. See [Message types](#message-types).                                                                                                 |
| `contenttype`         | string                                  | MIME-style hint. `text` for textual bodies, `application/cards+json` for adaptive cards, `application/user-properties+json` for control messages. |
| `content`             | string                                  | The body. Encoding depends on `messagetype`.                                                                                                      |
| `from`                | string                                  | Absolute URL pointing at the sender's contact resource. The sender's MRI is the trailing path segment.                                            |
| `imdisplayname`       | string                                  | Display name as it was at send time. Not a stable identity.                                                                                       |
| `conversationid`      | string                                  | The conversation MRI.                                                                                                                             |
| `conversationLink`    | string                                  | Self-link to the conversation.                                                                                                                    |
| `composetime`         | string                                  | ISO 8601 timestamp recorded by the sender's client. May drift; use `originalarrivaltime` for ordering.                                            |
| `originalarrivaltime` | string                                  | ISO 8601 timestamp at which the service first stored the message. Use this for ordering.                                                          |
| `version`             | string                                  | Stamp updated on every server-side mutation (edit, delete). Equal to `id` on first delivery.                                                      |
| `parentmessageid`     | string                                  | For replies: the `id` of the message being replied to. Absent on top-level messages.                                                              |
| `skypeeditedid`       | string                                  | For edits: the `id` of the original message. Equal to `id` for the first edit, then carried forward.                                              |
| `properties`          | [MessageProperties](#messageproperties) | Optional. Edit and delete timestamps and other extension data.                                                                                    |

### MessageProperties

| Key          | Type            | Description                                                                                                                      |
| ------------ | --------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `deletetime` | integer \| null | Unix-millisecond timestamp of deletion. Non-null for tombstoned messages; `content` is cleared in the same response.             |
| `edittime`   | integer \| null | Unix-millisecond timestamp of the most recent edit. Non-null for edited messages.                                                |
| `mentions`   | string          | JSON-encoded array of [Mention](#mention) objects (see [Mention a user](#mention-a-user)). Stored as a string, not a JSON value. |

### Message types

Clients should ignore unknown `messagetype` values rather than failing — the service adds new types over time.

| `messagetype`                | `content` encoding                                                                                                                                                                                         |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Text`                       | Plain text. The service stores the bytes as-is; clients render them without HTML interpretation.                                                                                                           |
| `RichText/Html`              | HTML fragment. Supported tags: `<p>`, `<b>`, `<i>`, `<u>`, `<s>`, `<a>`, `<ul>`, `<ol>`, `<li>`, `<blockquote>`, `<pre>`, `<code>`, `<br>`, `<at id="…">`, `<quote>`. Other tags are stripped on delivery. |
| `RichText/Media_GenericFile` | XML containing a `<URIObject>` element with `uri`, `urlthumbnail`, and `<OriginalName>`. Files are uploaded to the AMS service before sending.                                                             |
| `Control/Typing`             | Control message indicating the sender is typing. No `content`.                                                                                                                                             |
| `Control/ClearTyping`        | Control message indicating the sender stopped typing. No `content`.                                                                                                                                        |
| `ThreadActivity/AddMember`   | Membership-change marker. Body is XML.                                                                                                                                                                     |
| `Event/Call`                 | Call signaling. Body is XML.                                                                                                                                                                               |

### Mention

The `properties.mentions` string deserializes to an array of:

| Field         | Type    | Description                                                                                                       |
| ------------- | ------- | ----------------------------------------------------------------------------------------------------------------- |
| `@type`       | string  | Always `http://schema.skype.com/Mention`.                                                                         |
| `itemid`      | integer | Zero-based ordinal of the mention within the message. Matches the `id` attribute on the corresponding `<at>` tag. |
| `mri`         | string  | MRI of the mentioned user, group, or bot.                                                                         |
| `displayName` | string  | Display name to render. Should match the text content of the corresponding `<at>` tag.                            |

---

## List messages

```
GET /v1/users/ME/conversations/:id/messages
```

Returns messages in the conversation, **most recent first**.

### Path parameters

| Name | Type   | Description                              |
| ---- | ------ | ---------------------------------------- |
| `id` | string | The URL-encoded MRI of the conversation. |

### Query parameters

| Name        | Type    | Description                                                                                                                                  |
| ----------- | ------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `pageSize`  | integer | Page size. Default `20`. Service maximum is `200`; larger values are clamped silently.                                                       |
| `startTime` | integer | Unix-millisecond lower bound on `originalarrivaltime`. Returns messages with arrival time strictly greater than `startTime`.                 |
| `view`      | string  | One of `msnp24Equivalent` (compatibility), `supportsMessageProperties` (includes `properties`), `supportsMessagePolicies` (adds DLP fields). |
| `syncState` | string  | Continuation cursor from the previous page's `_metadata.syncState`.                                                                          |

### Returns

An object with a `messages` array and a `_metadata` object containing the pagination cursor.

To page backward through history, re-issue the request as `GET <syncState>` unmodified until the response carries no `syncState` or returns an empty `messages` array.

### Example request

```http
GET /v1/users/ME/conversations/8:live:exampleuser/messages?pageSize=20 HTTP/1.1
Host: msgapi.teams.live.com
Authentication: skypetoken=eyJhbGci...
BehaviorOverride: redirectAs404
```

### Example response

```json
{
  "messages": [
    {
      "id": "1715842456893",
      "type": "Message",
      "messagetype": "Text",
      "contenttype": "text",
      "from": "https://msgapi.teams.live.com/v1/users/ME/contacts/8:live:other",
      "imdisplayname": "Other Person",
      "conversationid": "8:live:exampleuser",
      "conversationLink": "https://msgapi.teams.live.com/v1/users/ME/conversations/8:live:exampleuser",
      "content": "ok sounds good",
      "composetime": "2026-05-04T07:04:11.213Z",
      "originalarrivaltime": "2026-05-04T07:04:11.213Z",
      "version": "1715842456893",
      "properties": { "deletetime": null, "edittime": null }
    },
    {
      "id": "1715842401007",
      "type": "Message",
      "messagetype": "RichText/Html",
      "contenttype": "text",
      "from": "https://msgapi.teams.live.com/v1/users/ME/contacts/8:live:exampleuser",
      "imdisplayname": "Example User",
      "conversationid": "8:live:exampleuser",
      "content": "<p>thanks!</p>",
      "composetime": "2026-05-04T07:03:21.007Z",
      "originalarrivaltime": "2026-05-04T07:03:21.007Z",
      "version": "1715842401007"
    }
  ],
  "_metadata": {
    "totalCount": 20,
    "syncState": "https://msgapi.teams.live.com/v1/users/ME/conversations/8:live:exampleuser/messages?cursor=..."
  }
}
```

### Edits and deletes

When a message is edited or deleted the service mutates the existing entry and bumps `version`; it does not insert a new collection entry. Diff by `(id, version)` rather than appending blindly:

- **Edited** messages have `properties.edittime` set, `content` replaced, and `skypeeditedid` populated.
- **Deleted** messages have `properties.deletetime` set; `content` is cleared.

For realtime updates, register a long-poll endpoint instead of polling messages directly; busy chats throttle pollers within minutes.

---

## Send a message

```
POST /v1/users/ME/conversations/:id/messages
```

Posts a new message to the conversation as the caller.

### Path parameters

| Name | Type   | Description                              |
| ---- | ------ | ---------------------------------------- |
| `id` | string | The URL-encoded MRI of the conversation. |

### Request body

| Field             | Type   | Description                                                                                                                                                                                                                        |
| ----------------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `content`         | string | The message body, encoded per `messagetype`. Required.                                                                                                                                                                             |
| `messagetype`     | string | Required. Usually `Text` or `RichText/Html`. See [Message types](#message-types).                                                                                                                                                  |
| `contenttype`     | string | Required. Almost always `text`. The service rejects bodies whose declared `contenttype` does not match `messagetype`.                                                                                                              |
| `clientmessageid` | string | Idempotency key. A 16–18 digit base-10 string generated client-side; the web client uses `Date.now() * 1000 + random(0, 999)`. Re-sending the same key returns the existing message rather than creating a duplicate. Recommended. |
| `imdisplayname`   | string | Display name to record on the message. Defaults to the caller's profile name. Some clients render mismatched names as a security warning; override sparingly.                                                                      |
| `parentmessageid` | string | The `id` of the message being replied to. Required when sending a reply.                                                                                                                                                           |
| `properties`      | object | Free-form `string → string` map persisted alongside the message. Total size capped at 32 KiB. Used for [mentions](#mention-a-user) and similar attachments.                                                                        |
| `amsreferences`   | array  | AMS object IDs to attach (images, files, voice clips). Used in conjunction with `RichText/Media_*` message types.                                                                                                                  |

### Returns

`201 Created` with an empty body. The new resource's URL is returned in the `Location` header; the new `id` is the trailing path segment of `Location` and (modulo same-millisecond collisions) equals the value in the `OriginalArrivalTime` header.

```http
HTTP/1.1 201 Created
OriginalArrivalTime: 1715842512004
Location: https://msgapi.teams.live.com/v1/users/ME/conversations/8:live:exampleuser/messages/1715842512004
```

### Example request

```http
POST /v1/users/ME/conversations/8:live:exampleuser/messages HTTP/1.1
Host: msgapi.teams.live.com
Authentication: skypetoken=eyJhbGci...
Content-Type: application/json

{
  "content": "Hello from the API",
  "messagetype": "Text",
  "contenttype": "text",
  "clientmessageid": "8472635198472635"
}
```

### Reply to a message

Set `parentmessageid` to the target message's `id` and send an HTML body that quotes the original:

```json
{
  "messagetype": "RichText/Html",
  "contenttype": "text",
  "content": "<quote author=\"Other Person\" timestamp=\"1715842456893\"><legacyquote>[2026-05-04 07:04:11] Other Person: </legacyquote>ok sounds good</quote>Sounds good.",
  "parentmessageid": "1715842456893",
  "clientmessageid": "8472635198472636"
}
```

The `<quote>` tag is rendered as a styled blockquote in the recipient's client; `<legacyquote>` is the fallback for older clients that render the message as plain text.

### Mention a user

Wrap the mentioned user's display name in `<at id="...">` (where `id` is the zero-based mention ordinal) and include a `mentions` entry in `properties`:

```json
{
  "messagetype": "RichText/Html",
  "contenttype": "text",
  "content": "<at id=\"0\">Other Person</at> can you check this?",
  "properties": {
    "mentions": "[{\"@type\":\"http://schema.skype.com/Mention\",\"itemid\":0,\"mri\":\"8:live:other\",\"displayName\":\"Other Person\"}]"
  },
  "clientmessageid": "8472635198472637"
}
```

`properties.mentions` is a JSON-encoded string, not a JSON value.
