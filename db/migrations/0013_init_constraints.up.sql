-- CONTACT
ALTER TABLE contacts
ADD CONSTRAINT fk_contacts_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE contacts
ADD CONSTRAINT fk_contacts_contact_user FOREIGN KEY (contact_user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE contacts
ADD CONSTRAINT chk_contacts_user_not_null CHECK (user_id IS NOT NULL AND contact_user_id IS NOT NULL);

-- CHATS
ALTER TABLE chats
ADD CONSTRAINT fk_chats_type FOREIGN KEY (type) REFERENCES chat_types(type);
ALTER TABLE chats
ADD CONSTRAINT fk_chats_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE RESTRICT;
ALTER TABLE chats
ADD CONSTRAINT fk_chats_last_message FOREIGN KEY (last_message_id) REFERENCES messages(id) ON DELETE SET NULL;

-- CHAT_MEMBERS
ALTER TABLE chat_members
ADD CONSTRAINT fk_chat_members_chat FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE;
ALTER TABLE chat_members
ADD CONSTRAINT fk_chat_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE chat_members
ADD CONSTRAINT fk_chat_members_role FOREIGN KEY (role) REFERENCES chat_member_roles(role);
ALTER TABLE chat_members
ADD CONSTRAINT chk_chat_members_not_null CHECK (chat_id IS NOT NULL AND user_id IS NOT NULL AND role IS NOT NULL);

-- MESSAGES
ALTER TABLE messages
ADD CONSTRAINT fk_messages_chat FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE;
ALTER TABLE messages
ADD CONSTRAINT fk_messages_sender FOREIGN KEY (sender_id) REFERENCES users(id);
ALTER TABLE messages
ADD CONSTRAINT fk_messages_sticker FOREIGN KEY (sticker_id) REFERENCES stickers(id);
ALTER TABLE messages
ADD CONSTRAINT chk_messages_content CHECK (content IS NOT NULL OR sticker_id IS NOT NULL);

-- ATTACHMENTS
ALTER TABLE attachments
ADD CONSTRAINT fk_attachments_message FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE;
ALTER TABLE attachments
ADD CONSTRAINT chk_attachments_not_null CHECK (message_id IS NOT NULL AND file_url IS NOT NULL);

-- REACTIONS
ALTER TABLE reactions
ADD CONSTRAINT fk_reactions_message FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE;
ALTER TABLE reactions
ADD CONSTRAINT fk_reactions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE reactions
ADD CONSTRAINT uq_reactions UNIQUE (message_id, user_id, emoji);
ALTER TABLE reactions
ADD CONSTRAINT chk_reactions_not_null CHECK (message_id IS NOT NULL AND user_id IS NOT NULL AND emoji IS NOT NULL);

-- STICKERS
ALTER TABLE stickers
ADD CONSTRAINT fk_stickers_pack FOREIGN KEY (pack_id) REFERENCES sticker_packs(id) ON DELETE CASCADE;
ALTER TABLE stickers
ADD CONSTRAINT chk_stickers_not_null CHECK (pack_id IS NOT NULL AND file_url IS NOT NULL);

-- NOTIFICATIONS
ALTER TABLE notifications
ADD CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE notifications
ADD CONSTRAINT chk_notifications_not_null CHECK (user_id IS NOT NULL AND type IS NOT NULL);

-- LAST READ MESSAGE
ALTER TABLE chat_members
ADD CONSTRAINT fk_chat_members_last_read FOREIGN KEY (last_read_message_id) REFERENCES messages(id) ON DELETE SET NULL;