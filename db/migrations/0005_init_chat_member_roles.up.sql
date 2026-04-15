CREATE TABLE chat_member_roles (
    role TEXT PRIMARY KEY
);

INSERT INTO chat_member_roles(role)
VALUES
('member'),
('admin'),
('owner');