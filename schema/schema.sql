-- categories of work that can be recorded.
CREATE TABLE IF NOT EXISTS categories (
                                          id   INTEGER PRIMARY KEY,
                                          name VARCHAR(50) NOT NULL
);

-- records of time spent.
CREATE TABLE IF NOT EXISTS records (
                                       id       INTEGER PRIMARY KEY,
                                       start    VARCHAR(35) NOT NULL,
                                       end      VARCHAR(35),
                                       duration VARCHAR(35),
                                       category UNSIGNED INTEGER NOT NULL REFERENCES categories(id),
                                       notes    TEXT
);


-- global configuration, things like default category, default view.
CREATE TABLE IF NOT EXISTS configuration (
                                             key VARCHAR(50) PRIMARY KEY,
                                             value VARCHAR(50) NOT NULL
);

-- Add the default data.
INSERT INTO categories (id, name) VALUES
                                      (1, 'Uncategorized'),
                                      (2, 'Work'),
                                      (3, 'Personal')
ON CONFLICT DO NOTHING;

-- Insert default configuration values
INSERT INTO configuration (key, value) VALUES ('default_category', '2');