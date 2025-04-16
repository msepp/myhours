-- categories of work that can be recorded.
CREATE TABLE IF NOT EXISTS categories (
    id   INTEGER PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    color_dark_fg VARCHAR(7) NOT NULL DEFAULT '#fff',
    color_dark_bg VARCHAR(7) NOT NULL DEFAULT '#000',
    color_light_fg VARCHAR(7) NOT NULL DEFAULT '#000',
    color_light_bg VARCHAR(7) NOT NULL DEFAULT '#fff'
);

-- records of time spent.
CREATE TABLE IF NOT EXISTS records (
    id       INTEGER PRIMARY KEY,
    start    VARCHAR(35)      NOT NULL,
    end      VARCHAR(35),
    duration VARCHAR(35),
    category UNSIGNED INTEGER NOT NULL REFERENCES categories (id),
    notes    TEXT
);


-- global configuration, things like default category, default view.
CREATE TABLE IF NOT EXISTS configuration (
    key   VARCHAR(50) PRIMARY KEY,
    value VARCHAR(50) NOT NULL
);

-- Add the default data.
INSERT INTO categories
(id, name, color_dark_bg, color_dark_fg, color_light_bg, color_light_fg)
VALUES
    (1, 'Uncategorized', '#000', '#fff', '#fff', '#000'),
    (2, 'Work',         '#0c0', '#fff', '#fff', '#0c0'),
    (3, 'Personal',     '#c00', '#fff', '#fff', '#c00')
ON CONFLICT DO NOTHING;

-- Insert default configuration values
INSERT INTO configuration
(key, value)
VALUES
    ('default_category', '2');