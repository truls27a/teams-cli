# Messages

A Message is a single entry in a [Conversation](./conversations.md). The same collection supports listing existing messages and posting new ones.

## The Message object

### Common attributes

| Field                       | Type                                    | Description                                                                                                                                                                                                |
| --------------------------- | --------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`                        | string                                  | Server-assigned message identifier. The numeric value is the message's Unix-millisecond arrival time; treat it as opaque. Equal to the trailing path segment of the `Location` header on the send response. |
| `sequenceId`                | integer                                 | Per-conversation monotonically increasing sequence number. Use this to detect gaps; use `originalarrivaltime` to sort.                                                                                      |
| `type`                      | string                                  | Always `Message`.                                                                                                                                                                                           |
| `messagetype`               | string                                  | Body format. See [Message types](#message-types).                                                                                                                                                           |
| `contenttype`               | string                                  | Body classification. `Text` for textual bodies (HTML or plain), `application/cards+json` for adaptive cards, `application/user-properties+json` for control messages. Capitalised exactly as shown.        |
| `content`                   | string                                  | The body. Encoding depends on `messagetype`.                                                                                                                                                                |
| `from`                      | string                                  | Absolute URL pointing at the sender's contact resource. The sender's MRI is the trailing path segment.                                                                                                      |
| `imdisplayname`             | string                                  | Display name as it was at send time. Not a stable identity.                                                                                                                                                 |
| `prioritizeImDisplayName`   | string                                  | `"true"` when the sender requests that `imdisplayname` override the recipient's contact-card name.                                                                                                          |
| `clientmessageid`           | string                                  | The idempotency key supplied by the sender. Echoed verbatim. See [Send a message](#send-a-message).                                                                                                         |
| `conversationid`            | string                                  | The conversation MRI.                                                                                                                                                                                        |
| `conversationLink`          | string                                  | Self-link to the conversation.                                                                                                                                                                              |
| `composetime`               | string                                  | ISO 8601 timestamp recorded by the sender's client. May drift; use `originalarrivaltime` for ordering.                                                                                                      |
| `originalarrivaltime`       | string                                  | ISO 8601 timestamp at which the service first stored the message.                                                                                                                                           |
| `version`                   | string                                  | Stamp updated on every server-side mutation (edit, delete). Equal to `id` on first delivery.                                                                                                                |
| `parentmessageid`           | string                                  | For replies: the `id` of the message being replied to. Absent on top-level messages.                                                                                                                        |
| `skypeeditedid`             | string                                  | For edits: the `id` of the original message. Equal to `id` for the first edit, then carried forward.                                                                                                        |
| `amsreferences`             | array \| null                           | AMS object identifiers attached to the message. `null` when no attachments.                                                                                                                                 |
| `properties`                | [MessageProperties](#messageproperties) | Per-message metadata.                                                                                                                                                                                       |

### MessageProperties

A string-to-string map. Each value is a JSON-encoded string (often the literal `"[]"` for unused arrays).

| Key             | Description                                                                                                                       |
| --------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `mentions`      | JSON-encoded array of [Mention](#mention) objects. See [Mention a user](#mention-a-user).                                         |
| `cards`         | JSON-encoded array of attached card objects.                                                                                      |
| `files`         | JSON-encoded array of file references. Used in tandem with `amsreferences`.                                                       |
| `links`         | JSON-encoded array of unfurled link previews.                                                                                     |
| `formatVariant` | Format hint (typically `Teams`).                                                                                                  |
| `languageStamp` | JSON-encoded language-detection result, including the detected ISO code and a confidence score.                                   |
| `deletetime`    | Unix-millisecond timestamp of deletion. Present only on deleted messages; `content` is cleared in the same response.              |
| `edittime`      | Unix-millisecond timestamp of the most recent edit. Present only on edited messages.                                              |

### Message types

Clients should ignore unknown `messagetype` values rather than failing — the service adds new types over time.

| `messagetype`                | `content` encoding                                                                                                                                                                                          |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `RichText/Html`              | HTML fragment. Supported tags: `<p>`, `<b>`, `<i>`, `<u>`, `<s>`, `<a>`, `<ul>`, `<ol>`, `<li>`, `<blockquote>`, `<pre>`, `<code>`, `<br>`, `<at id="…">`, `<quote>`. Other tags are stripped on delivery. The Teams web client emits this type for every user-composed message, including plain-text bodies. |
| `Text`                       | Plain text. The service stores the bytes as-is; clients render them without HTML interpretation. Accepted on send; not emitted by the Teams web client.                                                     |
| `RichText/Media_GenericFile` | XML containing a `<URIObject>` element with `uri`, `urlthumbnail`, and `<OriginalName>`. Files are uploaded to the AMS service before sending.                                                              |
| `Control/Typing`             | Control message indicating the sender is typing. No `content`.                                                                                                                                              |
| `Control/ClearTyping`        | Control message indicating the sender stopped typing. No `content`.                                                                                                                                         |
| `ThreadActivity/AddMember`   | Membership-change marker. Body is XML.                                                                                                                                                                      |
| `Event/Call`                 | Call signaling. Body is XML.                                                                                                                                                                                |

### Mention

Each entry of the `properties.mentions` array:

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

| Name        | Type    | Description                                                                                                                                 |
| ----------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `pageSize`  | integer | Page size. Default `20`. Service maximum is `200`; larger values are clamped silently.                                                      |
| `startTime` | integer | Unix-millisecond lower bound on `originalarrivaltime`. Returns messages with arrival time strictly greater than `startTime`.                |
| `view`      | string  | One of `msnp24Equivalent` (compatibility), `supportsMessageProperties` (includes `properties`), `supportsMessagePolicies` (adds DLP fields). |
| `syncState` | string  | Continuation cursor from the previous page's `_metadata.syncState`.                                                                         |

### Returns

```json
{
  "messages": [ /* Message objects */ ],
  "tenantId": "00000000-0000-0000-0000-000000000000",
  "_metadata": {
    "totalCount": 20,
    "syncState": "https://emea.ng.msg.teams.microsoft.com/v1/users/ME/conversations/.../messages?cursor=..."
  }
}
```

| Field            | Description                                                                                                                                          |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `messages`       | Array of [Message](#the-message-object) objects.                                                                                                     |
| `tenantId`       | The tenant on which the conversation is rooted. For native chats this is the caller's tenant; for federated chats this identifies the other tenant. |
| `_metadata`      | `totalCount` and the pagination `syncState`.                                                                                                         |

To page backward through history, re-issue the request as `GET <syncState>` unmodified until the response carries no `syncState` or returns an empty `messages` array.

### Example request

```http
GET /v1/users/ME/conversations/19%3Aabcd0123%E2%80%A6%40thread.v2/messages?pageSize=20 HTTP/1.1
Host: emea.ng.msg.teams.microsoft.com
Authentication: skypetoken=eyJhbGci...
BehaviorOverride: redirectAs404
```

### Example response

```json
{
  "messages": [
    {
      "sequenceId": 187,
      "id": "1715842456893",
      "type": "Message",
      "messagetype": "RichText/Html",
      "contenttype": "Text",
      "from": "https://emea.ng.msg.teams.microsoft.com/v1/users/ME/contacts/8:orgid:00000000-0000-0000-0000-000000000000",
      "imdisplayname": "Other Person",
      "clientmessageid": "8472635198472635",
      "conversationid": "19:abcd0123…@thread.v2",
      "conversationLink": "https://emea.ng.msg.teams.microsoft.com/v1/users/ME/conversations/19:abcd0123…@thread.v2",
      "content": "<p>ok sounds good</p>",
      "composetime": "2026-05-04T07:04:11.213Z",
      "originalarrivaltime": "2026-05-04T07:04:11.213Z",
      "version": "1715842456893",
      "amsreferences": null,
      "properties": {
        "mentions": "[]",
        "cards": "[]",
        "files": "[]",
        "links": "[]",
        "formatVariant": "Teams",
        "languageStamp": "{\"detected\":\"en\",\"confidence\":0.99}"
      }
    }
  ],
  "tenantId": "00000000-0000-0000-0000-000000000000",
  "_metadata": {
    "totalCount": 20,
    "syncState": "https://emea.ng.msg.teams.microsoft.com/v1/users/ME/conversations/19:abcd0123…@thread.v2/messages?cursor=..."
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

| Field             | Type   | Description                                                                                                                                                                                                                                                              |
| ----------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `content`         | string | The message body, encoded per `messagetype`. Required.                                                                                                                                                                                                                   |
| `messagetype`     | string | Required. Use `RichText/Html` for any message containing formatting, mentions, links, or replies; use `Text` for plain bodies sent without HTML interpretation. The Teams web client uses `RichText/Html` for every user-composed message.                                |
| `contenttype`     | string | Required. `Text` for textual bodies (with either `messagetype`). The service rejects bodies whose declared `contenttype` does not match `messagetype`.                                                                                                                   |
| `clientmessageid` | string | Idempotency key. A 16–18 digit base-10 string generated client-side; the web client uses `Date.now() * 1000 + random(0, 999)`. Re-sending the same key returns the existing message rather than creating a duplicate. Recommended.                                       |
| `imdisplayname`   | string | Display name to record on the message. Defaults to the caller's profile name.                                                                                                                                                                                            |
| `parentmessageid` | string | The `id` of the message being replied to. Required when sending a reply.                                                                                                                                                                                                 |
| `properties`      | object | Free-form `string → string` map persisted alongside the message. Total size capped at 32 KiB. Used for [mentions](#mention-a-user) and similar attachments.                                                                                                              |
| `amsreferences`   | array  | AMS object IDs to attach (images, files, voice clips). Used in conjunction with `RichText/Media_*` message types.                                                                                                                                                        |

### Returns

`201 Created` with a JSON body containing the assigned arrival time:

```http
HTTP/1.1 201 Created
Location: https://emea.ng.msg.teams.microsoft.com/v1/users/ME/conversations/19:abcd0123…@thread.v2/messages/1777987909877

{ "OriginalArrivalTime": 1777987909877 }
```

The new message's `id` is `OriginalArrivalTime` from the body, and equally the trailing path segment of `Location`. The `OriginalArrivalTime` HTTP response header is *not* exposed via CORS to browser callers; non-browser callers may read it from the header instead of the body.

### Example request

```http
POST /v1/users/ME/conversations/19%3Aabcd0123%E2%80%A6%40thread.v2/messages HTTP/1.1
Host: emea.ng.msg.teams.microsoft.com
Authentication: skypetoken=eyJhbGci...
Content-Type: application/json

{
  "content": "<p>Hello from the API</p>",
  "messagetype": "RichText/Html",
  "contenttype": "Text",
  "clientmessageid": "8472635198472635"
}
```

### Reply to a message

Set `parentmessageid` to the target message's `id` and send an HTML body that quotes the original:

```json
{
  "messagetype": "RichText/Html",
  "contenttype": "Text",
  "content": "<quote author=\"Other Person\" timestamp=\"1715842456893\"><legacyquote>[2026-05-04 07:04:11] Other Person: </legacyquote>ok sounds good</quote>Sounds good.",
  "parentmessageid": "1715842456893",
  "clientmessageid": "8472635198472636"
}
```

`<quote>` renders as a styled blockquote; `<legacyquote>` is the fallback used by clients that do not understand `<quote>`.

### Mention a user

Wrap the mentioned user's display name in `<at id="...">` (where `id` is the zero-based mention ordinal) and include a `mentions` entry in `properties`:

```json
{
  "messagetype": "RichText/Html",
  "contenttype": "Text",
  "content": "<at id=\"0\">Other Person</at> can you check this?",
  "properties": {
    "mentions": "[{\"@type\":\"http://schema.skype.com/Mention\",\"itemid\":0,\"mri\":\"8:orgid:00000000-0000-0000-0000-000000000000\",\"displayName\":\"Other Person\"}]"
  },
  "clientmessageid": "8472635198472637"
}
```

`properties.mentions` is a JSON-encoded string, not a JSON value.
