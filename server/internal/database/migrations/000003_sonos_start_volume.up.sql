ALTER TABLE alarms
    ADD COLUMN sonos_start_volume INTEGER NOT NULL DEFAULT 0
        CHECK (sonos_start_volume BETWEEN 0 AND 100);
