package contacts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	contactdom "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/domain/contact"
	contactssql "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/repository/contacts/sql"
)

type dbPool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Close()
}

type ContactsRepository struct {
	db     dbPool
	logger *zap.Logger
}

func NewContactsRepository(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*ContactsRepository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return &ContactsRepository{db: pool, logger: logger}, nil
}

func (r *ContactsRepository) GetAllContactsByUserID(ctx context.Context, userID int64) ([]*contactdom.Contact, error) {
	q := contactssql.GetAllContactsByUserID
	start := time.Now()
	rows, err := r.db.Query(ctx, q, userID)
	sqllog.LogQuery(ctx, r.log(ctx), "GetAllContactsByUserID", q, start, err, []any{userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get all contacts: %w", err)
	}
	defer rows.Close()

	var contacts []*contactdom.Contact
	for rows.Next() {
		contact := &ContactModel{}
		err := rows.Scan(
			&contact.UserID,
			&contact.FirstName,
			&contact.LastName,
			&contact.ContactUserID,
			&contact.ContactAvatarUrl,
			&contact.CreatedAt,
			&contact.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rows: %w", err)
		}
		contacts = append(contacts, toDomainContact(contact))
	}

	return contacts, nil
}

func (r *ContactsRepository) CreateContact(ctx context.Context, contact *contactdom.Contact) (*contactdom.Contact, error) {
	contactModel := toModelContact(contact)
	q := contactssql.InsertContact
	start := time.Now()
	err := r.db.QueryRow(ctx, q,
		contactModel.UserID, contactModel.ContactUserID, contactModel.FirstName, contactModel.LastName,
	).Scan(&contactModel.UserID, &contactModel.ContactUserID, &contactModel.FirstName, &contactModel.LastName, &contactModel.CreatedAt, &contactModel.UpdatedAt)
	sqllog.LogQuery(ctx, r.log(ctx), "CreateContact", q, start, err, []any{
		contactModel.UserID, contactModel.ContactUserID, contactModel.FirstName, contactModel.LastName,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	var contactAvatarUrl sql.NullString
	q = contactssql.GetUserAvatarURL
	start = time.Now()
	errr := r.db.QueryRow(ctx, q, contactModel.ContactUserID).Scan(&contactAvatarUrl)
	sqllog.LogQuery(ctx, r.log(ctx), "CreateContact.avatar", q, start, errr, []any{contactModel.ContactUserID})

	if errr != nil && errr != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get user avatar: %w", err)
	}
	contactModel.ContactAvatarUrl = contactAvatarUrl

	return toDomainContact(contactModel), nil
}

func (r *ContactsRepository) DeleteContact(ctx context.Context, userID, contactUserID int64) error {
	q := contactssql.DeleteContact
	start := time.Now()
	result, err := r.db.Exec(ctx, q, userID, contactUserID)
	sqllog.LogQuery(ctx, r.log(ctx), "DeleteContact", q, start, err, []any{userID, contactUserID})
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	if result.RowsAffected() == 0 {
		return contactdom.ErrContactNotFound
	}

	return nil
}

func (r *ContactsRepository) GetContact(ctx context.Context, userID, contactUserID int64) (*contactdom.Contact, error) {
	q := contactssql.GetContact
	start := time.Now()
	contact := &ContactModel{}
	err := r.db.QueryRow(ctx, q, userID, contactUserID).Scan(
		&contact.UserID,
		&contact.FirstName,
		&contact.LastName,
		&contact.ContactUserID,
		&contact.ContactAvatarUrl,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)
	sqllog.LogQuery(ctx, r.log(ctx), "GetContact", q, start, err, []any{userID, contactUserID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, contactdom.ErrContactNotFound
		}
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	return toDomainContact(contact), nil
}

func (r *ContactsRepository) IsContact(ctx context.Context, userID, contactUserID int64) (bool, error) {
	var exists bool

	q := contactssql.IsContact
	start := time.Now()
	err := r.db.QueryRow(ctx, q, userID, contactUserID).Scan(&exists)
	sqllog.LogQuery(ctx, r.log(ctx), "IsContact", q, start, err, []any{userID, contactUserID})

	return exists, err
}

func (r *ContactsRepository) Close() {
	r.db.Close()
}

func (r *ContactsRepository) log(ctx context.Context) *zap.Logger {
	base := r.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}
