# Microsoft Teams API

The Microsoft Teams API is an undocumented HTTP surface for work and school (Azure AD) accounts. It is not endorsed by Microsoft and may change without notice.

It accepts JSON-encoded request bodies, returns JSON responses, and uses standard HTTP response codes. All requests are authenticated. See [Authentication](./authentication.md).

## Services

The API is split across two HTTP services. They have different hosts, different authentication, and different ownership of the data they return.

| Service                       | Host                                                                                                                                                          | Authentication                     | Used for                                                                                         |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- | ------------------------------------------------------------------------------------------------ |
| Chat service                  | Region-stamped: `https://{region}.ng.msg.teams.microsoft.com`. Front Door proxy at `https://teams.microsoft.com/api/chatsvc/{region}` serves identical paths. | `Authentication: skypetoken=<jwt>` | Reading and sending messages, conversation properties, read markers.                             |
| Chat-service aggregator (CSA) | Global: `https://teams.microsoft.com/api/csa/api`. Also advertised as `https://chatsvcagg.teams.microsoft.com` in `regionGtms.chatServiceAggregator`.         | `Authorization: Bearer <jwt>`      | Listing chats with titles and member rosters; the team and channel tree.                          |
| Middle tier (MT)              | Region-stamped: `https://teams.microsoft.com/api/mt/{region}/beta`.                                                                                            | `Authorization: Bearer <jwt>`      | User profile lookups by MRI or AAD object ID; tenant discovery.                                  |

The chat service does not return chat titles or member display names; the aggregator does not serve message history or accept message sends. The middle tier covers identity lookups when neither has the data — for example, when the aggregator omits a peer's `friendlyName` from a one-to-one chat.

Do not hard-code the chat-service or middle-tier region. Discover the full service map at sign-in via [Authentication](./authentication.md#exchange-for-a-skype-token) and route subsequent calls through it. CSA is global and is not region-stamped.

## API version

This reference covers chat-service version `v1`, CSA version `v1`, and middle-tier version `beta`. Prepend the version segment to every path shown in the service's reference page.

## Reference

- [Authentication](./authentication.md) — Skype token via OAuth 2.0 device code, and the aggregator bearer token
- [Errors](./errors.md) — Status codes, error response format, rate limiting
- [Chats](./chats.md) — List chats from the aggregator; identify chats across the services
- [Messages](./messages.md) — Read and send messages within a chat
- [Users](./users.md) — Resolve member MRIs to display names via the middle tier
