package files

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"filetransfer.ir/api/internal/auth"
	"filetransfer.ir/api/internal/storage"
)

// ---- fakes -----------------------------------------------------------------

type fakeStore struct {
	uploads   map[string][]storage.Part // uploadID -> parts present in the bucket
	completed map[string]bool
	aborted   map[string]bool
	deleted   map[string]bool
	n         int
}

func newFakeStore() *fakeStore {
	return &fakeStore{uploads: map[string][]storage.Part{}, completed: map[string]bool{}, aborted: map[string]bool{}, deleted: map[string]bool{}}
}

func (s *fakeStore) CreateMultipart(context.Context, string, string) (string, error) {
	s.n++
	id := fmt.Sprintf("up-%d", s.n)
	s.uploads[id] = nil
	return id, nil
}
func (s *fakeStore) PresignPart(_ context.Context, key, id string, part int32, _ time.Duration) (string, error) {
	return fmt.Sprintf("https://bucket/%s?uploadId=%s&partNumber=%d", key, id, part), nil
}
func (s *fakeStore) ListParts(_ context.Context, _, id string) ([]storage.Part, error) {
	return s.uploads[id], nil
}
func (s *fakeStore) Complete(_ context.Context, _, id string, _ []storage.Part) error {
	s.completed[id] = true
	return nil
}
func (s *fakeStore) Abort(_ context.Context, _, id string) error { s.aborted[id] = true; return nil }
func (s *fakeStore) PresignDownload(_ context.Context, key, name string, _ time.Duration) (string, error) {
	return "https://bucket/" + key + "?dl=" + name, nil
}
func (s *fakeStore) Delete(_ context.Context, key string) error { s.deleted[key] = true; return nil }

type memRepo struct {
	files  map[string]File
	shares map[string]Share
	n      int
	now    func() time.Time
}

func newMemRepo() *memRepo {
	return &memRepo{files: map[string]File{}, shares: map[string]Share{}, now: time.Now}
}
func (m *memRepo) Create(_ context.Context, f File) (File, error) {
	m.n++
	f.ID, f.Status, f.CreatedAt = fmt.Sprintf("f%d", m.n), "uploading", m.now()
	m.files[f.ID] = f
	return f, nil
}
func (m *memRepo) Get(_ context.Context, id string) (File, error) {
	f, ok := m.files[id]
	if !ok || f.Status == "deleted" {
		return File{}, ErrNotFound
	}
	return f, nil
}
func (m *memRepo) ListReady(_ context.Context, owner string) ([]File, error) {
	var out []File
	for _, f := range m.files {
		if f.OwnerID == owner && f.Status == "ready" {
			out = append(out, f)
		}
	}
	return out, nil
}
func (m *memRepo) MarkReady(_ context.Context, id string) error {
	f := m.files[id]
	f.Status, f.UploadID = "ready", ""
	m.files[id] = f
	return nil
}
func (m *memRepo) MarkDeleted(_ context.Context, id string) error {
	f := m.files[id]
	f.Status = "deleted"
	m.files[id] = f
	return nil
}
func (m *memRepo) StaleUploads(_ context.Context, before time.Time, _ int) ([]File, error) {
	var out []File
	for _, f := range m.files {
		if f.Status == "uploading" && f.CreatedAt.Before(before) {
			out = append(out, f)
		}
	}
	return out, nil
}
func (m *memRepo) CreateShare(_ context.Context, s Share) error { m.shares[s.Token] = s; return nil }
func (m *memRepo) GetShare(_ context.Context, t string) (Share, error) {
	s, ok := m.shares[t]
	if !ok {
		return Share{}, ErrNotFound
	}
	return s, nil
}

// whoAmI reads the user from an X-User header so tests can switch identity.
type whoAmI struct{}

func (whoAmI) CurrentUser(r *http.Request) (auth.User, bool) {
	id := r.Header.Get("X-User")
	return auth.User{ID: id, Phone: "+98" + id}, id != ""
}

