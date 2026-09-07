package media

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Video struct {
	ClipID     string    `json:"clipId"`
	Title      string    `json:"title"`
	Streamer   string    `json:"streamer"`
	TwitchURL  string    `json:"twitchUrl"`
	StorageKey string    `json:"storageKey"`
	URL        string    `json:"url"`
	CreatedAt  time.Time `json:"createdAt"`
}
type Videos struct {
	db      *pgxpool.Pool
	storage Storage
}

func NewVideos(db *pgxpool.Pool, s Storage) *Videos { return &Videos{db, s} }
func (v *Videos) List(ctx context.Context) ([]Video, error) {
	rows, e := v.db.Query(ctx, `SELECT c.id,c.title,s.display_name,c.twitch_url,m.storage_key,m.created_at FROM clips c JOIN streamers s ON s.id=c.streamer_id JOIN media_files m ON m.clip_id=c.id AND m.type='render' ORDER BY m.created_at DESC`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Video{}
	for rows.Next() {
		var x Video
		if e = rows.Scan(&x.ClipID, &x.Title, &x.Streamer, &x.TwitchURL, &x.StorageKey, &x.CreatedAt); e != nil {
			return nil, e
		}
		x.URL, e = v.storage.PresignGet(ctx, x.StorageKey, 15*time.Minute)
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
