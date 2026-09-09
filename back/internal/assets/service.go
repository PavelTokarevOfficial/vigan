// Package assets implements the application boundary for reusable media assets.
package assets

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/finde-clip/finde-v2/back/internal/media"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Folder struct {
	ID        string    `json:"id"`
	ParentID  *string   `json:"parentId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Asset struct {
	ID         string    `json:"id"`
	FolderID   *string   `json:"folderId"`
	Name       string    `json:"name"`
	Kind       string    `json:"kind"`
	MIMEType   string    `json:"mimeType"`
	Size       int64     `json:"size"`
	StorageKey string    `json:"storageKey"`
	URL        string    `json:"url,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Service struct {
	db      *pgxpool.Pool
	storage media.Storage
}

func New(db *pgxpool.Pool, storage media.Storage) *Service { return &Service{db: db, storage: storage} }

func (s *Service) List(ctx context.Context) ([]Folder, []Asset, error) {
	folderRows, err := s.db.Query(ctx, `SELECT id,parent_id,name,created_at,updated_at FROM asset_folders ORDER BY name`)
	if err != nil {
		return nil, nil, err
	}
	defer folderRows.Close()
	folders := []Folder{}
	for folderRows.Next() {
		var item Folder
		if err = folderRows.Scan(&item.ID, &item.ParentID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, nil, err
		}
		folders = append(folders, item)
	}
	if err = folderRows.Err(); err != nil {
		return nil, nil, err
	}

	rows, err := s.db.Query(ctx, `SELECT id,folder_id,name,kind::text,mime_type,size,storage_key,created_at,updated_at FROM assets ORDER BY name`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := []Asset{}
	for rows.Next() {
		var item Asset
		if err = rows.Scan(&item.ID, &item.FolderID, &item.Name, &item.Kind, &item.MIMEType, &item.Size, &item.StorageKey, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, nil, err
		}
		item.URL, err = s.storage.PresignGet(ctx, item.StorageKey, 15*time.Minute)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	return folders, items, rows.Err()
}

func (s *Service) CreateFolder(ctx context.Context, name string, parentID *string) (Folder, error) {
	name = cleanName(name)
	if name == "" {
		return Folder{}, fmt.Errorf("folder name is required")
	}
	if err := s.ensureFolder(ctx, parentID); err != nil {
		return Folder{}, err
	}
	var item Folder
	err := s.db.QueryRow(ctx, `INSERT INTO asset_folders(parent_id,name) VALUES($1,$2) RETURNING id,parent_id,name,created_at,updated_at`, parentID, name).Scan(&item.ID, &item.ParentID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return Folder{}, friendlyUnique(err, "a folder with this name already exists")
	}
	return item, nil
}

func (s *Service) UpdateFolder(ctx context.Context, id, name string, parentID *string) error {
	name = cleanName(name)
	if name == "" {
		return fmt.Errorf("folder name is required")
	}
	if parentID != nil && *parentID == id {
		return fmt.Errorf("a folder cannot be its own parent")
	}
	if err := s.ensureFolder(ctx, parentID); err != nil {
		return err
	}
	var exists bool
	err := s.db.QueryRow(ctx, `WITH RECURSIVE descendants AS (
		SELECT id FROM asset_folders WHERE parent_id=$1 UNION ALL
		SELECT f.id FROM asset_folders f JOIN descendants d ON f.parent_id=d.id
	) SELECT EXISTS(SELECT 1 FROM descendants WHERE id=$2)`, id, parentID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("a folder cannot be moved into its descendant")
	}
	cmd, err := s.db.Exec(ctx, `UPDATE asset_folders SET name=$2,parent_id=$3,updated_at=now() WHERE id=$1`, id, name, parentID)
	if err != nil {
		return friendlyUnique(err, "a folder with this name already exists")
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("folder not found")
	}
	return nil
}

func (s *Service) DeleteFolder(ctx context.Context, id string) error {
	var hasChildren, hasAssets bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM asset_folders WHERE parent_id=$1), EXISTS(SELECT 1 FROM assets WHERE folder_id=$1)`, id).Scan(&hasChildren, &hasAssets); err != nil {
		return err
	}
	if hasChildren || hasAssets {
		return fmt.Errorf("move or delete child folders and assets before deleting this folder")
	}
	cmd, err := s.db.Exec(ctx, `DELETE FROM asset_folders WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("folder not found")
	}
	return nil
}