// ---- harness ---------------------------------------------------------------

type env struct {
	t     *testing.T
	mux   *http.ServeMux
	store *fakeStore
	repo  *memRepo
	now   time.Time
}

func newEnv(t *testing.T) *env {
	e := &env{t: t, store: newFakeStore(), repo: newMemRepo(), now: time.Now()}
	h := &Handlers{Repo: e.repo, Store: e.store, Users: whoAmI{}, MaxFileSize: 50 << 30,
		PublicBaseURL: "https://filetransfer.ir/", Now: func() time.Time { return e.now }}
	e.mux = http.NewServeMux()
	h.Register(e.mux)
	return e
}

func (e *env) do(user, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != "" {
		req.Header.Set("X-User", user)
	}
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *env) json(rec *httptest.ResponseRecorder) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		e.t.Fatalf("bad json %q: %v", rec.Body, err)
	}
	return m
}

func (e *env) startUpload(user string, size int64) string {
	rec := e.do(user, "POST", "/api/v1/uploads", fmt.Sprintf(`{"name":"a.zip","size":%d,"contentType":"application/zip"}`, size))
	if rec.Code != http.StatusCreated {
		e.t.Fatalf("init: %d %s", rec.Code, rec.Body)
	}
	return e.json(rec)["id"].(string)
}

// land pretends the browser PUT every part: the bucket now holds them.
func (e *env) land(fileID string) {
	f := e.repo.files[fileID]
	var parts []storage.Part
	left := f.SizeBytes
	for n := int32(1); left > 0; n++ {
		sz := min(f.PartSize, left)
		parts = append(parts, storage.Part{Number: n, ETag: fmt.Sprintf("e%d", n), Size: sz})
		left -= sz
	}
	e.store.uploads[f.UploadID] = parts
}

// ---- tests -----------------------------------------------------------------

func TestPartSizeKeepsEveryFileUnderPartLimit(t *testing.T) {
	for _, size := range []int64{1, 64 << 20, 5 << 30, 50 << 30, 5 << 40} {
		ps := PartSizeFor(size)
		if ps < minPartSize || partCount(size, ps) > maxParts {
			t.Errorf("size %d -> part %d, %d parts", size, ps, partCount(size, ps))
		}
	}
	if got := partCount(50<<30, PartSizeFor(50<<30)); got != 800 {
		t.Errorf("50 GB should be 800 parts, got %d", got)
	}
}

func TestRequiresLogin(t *testing.T) {
	e := newEnv(t)
	for _, c := range [][2]string{{"POST", "/api/v1/uploads"}, {"GET", "/api/v1/files"}, {"DELETE", "/api/v1/files/f1"}} {
		if rec := e.do("", c[0], c[1], `{}`); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d", c[0], c[1], rec.Code)
		}
	}
}

func TestInitValidation(t *testing.T) {
	e := newEnv(t)
	cases := map[string]int{
		`{"name":"a","size":0}`:                       http.StatusBadRequest,
		`{"name":"","size":10}`:                       http.StatusBadRequest,
		`{"name":"../","size":10}`:                    http.StatusBadRequest,
		fmt.Sprintf(`{"name":"a","size":%d}`, 51<<30): http.StatusRequestEntityTooLarge,
		`not json`:                    http.StatusBadRequest,
		`{"name":"ok.bin","size":10}`: http.StatusCreated,
	}
	for body, want := range cases {
		if rec := e.do("u1", "POST", "/api/v1/uploads", body); rec.Code != want {
			t.Errorf("%s -> %d, want %d (%s)", body, rec.Code, want, rec.Body)
		}
	}
}

func TestCleanNameStripsPaths(t *testing.T) {
	for in, want := range map[string]string{"report.pdf": "report.pdf", "../../etc/passwd": "passwd", `C:\\x\\y.zip`: "y.zip", "  گزارش.pdf ": "گزارش.pdf"} {
		if got := cleanName(in); got != want {
			t.Errorf("cleanName(%q) = %q, want %q", in, got, want)
		}
	}
	if cleanName("bad\x00name") != "" {
		t.Error("control characters accepted")
	}
}

