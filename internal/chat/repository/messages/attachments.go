package messages

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	messagessql "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/repository/messages/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
)

func (m *MessageRepository) CreateMessageWithAttachments(
	ctx context.Context,
	message *domain.Message,
	attachments []domain.MessageAttachment,
) (*domain.Message, error) {
	start := time.Now()
	messageModel := toModel(message)
	trx, err := m.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
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
	sqllog.LogQuery(ctx, m.log(ctx), "CreateMessageWithAttachments", messagessql.InsertMessage, start, err, []any{
		messageModel.ChatId, messageModel.SenderId, messageModel.Content,
	})
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	createdAttachments := make([]domain.MessageAttachment, 0, len(attachments))
	for i, att := range attachments {
		att.MessageId = messageModel.Id
		att.SortOrder = i
		model := attachmentFromDomain(att)
		var scanned AttachmentModel
		err = trx.QueryRow(ctx, messagessql.InsertMessageAttachment,
			model.MessageId,
			model.Type,
			att.SortOrder,
			model.FileURL,
			model.FileName,
			model.MimeType,
			model.FileSize,
			model.ContactUserID,
			model.ContactFirstName,
			model.ContactLastName,
			model.ContactAvatarURL,
			durationArg(model.DurationMs),
			waveformArg(model.Waveform),
			model.IsCapybara,
		).Scan(
			&scanned.Id,
			&scanned.MessageId,
			&scanned.Type,
			&scanned.SortOrder,
			&scanned.FileURL,
			&scanned.FileName,
			&scanned.MimeType,
			&scanned.FileSize,
			&scanned.ContactUserID,
			&scanned.ContactFirstName,
			&scanned.ContactLastName,
			&scanned.ContactAvatarURL,
			&scanned.DurationMs,
			&scanned.Waveform,
			&scanned.IsCapybara,
			&scanned.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("insert message attachment: %w", err)
		}
		createdAttachments = append(createdAttachments, attachmentToDomain(&scanned))
	}

	start = time.Now()
	if _, err = trx.Exec(ctx, messagessql.UpdateChatLastMessage, messageModel.Id, messageModel.ChatId); err != nil {
		sqllog.LogQuery(ctx, m.log(ctx), "CreateMessageWithAttachments.updateChat", messagessql.UpdateChatLastMessage, start, err, []any{messageModel.Id, messageModel.ChatId})
		return nil, fmt.Errorf("update chat last message: %w", err)
	}

	if err = trx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	result := toDomainModel(messageModel)
	result.Attachments = createdAttachments
	return result, nil
}

func (m *MessageRepository) GetAttachmentsByMessageIDs(ctx context.Context, messageIDs []int64) (map[int64][]domain.MessageAttachment, error) {
	if len(messageIDs) == 0 {
		return map[int64][]domain.MessageAttachment{}, nil
	}
	q := messagessql.GetAttachmentsByMessageIDs
	start := time.Now()
	rows, err := m.db.Query(ctx, q, messageIDs)
	sqllog.LogQuery(ctx, m.log(ctx), "GetAttachmentsByMessageIDs", q, start, err, []any{messageIDs})
	if err != nil {
		return nil, fmt.Errorf("query attachments: %w", err)
	}
	defer rows.Close()

	out := make(map[int64][]domain.MessageAttachment)
	for rows.Next() {
		model := &AttachmentModel{}
		if err = rows.Scan(
			&model.Id,
			&model.MessageId,
			&model.Type,
			&model.SortOrder,
			&model.FileURL,
			&model.FileName,
			&model.MimeType,
			&model.FileSize,
			&model.ContactUserID,
			&model.ContactFirstName,
			&model.ContactLastName,
			&model.ContactAvatarURL,
			&model.DurationMs,
			&model.Waveform,
			&model.Transcript,
			&model.IsCapybara,
			&model.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan attachment: %w", err)
		}
		out[model.MessageId] = append(out[model.MessageId], attachmentToDomain(model))
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attachments: %w", err)
	}
	return out, nil
}

func (m *MessageRepository) GetMessageByID(ctx context.Context, chatID, messageID int64) (*domain.Message, error) {
	q := messagessql.GetMessageByID
	start := time.Now()
	model := &MessageModel{}
	err := m.db.QueryRow(ctx, q, messageID, chatID).Scan(
		&model.Id,
		&model.ChatId,
		&model.SenderId,
		&model.Content,
		&model.StickerId,
		&model.Edited,
		&model.CreatedAt,
		&model.UpdatedAt,
		&model.DeletedAt,
	)
	sqllog.LogQuery(ctx, m.log(ctx), "GetMessageByID", q, start, err, []any{messageID, chatID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNoMessage
		}
		return nil, fmt.Errorf("get message by id: %w", err)
	}
	return toDomainModel(model), nil
}

func (m *MessageRepository) UpdateAttachmentTranscript(ctx context.Context, attachmentID int64, transcript string) (*domain.MessageAttachment, error) {
	q := messagessql.UpdateMessageAttachmentTranscript
	start := time.Now()
	model := &AttachmentModel{}
	err := m.db.QueryRow(ctx, q, attachmentID, transcript).Scan(
		&model.Id,
		&model.MessageId,
		&model.Type,
		&model.SortOrder,
		&model.FileURL,
		&model.FileName,
		&model.MimeType,
		&model.FileSize,
		&model.ContactUserID,
		&model.ContactFirstName,
		&model.ContactLastName,
		&model.ContactAvatarURL,
		&model.DurationMs,
		&model.Waveform,
		&model.Transcript,
		&model.CreatedAt,
	)
	sqllog.LogQuery(ctx, m.log(ctx), "UpdateAttachmentTranscript", q, start, err, []any{attachmentID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidAttachment
		}
		return nil, fmt.Errorf("update attachment transcript: %w", err)
	}
	out := attachmentToDomain(model)
	return &out, nil
}
