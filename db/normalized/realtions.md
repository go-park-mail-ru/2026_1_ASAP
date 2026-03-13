# Описание отношений БД

#### Таблица `user`
**Описание:** Хранит информацию о пользователях, включая учетные данные и профиль.

**Первичный ключ:**  
`{id}`

**Альтернативные ключи:**  
`{email}`, `{username}`

**Функциональные зависимости:**
```
{id} -> username, email, password_hash, avatar_url, bio, created_at, updated_at
{email} -> id, username, password_hash, avatar_url, bio, created_at, updated_at
{username} -> id, email, password_hash, avatar_url, bio, created_at, updated_at
```

#### Таблица `contact`
**Описание:** Хранит связи между пользователями (контакты).

**Первичный ключ:**  
```{(user_id, contact_user_id)}```

**Функциональные зависимости:**
```
{user_id, contact_user_id} -> created_at, updated_at
```


#### Таблица `chat_type`
**Описание:** Типы чатов.

**Первичный ключ:**  
```{type}```

**Функциональные зависимости:**
```{type} -> (нет дополнительных атрибутов)```

#### Таблица `chat`
**Описание:** Хранит информацию о чатах.

**Первичный ключ:**  
```{id}```

**Функциональные зависимости:**
```
{id} -> type, title, description, owner_id, avatar_url, created_at, updated_at
```

#### Таблица `chat_member_role`
**Описание:** Хранит роли участников чатов.

**Первичный ключ:**  
```{role}```

#### Таблица `chat_member`
**Описание:** Связь пользователей с чатами и их роли.

**Первичный ключ:**  
```{chat_id, user_id}```

**Функциональные зависимости:**
```
{chat_id, user_id} -> role, last_read_message_id, joined_at, created_at, updated_at
```

#### Таблица `message`
**Описание:** Хранит сообщения в чатах.

**Первичный ключ:**  
```{id}```

**Функциональные зависимости:**
```
{id} -> chat_id, sender_id, content, sticker_id, edited, created_at, updated_at, deleted_at
```

#### Таблица `attachment`
**Описание:** Вложения к сообщениям.

**Первичный ключ:**  
```{id}```

**Функциональные зависимости:**
```
{id} -> message_id, file_url, file_type, file_size, created_at, updated_at
```

#### Таблица `reaction`
**Описание:** Реакции пользователей на сообщения.

**Первичный ключ:**  
```{id}```

**Функциональные зависимости:**
```
{id} -> message_id, user_id, emoji, created_at, updated_at
```

#### Таблица `sticker_pack`
**Описание:** Наборы стикеров.

**Первичный ключ:**  
```{id}```

**Функциональные зависимости:**
```
{id} -> name, created_at, updated_at
```

#### Таблица `sticker`
**Описание:** Стикеры, принадлежащие наборам.

**Первичный ключ:**  
```{id}```

**Функциональные зависимости:**
```
{id} -> pack_id, file_url, created_at, updated_at
```

#### Таблица `notification`
**Описание:** Уведомления для пользователей.

**Первичный ключ:**  
```{id}```

**Функциональные зависимости:**
```
{id} -> user_id, type, entity_id, is_read, created_at, updated_at
```

#### Таблица `session (REDIS)`
**Описание:** Сессии пользователей для аутентификации. Хранятся в Redis в виде key-value, где ключ — session_id, значение — данные о пользователе и времени жизни сессии.

**Первичный ключ:**  
```{session_id}```

**Функциональные зависимости:**
```
{session_id} -> user_id, created_at, expires_at
```


```mermaid
erDiagram
    USER {
        uuid id PK
        text username
        text email
        text password_hash
        text avatar_url
        text bio
        timestamptz created_at
        timestamptz updated_at
    }

    CONTACT {
        uuid user_id PK, FK
        uuid contact_user_id PK, FK
        timestamptz created_at
        timestamptz updated_at
    }

    CHAT_TYPE {
        text type PK
    }

    CHAT {
        uuid id PK
        text type FK
        text title
        text description
        uuid owner_id FK
        text avatar_url
        timestamptz created_at
        timestamptz updated_at
    }

    CHAT_MEMBER_ROLE {
        text role PK
    }

    CHAT_MEMBER {
        uuid chat_id PK
        uuid user_id PK, FK
        text role FK
        bigint last_read_message_id FK
        timestamptz joined_at
        timestamptz created_at
        timestamptz updated_at
    }

    MESSAGE {
        bigint id PK
        uuid chat_id FK
        uuid sender_id FK
        text content
        uuid sticker_id FK
        boolean edited
        timestamptz created_at
        timestamptz updated_at
        timestamptz deleted_at
    }

    ATTACHMENT {
        bigint id PK
        bigint message_id FK
        text file_url
        text file_type
        bigint file_size
        timestamptz created_at
        timestamptz updated_at
    }

    REACTION {
        bigint id PK
        bigint message_id FK
        uuid user_id FK
        text emoji
        timestamptz created_at
        timestamptz updated_at
    }

    STICKER_PACK {
        uuid id PK
        text name
        timestamptz created_at
        timestamptz updated_at
    }

    STICKER {
        uuid id PK
        uuid pack_id FK
        text file_url
        timestamptz created_at
        timestamptz updated_at
    }

    NOTIFICATION {
        bigint id PK
        uuid user_id FK
        text type
        bigint entity_id
        boolean is_read
        timestamptz created_at
        timestamptz updated_at
    }

    SESSION {
        string session_id PK
        uuid user_id
        timestamptz created_at
        timestamptz expires_at
    }

    USER ||--o{ SESSION : has


    USER ||--o{ CONTACT : has
    USER ||--o{ CONTACT : added_contact

    USER ||--o{ CHAT_MEMBER : participates
    CHAT ||--o{ CHAT_MEMBER : contains
    CHAT_TYPE ||--o{ CHAT : defines
    CHAT_MEMBER_ROLE ||--o{ CHAT_MEMBER : defines_role

    CHAT ||--o{ MESSAGE : has
    USER ||--o{ MESSAGE : sends

    MESSAGE ||--o{ ATTACHMENT : contains
    MESSAGE ||--o{ REACTION : receives

    USER ||--o{ REACTION : reacts

    STICKER ||--o{ MESSAGE : used_in
    STICKER_PACK ||--o{ STICKER : contains

    USER ||--o{ NOTIFICATION : receives
```