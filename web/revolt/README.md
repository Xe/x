# revolt.go

Revolt.go is a Go package for writing bots and self-bots in Revolt
easily. This project is a mantained and re-worked version
[ben-forster's fork](https://github.com/ben-forster/revolt) of
5elenay's library [revoltgo](https://github.com/5elenay/revoltgo).

## Features

- Multiple event listen
- Easy to use
- Supports self bots
- Simple cache system

## API Reference

Click [here](https://pkg.go.dev/within.website/x/web/revolt) for api reference.

## Notice

Please note that you will need the Go 1.20 to use revolt.

This package is still under development and while you can create a
working bot, the library is not finished. Create an issue if you would
like to contribute.

## Ping Pong Example (Bot)

```go
package main

import (
    "os"
    "os/signal"
    "syscall"

    "within.website/x/web/revolt"
)

func main() {
    // Init a new client.
    client := revolt.Client{
        Token: "bot token",
    }

    // Listen a on message event.
    client.OnMessage(func(m *revolt.Message) {
        if m.Content == "!ping" {
            sendMsg := &revolt.SendMessage{}
            sendMsg.SetContent("🏓 Pong!")

            m.Reply(true, sendMsg)
        }
    })

    // Start the client.
    client.Start()

    // Wait for close.
    sc := make(chan os.Signal, 1)

    signal.Notify(
        sc,
        syscall.SIGINT,
        syscall.SIGTERM,
        os.Interrupt,
    )
    <-sc

    // Destroy client.
    client.Destroy()
}

```

## Ping Pong Example (Self-Bot)

```go
package main

import (
    "os"
    "os/signal"
    "syscall"

    "within.website/x/web/revolt"
)

func main() {
    // Init a new client.
    client := revolt.Client{
        SelfBot: &revolt.SelfBot{
            Id:           "session id",
            SessionToken: "session token",
            UserId:       "user id",
        },
    }

    // Listen a on message event.
    client.OnMessage(func(m *revolt.Message) {
        if m.Content == "!ping" {
            sendMsg := &revolt.SendMessage{}
            sendMsg.SetContent("🏓 Pong!")

            m.Reply(true, sendMsg)
        }
    })

    // Start the client.
    client.Start()

    // Wait for close.
    sc := make(chan os.Signal, 1)

    signal.Notify(
        sc,
        syscall.SIGINT,
        syscall.SIGTERM,
        os.Interrupt,
    )
    <-sc

    // Destroy client.
    client.Destroy()
}

```

## To-Do

- [x] OnReady
- [x] OnMessage
- [x] OnMessageUpdate
- [ ] OnMessageAppend
- [x] OnMessageDelete
- [x] OnChannelCreate
- [x] OnChannelUpdate
- [x] OnChannelDelete
- [ ] OnChannelGroupJoin
- [ ] OnChannelGroupLeave
- [x] OnChannelStartTyping
- [x] OnChannelStopTyping
- [ ] OnChannelAck
- [x] OnServerCreate
- [x] OnServerUpdate
- [x] OnServerDelete
- [x] OnServerMemberUpdate
- [x] OnServerMemberJoin
- [x] OnServerMemberLeave
- [ ] OnServerRoleUpdate
- [ ] OnServerRoleDelete
- [ ] OnUserUpdate
- [ ] OnUserRelationship
- [ ] OnEmojiCreate
- [ ] OnEmojiDelete
