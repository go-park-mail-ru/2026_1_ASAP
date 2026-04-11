package messages

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepository struct {
	db *pgxpool.Pool
}

func NewMessageRepository(ctx context.Context, cfg config.PostgresConfig) (*MessageRepository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return &MessageRepository{db: pool}, nil
}

func (m MessageRepository) CreateMessage(ctx context.Context, message *domain.Message) (*domain.Message, error) {
	messageModel := toModel(message)
	trx, err := m.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = trx.Rollback(ctx) }()

	err = trx.QueryRow(ctx,
		`INSERT INTO messages
		(chat_id, sender_id, content, sticker_id, edited, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, chat_id, sender_id, content, sticker_id, edited, created_at, updated_at, deleted_at`,
		messageModel.ChatId,
		messageModel.SenderId,
		messageModel.Content,
		messageModel.StickerId,
		messageModel.Edited,
		messageModel.CreatedAt,
		messageModel.UpdatedAt,
		messageModel.DeletedAt,
	).Scan(
		&messageModel.Id,
		&messageModel.ChatId,
		&messageModel.SenderId,
		&messageModel.Content,
		&messageModel.StickerId,
		&messageModel.Edited,
		&messageModel.CreatedAt,
		&messageModel.UpdatedAt,
		&messageModel.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	if _, err = trx.Exec(ctx,
		`UPDATE chats
		SET last_message_id = $1, updated_at = now()
		WHERE id = $2`,
		messageModel.Id, messageModel.ChatId,
	); err != nil {
		return nil, fmt.Errorf("update chat last message: %w", err)
	}

	if err = trx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return toDomainModel(messageModel), nil
}

func (m MessageRepository) GetMessagesByChatId(ctx context.Context, chatId int64, beforeID *int64, limit int) ([]*domain.Message, error) {
	var (
		rows pgx.Rows
		err  error
	)

	if beforeID != nil {
		rows, err = m.db.Query(ctx,
			`SELECT id, chat_id, sender_id, content, sticker_id, edited, created_at, updated_at, deleted_at
			FROM messages
			WHERE chat_id = $1 AND id < $2 AND deleted_at IS NULL
			ORDER BY id DESC
			LIMIT $3`,
			chatId, *beforeID, limit,
		)
	} else {
		rows, err = m.db.Query(ctx,
			`SELECT id, chat_id, sender_id, content, sticker_id, edited, created_at, updated_at, deleted_at
			FROM messages
			WHERE chat_id = $1 AND deleted_at IS NULL
			ORDER BY id DESC
			LIMIT $2`,
			chatId, limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("query messages by chat id: %w", err)
	}
	defer rows.Close()

	messages := make([]*domain.Message, 0, limit)
	for rows.Next() {
		model := &MessageModel{}
		if err = rows.Scan(
			&model.Id,
			&model.ChatId,
			&model.SenderId,
			&model.Content,
			&model.StickerId,
			&model.Edited,
			&model.CreatedAt,
			&model.UpdatedAt,
			&model.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message row: %w", err)
		}

		messages = append(messages, toDomainModel(model))
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate message rows: %w", err)
	}

	return messages, nil
}
