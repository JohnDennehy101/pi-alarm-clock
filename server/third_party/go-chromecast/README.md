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

3. **`application/application.go` — detached external media loads wait for
   the receiver's acknowledgement** (`loadAndWait`, `loadWaitTimeout`).
   Upstream fires the media `LOAD` and returns immediately, so every
   receiver-side failure is invisible: a board refresh loop believed
   each frame loaded while the receiver had long since failed it or
   exited the app to the backdrop. A load now waits (10s) for the
   correlated `MEDIA_STATUS`; `LOAD_FAILED`/`LOAD_CANCELLED` and timeouts
   return errors so the caller can reconnect and re-launch. The
   `application_test.go` suite pins this with a fake connection.

4. **`application/application.go` — correlated responses never block the
   receive loop.** A receiver may answer one request with more than one
   requestId-correlated message; the extra delivery now drops instead of
   hanging `recvMessages` forever on a full result channel.

## Applying upstream updates

If a new version is ever published: diff
`cast/connection.go` and `application/application.go` against upstream
first — both local changes are marked with `Pi Alarm Clock local patch`
comments and must be re-applied.