func (s *Service) Upload(ctx context.Context, folderID *string, name, mime string, size int64, body io.Reader) (Asset, error) {
	if err := s.ensureFolder(ctx, folderID); err != nil {
		return Asset{}, err
	}
	name = cleanName(name)
	if name == "" {
		return Asset{}, fmt.Errorf("asset name is required")
	}
	kind, err := kindForMIME(mime)
	if err != nil {
		return Asset{}, err
	}
	if size < 1 {
		return Asset{}, fmt.Errorf("asset must not be empty")
	}
	id, err := newUUID()
	if err != nil {
		return Asset{}, err
	}
	key := "assets/" + id + "/original"
	if err = s.storage.Put(ctx, key, body, mime); err != nil {
		return Asset{}, fmt.Errorf("upload asset: %w", err)
	}
	var item Asset
	err = s.db.QueryRow(ctx, `INSERT INTO assets(id,folder_id,name,kind,mime_type,size,storage_key) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,folder_id,name,kind::text,mime_type,size,storage_key,created_at,updated_at`, id, folderID, name, kind, mime, size, key).Scan(&item.ID, &item.FolderID, &item.Name, &item.Kind, &item.MIMEType, &item.Size, &item.StorageKey, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		_ = s.storage.Delete(ctx, key)
		return Asset{}, friendlyUnique(err, "an asset with this name already exists in this folder")
	}
	item.URL, err = s.storage.PresignGet(ctx, item.StorageKey, 15*time.Minute)
	if err != nil {
		return Asset{}, err
	}
	return item, nil
}

func (s *Service) UpdateAsset(ctx context.Context, id, name string, folderID *string) error {
	if err := s.ensureFolder(ctx, folderID); err != nil {
		return err
	}
	name = cleanName(name)
	if name == "" {
		return fmt.Errorf("asset name is required")
	}
	cmd, err := s.db.Exec(ctx, `UPDATE assets SET name=$2,folder_id=$3,updated_at=now() WHERE id=$1`, id, name, folderID)
	if err != nil {
		return friendlyUnique(err, "an asset with this name already exists in this folder")
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("asset not found")
	}
	return nil
}

func (s *Service) DeleteAsset(ctx context.Context, id string) error {
	var key string
	if err := s.db.QueryRow(ctx, `SELECT storage_key FROM assets WHERE id=$1`, id).Scan(&key); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("asset not found")
		}
		return err
	}
	var referenced bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM template_asset_references WHERE asset_id=$1)`, id).Scan(&referenced); err != nil {
		return err
	}
	if referenced {
		return fmt.Errorf("asset is used by a template; remove it from the template first")
	}
	if err := s.storage.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete asset from object storage: %w", err)
	}
	_, err := s.db.Exec(ctx, `DELETE FROM assets WHERE id=$1`, id)
	return err
}

func (s *Service) ensureFolder(ctx context.Context, id *string) error {
	if id == nil || *id == "" {
		return nil
	}
	var exists bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM asset_folders WHERE id=$1)`, *id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("folder not found")
	}
	return nil
}

func cleanName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	return strings.Trim(name, ".")
}

func kindForMIME(mime string) (string, error) {
	mime = strings.ToLower(strings.TrimSpace(strings.Split(mime, ";")[0]))
	if mime == "image/gif" {
		return "gif", nil
	}
	if strings.HasPrefix(mime, "image/") {
		return "image", nil
	}
	if strings.HasPrefix(mime, "video/") {
		return "video", nil
	}
	if strings.HasPrefix(mime, "audio/") {
		return "audio", nil
	}
	return "", fmt.Errorf("unsupported asset type %q", mime)
}

func friendlyUnique(err error, message string) error {
	if strings.Contains(err.Error(), "duplicate key") {
		return errors.New(message)
	}
	return err
}

func newUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
