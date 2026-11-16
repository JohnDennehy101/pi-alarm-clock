ALTER TABLE alarms
    ADD COLUMN sonos_start_offset INTEGER NOT NULL DEFAULT 0
        CHECK (sonos_start_offset BETWEEN 0 AND 120);

ALTER TABLE alarms
    ADD COLUMN hue_start_offset INTEGER NOT NULL DEFAULT 0
        CHECK (hue_start_offset BETWEEN 0 AND 120);
