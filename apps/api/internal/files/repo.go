package files

import (
	"context"
	"errors"
	"time"

	"filetransfer.ir/api/internal/plans"
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
	// ExpiresAt is when the plan's retention removes the file. Zero means no retention date.
	ExpiresAt time.Time `json:"expiresAt,omitzero"`
}

// Download is one hand-out of a file through a share link.
type Download struct {
	FileID     string
	OwnerID    string
	ShareToken string
	SizeBytes  int64
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

	// LatestSubscription returns the owner's most recent subscription that has started by now.
	LatestSubscription(ctx context.Context, userID string, now time.Time) (plans.Subscription, error)
	// StorageUsed is the space the owner holds: finished files plus uploads in progress.
	StorageUsed(ctx context.Context, ownerID string) (int64, error)
	// TrafficUsed is the download volume handed out for the owner since the given time.
	TrafficUsed(ctx context.Context, ownerID string, since time.Time) (int64, error)
	RecordDownload(ctx context.Context, d Download) error
	// ExpiredFiles lists files past their retention date.
	ExpiredFiles(ctx context.Context, now time.Time, limit int) ([]File, error)
	// LapsedFiles lists files of owners who once had a plan and whose last plan ended before cutoff.
	LapsedFiles(ctx context.Context, cutoff time.Time, limit int) ([]File, error)
}

type PG struct{ Pool *pgxpool.Pool }

const cols = `id::text, owner_id::text, object_key, name, size_bytes, content_type, status, coalesce(upload_id, ''), part_size, created_at, expires_at`

func scan(row pgx.Row) (File, error) {
	var f File
	var expires *time.Time
	err := row.Scan(&f.ID, &f.OwnerID, &f.ObjectKey, &f.Name, &f.SizeBytes, &f.ContentType, &f.Status, &f.UploadID, &f.PartSize, &f.CreatedAt, &expires)
	if expires != nil {
		f.ExpiresAt = *expires
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return f, ErrNotFound
	}
	return f, err
}

func (p PG) Create(ctx context.Context, f File) (File, error) {
	return scan(p.Pool.QueryRow(ctx, `
		INSERT INTO files (owner_id, object_key, name, size_bytes, content_type, status, upload_id, part_size, expires_at)
		VALUES ($1::uuid, $2, $3, $4, $5, 'uploading', $6, $7, $8)
		RETURNING `+cols, f.OwnerID, f.ObjectKey, f.Name, f.SizeBytes, f.ContentType, f.UploadID, f.PartSize, nullTime(f.ExpiresAt)))
}

func (p PG) Get(ctx context.Context, id string) (File, error) {
	if !validUUID(id) {
		return File{}, ErrNotFound
	}
	return scan(p.Pool.QueryRow(ctx, `SELECT `+cols+` FROM files WHERE id = $1::uuid AND status <> 'deleted'`, id))
}

func (p PG) ListReady(ctx context.Context, ownerID string) ([]File, error) {
	return p.query(ctx, `SELECT `+cols+` FROM files WHERE owner_id = $1::uuid AND status = 'ready' ORDER BY created_at DESC LIMIT 500`, ownerID)
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
	return p.query(ctx, `SELECT `+cols+` FROM files WHERE status = 'uploading' AND created_at < $1 ORDER BY created_at LIMIT $2`, olderThan, limit)
}

func (p PG) query(ctx context.Context, sql string, args ...any) ([]File, error) {
	rows, err := p.Pool.Query(ctx, sql, args...)
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

func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func (p PG) LatestSubscription(ctx context.Context, userID string, now time.Time) (plans.Subscription, error) {
	var s plans.Subscription
	err := p.Pool.QueryRow(ctx, `
		SELECT id::text, user_id::text, plan, starts_at, expires_at FROM subscriptions
		WHERE user_id = $1::uuid AND starts_at <= $2
		ORDER BY starts_at DESC, created_at DESC LIMIT 1`, userID, now).Scan(&s.ID, &s.UserID, &s.PlanID, &s.StartsAt, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	return s, err
}

func (p PG) StorageUsed(ctx context.Context, ownerID string) (int64, error) {
	var n int64
	err := p.Pool.QueryRow(ctx, `SELECT coalesce(sum(size_bytes), 0)::bigint FROM files WHERE owner_id = $1::uuid AND status IN ('uploading', 'ready')`, ownerID).Scan(&n)
	return n, err
}

func (p PG) TrafficUsed(ctx context.Context, ownerID string, since time.Time) (int64, error) {
	var n int64
	err := p.Pool.QueryRow(ctx, `SELECT coalesce(sum(size_bytes), 0)::bigint FROM downloads WHERE owner_id = $1::uuid AND created_at >= $2`, ownerID, since).Scan(&n)
	return n, err
}

func (p PG) RecordDownload(ctx context.Context, d Download) error {
	_, err := p.Pool.Exec(ctx, `INSERT INTO downloads (file_id, owner_id, share_token, size_bytes) VALUES ($1::uuid, $2::uuid, $3, $4)`, d.FileID, d.OwnerID, d.ShareToken, d.SizeBytes)
	return err
}

func (p PG) ExpiredFiles(ctx context.Context, now time.Time, limit int) ([]File, error) {
	return p.query(ctx, `SELECT `+cols+` FROM files WHERE status = 'ready' AND expires_at < $1 ORDER BY expires_at LIMIT $2`, now, limit)
}

// LapsedFiles never touches owners without any subscription row, so files from before plans existed are left alone.
func (p PG) LapsedFiles(ctx context.Context, cutoff time.Time, limit int) ([]File, error) {
	return p.query(ctx, `
		SELECT `+cols+` FROM files f
		WHERE f.status IN ('uploading', 'ready')
		  AND EXISTS (SELECT 1 FROM subscriptions s WHERE s.user_id = f.owner_id)
		  AND NOT EXISTS (SELECT 1 FROM subscriptions s WHERE s.user_id = f.owner_id AND s.expires_at >= $1)
		ORDER BY f.created_at LIMIT $2`, cutoff, limit)
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
