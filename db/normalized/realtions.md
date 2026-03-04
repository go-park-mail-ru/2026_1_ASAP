# Описание отношений БД

```mermaid
erDiagram
    %% Профиль пользователя
    user {
        bigint id PK
        text username
        text email
        text password_hash
        timestampz created_at
        timestampz updated_at
    }

    %% Сущность чата
    chat {
        bigint id PK

    }

    %% Связка реализация списка контактов
    contact {

    }

    %% Стикеры
    sticker {

    }

    messsage {

    }

    %% Тип чата: группа, канал, личка
    chat_type {

    }

    %% Вложение
    attachments {

    }

    %% Отношение хранится в redis
    session {

    }
```