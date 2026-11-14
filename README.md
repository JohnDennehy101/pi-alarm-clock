Raspberry PI Alarm Clock

## Running locally

To simulate Raspberry Pi discovery (which will be enabled through DNS advertisement on service on the Pi, run the locally on Mac)

```
dns-sd -R "Pi Alarm Clock Test" _pialarmclock._tcp . 8080
```

## Deployment & Distribution

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
