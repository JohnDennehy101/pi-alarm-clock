CREATE TABLE alarms (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    time TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    days TEXT NOT NULL DEFAULT '[]',
    date TEXT,

    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,

    source_type TEXT NOT NULL,
    source_uri TEXT,
    source_name TEXT,

    volume INTEGER NOT NULL DEFAULT 50,
    gradual_volume INTEGER NOT NULL DEFAULT 0,
    volume_duration INTEGER NOT NULL DEFAULT 10,

    hue_enabled INTEGER NOT NULL DEFAULT 0,
    hue_light_ids TEXT NOT NULL DEFAULT '[]',
    hue_start_brightness INTEGER NOT NULL DEFAULT 5,
    hue_brightness INTEGER NOT NULL DEFAULT 100,
    hue_gradual_brightness INTEGER NOT NULL DEFAULT 0,
    hue_brightness_duration INTEGER NOT NULL DEFAULT 20,
    hue_colour_temperature INTEGER,

    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,

    CHECK (enabled IN (0, 1)),
    CHECK (gradual_volume IN (0, 1)),
    CHECK (volume BETWEEN 0 AND 100),

    CHECK (hue_enabled IN (0, 1)),
    CHECK (hue_gradual_brightness IN (0, 1)),
    CHECK (hue_start_brightness BETWEEN 0 AND 100),
    CHECK (hue_brightness BETWEEN 0 AND 100),
    CHECK (hue_brightness_duration >= 0),
    CHECK (
        hue_colour_temperature IS NULL
        OR hue_colour_temperature BETWEEN 2000 AND 6500
    )
);

CREATE INDEX idx_alarms_enabled ON alarms(enabled);