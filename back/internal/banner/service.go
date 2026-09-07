package banner

import (
	"context"
	"fmt"
	"github.com/finde-clip/finde-v2/back/internal/media"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"path"
)

type Banner struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	StorageKey string `json:"storageKey"`
	MimeType   string `json:"mimeType"`
	Size       int64  `json:"size"`
}
type Service struct {
	db *pgxpool.Pool
	s  media.Storage
}

func New(db *pgxpool.Pool, s media.Storage) *Service { return &Service{db, s} }
func (s *Service) List(ctx context.Context) ([]Banner, error) {
	rows, e := s.db.Query(ctx, "SELECT id,name,storage_key,mime_type,size FROM banners ORDER BY created_at DESC")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Banner{}
	for rows.Next() {
		var b Banner
		if e = rows.Scan(&b.ID, &b.Name, &b.StorageKey, &b.MimeType, &b.Size); e != nil {
			return nil, e
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
func (s *Service) Create(ctx context.Context, name, mime string, size int64, body io.Reader) (Banner, error) {
	if name == "" {
		return Banner{}, fmt.Errorf("banner name is required")
	}
	var b Banner
	b.Name = name
	b.MimeType = mime
	b.Size = size
	e := s.db.QueryRow(ctx, "INSERT INTO banners(name,storage_key,mime_type,size) VALUES($1,'pending',$2,$3) RETURNING id", name, mime, size).Scan(&b.ID)
	if e != nil {
		return b, e
	}
	b.StorageKey = "banners/" + b.ID + path.Ext(name)
	if e = s.s.Put(ctx, b.StorageKey, body, mime); e != nil {
		_, _ = s.db.Exec(ctx, "DELETE FROM banners WHERE id=$1", b.ID)
		return b, e
	}
	_, e = s.db.Exec(ctx, "UPDATE banners SET storage_key=$2 WHERE id=$1", b.ID, b.StorageKey)
	return b, e
}
func (s *Service) Delete(ctx context.Context, id string) error {
	var key string
	if e := s.db.QueryRow(ctx, "DELETE FROM banners WHERE id=$1 RETURNING storage_key", id).Scan(&key); e != nil {
		return e
	}
	return s.s.Delete(ctx, key)
}
