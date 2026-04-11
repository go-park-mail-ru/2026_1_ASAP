-- CONTACT
ALTER TABLE contacts DROP CONSTRAINT fk_contacts_user;
ALTER TABLE contacts DROP CONSTRAINT fk_contacts_contact_user;
ALTER TABLE contacts DROP CONSTRAINT chk_contacts_user_not_null;

-- CHATS
ALTER TABLE chats DROP CONSTRAINT fk_chats_type;
ALTER TABLE chats DROP CONSTRAINT fk_chats_owner;

-- CHAT_MEMBERS
ALTER TABLE chat_members DROP CONSTRAINT fk_chat_members_chat;
ALTER TABLE chat_members DROP CONSTRAINT fk_chat_members_user;
ALTER TABLE chat_members DROP CONSTRAINT fk_chat_members_role;
ALTER TABLE chat_members DROP CONSTRAINT chk_chat_members_not_null;

-- MESSAGES
ALTER TABLE messages DROP CONSTRAINT fk_messages_chat;
ALTER TABLE messages DROP CONSTRAINT fk_messages_sender;
ALTER TABLE messages DROP CONSTRAINT fk_messages_sticker;
ALTER TABLE messages DROP CONSTRAINT chk_messages_content;

-- ATTACHMENTS
ALTER TABLE attachments DROP CONSTRAINT fk_attachments_message;
ALTER TABLE attachments DROP CONSTRAINT chk_attachments_not_null;

-- REACTIONS
ALTER TABLE reactions DROP CONSTRAINT fk_reactions_message;
ALTER TABLE reactions DROP CONSTRAINT fk_reactions_user;
ALTER TABLE reactions DROP CONSTRAINT uq_reactions;
ALTER TABLE reactions DROP CONSTRAINT chk_reactions_not_null;

-- STICKERS
ALTER TABLE stickers DROP CONSTRAINT fk_stickers_pack;
ALTER TABLE stickers DROP CONSTRAINT chk_stickers_not_null;

-- NOTIFICATIONS
ALTER TABLE notifications DROP CONSTRAINT fk_notifications_user;
ALTER TABLE notifications DROP CONSTRAINT chk_notifications_not_null;

-- LAST READ MESSAGE
ALTER TABLE chat_members DROP CONSTRAINT fk_chat_members_last_read;