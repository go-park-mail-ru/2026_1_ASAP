package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
)

type ChatRepository struct {
	db *pgxpool.Pool
}

func NewChatRepository(ctx context.Context, cfg config.PostgresConfig) (*ChatRepository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return &ChatRepository{db: pool}, nil
}

func (r *ChatRepository) GetAllChatsByUserID(ctx context.Context, id int64) ([]*domain.Chat, error) {
	rows, err := r.db.Query(ctx,
		`SELECT c.id, c.type, c.title, c.description, c.owner_id, c.avatar_url, c.created_at, c.updated_at
	 FROM chats c
	 INNER JOIN chat_members cm ON c.id = cm.chat_id
	 WHERE cm.user_id=$1`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get all chats by userID: %w", err)
	}
	defer rows.Close()

	var chats []*domain.Chat
	for rows.Next() {
		chatModel := &ChatModel{}
		err := rows.Scan(
			&chatModel.Id,
			&chatModel.Type,
			&chatModel.Title,
			&chatModel.Description,
			&chatModel.OwnerId,
			&chatModel.AvatarUrl,
			&chatModel.CreatedAt,
			&chatModel.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rows: %w", err)
		}

		chat := toDomainChat(chatModel)
		chats = append(chats, chat)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return chats, nil
}

func (r *ChatRepository) GetChatByID(ctx context.Context, chatID int64) (*domain.Chat, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, type, title, description, owner_id, avatar_url, created_at, updated_at
	 FROM chats
	 WHERE id=$1`, chatID)

	chatModel := &ChatModel{}
	if err := row.Scan(
		&chatModel.Id,
		&chatModel.Type,
		&chatModel.Title,
		&chatModel.Description,
		&chatModel.OwnerId,
		&chatModel.AvatarUrl,
		&chatModel.CreatedAt,
		&chatModel.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrChatNotFound
		}

		return nil, fmt.Errorf("failed to get chat by ID: %w", err)
	}

	return toDomainChat(chatModel), nil
}

func (r *ChatRepository) CreateChat(ctx context.Context, chat *domain.Chat) (*domain.Chat, error) {
	chatModel := toModelChat(chat)
	err := r.db.QueryRow(ctx,
		`INSERT INTO chats
	 (type, title, description, owner_id, avatar_url, created_at, updated_at)
	 VALUES ($1, $2, $3, $4, $5, $6, $7)
	 RETURNING id`,
		chatModel.Type, chatModel.Title, chatModel.Description, chatModel.OwnerId, chatModel.AvatarUrl, chatModel.CreatedAt, chatModel.UpdatedAt).Scan(&chatModel.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	return toDomainChat(chatModel), nil
}

func (r *ChatRepository) GetLastMessageOfChat(ctx context.Context, chatID int64) (*domain.Message, error) {
	row := r.db.QueryRow(ctx,
		`SELECT m.id, m.chat_id, m.sender_id, m.content, m.sticker_id, m.edited, m.created_at, m.updated_at, m.deleted_at
	 FROM messages m
	 JOIN chats c ON m.id = c.last_message_id
	 WHERE c.id=$1`, chatID)

	msg := &MessageModel{}
	err := row.Scan(
		&msg.Id,
		&msg.ChatId,
		&msg.SenderId,
		&msg.Content,
		&msg.StickerId,
		&msg.Edited,
		&msg.CreatedAt,
		&msg.UpdatedAt,
		&msg.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return toDomainMessage(msg), nil
		}
		return nil, fmt.Errorf("failed to get last message: %w", err)
	}

	return toDomainMessage(msg), nil
}

func (r *ChatRepository) GetLastMessagesOfChats(ctx context.Context, id int64) ([]*domain.Message, error) {
	rows, err := r.db.Query(ctx,
		`SELECT m.id, m.chat_id, m.sender_id, m.content, m.sticker_id, m.edited, m.created_at, m.updated_at, m.deleted_at
	 FROM messages m
	 JOIN chats c ON m.id = c.last_message_id
	 JOIN chat_members cm ON c.id = cm.chat_id
	 WHERE cm.user_id=$1`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get last messages: %w", err)
	}
	defer rows.Close()

	var lastMessages []*domain.Message
	for rows.Next() {
		lastMessageModel := &MessageModel{}
		err := rows.Scan(
			&lastMessageModel.Id,
			&lastMessageModel.ChatId,
			&lastMessageModel.SenderId,
			&lastMessageModel.Content,
			&lastMessageModel.StickerId,
			&lastMessageModel.Edited,
			&lastMessageModel.CreatedAt,
			&lastMessageModel.UpdatedAt,
			&lastMessageModel.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rows: %w", err)
		}

		lastMessage := toDomainMessage(lastMessageModel)
		lastMessages = append(lastMessages, lastMessage)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return lastMessages, nil
}

func (r *ChatRepository) GetChatMembers(ctx context.Context, chatID int64) ([]int64, error) {
	rows, err := r.db.Query(ctx,
		`SELECT user_id
	 FROM chat_members
	 WHERE chat_id=$1`, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat members: %w", err)
	}
	defer rows.Close()

	var members []int64
	for rows.Next() {
		var userID int64
		err := rows.Scan(
			&userID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, userID)
	}
	return members, nil
}

func (r *ChatRepository) AddMember(ctx context.Context, chatID, userID int64, role string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO chat_members
	 (chat_id, user_id, role, joined_at)
	 VALUES ($1, $2, $3, $4)`, chatID, userID, role, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert chat member: %w", err)
	}

	return nil
}

