package files

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"filetransfer.ir/api/internal/auth"
	"filetransfer.ir/api/internal/storage"
)

const (
	minPartSize     = 64 << 20 // 64 MiB: a 50 GB file is ~800 requests, not thousands
	maxParts        = 10000    // S3 hard limit
	maxPartsPerCall = 100
	partURLTTL      = 2 * time.Hour
	downloadURLTTL  = time.Hour
	defaultShare    = 7 * 24 * time.Hour
	maxShare        = 30 * 24 * time.Hour
	minShare        = time.Hour
)

// Users resolves the logged-in user (satisfied by *auth.Handlers).
type Users interface {
	CurrentUser(r *http.Request) (auth.User, bool)
}

type Handlers struct {
	Repo        Repo
	Store       storage.Store
	Users       Users
	MaxFileSize int64
	// PublicBaseURL is where shared links point (the web app), e.g. https://filetransfer.ir
	PublicBaseURL string
	Now           func() time.Time
}

func NewHandlers(repo Repo, store storage.Store, users Users, maxFileSize int64, publicBaseURL string) *Handlers {
	return &Handlers{Repo: repo, Store: store, Users: users, MaxFileSize: maxFileSize, PublicBaseURL: publicBaseURL}
}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/uploads", h.authed(h.initUpload))
	mux.HandleFunc("GET /api/v1/uploads/{id}", h.authed(h.uploadStatus))
	mux.HandleFunc("POST /api/v1/uploads/{id}/parts", h.authed(h.presignParts))
	mux.HandleFunc("POST /api/v1/uploads/{id}/complete", h.authed(h.complete))
	mux.HandleFunc("DELETE /api/v1/uploads/{id}", h.authed(h.abort))
	mux.HandleFunc("GET /api/v1/files", h.authed(h.list))
	mux.HandleFunc("DELETE /api/v1/files/{id}", h.authed(h.deleteFile))
	mux.HandleFunc("POST /api/v1/files/{id}/shares", h.authed(h.createShare))
	mux.HandleFunc("GET /api/v1/shares/{token}", h.shareInfo)
	mux.HandleFunc("GET /api/v1/shares/{token}/download", h.shareDownload)
}

type authedFunc func(http.ResponseWriter, *http.Request, auth.User)