func TestOwnershipIsolation(t *testing.T) {
	e := newEnv(t)
	id := e.startUpload("u1", 10)
	for _, c := range [][2]string{{"GET", "/api/v1/uploads/" + id}, {"POST", "/api/v1/uploads/" + id + "/parts"}, {"POST", "/api/v1/uploads/" + id + "/complete"}, {"DELETE", "/api/v1/uploads/" + id}} {
		if rec := e.do("u2", c[0], c[1], `{"parts":[1]}`); rec.Code != http.StatusNotFound {
			t.Errorf("other user %s %s = %d, want 404", c[0], c[1], rec.Code)
		}
	}
}

func TestPresignPartsBounds(t *testing.T) {
	e := newEnv(t)
	id := e.startUpload("u1", 3*minPartSize) // 3 parts
	ok := e.do("u1", "POST", "/api/v1/uploads/"+id+"/parts", `{"parts":[1,3]}`)
	if ok.Code != http.StatusOK || len(e.json(ok)["urls"].(map[string]any)) != 2 {
		t.Fatalf("presign: %d %s", ok.Code, ok.Body)
	}
	for _, body := range []string{`{"parts":[0]}`, `{"parts":[4]}`, `{"parts":[]}`, `{"parts":[-1]}`} {
		if rec := e.do("u1", "POST", "/api/v1/uploads/"+id+"/parts", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s -> %d", body, rec.Code)
		}
	}
}

func TestResumeListsUploadedParts(t *testing.T) {
	e := newEnv(t)
	id := e.startUpload("u1", 3*minPartSize)
	e.store.uploads[e.repo.files[id].UploadID] = []storage.Part{{Number: 1, ETag: "e1", Size: minPartSize}}
	rec := e.do("u1", "GET", "/api/v1/uploads/"+id, "")
	up := e.json(rec)["uploaded"].([]any)
	if rec.Code != http.StatusOK || len(up) != 1 {
		t.Fatalf("status: %d %s", rec.Code, rec.Body)
	}
}

func TestCompleteChecksWhatReallyArrived(t *testing.T) {
	e := newEnv(t)
	id := e.startUpload("u1", 3*minPartSize)

	if rec := e.do("u1", "POST", "/api/v1/uploads/"+id+"/complete", ``); rec.Code != http.StatusConflict {
		t.Fatalf("complete with nothing uploaded = %d", rec.Code)
	}
	up := e.repo.files[id].UploadID
	e.store.uploads[up] = []storage.Part{{Number: 1, Size: minPartSize}, {Number: 3, Size: minPartSize}, {Number: 2, Size: 1}}
	if rec := e.do("u1", "POST", "/api/v1/uploads/"+id+"/complete", ``); rec.Code != http.StatusConflict {
		t.Fatalf("complete with wrong sizes = %d", rec.Code)
	}
	if e.store.completed[up] {
		t.Fatal("sealed a bad upload")
	}

	e.land(id)
	rec := e.do("u1", "POST", "/api/v1/uploads/"+id+"/complete", ``)
	if rec.Code != http.StatusOK || !e.store.completed[up] || e.repo.files[id].Status != "ready" {
		t.Fatalf("complete: %d %s", rec.Code, rec.Body)
	}
	if rec := e.do("u1", "POST", "/api/v1/uploads/"+id+"/complete", ``); rec.Code != http.StatusConflict {
		t.Fatalf("double complete = %d", rec.Code)
	}
}

