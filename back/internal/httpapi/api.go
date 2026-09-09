package httpapi

import (
	"encoding/json"
	"github.com/finde-clip/finde-v2/back/infrastructure/twitch"
	"github.com/finde-clip/finde-v2/back/internal/clip"
	"github.com/finde-clip/finde-v2/back/internal/media"
	"github.com/finde-clip/finde-v2/back/internal/processing"
	"github.com/finde-clip/finde-v2/back/internal/streamer"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
)

type API struct {
	streamers *streamer.Service
	clips     *clip.Service
	library   *media.Library
	videos    *media.Videos
	jobs      *processing.Jobs
	log       *slog.Logger
}

func New(s *streamer.Service, c *clip.Service, library *media.Library, v *media.Videos, j *processing.Jobs, l *slog.Logger) *API {
	return &API{streamers: s, clips: c, library: library, videos: v, jobs: j, log: l}
}
func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]bool{"ok": true}) })
	r.Route("/api/streamers", func(r chi.Router) {
		r.Get("/", a.list)
		r.Post("/", a.create)
		r.Put("/{id}", a.update)
		r.Delete("/{id}", a.delete)
	})
	r.Get("/api/streamers/{id}/clips", a.remoteClips)
	r.Post("/api/clips/import", a.importClip)
	r.Get("/api/clips", a.localClips)
	r.Delete("/api/clips/{id}", a.deleteClip)
	r.Post("/api/clips/{id}/download", a.download)
	r.Post("/api/clips/{id}/process", a.process)
	r.Post("/api/clips/{id}/retry", a.retry)
	r.Get("/api/jobs", a.listJobs)
	r.Get("/api/jobs/{id}", a.getJob)
	r.Get("/api/videos", a.readyVideos)
	return r
}
func (a *API) listJobs(w http.ResponseWriter, r *http.Request) {
	x, e := a.jobs.List(r.Context())
	if e != nil {
		fail(w, 500, e)
		return
	}
	write(w, 200, map[string]any{"data": x})
}
func (a *API) getJob(w http.ResponseWriter, r *http.Request) {
	x, e := a.jobs.Get(r.Context(), chi.URLParam(r, "id"))
	if e != nil {
		fail(w, 404, errText("job not found"))
		return
	}
	write(w, 200, map[string]any{"data": x})
}
func (a *API) readyVideos(w http.ResponseWriter, r *http.Request) {
	x, e := a.videos.List(r.Context())
	if e != nil {
		fail(w, 500, e)
		return
	}
	write(w, 200, map[string]any{"data": x})
}
func (a *API) remoteClips(w http.ResponseWriter, r *http.Request) {
	x, e := a.clips.Remote(r.Context(), chi.URLParam(r, "id"))
	if e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 200, map[string]any{"data": x})
}

type importInput struct {
	StreamerID string      `json:"streamerId"`
	Clip       twitch.Clip `json:"clip"`
}

func (a *API) importClip(w http.ResponseWriter, r *http.Request) {
	var in importInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, errText("invalid JSON"))
		return
	}
	if in.StreamerID == "" || in.Clip.ID == "" || in.Clip.URL == "" || in.Clip.Title == "" {
		fail(w, 400, errText("streamerId, clip.id, clip.url and clip.title are required"))
		return
	}
	id, e := a.clips.Import(r.Context(), in.StreamerID, in.Clip)
	if e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 201, map[string]string{"id": id})
}
func (a *API) localClips(w http.ResponseWriter, r *http.Request) {
	x, e := a.clips.List(r.Context())
	if e != nil {
		fail(w, 500, e)
		return
	}
	write(w, 200, map[string]any{"data": x})
}

func (a *API) process(w http.ResponseWriter, r *http.Request) {
	if e := a.clips.EnqueueProcess(r.Context(), chi.URLParam(r, "id")); e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 202, map[string]string{"status": "queued"})
}
func (a *API) retry(w http.ResponseWriter, r *http.Request) {
	if e := a.clips.Retry(r.Context(), chi.URLParam(r, "id")); e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 202, map[string]string{"status": "queued"})
}
func (a *API) download(w http.ResponseWriter, r *http.Request) {
	if e := a.clips.EnqueueDownload(r.Context(), chi.URLParam(r, "id")); e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 202, map[string]string{"status": "queued"})
}
func (a *API) deleteClip(w http.ResponseWriter, r *http.Request) {
	if e := a.library.DeleteClip(r.Context(), chi.URLParam(r, "id")); e != nil {
		fail(w, 422, e)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (a *API) list(w http.ResponseWriter, r *http.Request) {
	x, e := a.streamers.List(r.Context())
	if e != nil {
		fail(w, 500, e)
		return
	}
	write(w, 200, map[string]any{"data": x})
}

type streamerInput struct {
	TwitchLogin string `json:"twitchLogin"`
	DisplayName string `json:"displayName"`
}

func (a *API) create(w http.ResponseWriter, r *http.Request) {
	var in streamerInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, errText("invalid JSON"))
		return
	}
	x, e := a.streamers.Create(r.Context(), in.TwitchLogin, in.DisplayName)
	if e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 201, map[string]any{"data": x})
}
func (a *API) update(w http.ResponseWriter, r *http.Request) {
	var in streamerInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, errText("invalid JSON"))
		return
	}
	x, e := a.streamers.Update(r.Context(), chi.URLParam(r, "id"), in.TwitchLogin, in.DisplayName)
	if e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 200, map[string]any{"data": x})
}
func (a *API) delete(w http.ResponseWriter, r *http.Request) {
	if e := a.streamers.Delete(r.Context(), chi.URLParam(r, "id")); e != nil {
		fail(w, 500, e)
		return
	}
	w.WriteHeader(204)
}

type errText string

func (e errText) Error() string { return string(e) }
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, e error) {
	write(w, status, map[string]any{"error": map[string]string{"message": e.Error()}})
}
