CREATE TABLE chat_types (
    type TEXT PRIMARY KEY
);

INSERT INTO chat_types(type)
VALUES
('private'),
('group'),
('channel');