func TestAbortAndDelete(t *testing.T) {
	e := newEnv(t)
	id := e.startUpload("u1", 10)
	up := e.repo.files[id].UploadID
	if rec := e.do("u1", "DELETE", "/api/v1/uploads/"+id, ""); rec.Code != http.StatusOK || !e.store.aborted[up] {
		t.Fatalf("abort: %d", rec.Code)
	}

	id2 := e.startUpload("u1", 10)
	e.land(id2)
	e.do("u1", "POST", "/api/v1/uploads/"+id2+"/complete", ``)
	key := e.repo.files[id2].ObjectKey
	if rec := e.do("u1", "DELETE", "/api/v1/files/"+id2, ""); rec.Code != http.StatusOK || !e.store.deleted[key] {
		t.Fatalf("delete: %d", rec.Code)
	}
	if rec := e.do("u1", "GET", "/api/v1/files", ""); len(e.json(rec)["files"].([]any)) != 0 {
		t.Fatalf("deleted file still listed: %s", rec.Body)
	}
}

func TestShareLifecycle(t *testing.T) {
	e := newEnv(t)
	id := e.startUpload("u1", 10)
	if rec := e.do("u1", "POST", "/api/v1/files/"+id+"/shares", ``); rec.Code != http.StatusConflict {
		t.Fatalf("share before ready = %d", rec.Code)
	}
	e.land(id)
	e.do("u1", "POST", "/api/v1/uploads/"+id+"/complete", ``)

	if rec := e.do("u1", "POST", "/api/v1/files/"+id+"/shares", `{"expiresInHours":0}`); rec.Code != http.StatusCreated {
		t.Fatalf("default expiry: %d", rec.Code)
	}
	if rec := e.do("u1", "POST", "/api/v1/files/"+id+"/shares", `{"expiresInHours":99999}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("huge expiry accepted: %d", rec.Code)
	}
	if rec := e.do("u2", "POST", "/api/v1/files/"+id+"/shares", ``); rec.Code != http.StatusNotFound {
		t.Fatalf("share someone else's file = %d", rec.Code)
	}

	rec := e.do("u1", "POST", "/api/v1/files/"+id+"/shares", `{"expiresInHours":2}`)
	out := e.json(rec)
	token := out["token"].(string)
	if !strings.HasPrefix(out["url"].(string), "https://filetransfer.ir/s/") {
		t.Fatalf("share url %v", out["url"])
	}

	// Public: no login needed.
	if info := e.do("", "GET", "/api/v1/shares/"+token, ""); info.Code != http.StatusOK || e.json(info)["name"] != "a.zip" {
		t.Fatalf("info: %d %s", info.Code, info.Body)
	}
	dl := e.do("", "GET", "/api/v1/shares/"+token+"/download", "")
	if dl.Code != http.StatusFound || !strings.Contains(dl.Header().Get("Location"), "dl=a.zip") {
		t.Fatalf("download: %d %q", dl.Code, dl.Header().Get("Location"))
	}

	e.now = e.now.Add(3 * time.Hour)
	if dl := e.do("", "GET", "/api/v1/shares/"+token+"/download", ""); dl.Code != http.StatusNotFound {
		t.Fatalf("expired share still works: %d", dl.Code)
	}
	if info := e.do("", "GET", "/api/v1/shares/nope", ""); info.Code != http.StatusNotFound {
		t.Fatalf("unknown token: %d", info.Code)
	}
}

func TestSweepAbortsOnlyStaleUploads(t *testing.T) {
	e := newEnv(t)
	old := e.startUpload("u1", 10)
	fresh := e.startUpload("u1", 10)
	f := e.repo.files[old]
	f.CreatedAt = time.Now().Add(-48 * time.Hour)
	e.repo.files[old] = f

	n := SweepStale(context.Background(), e.repo, e.store, time.Now().Add(-24*time.Hour))
	if n != 1 || !e.store.aborted[f.UploadID] || e.repo.files[old].Status != "deleted" {
		t.Fatalf("sweep n=%d", n)
	}
	if e.repo.files[fresh].Status != "uploading" || e.store.aborted[e.repo.files[fresh].UploadID] {
		t.Fatal("fresh upload was swept")
	}
}
