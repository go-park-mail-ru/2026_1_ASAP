package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	searchdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/domain/search"
	searchsql "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/repository/postgres/sql"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type dbPool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Close()
}

type Repository struct {
	db     dbPool
	logger *zap.Logger
}

func NewRepository(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*Repository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	poolCfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}
	return &Repository{db: pool, logger: logger}, nil
}

func (r *Repository) Close() {
	r.db.Close()
}

func (r *Repository) log(ctx context.Context) *zap.Logger {
	base := r.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}

func kindsToStrings(kinds []searchdomain.ChatType) []string {
	if len(kinds) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, string(k))
	}
	return out
}

func (r *Repository) SearchChats(ctx context.Context, params *searchdomain.SearchChatsParams) (*searchdomain.SearchChatsResult, error) {
	if params == nil {
		return nil, fmt.Errorf("search chats: params is nil")
	}

	q := searchsql.SearchChats
	pattern := likePattern(params.Query)
	kindsArg := kindsToStrings(params.Kinds)
	want := int(params.Limit) + 1
	start := time.Now()
	rows, err := r.db.Query(ctx, q, params.UserID, pattern, kindsArg, params.BeforeID, want)
	sqllog.LogQuery(ctx, r.log(ctx), "SearchChats", q, start, err,
		[]any{params.UserID, "[like]", kindsArg, params.BeforeID, want})
	if err != nil {
		return nil, fmt.Errorf("search chats query: %w", err)
	}
	defer rows.Close()

	var chats []searchdomain.ChatHit
	for rows.Next() {
		row := &chatSearchRow{}
		if scanErr := rows.Scan(
			&row.ID,
			&row.Type,
			&row.Title,
			&row.AvatarURL,
			&row.LastMessagePreview,
			&row.LastMessageAt,
		); scanErr != nil {
			return nil, fmt.Errorf("search chats scan: %w", scanErr)
		}
		chats = append(chats, rowToChatHit(row))
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("search chats rows: %w", err)
	}

	var nextBeforeID int64
	if len(chats) > int(params.Limit) {
		nextBeforeID = chats[params.Limit-1].ChatID
		chats = chats[:params.Limit]
	}

	return &searchdomain.SearchChatsResult{
		Chats:        chats,
		NextBeforeID: nextBeforeID,
	}, nil
}

func (r *Repository) SearchGlobalChannels(ctx context.Context, params *searchdomain.SearchGlobalChannelsParams) (*searchdomain.SearchGlobalChannelsResult, error) {
	if params == nil {
		return nil, fmt.Errorf("search global channels: params is nil")
	}

	q := searchsql.SearchGlobalChannels
	pattern := likePattern(params.Query)
	want := int(params.Limit) + 1
	start := time.Now()
	rows, err := r.db.Query(ctx, q, params.UserID, pattern, params.BeforeID, want)
	sqllog.LogQuery(ctx, r.log(ctx), "SearchGlobalChannels", q, start, err,
		[]any{params.UserID, "[like]", params.BeforeID, want})
	if err != nil {
		return nil, fmt.Errorf("search global channels query: %w", err)
	}
	defer rows.Close()

	var channels []searchdomain.GlobalChannelHit
	for rows.Next() {
		row := &globalChannelSearchRow{}
		if scanErr := rows.Scan(
			&row.ID,
			&row.Title,
			&row.AvatarURL,
			&row.LastMessagePreview,
			&row.LastMessageAt,
			&row.IsMember,
		); scanErr != nil {
			return nil, fmt.Errorf("search global channels scan: %w", scanErr)
		}
		channels = append(channels, rowToGlobalChannelHit(row))
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("search global channels rows: %w", err)
	}

	var nextBeforeID int64
	if len(channels) > int(params.Limit) {
		nextBeforeID = channels[params.Limit-1].ChatID
		channels = channels[:params.Limit]
	}

	return &searchdomain.SearchGlobalChannelsResult{
		Channels:     channels,
		NextBeforeID: nextBeforeID,
	}, nil
}

