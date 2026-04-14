package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
	chatssql "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/chat/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/sqllog"
	"github.com/jackc/pgx/v5/pgconn"
)

type dbPool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Close()
}

type ChatRepository struct {
	db     dbPool
	logger *zap.Logger
}

func NewChatRepository(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*ChatRepository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return &ChatRepository{db: pool, logger: logger}, nil
}

func (r *ChatRepository) GetAllChatsByUserID(ctx context.Context, id int64) ([]*domain.Chat, error) {
	q := chatssql.GetAllChatsByUserID
	start := time.Now()
	rows, err := r.db.Query(ctx, q, id)
	sqllog.LogQuery(ctx, r.log(ctx), "GetAllChatsByUserID", q, start, err, []any{id})
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
	q := chatssql.GetChatByID
	start := time.Now()
	row := r.db.QueryRow(ctx, q, chatID)

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
	sqllog.LogQuery(ctx, r.log(ctx), "GetChatByID", q, start, err, []any{chatID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrChatNotFound
		}

		return nil, fmt.Errorf("failed to get chat by ID: %w", err)
	}

	return toDomainChat(chatModel), nil
}

func (r *ChatRepository) CreateChat(ctx context.Context, chat *domain.Chat) (*domain.Chat, error) {
	chatModel := toModelChat(chat)
	q := chatssql.InsertChat
	start := time.Now()
	err := r.db.QueryRow(ctx, q,
		string(chatModel.Type), chatModel.Title, chatModel.Description, chatModel.OwnerId, chatModel.AvatarUrl,
	).Scan(&chatModel.Id, &chatModel.CreatedAt, &chatModel.UpdatedAt)
	sqllog.LogQuery(ctx, r.log(ctx), "CreateChat", q, start, err, []any{
		string(chatModel.Type), chatModel.Title, chatModel.Description, chatModel.OwnerId, chatModel.AvatarUrl,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	return toDomainChat(chatModel), nil
}

func (r *ChatRepository) GetLastMessageOfChat(ctx context.Context, chatID int64) (*domain.Message, error) {
	q := chatssql.GetLastMessageOfChat
	start := time.Now()
	row := r.db.QueryRow(ctx, q, chatID)

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
	sqllog.LogQuery(ctx, r.log(ctx), "GetLastMessageOfChat", q, start, err, []any{chatID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return toDomainMessage(msg), nil
		}
		return nil, fmt.Errorf("failed to get last message: %w", err)
	}

	return toDomainMessage(msg), nil
}

func (r *ChatRepository) GetLastMessagesOfChats(ctx context.Context, id int64) ([]*domain.Message, error) {
	q := chatssql.GetLastMessagesOfChats
	start := time.Now()
	rows, err := r.db.Query(ctx, q, id)
	sqllog.LogQuery(ctx, r.log(ctx), "GetLastMessagesOfChats", q, start, err, []any{id})
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
	q := chatssql.GetChatMembers
	start := time.Now()
	rows, err := r.db.Query(ctx, q, chatID)
	sqllog.LogQuery(ctx, r.log(ctx), "GetChatMembers", q, start, err, []any{chatID})
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
	q := chatssql.InsertChatMember
	start := time.Now()
	_, err := r.db.Exec(ctx, q, chatID, userID, role)
	sqllog.LogQuery(ctx, r.log(ctx), "AddMember", q, start, err, []any{chatID, userID, role})
	if err != nil {
		return fmt.Errorf("failed to insert chat member: %w", err)
	}

	return nil
}

func (r *ChatRepository) IsMember(ctx context.Context, chatID, userID int64) (bool, error) {
	var exists bool
	q := chatssql.IsMember
	start := time.Now()
	err := r.db.QueryRow(ctx, q, chatID, userID).Scan(&exists)
	sqllog.LogQuery(ctx, r.log(ctx), "IsMember", q, start, err, []any{chatID, userID})
	if err != nil {
		return false, fmt.Errorf("failed to check if member exists: %w", err)
	}

	return exists, nil
}

func (r *ChatRepository) GetDialogBetweenUsers(ctx context.Context, user1ID, user2ID int64) (*domain.Chat, error) {
	q := chatssql.GetDialogBetweenUsers
	start := time.Now()
	row := r.db.QueryRow(ctx, q, user1ID, user2ID)

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
	sqllog.LogQuery(ctx, r.log(ctx), "GetDialogBetweenUsers", q, start, err, []any{user1ID, user2ID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get dialog between users: %w", err)
	}

	return toDomainChat(chatModel), nil
}

func (r *ChatRepository) DeleteChat(ctx context.Context, chatID int64) error {
	trx, err := r.db.BeginTx(ctx, pgx.TxOptions{
        IsoLevel: pgx.RepeatableRead,
    })
    if err != nil {
        return fmt.Errorf("failed to create transaction: %w", err)
    }
    defer trx.Rollback(ctx)

	q := chatssql.DeleteMessagesByChatID
	start := time.Now()
	_, err = trx.Exec(ctx, q, chatID)
	sqllog.LogQuery(ctx, r.log(ctx), "DeleteChat.messages", q, start, err, []any{chatID})
	if err != nil {
		return fmt.Errorf("failed to delete messages from chat: %w", err)
	}

	q = chatssql.DeleteChatMembersByChatID
	start = time.Now()
	_, err = trx.Exec(ctx, q, chatID)
	sqllog.LogQuery(ctx, r.log(ctx), "DeleteChat.members", q, start, err, []any{chatID})
	if err != nil {
		return fmt.Errorf("failed to delete members from chat: %w", err)
	}

	q = chatssql.DeleteChatByID
	start = time.Now()
	_, err = trx.Exec(ctx, q, chatID)
	sqllog.LogQuery(ctx, r.log(ctx), "DeleteChat.chat", q, start, err, []any{chatID})
	if err != nil {
		return fmt.Errorf("failed to delete chat: %w", err)
	}

	start = time.Now()
	err = trx.Commit(ctx)
	sqllog.LogQuery(ctx, r.log(ctx), "DeleteChat.commit", "COMMIT", start, err, []any{chatID})
	if err != nil {
		return fmt.Errorf("failed to commit delete chat: %w", err)
	}
	return nil
}

func (r *ChatRepository) GetMemberRole(ctx context.Context, userID, chatID int64) (string, error) {
	var role string
	q := chatssql.GetMemberRole
	start := time.Now()
	err := r.db.QueryRow(ctx, q, chatID, userID).Scan(&role)
	sqllog.LogQuery(ctx, r.log(ctx), "GetMemberRole", q, start, err, []any{chatID, userID})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

func (r *ChatRepository) UploadAvatarUrl(ctx context.Context, chatID int64, avatarURL string) (*domain.Chat, error) {
	q := chatssql.UpdateChatAvatarURL
	start := time.Now()
	row := r.db.QueryRow(ctx, q, chatID, avatarURL)

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
	sqllog.LogQuery(ctx, r.log(ctx), "UploadAvatarUrl", q, start, err, []any{chatID, avatarURL})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrChatNotFound
		}
		return nil, fmt.Errorf("chatRepo failed upload avatar url: %w", err)
	}
	return toDomainChat(chatModel), nil
}

func (r *ChatRepository) UpdateTitle(ctx context.Context, chatID int64, title string) (*domain.Chat, error) {
	q := chatssql.UpdateChatTitle
	start := time.Now()
	row := r.db.QueryRow(ctx, q, chatID, title)

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
	sqllog.LogQuery(ctx, r.log(ctx), "UpdateTitle", q, start, err, []any{chatID, title})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrChatNotFound
		}
		return nil, fmt.Errorf("chatRepo failed update title: %w", err)
	}
	return toDomainChat(chatModel), nil
}

func (r *ChatRepository) DeleteMember(ctx context.Context, chatID, userID int64) error {
	q := chatssql.DeleteChatMember
	start := time.Now()
	result, err := r.db.Exec(ctx, q, chatID, userID)
	sqllog.LogQuery(ctx, r.log(ctx), "DeleteMember", q, start, err, []any{chatID, userID})
	if err != nil {
		return fmt.Errorf("failed to delete members from chat: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrMemberNotFound
	}

	return nil
}

func (r *ChatRepository) Close() {
	r.db.Close()
}

func (r *ChatRepository) log(ctx context.Context) *zap.Logger {
	base := r.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}
