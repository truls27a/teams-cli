# Microsoft Teams API

The Microsoft Teams API is an undocumented HTTP surface for work and school (Azure AD) accounts. It is not endorsed by Microsoft and may change without notice.

It accepts JSON-encoded request bodies, returns JSON responses, and uses standard HTTP response codes. All requests are authenticated. See [Authentication](./authentication.md).

## Services

The API is split across two HTTP services. They have different hosts, different authentication, and different ownership of the data they return.

| Service                       | Host                                                                                                                                                          | Authentication                     | Used for                                                                                         |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- | ------------------------------------------------------------------------------------------------ |
| Chat service                  | Region-stamped: `https://{region}.ng.msg.teams.microsoft.com`. Front Door proxy at `https://teams.microsoft.com/api/chatsvc/{region}` serves identical paths. | `Authentication: skypetoken=<jwt>` | Reading and sending messages, conversation properties, read markers.                             |
| Chat-service aggregator (CSA) | Global: `https://teams.microsoft.com/api/csa/api`. Also advertised as `https://chatsvcagg.teams.microsoft.com` in `regionGtms.chatServiceAggregator`.         | `Authorization: Bearer <jwt>`      | Listing chats with titles and member rosters; the team and channel tree.                          |

The chat service does not return chat titles or member display names; the aggregator does not serve message history or accept message sends. A typical caller uses both: CSA to list chats, the chat service to read and send within one.

A small set of identity and discovery endpoints lives on the middle tier at `https://teams.microsoft.com/api/mt/{region}`. Where used, the full path is shown.

Do not hard-code the chat-service region. Discover the full service map at sign-in via [Authentication](./authentication.md#exchange-for-a-skype-token) and route subsequent chat-service calls through it. CSA is global and is not region-stamped.

## API version

This reference covers chat-service version `v1` and CSA version `v1`. Prepend `/v1/` to every path shown below.

## Reference

- [Authentication](./authentication.md) — Skype token via OAuth 2.0 device code, and the aggregator bearer token
- [Errors](./errors.md) — Status codes, error response format, rate limiting
- [Chats](./chats.md) — List chats from the aggregator; identify chats across the two services
- [Messages](./messages.md) — Read and send messages within a chat
