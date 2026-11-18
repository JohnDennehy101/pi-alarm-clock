ALTER TABLE alarms
    ADD COLUMN chromecast_enabled INTEGER NOT NULL DEFAULT 0
        CHECK (chromecast_enabled IN (0, 1));

ALTER TABLE alarms
    ADD COLUMN chromecast_device_id TEXT NOT NULL DEFAULT '';

ALTER TABLE alarms
    ADD COLUMN chromecast_display_offset INTEGER NOT NULL DEFAULT 0
        CHECK (chromecast_display_offset BETWEEN 0 AND 120);

ALTER TABLE alarms
    ADD COLUMN chromecast_display_duration INTEGER NOT NULL DEFAULT 60
        CHECK (chromecast_display_duration BETWEEN 0 AND 240);
