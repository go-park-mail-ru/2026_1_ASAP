# Описание отношений БД

#### Таблица `users`
**Описание:** Хранит информацию о пользователях, включая учетные данные и профиль.

**Первичный ключ:**  
`{id}`

**Альтернативные ключи:**  
`{email}`, `{login}`

**Функциональные зависимости:**
```
{id} -> login, first_name, last_name, email, password_hash, avatar_url, bio, birth_date, last_seen, created_at, updated_at
{email} -> id, login, first_name, last_name, password_hash, avatar_url, bio, birth_date, last_seen, created_at, updated_at
{login} -> id, email, first_name, last_name, password_hash, avatar_url, bio, birth_date, last_seen, created_at, updated_at
```

#### Таблица `contacts`
**Описание:** Хранит связи между пользователями (контакты). Пользователь может дать контакту своё имя (first_name, last_name), отличное от оригинального.

**Первичный ключ:**  
`{(user_id, contact_user_id)}`

**Функциональные зависимости:**
```
{user_id, contact_user_id} -> first_name, last_name, created_at, updated_at
```

#### Таблица `chat_types`
**Описание:** Справочник типов чатов (dialog, group, channel).

**Первичный ключ:**  
`{type}`

**Функциональные зависимости:**
```
{type} -> (нет дополнительных атрибутов)
```

#### Таблица `chat_member_roles`
**Описание:** Справочник ролей участников чата (member, admin, owner).

**Первичный ключ:**  
`{role}`

**Функциональные зависимости:**
{role} -> (нет дополнительных атрибутов)


#### Таблица `chats`
**Описание:** Хранит информацию о чатах.

**Первичный ключ:**  
`{id}`

**Функциональные зависимости:**
```
{id} -> type, title, description, owner_id, avatar_url, last_message_id, created_at, updated_at
```

#### Таблица `chat_members`
**Описание:** Связь пользователей с чатами и их роли.

**Первичный ключ:**  
`{chat_id, user_id}`

**Функциональные зависимости:**
```
{chat_id, user_id} -> role, last_read_message_id, joined_at, created_at, updated_at
```

#### Таблица `messages`
**Описание:** Хранит сообщения в чатах.

**Первичный ключ:**  
`{id}`

**Функциональные зависимости:**
```
{id} -> chat_id, sender_id, content, sticker_id, edited, created_at, updated_at, deleted_at
```

#### Таблица `attachments`
**Описание:** Вложения к сообщениям.

**Первичный ключ:**  
`{id}`

**Функциональные зависимости:**
```
{id} -> message_id, file_url, file_type, file_size, created_at, updated_at
```

#### Таблица `reactions`
**Описание:** Реакции пользователей на сообщения.

**Первичный ключ:**  
`{id}`

**Альтернативные ключи:**  
`{message_id, user_id, emoji}`

**Функциональные зависимости:**
```
{id} -> message_id, user_id, emoji, created_at, updated_at
```

#### Таблица `sticker_packs`
**Описание:** Наборы стикеров.

**Первичный ключ:**  
`{id}`

**Функциональные зависимости:**
```
{id} -> name, created_at, updated_at
```

#### Таблица `stickers`
**Описание:** Стикеры, принадлежащие наборам.

**Первичный ключ:**  
`{id}`

**Функциональные зависимости:**
```
{id} -> pack_id, file_url, created_at, updated_at
```

#### Таблица `notifications`
**Описание:** Уведомления для пользователей.

**Первичный ключ:**  
`{id}`

**Функциональные зависимости:**
```
{id} -> user_id, type, entity_id, is_read, created_at, updated_at
```

#### `session (REDIS)`
**Описание:** Сессии пользователей для аутентификации. Хранятся в Redis в виде key-value, где ключ — session_id, значение — данные о пользователе и времени жизни сессии.

**Первичный ключ:**  
`{session_id}`

**Функциональные зависимости:**
```
{session_id} -> user_id, created_at, expires_at
```

#### `user_status (REDIS)`
**Описание:** Хранит текущий статус пользователя и время последней активности.

**Первичный ключ:**  
`{user_id}`

**Функциональные зависимости:**
```
{user_id} -> status, last_active
```

```mermaid
erDiagram
    USERS {
        bigint id PK
        text login UK
        text first_name
        text last_name
        text email UK
        text password_hash
        text avatar_url
        text bio
        date birth_date
        timestamptz last_seen
        timestamptz created_at
        timestamptz updated_at
    }

    CONTACTS {
        bigint user_id PK,FK
        bigint contact_user_id PK,FK
        text first_name
        text last_name
        timestamptz created_at
        timestamptz updated_at
    }

    CHAT_TYPES {
        text type PK
    }

    CHAT_MEMBER_ROLES {
        text role PK
    }

    CHATS {
        bigint id PK
        text type FK
        text title
        text description
        bigint owner_id FK
        text avatar_url
        bigint last_message_id FK
        timestamptz created_at
        timestamptz updated_at
    }

    CHAT_MEMBERS {
        bigint chat_id PK,FK
        bigint user_id PK,FK
        text role FK
        bigint last_read_message_id FK
        timestamptz joined_at
        timestamptz created_at
        timestamptz updated_at
    }

    MESSAGES {
        bigint id PK
        bigint chat_id FK
        bigint sender_id FK
        text content
        bigint sticker_id FK
        boolean edited
        timestamptz created_at
        timestamptz updated_at
        timestamptz deleted_at
    }

    ATTACHMENTS {
        bigint id PK
        bigint message_id FK
        text file_url
        text file_type
        bigint file_size
        timestamptz created_at
        timestamptz updated_at
    }

    REACTIONS {
        bigint id PK
        bigint message_id FK
        bigint user_id FK
        text emoji
        timestamptz created_at
        timestamptz updated_at
    }

    STICKER_PACKS {
        bigint id PK
        text name
        timestamptz created_at
        timestamptz updated_at
    }

    STICKERS {
        bigint id PK
        bigint pack_id FK
        text file_url
        timestamptz created_at
        timestamptz updated_at
    }

    NOTIFICATIONS {
        bigint id PK
        bigint user_id FK
        text type
        bigint entity_id
        boolean is_read
        timestamptz created_at
        timestamptz updated_at
    }

    SESSION {
        string session_id PK
        bigint user_id
        timestamptz created_at
        timestamptz expires_at
    }

    USER_STATUS {
        bigint user_id PK
        text status
        timestamptz last_active
    }

    USERS ||--o{ SESSION : has
    USERS ||--o{ USER_STATUS : has
    USERS ||--o{ CONTACTS : "owns (user_id)"
    USERS ||--o{ CONTACTS : "is_contact (contact_user_id)"
    USERS ||--o{ CHAT_MEMBERS : participates
    USERS ||--o{ MESSAGES : sends
    USERS ||--o{ REACTIONS : reacts
    USERS ||--o{ NOTIFICATIONS : receives

    CHATS ||--|| CHAT_TYPES : "has type"
    CHATS ||--o{ CHAT_MEMBERS : contains
    CHATS ||--o{ MESSAGES : has
    CHATS ||--o| USERS : "owned by (owner_id)"

    CHAT_MEMBERS ||--|| CHAT_MEMBER_ROLES : "has role"
    CHAT_MEMBERS ||--o| MESSAGES : "last read"

    MESSAGES ||--o{ ATTACHMENTS : contains
    MESSAGES ||--o{ REACTIONS : receives
    MESSAGES ||--o| STICKERS : "uses sticker"

    STICKER_PACKS ||--o{ STICKERS : contains