func (h *Handlers) authed(next authedFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := h.Users.CurrentUser(r)
		if !ok {
			fail(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		next(w, r, u)
	}
}

// PartSizeFor picks a part size that keeps any file within the 10,000-part limit.
func PartSizeFor(size int64) int64 {
	part := int64(minPartSize)
	if need := (size + maxParts - 2) / (maxParts - 1); need > part {
		const mib = 1 << 20
		part = (need + mib - 1) / mib * mib
	}
	return part
}

func partCount(size, partSize int64) int64 { return (size + partSize - 1) / partSize }

func (h *Handlers) initUpload(w http.ResponseWriter, r *http.Request, u auth.User) {
	var body struct {
		Name        string `json:"name"`
		Size        int64  `json:"size"`
		ContentType string `json:"contentType"`
	}
	if !decode(w, r, &body) {
		return
	}
	name := cleanName(body.Name)
	if name == "" {
		fail(w, http.StatusBadRequest, "invalid_name")
		return
	}
	if body.Size <= 0 {
		fail(w, http.StatusBadRequest, "invalid_size")
		return
	}
	if body.Size > h.MaxFileSize {
		fail(w, http.StatusRequestEntityTooLarge, "file_too_large")
		return
	}
	ct := body.ContentType
	if ct == "" || len(ct) > 200 {
		ct = "application/octet-stream"
	}

	key := "files/" + randomHex(16)
	uploadID, err := h.Store.CreateMultipart(r.Context(), key, ct)
	if err != nil {
		log.Printf("create multipart: %v", err)
		fail(w, http.StatusBadGateway, "storage_unavailable")
		return
	}
	ps := PartSizeFor(body.Size)
	f, err := h.Repo.Create(r.Context(), File{
		OwnerID: u.ID, ObjectKey: key, Name: name, SizeBytes: body.Size,
		ContentType: ct, UploadID: uploadID, PartSize: ps,
	})
	if err != nil {
		_ = h.Store.Abort(r.Context(), key, uploadID)
		log.Printf("create file row: %v", err)
		fail(w, http.StatusInternalServerError, "internal")
		return
	}
	reply(w, http.StatusCreated, map[string]any{
		"id": f.ID, "partSize": ps, "partCount": partCount(body.Size, ps),
	})
}

// load returns the caller's own file; anyone else's is reported as not found.
func (h *Handlers) load(w http.ResponseWriter, r *http.Request, u auth.User) (File, bool) {
	f, err := h.Repo.Get(r.Context(), r.PathValue("id"))
	if err != nil || f.OwnerID != u.ID {
		fail(w, http.StatusNotFound, "not_found")
		return File{}, false
	}
	return f, true
}

func (h *Handlers) uploadStatus(w http.ResponseWriter, r *http.Request, u auth.User) {
	f, ok := h.load(w, r, u)
	if !ok {
		return
	}
	resp := map[string]any{
		"status": f.Status, "partSize": f.PartSize, "partCount": partCount(f.SizeBytes, f.PartSize),
	}
	if f.Status == "uploading" {
		parts, err := h.Store.ListParts(r.Context(), f.ObjectKey, f.UploadID)
		if err != nil {
			fail(w, http.StatusBadGateway, "storage_unavailable")
			return
		}
		uploaded := make([]map[string]any, len(parts))
		for i, p := range parts {
			uploaded[i] = map[string]any{"number": p.Number, "size": p.Size}
		}
		resp["uploaded"] = uploaded
	}
	reply(w, http.StatusOK, resp)
}

func (h *Handlers) presignParts(w http.ResponseWriter, r *http.Request, u auth.User) {
	f, ok := h.load(w, r, u)
	if !ok {
		return
	}
	if f.Status != "uploading" {
		fail(w, http.StatusConflict, "not_uploading")
		return
	}
	var body struct {
		Parts []int32 `json:"parts"`
	}
	if !decode(w, r, &body) {
		return
	}
	total := partCount(f.SizeBytes, f.PartSize)
	if len(body.Parts) == 0 || len(body.Parts) > maxPartsPerCall {
		fail(w, http.StatusBadRequest, "invalid_parts")
		return
	}
	urls := make(map[int32]string, len(body.Parts))
	for _, n := range body.Parts {
		if int64(n) < 1 || int64(n) > total {
			fail(w, http.StatusBadRequest, "invalid_parts")
			return
		}
		url, err := h.Store.PresignPart(r.Context(), f.ObjectKey, f.UploadID, n, partURLTTL)
		if err != nil {
			fail(w, http.StatusInternalServerError, "internal")
			return
		}
		urls[n] = url
	}
	reply(w, http.StatusOK, map[string]any{"urls": urls, "expiresInSeconds": int(partURLTTL.Seconds())})
}

// complete trusts the bucket, not the client: it lists the parts that really
// arrived, checks count and total size against what was declared, then seals.
func (h *Handlers) complete(w http.ResponseWriter, r *http.Request, u auth.User) {
	f, ok := h.load(w, r, u)
	if !ok {
		return
	}
	if f.Status != "uploading" {
		fail(w, http.StatusConflict, "not_uploading")
		return
	}
	parts, err := h.Store.ListParts(r.Context(), f.ObjectKey, f.UploadID)
	if err != nil {
		fail(w, http.StatusBadGateway, "storage_unavailable")
		return
	}
	var sum int64
	for i, p := range parts {
		if int64(p.Number) != int64(i)+1 {
			fail(w, http.StatusConflict, "missing_parts")
			return
		}
		sum += p.Size
	}
	if int64(len(parts)) != partCount(f.SizeBytes, f.PartSize) || sum != f.SizeBytes {
		fail(w, http.StatusConflict, "size_mismatch")
		return
	}
	if err := h.Store.Complete(r.Context(), f.ObjectKey, f.UploadID, parts); err != nil {
		log.Printf("complete multipart: %v", err)
		fail(w, http.StatusBadGateway, "storage_unavailable")
		return
	}
	if err := h.Repo.MarkReady(r.Context(), f.ID); err != nil {
		fail(w, http.StatusInternalServerError, "internal")
		return
	}
	f.Status = "ready"
	reply(w, http.StatusOK, map[string]any{"file": f})
}

func (h *Handlers) abort(w http.ResponseWriter, r *http.Request, u auth.User) {
	f, ok := h.load(w, r, u)
	if !ok {
		return
	}
	if f.Status != "uploading" {
		fail(w, http.StatusConflict, "not_uploading")
		return
	}
	if err := h.Store.Abort(r.Context(), f.ObjectKey, f.UploadID); err != nil {
		fail(w, http.StatusBadGateway, "storage_unavailable")
		return
	}
	_ = h.Repo.MarkDeleted(r.Context(), f.ID)
	reply(w, http.StatusOK, map[string]string{"status": "aborted"})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request, u auth.User) {
	fs, err := h.Repo.ListReady(r.Context(), u.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "internal")
		return
	}
	if fs == nil {
		fs = []File{}
	}
	reply(w, http.StatusOK, map[string]any{"files": fs})
}