func (r *ChatRepository) IsMember(ctx context.Context, chatID, userID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM chat_members WHERE chat_id=$1 AND user_id=$2)`, chatID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if member exists: %w", err)
	}

	return exists, nil
}

func (r *ChatRepository) GetDialogBetweenUsers(ctx context.Context, user1ID, user2ID int64) (*domain.Chat, error) {
	row := r.db.QueryRow(ctx,
		`SELECT c.id, c.type, c.title, c.description, c.owner_id, c.avatar_url, c.created_at, c.updated_at
	 FROM chats c
	 JOIN chat_members cm1 ON c.id = cm1.chat_id
	 JOIN chat_members cm2 ON c.id = cm2.chat_id
	 WHERE c.type = 'dialog' 
  	 AND cm1.user_id IN ($1, $2) 
  	 AND cm2.user_id IN ($1, $2) 
  	 AND cm1.user_id != cm2.user_id
	 LIMIT 1`, user1ID, user2ID)

	chatModel := &ChatModel{}
	err := row.Scan(
		&chatModel.Id,
		&chatModel.Type,
		&chatModel.Title,
		&chatModel.Description,
		&chatModel.OwnerId,
		&chatModel.AvatarUrl,
		&chatModel.CreatedAt,
		&chatModel.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get dialog between users: %w", err)
	}

	return toDomainChat(chatModel), nil
}

func (r *ChatRepository) DeleteChat(ctx context.Context, chatID int64) error {
	trx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	defer trx.Rollback(ctx)

	_, err = trx.Exec(ctx,
		`DELETE FROM messages WHERE chat_id=$1`, chatID)
	if err != nil {
		return fmt.Errorf("failed to delete messages from chat: %w", err)
	}

	_, err = trx.Exec(ctx,
		`DELETE FROM chat_members WHERE chat_id=$1`, chatID)
	if err != nil {
		return fmt.Errorf("failed to delete members from chat: %w", err)
	}

	_, err = trx.Exec(ctx,
		`DELETE FROM chats WHERE id=$1`, chatID)
	if err != nil {
		return fmt.Errorf("failed to delete chat: %w", err)
	}

	return trx.Commit(ctx)
}

func (r *ChatRepository) GetMemberRole(ctx context.Context, userID, chatID int64) (string, error) {
	var role string
	err := r.db.QueryRow(ctx,
		`SELECT role
	 FROM chat_members
	 WHERE chat_id=$1 AND user_id=$2`, chatID, userID).Scan(&role)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

func (r *ChatRepository) Close() {
	r.db.Close()
}
