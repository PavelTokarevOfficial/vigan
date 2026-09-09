package media

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Files struct{ db *pgxpool.Pool }

func NewFiles(db *pgxpool.Pool) *Files { return &Files{db} }
func (f *Files) Upsert(ctx context.Context, clipID, jobID, kind, key, mime string, size int64) error {
	_, e := f.db.Exec(ctx, `INSERT INTO media_files(clip_id,processing_job_id,type,storage_key,mime_type,size) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(storage_key) DO UPDATE SET processing_job_id=EXCLUDED.processing_job_id,mime_type=EXCLUDED.mime_type,size=EXCLUDED.size`, clipID, jobID, kind, key, mime, size)
	return e
}
