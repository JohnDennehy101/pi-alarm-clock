# go-chromecast (patched fork)

Patched copy of `github.com/vishen/go-chromecast` **v0.3.4**, wired in via a
`replace` directive in `server/go.mod`. v0.3.4 is the newest published
release and the project is no longer maintained, so the fixes below live
here.

Only the packages this server imports are kept (`application`, `cast`,
`dns`, `playlists`, `storage`); the CLI, UI, TTS and HTTP-server packages
were dropped, along with tests, mocks and testdata.

## Local changes against upstream v0.3.4

1. **`cast/connection.go` — `receiveLoop` no longer panics on teardown.**
   Upstream `Close()` closes `recvMsgChan` while the receive loop may
   still be delivering an in-flight message; the send then panics with
   `send on closed channel`, which crashes the whole server. Reproduced
   whenever a board load fails and the client is closed (library author's
   own `TODO: nothing here is concurrent safe, fix?`). The receive loop
   now recovers during teardown and drops the message.

2. **`application/application.go` — request timeout raised 5s → 15s**
   (`requestWaitTimeout` in `sendAndWait`). A Chromecast waking up (e.g.
   for an alarm board cast) regularly needs longer than 5 seconds to
   acknowledge a receiver LAUNCH, which surfaced as
   `unable to change to appID "CC1AD845": context deadline exceeded`.

## Applying upstream updates

If a new version is ever published: diff
`cast/connection.go` and `application/application.go` against upstream
first — both local changes are marked with `Pi Alarm Clock local patch`
comments and must be re-applied.
