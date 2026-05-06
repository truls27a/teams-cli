# Microsoft Teams APIs

Microsoft Teams for work and school (Azure AD) accounts is served by several independent HTTP APIs. They share an OAuth sign-in but have different hosts, different authentication schemes, different error envelopes, and different ownership of the data they return. None are publicly endorsed by Microsoft and any may change without notice.

All three accept JSON-encoded request bodies, return JSON responses, and use standard HTTP response codes. All require authentication.

## APIs

| API                                | Reference                          | Used for                                                                                              |
| ---------------------------------- | ---------------------------------- | ----------------------------------------------------------------------------------------------------- |
| Authentication (shared OAuth)      | [authentication.md](./authentication.md) | Device-code sign-in, AAD access-token mints, Skype-token exchange, refresh, identity-platform errors. |
| Chat service                       | [chatsvc.md](./chatsvc.md)         | Reading and sending messages, conversation properties, read markers.                                  |
| Chat-service aggregator (CSA)      | [csa.md](./csa.md)                 | Listing chats with titles and member rosters; the team and channel tree.                              |
| Middle tier (MT)                   | [mt.md](./mt.md)                   | User profile lookups by MRI or AAD object ID; tenant discovery.                                       |

See [authentication.md](./authentication.md) for the credentials each API requires.
