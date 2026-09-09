package httpapi

import (
	"encoding/json"
	"github.com/finde-clip/finde-v2/back/infrastructure/twitch"
	"github.com/finde-clip/finde-v2/back/internal/assets"
	"github.com/finde-clip/finde-v2/back/internal/clip"
	"github.com/finde-clip/finde-v2/back/internal/composition"
	"github.com/finde-clip/finde-v2/back/internal/media"
	"github.com/finde-clip/finde-v2/back/internal/processing"
	"github.com/finde-clip/finde-v2/back/internal/streamer"
	"github.com/finde-clip/finde-v2/back/internal/videotemplate"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
)

type API struct {
	streamers *streamer.Service
	clips     *clip.Service
	assets    *assets.Service
	templates *videotemplate.Service
	library   *media.Library
	videos    *media.Videos
	jobs      *processing.Jobs
	log       *slog.Logger
}

func New(s *streamer.Service, c *clip.Service, assets *assets.Service, templates *videotemplate.Service, library *media.Library, v *media.Videos, j *processing.Jobs, l *slog.Logger) *API {
	return &API{streamers: s, clips: c, assets: assets, templates: templates, library: library, videos: v, jobs: j, log: l}
}
func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]bool{"ok": true}) })
	r.Route("/api/streamers", func(r chi.Router) {
		r.Get("/", a.list)
		r.Post("/", a.create)
		r.Post("/bulk", a.createMany)
		r.Put("/{id}", a.update)
		r.Delete("/{id}", a.delete)
	})
	r.Get("/api/streamers/{id}/clips", a.remoteClips)
	r.Route("/api/assets", func(r chi.Router) {
		r.Get("/", a.listAssets)
		r.Post("/", a.uploadAsset)
		r.Patch("/{id}", a.updateAsset)
		r.Delete("/{id}", a.deleteAsset)
	})
	r.Route("/api/asset-folders", func(r chi.Router) {
		r.Post("/", a.createFolder)
		r.Patch("/{id}", a.updateFolder)
		r.Delete("/{id}", a.deleteFolder)
	})
	r.Route("/api/templates", func(r chi.Router) {
		r.Get("/", a.listTemplates)
		r.Post("/", a.createTemplate)
		r.Get("/{id}", a.getTemplate)
		r.Put("/{id}", a.updateTemplate)
		r.Post("/{id}/duplicate", a.duplicateTemplate)
		r.Delete("/{id}", a.deleteTemplate)
	})
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
	var in processInput
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.TemplateID == "" {
		fail(w, 400, errText("templateId is required"))
		return
	}
	if e := a.clips.EnqueueProcess(r.Context(), chi.URLParam(r, "id"), in.TemplateID); e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 202, map[string]string{"status": "queued"})
}

type processInput struct {
	TemplateID string `json:"templateId"`
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

type streamerBulkInput struct {
	TwitchLogins []string `json:"twitchLogins"`
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
func (a *API) createMany(w http.ResponseWriter, r *http.Request) {
	var in streamerBulkInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, errText("invalid JSON"))
		return
	}
	created, e := a.streamers.CreateMany(r.Context(), in.TwitchLogins)
	if e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 201, map[string]any{"data": map[string]any{"created": len(created)}})
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

type folderInput struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parentId"`
}

type assetInput struct {
	Name     string  `json:"name"`
	FolderID *string `json:"folderId"`
}

