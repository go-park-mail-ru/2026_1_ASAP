package contacts

import (
	"context"
	"fmt"

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
	`SELECT user_id, contact_name, contact_user_id, created_at, updated_at
	 FROM contacts
	 WHERE user_id=$1
	 ORDER BY contact_name ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all contacts: %w", err)
	}
	defer rows.Close()

	var contacts []*domain.Contact
	for rows.Next() {
		contact := &ContactModel{}
		err := rows.Scan(
			&contact.UserID,
			&contact.ContactName,
			&contact.ContactUserID,
			&contact.CreatedAt,
			&contact.UpdatedAt,
		)
		if err != nil {
			return nil ,fmt.Errorf("failed to scan rows: %w", err)
		}
		contacts = append(contacts, toDomainContact(contact))
	}

	return contacts, nil
}

func (r *ContactsRepository) CreateContact(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	contactModel := toModelContact(contact)
	err := r.db.QueryRow(ctx, 
	`INSERT INTO contacts
	 (user_id, contact_user_id, contact_name, created_at, updated_at)
	 VALUES ($1, $2, $3, $4, $5)
	 RETURNING user_id, contact_user_id, contact_name`, contactModel.UserID, contactModel.ContactUserID, contactModel.ContactName, contactModel.CreatedAt, contactModel.UpdatedAt).Scan(&contactModel.UserID, &contactModel.ContactUserID, &contactModel.ContactName,)

	if err != nil {
		return nil, fmt.Errorf("failed to creaye contact: %w", err)
	}

	return toDomainContact(contactModel), nil
}

func (r *ContactsRepository) DeleteContact(ctx context.Context, userID, contactUserID int64) (error) {
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