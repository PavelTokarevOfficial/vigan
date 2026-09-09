package streamer

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type Streamer struct {
	ID           string `json:"id"`
	TwitchLogin  string `json:"twitchLogin"`
	DisplayName  string `json:"displayName"`
	TwitchUserID string `json:"twitchUserId"`
}
type Service struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Service { return &Service{db} }
func (s *Service) List(ctx context.Context) ([]Streamer, error) {
	rows, e := s.db.Query(ctx, "SELECT id,twitch_login,display_name,COALESCE(twitch_user_id,'') FROM streamers ORDER BY created_at DESC")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	r := []Streamer{}
	for rows.Next() {
		var x Streamer
		if e = rows.Scan(&x.ID, &x.TwitchLogin, &x.DisplayName, &x.TwitchUserID); e != nil {
			return nil, e
		}
		r = append(r, x)
	}
	return r, rows.Err()
}
func (s *Service) Create(ctx context.Context, login, name string) (Streamer, error) {
	login = strings.ToLower(strings.TrimSpace(login))
	name = strings.TrimSpace(name)
	if login == "" {
		return Streamer{}, fmt.Errorf("streamer nickname is required")
	}
	if name == "" {
		name = login
	}
	var x Streamer
	e := s.db.QueryRow(ctx, "INSERT INTO streamers(twitch_login,display_name) VALUES($1,$2) RETURNING id,twitch_login,display_name,COALESCE(twitch_user_id,'')", login, name).Scan(&x.ID, &x.TwitchLogin, &x.DisplayName, &x.TwitchUserID)
	return x, e
}
func (s *Service) Update(ctx context.Context, id, login, name string) (Streamer, error) {
	login = strings.ToLower(strings.TrimSpace(login))
	name = strings.TrimSpace(name)
	if login == "" {
		return Streamer{}, fmt.Errorf("streamer nickname is required")
	}
	if name == "" {
		name = login
	}
	var x Streamer
	e := s.db.QueryRow(ctx, `UPDATE streamers
		SET twitch_login=$2,display_name=$3,
			twitch_user_id=CASE WHEN twitch_login <> $2 THEN NULL ELSE twitch_user_id END,
			updated_at=now()
		WHERE id=$1
		RETURNING id,twitch_login,display_name,COALESCE(twitch_user_id,'')`, id, login, name).Scan(&x.ID, &x.TwitchLogin, &x.DisplayName, &x.TwitchUserID)
	return x, e
}
func (s *Service) Delete(ctx context.Context, id string) error {
	_, e := s.db.Exec(ctx, "DELETE FROM streamers WHERE id=$1", id)
	return e
}
