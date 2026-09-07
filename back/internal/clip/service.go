package clip

import (
	"context"
	"fmt"
	"github.com/finde-clip/finde-v2/back/infrastructure/twitch"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db     *pgxpool.Pool
	twitch *twitch.Client
}
type Local struct {
	ID           string `json:"id"`
	StreamerID   string `json:"streamerId"`
	StreamerName string `json:"streamerName"`
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnailUrl"`
	Status       string `json:"status"`
	Error        string `json:"error"`
	CurrentStep  string `json:"currentStep"`
	Progress     int    `json:"progress"`
}

func (s *Service) List(ctx context.Context) ([]Local, error) {
	rows, e := s.db.Query(ctx, `SELECT c.id,c.streamer_id,s.display_name,c.title,COALESCE(c.thumbnail_url,''),c.status,COALESCE(c.error,''),
		COALESCE(j.current_step,''),COALESCE(j.progress,0)
		FROM clips c JOIN streamers s ON s.id=c.streamer_id
		LEFT JOIN LATERAL (SELECT current_step,progress FROM processing_jobs WHERE clip_id=c.id ORDER BY created_at DESC LIMIT 1) j ON true
		ORDER BY c.created_at DESC`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Local{}
	for rows.Next() {
		var x Local
		if e = rows.Scan(&x.ID, &x.StreamerID, &x.StreamerName, &x.Title, &x.ThumbnailURL, &x.Status, &x.Error, &x.CurrentStep, &x.Progress); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *Service) EnqueueProcess(ctx context.Context, id, bannerID string, retry bool) error {
	if retry {
		_, e := s.db.Exec(ctx, "UPDATE clips SET status='downloaded',error=NULL,updated_at=now() WHERE id=$1", id)
		if e != nil {
			return e
		}
	}
	var jobID string
	e := s.db.QueryRow(ctx, `INSERT INTO processing_jobs(clip_id,banner_id,type)
		SELECT c.id,NULLIF($2,''),'process' FROM clips c
		WHERE c.id=$1
		AND NOT EXISTS (SELECT 1 FROM processing_jobs j WHERE j.clip_id=c.id AND j.type='process' AND j.status IN ('pending','running'))
		RETURNING id`, id, bannerID).Scan(&jobID)
	if e == pgx.ErrNoRows {
		return fmt.Errorf("clip does not exist or already has an active processing job")
	}
	return e
}

func New(db *pgxpool.Pool, t *twitch.Client) *Service { return &Service{db, t} }
func (s *Service) Remote(ctx context.Context, streamerID string) ([]twitch.Clip, error) {
	var tid, login string
	if e := s.db.QueryRow(ctx, "SELECT COALESCE(twitch_user_id,''),twitch_login FROM streamers WHERE id=$1", streamerID).Scan(&tid, &login); e != nil {
		return nil, e
	}
	if tid == "" {
		u, e := s.twitch.User(ctx, login)
		if e != nil {
			return nil, e
		}
		tid = u.ID
		_, e = s.db.Exec(ctx, "UPDATE streamers SET twitch_user_id=$2,display_name=$3,updated_at=now() WHERE id=$1", streamerID, u.ID, u.DisplayName)
		if e != nil {
			return nil, e
		}
	}
	clips, e := s.twitch.Clips(ctx, tid)
	if e != nil {
		return nil, e
	}
	for i := range clips {
		if e = s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM clips WHERE twitch_clip_id=$1)", clips[i].ID).Scan(&clips[i].Saved); e != nil {
			return nil, e
		}
	}
	return clips, nil
}
func (s *Service) Import(ctx context.Context, streamerID string, c twitch.Clip) (string, error) {
	var id string
	e := s.db.QueryRow(ctx, `INSERT INTO clips(streamer_id,twitch_clip_id,title,twitch_url,thumbnail_url,duration,twitch_created_at,status)
		VALUES($1,$2,$3,$4,$5,$6,$7,'saved') ON CONFLICT(twitch_clip_id) DO NOTHING RETURNING id`, streamerID, c.ID, c.Title, c.URL, c.ThumbnailURL, c.Duration, c.CreatedAt).Scan(&id)
	if e == pgx.ErrNoRows {
		e = s.db.QueryRow(ctx, "SELECT id FROM clips WHERE twitch_clip_id=$1", c.ID).Scan(&id)
		return id, e
	}
	if e != nil {
		return "", e
	}
	_, e = s.db.Exec(ctx, "INSERT INTO processing_jobs(clip_id,type) VALUES($1,'download')", id)
	return id, e
}
