package messages

import (
	"context"
	"fmt"
	"time"

	messagessql "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/repository/messages/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
)

func (m *MessageRepository) CanUserAccessAttachment(ctx context.Context, userID int64, objectKey, attachmentRef string) (bool, error) {
	q := messagessql.CanAccessAttachment
	start := time.Now()
	var exists bool
	err := m.db.QueryRow(ctx, q, userID, objectKey, attachmentRef).Scan(&exists)
	sqllog.LogQuery(ctx, m.log(ctx), "CanUserAccessAttachment", q, start, err, []any{userID, objectKey, attachmentRef})
	if err != nil {
		return false, fmt.Errorf("can user access attachment: %w", err)
	}
	return exists, nil
}
