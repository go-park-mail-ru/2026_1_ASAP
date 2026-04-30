package messages

import (
	"context"
	"fmt"
	"time"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	messagessql "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/repository/messages/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
)

type dbPool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Close()
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	QueryRow(ctx context.Context, sql string, args ...any) (pgx.Row)
}

type MessageRepository struct {
	db     dbPool
	logger *zap.Logger
}

func NewMessageRepository(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*MessageRepository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return &MessageRepository{db: pool, logger: logger}, nil
}

func (m *MessageRepository) CreateMessage(ctx context.Context, message *domain.Message) (*domain.Message, error) {
	start := time.Now()
	messageModel := toModel(message)
	trx, err := m.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead,
	})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = trx.Rollback(ctx) }()

	err = trx.QueryRow(ctx, messagessql.InsertMessage,
		messageModel.ChatId,
		messageModel.SenderId,
		messageModel.Content,
		messageModel.StickerId,
		messageModel.Edited,
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
	sqllog.LogQuery(ctx, m.log(ctx), "CreateMessage", messagessql.CreateMessageTxDescription, start, err, []any{
		messageModel.ChatId, messageModel.SenderId, messageModel.Content, messageModel.StickerId, messageModel.Edited,
	})
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	start = time.Now()
	if _, err = trx.Exec(ctx, messagessql.UpdateChatLastMessage,
		messageModel.Id, messageModel.ChatId,
	); err != nil {
		sqllog.LogQuery(ctx, m.log(ctx), "CreateMessage.updateChat", messagessql.UpdateChatLastMessage, start, err, []any{messageModel.Id, messageModel.ChatId})
		return nil, fmt.Errorf("update chat last message: %w", err)
	}
	sqllog.LogQuery(ctx, m.log(ctx), "CreateMessage.updateChat", messagessql.UpdateChatLastMessage, start, err, []any{messageModel.Id, messageModel.ChatId})

	start = time.Now()
	if err = trx.Commit(ctx); err != nil {
		sqllog.LogQuery(ctx, m.log(ctx), "CreateMessage.commit", "COMMIT", start, err, []any{messageModel.ChatId})
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	sqllog.LogQuery(ctx, m.log(ctx), "CreateMessage.commit", "COMMIT", start, nil, []any{messageModel.ChatId})

	return toDomainModel(messageModel), nil
}

func (m *MessageRepository) UpdateMessage(ctx context.Context, message *domain.Message) (*domain.Message, error) {
	q := messagessql.UpdateMessage
	messageModel := toModel(message)
	start := time.Now()
	row := m.db.QueryRow(ctx, q, messageModel.Content, messageModel.Id, messageModel.ChatId, messageModel.SenderId)
	err := row.Scan(
		&messageModel.Id,
		&messageModel.ChatId,
		&messageModel.SenderId,
		&messageModel.Content,
		&messageModel.CreatedAt,
		&messageModel.UpdatedAt,
		&messageModel.Edited,
	)
	sqllog.LogQuery(ctx, m.log(ctx), "UpdateMessage", q, start, err, []any{messageModel.Content, messageModel.Id, messageModel.ChatId, messageModel.SenderId})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNoMessage
		}
		return nil, fmt.Errorf("failed to scan message: %w", err)
	}

	return toDomainModel(messageModel), nil
}

func (m *MessageRepository) GetMessagesByChatId(ctx context.Context, chatId int64, beforeID *int64, limit int) ([]*domain.Message, error) {
	var (
		rows pgx.Rows
		err  error
		q    string
	)
	start := time.Now()

	if beforeID != nil {
		q = messagessql.GetMessagesByChatBeforeID
		rows, err = m.db.Query(ctx, q,
			chatId, *beforeID, limit,
		)
		sqllog.LogQuery(ctx, m.log(ctx), "GetMessagesByChatId", q, start, err, []any{chatId, *beforeID, limit})
	} else {
		q = messagessql.GetMessagesByChat
		rows, err = m.db.Query(ctx, q,
			chatId, limit,
		)
		sqllog.LogQuery(ctx, m.log(ctx), "GetMessagesByChatId", q, start, err, []any{chatId, limit})
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

func (m *MessageRepository) log(ctx context.Context) *zap.Logger {
	base := m.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}

func (m *MessageRepository) Close() {
	m.db.Close()
}