func (r *Repository) SearchContacts(ctx context.Context, params *searchdomain.SearchContactsParams) (*searchdomain.SearchContactsResult, error) {
	if params == nil {
		return nil, fmt.Errorf("search contacts: params is nil")
	}

	var sqlText string
	switch params.Scope {
	case searchdomain.ContactScopeContacts:
		sqlText = searchsql.SearchContacts
	case searchdomain.ContactScopeLocal:
		sqlText = searchsql.SearchUsers
	default:
		return nil, searchdomain.ErrInvalidInput
	}

	res, err := r.queryUserHits(ctx, "SearchContacts", sqlText, params.UserID, params.Query, params.BeforeID, params.Limit)
	if err != nil {
		return nil, err
	}
	return &searchdomain.SearchContactsResult{
		Contacts:     res.hits,
		NextBeforeID: res.nextBeforeID,
	}, nil
}

func (r *Repository) SearchUsers(ctx context.Context, params *searchdomain.SearchUsersParams) (*searchdomain.SearchUsersResult, error) {
	if params == nil {
		return nil, fmt.Errorf("search users: params is nil")
	}

	res, err := r.queryUserHits(ctx, "SearchUsers", searchsql.SearchUsers, params.RequesterID, params.Query, params.BeforeID, params.Limit)
	if err != nil {
		return nil, err
	}
	return &searchdomain.SearchUsersResult{
		Users:        res.hits,
		NextBeforeID: res.nextBeforeID,
	}, nil
}

type userHitsQueryResult struct {
	hits         []searchdomain.ContactHit
	nextBeforeID int64
}

func (r *Repository) queryUserHits(ctx context.Context, opName, sqlText string, scopedUserID int64, query string, beforeID int64, limit int32) (userHitsQueryResult, error) {
	pattern := likePattern(query)
	want := int(limit) + 1
	start := time.Now()
	rows, err := r.db.Query(ctx, sqlText, scopedUserID, pattern, beforeID, want)
	sqllog.LogQuery(ctx, r.log(ctx), opName, sqlText, start, err,
		[]any{scopedUserID, "[like]", beforeID, want})
	if err != nil {
		return userHitsQueryResult{}, fmt.Errorf("%s query: %w", opName, err)
	}
	defer rows.Close()

	var hits []searchdomain.ContactHit
	for rows.Next() {
		row := &contactSearchRow{}
		if scanErr := rows.Scan(
			&row.UserID,
			&row.FName,
			&row.LastName,
			&row.Login,
			&row.Avatar,
			&row.LastSeen,
		); scanErr != nil {
			return userHitsQueryResult{}, fmt.Errorf("%s scan: %w", opName, scanErr)
		}
		hits = append(hits, rowToContactHit(row))
	}
	if err = rows.Err(); err != nil {
		return userHitsQueryResult{}, fmt.Errorf("%s rows: %w", opName, err)
	}

	var nextBeforeID int64
	if len(hits) > int(limit) {
		nextBeforeID = hits[limit-1].UserID
		hits = hits[:limit]
	}

	return userHitsQueryResult{hits: hits, nextBeforeID: nextBeforeID}, nil
}

func (r *Repository) SearchMessagesInChat(ctx context.Context, params *searchdomain.SearchMessagesInChatParams) (*searchdomain.SearchMessagesInChatResult, error) {
	if params == nil {
		return nil, fmt.Errorf("search messages in chat: params is nil")
	}

	q := searchsql.SearchMessagesInChat
	want := int(params.Limit) + 1
	start := time.Now()
	rows, err := r.db.Query(ctx, q, params.ChatID, params.Query, params.UserID, params.BeforeID, want)
	sqllog.LogQuery(ctx, r.log(ctx), "SearchMessagesInChat", q, start, err,
		[]any{params.ChatID, "[fts]", params.UserID, params.BeforeID, want})
	if err != nil {
		return nil, fmt.Errorf("search messages query: %w", err)
	}
	defer rows.Close()

	var msgs []searchdomain.MessageHit
	for rows.Next() {
		row := &messageSearchRow{}
		if scanErr := rows.Scan(
			&row.ID,
			&row.ChatID,
			&row.SenderID,
			&row.Content,
			&row.CreatedAt,
			&row.RankDiscard,
		); scanErr != nil {
			return nil, fmt.Errorf("search messages scan: %w", scanErr)
		}
		msgs = append(msgs, rowToMessageHit(row))
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("search messages rows: %w", err)
	}

	var nextBeforeID int64
	if len(msgs) > int(params.Limit) {
		nextBeforeID = msgs[params.Limit-1].MessageID
		msgs = msgs[:params.Limit]
	}

	return &searchdomain.SearchMessagesInChatResult{
		Messages:     msgs,
		NextBeforeID: nextBeforeID,
	}, nil
}
