Raspberry PI Alarm Clock

## Running locally

To simulate Raspberry Pi discovery (which will be enabled through DNS advertisement on service on the Pi, run the locally on Mac)

```
dns-sd -R "Pi Alarm Clock Test" _pialarmclock._tcp . 8080
```

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
