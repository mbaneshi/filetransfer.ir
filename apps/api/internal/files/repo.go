package files

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type File struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"-"`
	ObjectKey   string    `json:"-"`
	Name        string    `json:"name"`
	SizeBytes   int64     `json:"sizeBytes"`
	ContentType string    `json:"contentType"`
	Status      string    `json:"status"`
	UploadID    string    `json:"-"`
	PartSize    int64     `json:"partSize"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Share struct {
	Token     string
	FileID    string
	ExpiresAt time.Time
}

// Repo is the persistence the handlers need.
type Repo interface {
	Create(ctx context.Context, f File) (File, error)
	Get(ctx context.Context, id string) (File, error)
	ListReady(ctx context.Context, ownerID string) ([]File, error)
	MarkReady(ctx context.Context, id string) error
	MarkDeleted(ctx context.Context, id string) error
	StaleUploads(ctx context.Context, olderThan time.Time, limit int) ([]File, error)
	CreateShare(ctx context.Context, s Share) error
	GetShare(ctx context.Context, token string) (Share, error)
}

type PG struct{ Pool *pgxpool.Pool }

const cols = `id::text, owner_id::text, object_key, name, size_bytes, content_type, status, coalesce(upload_id, ''), part_size, created_at`

func scan(row pgx.Row) (File, error) {
	var f File
	err := row.Scan(&f.ID, &f.OwnerID, &f.ObjectKey, &f.Name, &f.SizeBytes, &f.ContentType, &f.Status, &f.UploadID, &f.PartSize, &f.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, ErrNotFound
	}
	return f, err
}

func (p PG) Create(ctx context.Context, f File) (File, error) {
	return scan(p.Pool.QueryRow(ctx, `
		INSERT INTO files (owner_id, object_key, name, size_bytes, content_type, status, upload_id, part_size)
		VALUES ($1::uuid, $2, $3, $4, $5, 'uploading', $6, $7)
		RETURNING `+cols, f.OwnerID, f.ObjectKey, f.Name, f.SizeBytes, f.ContentType, f.UploadID, f.PartSize))
}

func (p PG) Get(ctx context.Context, id string) (File, error) {
	if !validUUID(id) {
		return File{}, ErrNotFound
	}
	return scan(p.Pool.QueryRow(ctx, `SELECT `+cols+` FROM files WHERE id = $1::uuid AND status <> 'deleted'`, id))
}

func (p PG) ListReady(ctx context.Context, ownerID string) ([]File, error) {
	rows, err := p.Pool.Query(ctx, `SELECT `+cols+` FROM files WHERE owner_id = $1::uuid AND status = 'ready' ORDER BY created_at DESC LIMIT 500`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []File
	for rows.Next() {
		f, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (p PG) MarkReady(ctx context.Context, id string) error {
	_, err := p.Pool.Exec(ctx, `UPDATE files SET status = 'ready', completed_at = now(), upload_id = NULL WHERE id = $1::uuid`, id)
	return err
}

func (p PG) MarkDeleted(ctx context.Context, id string) error {
	_, err := p.Pool.Exec(ctx, `UPDATE files SET status = 'deleted', upload_id = NULL WHERE id = $1::uuid`, id)
	return err
}

func (p PG) StaleUploads(ctx context.Context, olderThan time.Time, limit int) ([]File, error) {
	rows, err := p.Pool.Query(ctx, `SELECT `+cols+` FROM files WHERE status = 'uploading' AND created_at < $1 ORDER BY created_at LIMIT $2`, olderThan, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []File
	for rows.Next() {
		f, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (p PG) CreateShare(ctx context.Context, s Share) error {
	_, err := p.Pool.Exec(ctx, `INSERT INTO shares (token, file_id, expires_at) VALUES ($1, $2::uuid, $3)`, s.Token, s.FileID, s.ExpiresAt)
	return err
}

func (p PG) GetShare(ctx context.Context, token string) (Share, error) {
	var s Share
	err := p.Pool.QueryRow(ctx, `SELECT token, file_id::text, expires_at FROM shares WHERE token = $1`, token).Scan(&s.Token, &s.FileID, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	return s, err
}

func validUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if r != '-' {
				return false
			}
		case !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'):
			return false
		}
	}
	return true
}
