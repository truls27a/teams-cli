# Chats

A Chat is a thread of messages returned by the chat-service aggregator (CSA). The same model covers one-to-one chats, federated one-to-one chats with consumer Microsoft accounts, group chats, and meeting chats. Threads are distinguished by `chatType`, `isOneOnOne`, and the prefix of the chat MRI.

CSA does not surface bot chats (`48:bot:`) or system feeds (`48:notifications`, `48:mentions`, …); those remain available through the Skype-token chat service. Team channels are returned in the same response under `teams[].channels[]` rather than in `chats[]`.

## The Chat object

### Common attributes

| Field                 | Type                                                       | Description                                                                                                                       |
| --------------------- | ---------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `id`                  | string                                                     | The chat MRI. See [Identifying chats](#identifying-chats).                                                                        |
| `chatType`            | string                                                     | `chat` for one-to-one and group chats, `meeting` for calendar meeting chats.                                                      |
| `threadType`          | string                                                     | Mirrors `chatType` in current responses.                                                                                          |
| `chatSubType`         | integer                                                    | Sub-classifier within `chatType`. Treat as opaque.                                                                                |
| `isOneOnOne`          | boolean                                                    | `true` for one-to-one chats (native or federated). Combine with `chatType` to identify group chats (`chatType: "chat"` and `isOneOnOne: false`). |
| `isLastMessageFromMe` | boolean                                                    | `true` when the most recent message in the chat was sent by the caller.                                                           |
| `isRead`              | boolean                                                    | `true` when the caller has consumed up to the last delivered message.                                                             |
| `isEmptyConversation` | boolean                                                    | `true` when the chat has no messages.                                                                                             |
| `isExternal`          | boolean                                                    | `true` when at least one member belongs to a different tenant than the caller.                                                    |
| `isMessagingDisabled` | boolean                                                    | `true` when sending is administratively blocked for this chat.                                                                    |
| `hidden`              | boolean                                                    | `true` when the caller has hidden the chat from their list.                                                                       |
| `isSticky`            | boolean                                                    | `true` when the chat is pinned for the caller.                                                                                    |
| `title`               | string \| null                                             | Chat title. `null` for one-to-one chats and for groups without a custom name.                                                     |
| `creator`             | string                                                     | MRI of the user who created the chat.                                                                                             |
| `tenantId`            | string                                                     | Tenant ID of the chat's home tenant.                                                                                              |
| `createdAt`           | string                                                     | Unix-millisecond creation timestamp, as a string.                                                                                 |
| `lastJoinAt`          | string                                                     | Unix-millisecond timestamp at which the caller most recently joined, as a string.                                                 |
| `version`             | integer                                                    | Server-side version stamp (Unix milliseconds), bumped on every server-side mutation.                                              |
| `threadVersion`       | integer                                                    | Thread-metadata version stamp.                                                                                                    |
| `rosterVersion`       | integer                                                    | Membership-list version stamp.                                                                                                    |
| `members`             | array of [ChatMember](#chatmember)                         | Roster of the chat. The caller is included.                                                                                       |
| `lastMessage`         | [LastMessage](#lastmessage) \| null                        | Preview of the most recent message. `null` for empty chats.                                                                       |
| `meetingInformation`  | [MeetingInformation](#meetinginformation) \| null          | Calendar metadata. Present only when `chatType` is `meeting`.                                                                     |
| `consumptionHorizon`  | object                                                     | Read marker. Object with `originalArrivalTime` (integer), `timeStamp` (integer), and `clientMessageId` (string).                  |
| `relationshipState`   | object                                                     | Federation relationship with the chat. Contains `inQuarantine` (boolean) and `hasImpersonation` (string).                         |

The CSA response carries additional internal flags (`isMigrated`, `isGapDetectionEnabled`, `isSmsThread`, `isLiveChatEnabled`, `isHighImportance`, `interopType`, `interopConversationStatus`, `importState`, `templateType`, `productContext`, `meetingPolicy`, `identityMaskEnabled`, `hasTranscript`, `isConversationDeleted`, `isDisabled`, `conversationBlockedAt`, `lastL2MessageIdNFS`, `lastRcMetadataVersion`, `retentionHorizon`, `retentionHorizonV2`, `fileReferences`). These are returned for compatibility with the web client and are not required to render a chat list.

### Identifying chats

CSA returns chats under three `19:` shapes. Bot threads (`48:bot:`) and system feeds (`48:`) are excluded from the response.

| Prefix                                | Kind                                                                                                                            | Example                                       |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------- |
| `19:<guid>_<guid>@unq.gbl.spaces`     | Native one-to-one chat. The two GUIDs are the participant AAD object IDs in lexicographic order.                                | `19:<guid>_<guid>@unq.gbl.spaces`             |
| `19:<hex>@thread.v2`                  | Group chat. `chatType: "chat"`, `isOneOnOne: false`.                                                                            | `19:<hex>@thread.v2`                          |
| `19:meeting_<base64>@thread.v2`       | Meeting chat. `chatType: "meeting"`. `meetingInformation` is populated.                                                         | `19:meeting_<base64>@thread.v2`               |
| `19:uni01_<base64>@thread.v2`         | Federated one-to-one chat with a consumer Microsoft account. `isOneOnOne: true`. The consumer participant's MRI is `8:live:…`. | `19:uni01_<base64>@thread.v2`                 |

### ChatMember

| Field              | Type    | Description                                                                                                                                  |
| ------------------ | ------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `mri`              | string  | Member MRI. `8:orgid:<guid>` for AAD users, `8:live:<id>` for consumer Microsoft accounts, `28:<id>` for bots and apps.                       |
| `objectId`         | string  | AAD object ID of the member, when applicable.                                                                                                |
| `tenantId`         | string  | Tenant ID of the member. Absent for the caller's own entry in some chats.                                                                    |
| `role`             | string  | Member role. Known values: `Admin`, `User`.                                                                                                  |
| `friendlyName`     | string  | Display name shown for this member in this chat. Absent for the caller's own entry, and for some interop members. Treat as optional.        |
| `isMuted`          | boolean | `true` when notifications from this member are muted for the caller.                                                                         |
| `isIdentityMasked` | boolean | `true` when the member's identity is hidden from other participants.                                                                         |
| `shareHistoryTime` | string  | Unix-millisecond timestamp from which prior message history is shared with this member, as a string. Absent when no history is shared.       |

### MeetingInformation

Present only when `chatType` is `meeting`.

| Field                       | Type    | Description                                                                                  |
| --------------------------- | ------- | -------------------------------------------------------------------------------------------- |
| `subject`                   | string  | Meeting subject.                                                                             |
| `location`                  | string  | Meeting location string.                                                                     |
| `startTime`                 | string  | ISO 8601 start time.                                                                         |
| `endTime`                   | string  | ISO 8601 end time.                                                                           |
| `iCalUid`                   | string  | iCalendar UID of the calendar event.                                                         |
| `meetingJoinUrl`            | string  | Join URL.                                                                                    |
| `organizerId`               | string  | MRI of the meeting organiser.                                                                |
| `coOrganizerIds`            | array   | MRIs of co-organisers.                                                                       |
| `tenantId`                  | string  | Tenant ID of the organiser.                                                                  |
| `appointmentType`           | integer | Appointment classification.                                                                  |
| `meetingType`               | integer | Meeting classification.                                                                      |
| `scenario`                  | string  | Originating scenario (e.g. `Default`).                                                       |
| `isCancelled`               | boolean | `true` when the calendar event has been cancelled.                                           |
| `isCopyRestrictionEnforced` | boolean | `true` when copy/paste restrictions apply.                                                   |
| `enableMultiLingualMeeting` | boolean | `true` when multilingual features are enabled.                                               |
| `groupCopilotDetails`       | object  | Copilot configuration. Opaque.                                                               |
| `exchangeId`                | string  | Exchange identifier of the meeting series. May be `null`.                                    |

### LastMessage

A summary of the chat's most recent message. Distinct from the [Message](./messages.md#the-message-object) returned by the chat-service messages endpoint; the field set is smaller and the JSON keys differ in case.

| Field                  | Type    | Description                                                                                       |
| ---------------------- | ------- | ------------------------------------------------------------------------------------------------- |
| `id`                   | string  | Server-assigned message identifier.                                                               |
| `type`                 | string  | Always `Message`.                                                                                 |
| `messageType`          | string  | Body format (e.g. `RichText/Html`, `Text`).                                                       |
| `content`              | string  | Body content. May contain HTML when `messageType` is `RichText/Html`.                             |
| `composeTime`          | string  | ISO 8601 timestamp when the sender composed the message.                                          |
| `originalArrivalTime`  | string  | ISO 8601 timestamp when the chat service first received the message.                              |
| `clientMessageId`      | string  | Client-supplied idempotency token.                                                                |
| `parentMessageId`      | string  | Parent message ID for replies; `"0"` when the message is a top-level post.                        |
| `containerId`          | string  | Server-side container identifier.                                                                 |
| `from`                 | string  | Sender MRI.                                                                                       |
| `imDisplayName`        | string  | Display name shown for the sender at compose time.                                                |
| `sequenceId`           | integer | Per-conversation monotonically increasing sequence number.                                        |
| `version`              | integer | Server-side version stamp.                                                                        |
| `threadType`           | string  | Often `null` in this preview shape.                                                               |
| `fromDisplayNameInToken` | string \| null | Display name from the sender's token at send time. May be `null`.                          |
| `fromGivenNameInToken`   | string \| null | Given name from the sender's token at send time. May be `null`.                            |
| `fromFamilyNameInToken`  | string \| null | Family name from the sender's token at send time. May be `null`.                           |
| `isEscalationToNewPerson` | boolean | `true` when the message escalated the chat to include a new participant.                        |

---

## List chats

```
GET /v1/teams/users/me
```

Returns the caller's chats together with their team list and metadata. Hosted on the chat-service aggregator at `https://teams.microsoft.com/api/csa/api`, not the regional chat-service host. The `regionGtms` map advertises the aggregator's underlying service at `https://chatsvcagg.teams.microsoft.com`; both fronts serve identical paths. The aggregator is global and not region-stamped.

### Authentication

The aggregator is bearer-authenticated. The `Authentication: skypetoken=...` header used by the chat service is rejected. Acquire an Azure AD access token whose audience is `chatsvcagg.teams.microsoft.com` and pass it as `Authorization: Bearer <jwt>`. See [Authentication](./authentication.md#aggregator-bearer-token).

### Query parameters

| Name                      | Type    | Description                                                                                                                                         |
| ------------------------- | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `isPrefetch`              | boolean | When `false` (default for interactive callers), the response is computed live. When `true`, the service may return a cached snapshot.               |
| `enableMembershipSummary` | boolean | When `true`, populates `members[].friendlyName` and the team `membershipSummary` object. Without this flag, member names are omitted.               |

### Returns

An object with `chats`, `teams`, `users`, `privateFeeds`, and a `metadata` object containing pagination state.

```json
{
  "chats": [ /* Chat objects */ ],
  "teams": [ /* Team objects with nested channels */ ],
  "users": [ /* supplementary user records */ ],
  "privateFeeds": [ /* system feeds: notifications, mentions, etc. */ ],
  "metadata": {
    "syncToken": "eyJk...",
    "forwardSyncToken": null,
    "isPartialData": false,
    "hasMoreChats": false
  }
}
```

To page forward, persist `metadata.syncToken` and re-issue the request with `syncToken=<value>`. When `metadata.hasMoreChats` is `false`, the caller has the full set.

### Example request

```http
GET /v1/teams/users/me?isPrefetch=false&enableMembershipSummary=true HTTP/1.1
Host: teams.microsoft.com
Authorization: Bearer eyJ0eXAi...
```

### Example response

```json
{
  "chats": [
    {
      "id": "19:abcd0123…@thread.v2",
      "chatType": "chat",
      "threadType": "chat",
      "chatSubType": 0,
      "isOneOnOne": false,
      "isLastMessageFromMe": true,
      "isRead": true,
      "isEmptyConversation": false,
      "isExternal": false,
      "hidden": false,
      "isSticky": false,
      "title": "Project Atlas",
      "creator": "8:orgid:00000000-0000-0000-0000-000000000000",
      "tenantId": "00000000-0000-0000-0000-000000000000",
      "createdAt": "1714000000000",
      "lastJoinAt": "1714000000000",
      "version": 1715842456893,
      "threadVersion": 1715842456893,
      "rosterVersion": 1714000000000,
      "members": [
        {
          "mri": "8:orgid:00000000-0000-0000-0000-000000000000",
          "objectId": "00000000-0000-0000-0000-000000000000",
          "tenantId": "00000000-0000-0000-0000-000000000000",
          "role": "Admin",
          "friendlyName": "Other Person",
          "isMuted": false,
          "isIdentityMasked": false,
          "shareHistoryTime": "1714000000000"
        }
      ],
      "lastMessage": {
        "id": "1715842456893",
        "type": "Message",
        "messageType": "RichText/Html",
        "content": "<p>ok sounds good</p>",
        "composeTime": "2026-05-04T07:04:11.213Z",
        "originalArrivalTime": "2026-05-04T07:04:11.213Z",
        "clientMessageId": "8472635198472635",
        "parentMessageId": "0",
        "containerId": "19:abcd0123…@thread.v2",
        "from": "8:orgid:00000000-0000-0000-0000-000000000000",
        "imDisplayName": "Caller",
        "sequenceId": 187,
        "version": 1715842456893,
        "threadType": null,
        "fromDisplayNameInToken": null,
        "fromGivenNameInToken": null,
        "fromFamilyNameInToken": null,
        "isEscalationToNewPerson": false
      },
      "meetingInformation": null,
      "consumptionHorizon": {
        "originalArrivalTime": 1715842456893,
        "timeStamp": 1715842456893,
        "clientMessageId": "8472635198472635"
      },
      "relationshipState": {
        "inQuarantine": false,
        "hasImpersonation": "False"
      }
    }
  ],
  "teams": [],
  "users": [],
  "privateFeeds": [],
  "metadata": {
    "syncToken": "eyJk...",
    "forwardSyncToken": null,
    "isPartialData": false,
    "hasMoreChats": false
  }
}
```
