package processing

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type ClaimedJob struct{ ID, ClipID, Type, ClipURL, BannerKey string }
type Jobs struct{ db *pgxpool.Pool }

type Info struct {
	ID          string     `json:"id"`
	ClipID      string     `json:"clipId"`
	ClipTitle   string     `json:"clipTitle"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	CurrentStep string     `json:"currentStep"`
	Error       string     `json:"error"`
	Progress    int        `json:"progress"`
	Attempts    int        `json:"attempts"`
	CreatedAt   time.Time  `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt"`
	FinishedAt  *time.Time `json:"finishedAt"`
}

func NewJobs(db *pgxpool.Pool) *Jobs { return &Jobs{db} }
func (j *Jobs) List(ctx context.Context) ([]Info, error) {
	rows, err := j.db.Query(ctx, `SELECT j.id,j.clip_id,c.title,j.type,j.status,COALESCE(j.current_step,''),j.progress,j.attempts,COALESCE(j.error,''),j.created_at,j.started_at,j.finished_at
		FROM processing_jobs j JOIN clips c ON c.id=j.clip_id ORDER BY j.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Info{}
	for rows.Next() {
		var x Info
		if err = rows.Scan(&x.ID, &x.ClipID, &x.ClipTitle, &x.Type, &x.Status, &x.CurrentStep, &x.Progress, &x.Attempts, &x.Error, &x.CreatedAt, &x.StartedAt, &x.FinishedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (j *Jobs) Get(ctx context.Context, id string) (Info, error) {
	var x Info
	err := j.db.QueryRow(ctx, `SELECT j.id,j.clip_id,c.title,j.type,j.status,COALESCE(j.current_step,''),j.progress,j.attempts,COALESCE(j.error,''),j.created_at,j.started_at,j.finished_at
		FROM processing_jobs j JOIN clips c ON c.id=j.clip_id WHERE j.id=$1`, id).Scan(&x.ID, &x.ClipID, &x.ClipTitle, &x.Type, &x.Status, &x.CurrentStep, &x.Progress, &x.Attempts, &x.Error, &x.CreatedAt, &x.StartedAt, &x.FinishedAt)
	return x, err
}
func (j *Jobs) Claim(ctx context.Context) (ClaimedJob, error) {
	var x ClaimedJob
	e := j.db.QueryRow(ctx, `WITH next AS (SELECT id,clip_id,banner_id FROM processing_jobs WHERE status='pending' ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1)
		UPDATE processing_jobs j SET status='running',attempts=attempts+1,started_at=now(),updated_at=now()
		FROM next JOIN clips c ON c.id=next.clip_id LEFT JOIN banners b ON b.id=next.banner_id
		WHERE j.id=next.id
		RETURNING j.id,j.clip_id,j.type,c.twitch_url,COALESCE(b.storage_key,'')`).Scan(&x.ID, &x.ClipID, &x.Type, &x.ClipURL, &x.BannerKey)
	return x, e
}
func (j *Jobs) Step(ctx context.Context, id, clipID, step, status string, progress int) error {
	_, e := j.db.Exec(ctx, "UPDATE processing_jobs SET current_step=$2,progress=$3,updated_at=now() WHERE id=$1", id, step, progress)
	if e != nil {
		return e
	}
	_, e = j.db.Exec(ctx, "UPDATE clips SET status=$2,updated_at=now() WHERE id=$1", clipID, status)
	return e
}
func (j *Jobs) Complete(ctx context.Context, id, clipID string) error {
	if e := j.Step(ctx, id, clipID, "completed", "completed", 100); e != nil {
		return e
	}
	_, e := j.db.Exec(ctx, "UPDATE processing_jobs SET status='completed',finished_at=now(),updated_at=now() WHERE id=$1", id)
	return e
}
func (j *Jobs) Fail(ctx context.Context, id, clipID, msg string) error {
	_, e := j.db.Exec(ctx, "UPDATE processing_jobs SET status='failed',error=$2,finished_at=now(),updated_at=now() WHERE id=$1", id, msg)
	if e != nil {
		return e
	}
	_, e = j.db.Exec(ctx, "UPDATE clips SET status='failed',error=$2,updated_at=now() WHERE id=$1", clipID, msg)
	return e
}

// Requeue is used only during graceful worker shutdown. Artifacts already persisted by Runner remain reusable.
func (j *Jobs) Requeue(ctx context.Context, id string) error {
	_, err := j.db.Exec(ctx, `UPDATE processing_jobs
		SET status='pending',current_step='interrupted',error='worker interrupted; retrying',started_at=NULL,updated_at=now()
		WHERE id=$1 AND status='running'`, id)
	return err
}