func (a *API) listAssets(w http.ResponseWriter, r *http.Request) {
	folders, items, e := a.assets.List(r.Context())
	if e != nil {
		fail(w, 500, e)
		return
	}
	write(w, 200, map[string]any{"data": map[string]any{"folders": folders, "assets": items}})
}
func (a *API) createFolder(w http.ResponseWriter, r *http.Request) {
	var in folderInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, errText("invalid JSON"))
		return
	}
	item, e := a.assets.CreateFolder(r.Context(), in.Name, emptyToNil(in.ParentID))
	if e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 201, map[string]any{"data": item})
}
func (a *API) updateFolder(w http.ResponseWriter, r *http.Request) {
	var in folderInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, errText("invalid JSON"))
		return
	}
	if e := a.assets.UpdateFolder(r.Context(), chi.URLParam(r, "id"), in.Name, emptyToNil(in.ParentID)); e != nil {
		fail(w, 422, e)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (a *API) deleteFolder(w http.ResponseWriter, r *http.Request) {
	if e := a.assets.DeleteFolder(r.Context(), chi.URLParam(r, "id")); e != nil {
		fail(w, 422, e)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (a *API) uploadAsset(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024<<20)
	if e := r.ParseMultipartForm(32 << 20); e != nil {
		fail(w, 400, errText("invalid or too large multipart upload"))
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, e := r.FormFile("file")
	if e != nil {
		fail(w, 400, errText("file is required"))
		return
	}
	defer file.Close()
	item, e := a.assets.Upload(r.Context(), emptyToNilString(r.FormValue("folderId")), header.Filename, header.Header.Get("Content-Type"), header.Size, file)
	if e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 201, map[string]any{"data": item})
}
func (a *API) updateAsset(w http.ResponseWriter, r *http.Request) {
	var in assetInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, errText("invalid JSON"))
		return
	}
	if e := a.assets.UpdateAsset(r.Context(), chi.URLParam(r, "id"), in.Name, emptyToNil(in.FolderID)); e != nil {
		fail(w, 422, e)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (a *API) deleteAsset(w http.ResponseWriter, r *http.Request) {
	if e := a.assets.DeleteAsset(r.Context(), chi.URLParam(r, "id")); e != nil {
		fail(w, 422, e)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type templateInput struct {
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	PreviewAssetID *string         `json:"previewAssetId"`
	Config         json.RawMessage `json:"config"`
}

func decodeTemplate(r *http.Request) (videotemplate.Input, error) {
	var in templateInput
	if e := json.NewDecoder(r.Body).Decode(&in); e != nil {
		return videotemplate.Input{}, errText("invalid JSON")
	}
	config, e := composition.ParseConfig(in.Config)
	if e != nil {
		return videotemplate.Input{}, e
	}
	return videotemplate.Input{Name: in.Name, Description: in.Description, PreviewAssetID: emptyToNil(in.PreviewAssetID), Config: config}, nil
}
func (a *API) listTemplates(w http.ResponseWriter, r *http.Request) {
	items, e := a.templates.List(r.Context())
	if e != nil {
		fail(w, 500, e)
		return
	}
	write(w, 200, map[string]any{"data": items})
}
func (a *API) getTemplate(w http.ResponseWriter, r *http.Request) {
	item, e := a.templates.Get(r.Context(), chi.URLParam(r, "id"))
	if e != nil {
		fail(w, 404, e)
		return
	}
	write(w, 200, map[string]any{"data": item})
}
func (a *API) createTemplate(w http.ResponseWriter, r *http.Request) {
	in, e := decodeTemplate(r)
	if e != nil {
		fail(w, 400, e)
		return
	}
	item, e := a.templates.Create(r.Context(), in)
	if e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 201, map[string]any{"data": item})
}
func (a *API) updateTemplate(w http.ResponseWriter, r *http.Request) {
	in, e := decodeTemplate(r)
	if e != nil {
		fail(w, 400, e)
		return
	}
	item, e := a.templates.Update(r.Context(), chi.URLParam(r, "id"), in)
	if e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 200, map[string]any{"data": item})
}
func (a *API) duplicateTemplate(w http.ResponseWriter, r *http.Request) {
	item, e := a.templates.Duplicate(r.Context(), chi.URLParam(r, "id"))
	if e != nil {
		fail(w, 422, e)
		return
	}
	write(w, 201, map[string]any{"data": item})
}
func (a *API) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	if e := a.templates.Delete(r.Context(), chi.URLParam(r, "id")); e != nil {
		fail(w, 422, e)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func emptyToNil(v *string) *string {
	if v == nil || *v == "" {
		return nil
	}
	return v
}
func emptyToNilString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
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