func (h *Handlers) deleteFile(w http.ResponseWriter, r *http.Request, u auth.User) {
	f, ok := h.load(w, r, u)
	if !ok {
		return
	}
	var err error
	if f.Status == "uploading" {
		err = h.Store.Abort(r.Context(), f.ObjectKey, f.UploadID)
	} else {
		err = h.Store.Delete(r.Context(), f.ObjectKey)
	}
	if err != nil {
		fail(w, http.StatusBadGateway, "storage_unavailable")
		return
	}
	_ = h.Repo.MarkDeleted(r.Context(), f.ID)
	reply(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handlers) createShare(w http.ResponseWriter, r *http.Request, u auth.User) {
	f, ok := h.load(w, r, u)
	if !ok {
		return
	}
	if f.Status != "ready" {
		fail(w, http.StatusConflict, "not_ready")
		return
	}
	var body struct {
		ExpiresInHours int `json:"expiresInHours"`
	}
	if r.ContentLength != 0 && !decode(w, r, &body) {
		return
	}
	ttl := defaultShare
	if body.ExpiresInHours != 0 {
		ttl = time.Duration(body.ExpiresInHours) * time.Hour
	}
	if ttl < minShare || ttl > maxShare {
		fail(w, http.StatusBadRequest, "invalid_expiry")
		return
	}
	s := Share{Token: randomToken(), FileID: f.ID, ExpiresAt: h.now().Add(ttl)}
	if err := h.Repo.CreateShare(r.Context(), s); err != nil {
		fail(w, http.StatusInternalServerError, "internal")
		return
	}
	reply(w, http.StatusCreated, map[string]any{
		"token": s.Token, "url": strings.TrimRight(h.PublicBaseURL, "/") + "/s/" + s.Token, "expiresAt": s.ExpiresAt,
	})
}

func (h *Handlers) shareFile(ctx context.Context, token string) (File, bool) {
	s, err := h.Repo.GetShare(ctx, token)
	if err != nil || !h.now().Before(s.ExpiresAt) {
		return File{}, false
	}
	f, err := h.Repo.Get(ctx, s.FileID)
	if err != nil || f.Status != "ready" {
		return File{}, false
	}
	return f, true
}

func (h *Handlers) shareInfo(w http.ResponseWriter, r *http.Request) {
	f, ok := h.shareFile(r.Context(), r.PathValue("token"))
	if !ok {
		fail(w, http.StatusNotFound, "not_found")
		return
	}
	reply(w, http.StatusOK, map[string]any{"name": f.Name, "sizeBytes": f.SizeBytes, "contentType": f.ContentType})
}

func (h *Handlers) shareDownload(w http.ResponseWriter, r *http.Request) {
	f, ok := h.shareFile(r.Context(), r.PathValue("token"))
	if !ok {
		fail(w, http.StatusNotFound, "not_found")
		return
	}
	url, err := h.Store.PresignDownload(r.Context(), f.ObjectKey, f.Name, downloadURLTTL)
	if err != nil {
		fail(w, http.StatusInternalServerError, "internal")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, url, http.StatusFound)
}

func (h *Handlers) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

// cleanName keeps only the base name and rejects empty or control-char names.
func cleanName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimSpace(path.Base(name))
	if name == "." || name == ".." || name == "/" || name == "" || len(name) > 255 {
		return ""
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	return name
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func randomToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		fail(w, http.StatusBadRequest, "bad_request")
		return false
	}
	return true
}

func fail(w http.ResponseWriter, status int, code string) {
	reply(w, status, map[string]string{"error": code})
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
