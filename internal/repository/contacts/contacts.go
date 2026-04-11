package contacts

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/contacts"
)

type ContactsRepository struct {
	db *pgxpool.Pool
}

func NewContactsRepository(ctx context.Context, cfg config.PostgresConfig) (*ContactsRepository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return &ContactsRepository{db: pool}, nil
}

func (r *ContactsRepository) GetAllContactsByUserID(ctx context.Context, userID int64) ([]*domain.Contact, error) {
	rows, err := r.db.Query(ctx,
		`SELECT c.user_id, c.first_name, c.last_name, c.contact_user_id, u.avatar_url, c.created_at, c.updated_at
	 FROM contacts c
	 JOIN users u ON c.contact_user_id=u.id
	 WHERE c.user_id=$1
	 ORDER BY first_name ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all contacts: %w", err)
	}
	defer rows.Close()

	var contacts []*domain.Contact
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

func (r *ContactsRepository) CreateContact(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	contactModel := toModelContact(contact)
	err := r.db.QueryRow(ctx,
		`INSERT INTO contacts
	 (user_id, contact_user_id, first_name, last_name)
	 VALUES ($1, $2, $3, $4)
	 RETURNING user_id, contact_user_id, first_name, last_name, created_at, updated_at`,
		contactModel.UserID, contactModel.ContactUserID, contactModel.FirstName, contactModel.LastName,
	).Scan(&contactModel.UserID, &contactModel.ContactUserID, &contactModel.FirstName, &contactModel.LastName, &contactModel.CreatedAt, &contactModel.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	var contactAvatarUrl sql.NullString
	errr := r.db.QueryRow(ctx,
		`SELECT avatar_url
	 FROM users
	 WHERE id=$1`, contactModel.ContactUserID).Scan(&contactAvatarUrl)

	if errr != nil && errr != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get user avatar: %w", err)
	}
	contactModel.ContactAvatarUrl = contactAvatarUrl

	return toDomainContact(contactModel), nil
}

func (r *ContactsRepository) DeleteContact(ctx context.Context, userID, contactUserID int64) error {
	result, err := r.db.Exec(ctx,
		`DELETE FROM contacts
	 WHERE user_id=$1 AND contact_user_id=$2`, userID, contactUserID)
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrContactNotFound
	}

	return nil
}

func (r *ContactsRepository) IsContact(ctx context.Context, userID, contactUserID int64) (bool, error) {
	var exists bool

	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM contacts WHERE user_id=$1 AND contact_user_id=$2)`, userID, contactUserID).Scan(&exists)

	return exists, err
}

func (r *ContactsRepository) Close() {
	r.db.Close()
}
