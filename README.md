# teams-cli

A Microsoft Teams CLI, born out of spite for the Electron desktop app and the way it eats RAM for breakfast.

## Commands

```
teams auth login        # sign in via device code
teams auth status       # show auth status
teams auth logout       # clear stored tokens

teams chat list         # list chats
teams chat view <id>    # view messages
teams chat send <id>    # send a message (stdin or $EDITOR)

teams watch             # poll chats and notify on new messages
```
