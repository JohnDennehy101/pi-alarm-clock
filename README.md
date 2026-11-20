Raspberry PI Alarm Clock

## Running locally

To simulate Raspberry Pi discovery (which will be enabled through DNS advertisement on service on the Pi, run the locally on Mac)

```
dns-sd -R "Pi Alarm Clock Test" _pialarmclock._tcp . 8080
```

### macOS firewall

The macOS application firewall prompts per binary, and `go run` compiles
to a new path on every start — so each restart loses the exception and
LAN devices (Sonos, Chromecast) silently fail to reach the served
board/audio files. Build to a fixed path once and allow it:

```bash
go build -o bin/pi-alarm-api ./cmd/api
./bin/pi-alarm-api
```

When macOS asks, click "Allow" (System Settings > Network > Firewall
shows the list afterwards). On the Pi deployment target there is no
macOS firewall.

## Board preview (dev)

To test dashboard/board PNG rendering without casting to a Chromecast
device, run the preview command. It writes one PNG per minute — the
same sequence the alarm flow would cast — to `data/boards/<session>/`:
each minute reflects the `BOARD_SCHEDULE` rotation (timetable, bus
position map, per-bus tracking progress, weather), so bus due times
count down, map positions advance, and the progress view's bus rotates
across the approaching fleet:

```bash
cd server
go run ./cmd/dashboard/preview -minutes 25        # optional: -stop <number>
```

It loads `.env` and the bus timetable exactly like the API. The first
run parses the GTFS dataset and writes the binary timetable cache
(`data/bus/timetable.cache`); later runs mmap that cache and start in
under a second. By design it never contacts any Chromecast device.

### Board schedule

`BOARD_SCHEDULE` rotates panels on the alarm display. A bus panel's
time can be split between views with a nested group: the timetable
view (`schedule`), the live bus position map (`map`), and the per-bus
tracking progress view (`progress`):

```
BOARD_SCHEDULE="bus:{map:2,schedule:2,progress:3},weather:1"
BOARD_SCHEDULE="bus:5,weather:1"                   # shorthand: all bus time on the timetable
```

The map view shows buses approaching the configured stop within ~15
minutes, drawn against the GTFS route shapes: the current position, a
dashed ghost marker for the previous minute's position, a minutes-due
label per bus, and the departure stop as a hollow square labelled with
its stop number. Buses are coloured by punctuality: green on time or
early, red late, and neutral white when only scheduled data exists.

The progress view tracks one bus at a time, rotating across buses due
within ~25 minutes: a route progress line showing how far along the
route the bus has travelled toward your stop (with its previous-minute
position), the expected vs scheduled arrival, and its realtime
lateness — same green/red semantics as the map.

The map's basemap background uses the Mapbox Static Images API, which
needs a (free) access token and is cached to `data/basemap` for six
hours, so board renders stay far below provider rate limits:

```
MAPBOX_TOKEN=pk....
```

Without a token the map panel still renders positions on a plain dark
background.

### Memory footprint

The GTFS loader streams `stop_times.txt` and `shapes.txt` (the two
multi-million-row files) record by record instead of loading them
wholly, and each stop-time row persists as an eight-byte trip
reference. The parsed result is then cached to
`data/bus/timetable.cache` (one ~110 MB file), keyed by the sha256 of
the GTFS zip, the schema version and the focus stop list. Subsequent
boots mmap the cache — departures stay in the kernel page cache, not
the Go heap: a warm start loads in under a second at around 150 MB
resident, with a cold start peaking around 350 MB (and roughly half a
minute of Pi CPU) only when the dataset changes. Only trips serving
the configured `BUS_STOP_NUMBER` keep their route shapes and stop
sequences beyond that.

## Deployment & Distribution

The API is deployed to the Pi as a Docker image, built on the Mac with buildx
(target `linux/arm64` for a 64-bit Pi OS). The container runs with host
networking so that Sonos/Hue UPnP discovery works and the API reports the Pi's
real LAN IP for the audio server.

```bash
# One-off: copy the deploy target config (deploy.env is gitignored)
cd server/deploy
cp deploy.env.example deploy.env   # then set PI_SSH in it

./deploy.sh
```

The Pi address is not hardcoded in the repo: `deploy.env` is gitignored and
holds the default `PI_SSH`; exported variables (`PI_SSH=... ./deploy.sh`)
always take precedence, along with `PI_DIR` and `PLATFORM`.

### SSH access

The script opens several SSH/SCP connections per deploy, but they are
multiplexed over one shared connection — so with password auth you are
prompted at most once per deploy.

For passwordless deploys, copy your key to the Pi once:

```bash
ssh-copy-id <user>@<pi-ip>
```

The script builds the image, ships it plus the compose file and `.env` to the
Pi via SSH, loads it, and recreates the container with
`docker compose up -d --force-recreate`. State (SQLite DB, TLS cert, paired
devices) persists in `~/pi-alarm/data` on the Pi.

Prerequisites on the Pi: 64-bit OS (aarch64), Docker with the compose plugin,
and key-based SSH access from the Mac.

### Verifying the deployment & pairing

SSH into the Pi and check the container logs for the pairing PIN:

```bash
ssh <user>@<pi-ip>            # enter the Pi password

# then on the Pi:
cd ~/pi-alarm
docker compose logs api | grep -A2 "Pairing PIN"
```

A new PIN is printed at every container start, so if there are multiple
matches, the last one is the current one. Paired devices are stored in
`~/pi-alarm/data/devices.json` on the Pi, so they keep working across
redeploys — the PIN is only needed to pair a new device.

To pair a device and get a bearer token (from the Mac):

```bash
curl -sk -X POST https://<pi-ip>:8080/api/v1/auth/pair \
  -H 'Content-Type: application/json' \
  -d '{"pin":"<PIN>"}'

# smoke test with the returned token:
curl -sk https://<pi-ip>:8080/api/v1/alarms \
  -H "Authorization: Bearer <token>"
```

### Timezone

Alarms are scheduled in the container's local time (`TZ` in
`server/deploy/docker-compose.yml`). Set it to your wall-clock zone.

Once configured on the Pi, make sure to enable DNS advertisement from the Pi so that the Flutter app can discover the IP address set to the Pi.

For example:

Using Avahi

```
/etc/avahi/services/pialarmclock.service
<?xml version="1.0" standalone='no'?>
<!DOCTYPE service-group SYSTEM "avahi-service.dtd">

<service-group>
    <name replace-wildcards="yes">Pi Alarm Clock</name>

    <service>
        <type>_pialarmclock._tcp</type>
        <port>8080</port>
    </service>
</service-group>
```

And then restart Avahi on the Pi

```
sudo systemctl restart avahi-daemon
```
