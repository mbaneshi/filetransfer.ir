package files

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"filetransfer.ir/api/internal/plans"
)

const gib = int64(1) << 30

func (e *env) tryUpload(user string, size int64) int {
	return e.do(user, "POST", "/api/v1/uploads", fmt.Sprintf(`{"name":"a.zip","size":%d}`, size)).Code
}

// ready uploads a file and completes it, returning its id.
func (e *env) ready(user string, size int64) string {
	id := e.startUpload(user, size)
	e.land(id)
	if rec := e.do(user, "POST", "/api/v1/uploads/"+id+"/complete", ``); rec.Code != http.StatusOK {
		e.t.Fatalf("complete: %d %s", rec.Code, rec.Body)
	}
	return id
}

func (e *env) share(user, id, body string) (token string, expires time.Time) {
	rec := e.do(user, "POST", "/api/v1/files/"+id+"/shares", body)
	if rec.Code != http.StatusCreated {
		e.t.Fatalf("share: %d %s", rec.Code, rec.Body)
	}
	out := e.json(rec)
	expires, err := time.Parse(time.RFC3339, out["expiresAt"].(string))
	if err != nil {
		e.t.Fatal(err)
	}
	return out["token"].(string), expires
}

func TestUploadNeedsAnActivePlan(t *testing.T) {
	e := newPlanEnv(t)
	if code := e.tryUpload("u1", 10); code != http.StatusPaymentRequired {
		t.Fatalf("upload without a plan = %d, want 402", code)
	}
	e.activate("u1", "basic")
	if code := e.tryUpload("u1", 10); code != http.StatusCreated {
		t.Fatalf("upload with a plan = %d", code)
	}
	if code := e.tryUpload("u2", 10); code != http.StatusPaymentRequired {
		t.Fatalf("another user rides on u1's plan: %d", code)
	}
	e.now = e.now.Add(plans.Period)
	if code := e.tryUpload("u1", 10); code != http.StatusPaymentRequired {
		t.Fatalf("upload at the moment of expiry = %d, want 402", code)
	}
}

func TestPlanLimitsFileSizeAndStorage(t *testing.T) {
	e := newPlanEnv(t)
	e.activate("u1", "basic") // 5 GB per file, 10 GB storage
	if code := e.tryUpload("u1", 5*gib+1); code != http.StatusRequestEntityTooLarge {
		t.Fatalf("file over the plan's largest size = %d, want 413", code)
	}
	if code := e.tryUpload("u1", 5*gib); code != http.StatusCreated {
		t.Fatalf("file at the plan's largest size = %d", code)
	}
	// An upload in progress already holds its space.
	if code := e.tryUpload("u1", 5*gib); code != http.StatusCreated {
		t.Fatalf("second 5 GB file = %d", code)
	}
	if code := e.tryUpload("u1", 1); code != http.StatusConflict {
		t.Fatalf("upload into a full quota = %d, want 409", code)
	}
	// Deleting a file frees its space.
	for id := range e.repo.files {
		e.do("u1", "DELETE", "/api/v1/files/"+id, "")
		break
	}
	if code := e.tryUpload("u1", gib); code != http.StatusCreated {
		t.Fatalf("upload after freeing space = %d", code)
	}
}

func TestShareCannotOutliveRetention(t *testing.T) {
	e := newPlanEnv(t)
	e.activate("u1", "basic") // 7 days retention
	id := e.ready("u1", 10)
	want := e.now.Add(7 * 24 * time.Hour)
	if got := e.repo.files[id].ExpiresAt; !got.Equal(want) {
		t.Fatalf("file retention date = %v, want %v", got, want)
	}
	if _, exp := e.share("u1", id, `{"expiresInHours":720}`); !exp.Equal(want) {
		t.Fatalf("30-day link on a 7-day plan expires %v, want it cut to %v", exp, want)
	}
	if _, exp := e.share("u1", id, `{"expiresInHours":2}`); !exp.Equal(e.now.Add(2 * time.Hour)) {
		t.Fatalf("short link changed: %v", exp)
	}
}

func TestDownloadTrafficIsCountedAndCapped(t *testing.T) {
	e := newPlanEnv(t)
	e.activate("u1", "basic") // 20 GB traffic
	id := e.ready("u1", 5*gib)
	token, _ := e.share("u1", id, ``)
	for i := 1; i <= 4; i++ {
		if dl := e.do("", "GET", "/api/v1/shares/"+token+"/download", ""); dl.Code != http.StatusFound {
			t.Fatalf("download %d = %d", i, dl.Code)
		}
	}
	dl := e.do("", "GET", "/api/v1/shares/"+token+"/download", "")
	if dl.Code != http.StatusForbidden || e.json(dl)["error"] != "traffic_exhausted" {
		t.Fatalf("download past the traffic quota = %d %s", dl.Code, dl.Body)
	}
	if info := e.json(e.do("", "GET", "/api/v1/shares/"+token, "")); info["blocked"] != "traffic_exhausted" {
		t.Fatalf("share info blocked = %v", info["blocked"])
	}
	acct := e.json(e.do("u1", "GET", "/api/v1/account", ""))
	if acct["trafficUsedBytes"].(float64) != float64(20*gib) || acct["storageUsedBytes"].(float64) != float64(5*gib) {
		t.Fatalf("account usage: %v", acct)
	}
}

func TestLinksWorkThroughGraceThenStop(t *testing.T) {
	e := newPlanEnv(t)
	e.activate("u1", "ultra") // 30 days retention, so retention does not interfere
	id := e.ready("u1", 10)
	token, _ := e.share("u1", id, `{"expiresInHours":720}`)

	e.now = e.now.Add(plans.Period - time.Hour) // still inside the link's and the file's life
	if dl := e.do("", "GET", "/api/v1/shares/"+token+"/download", ""); dl.Code != http.StatusFound {
		t.Fatalf("download before expiry = %d", dl.Code)
	}
	if info := e.json(e.do("", "GET", "/api/v1/shares/"+token, "")); info["blocked"] != "" {
		t.Fatalf("blocked before expiry: %v", info["blocked"])
	}
}

func TestPlanExpiryBlocksDownloadsAfterGrace(t *testing.T) {
	e := newPlanEnv(t)
	// A 10-day plan with a 30-day link, so only the plan's end can stop the download.
	e.repo.subs = append(e.repo.subs, plans.Subscription{UserID: "u1", PlanID: "ultra", StartsAt: e.now, ExpiresAt: e.now.Add(10 * 24 * time.Hour)})
	id := e.ready("u1", 10)
	token, _ := e.share("u1", id, `{"expiresInHours":720}`)

	e.now = e.now.Add(10*24*time.Hour + plans.Grace - time.Minute)
	if dl := e.do("", "GET", "/api/v1/shares/"+token+"/download", ""); dl.Code != http.StatusFound {
		t.Fatalf("download in the grace days = %d", dl.Code)
	}
	if code := e.tryUpload("u1", 10); code != http.StatusPaymentRequired {
		t.Fatalf("upload in the grace days = %d, want 402", code)
	}
	e.now = e.now.Add(2 * time.Minute)
	dl := e.do("", "GET", "/api/v1/shares/"+token+"/download", "")
	if dl.Code != http.StatusForbidden || e.json(dl)["error"] != "plan_expired" {
		t.Fatalf("download after grace = %d %s", dl.Code, dl.Body)
	}
}

func TestSweepExpiredRemovesRetentionAndLapsedFiles(t *testing.T) {
	e := newPlanEnv(t)
	e.activate("u1", "basic") // 7 days retention, 30-day plan
	old := e.ready("u1", 10)

	e.now = e.now.Add(6 * 24 * time.Hour)
	fresh := e.ready("u1", 10)
	if n := SweepExpired(t.Context(), e.repo, e.store, e.now); n != 0 {
		t.Fatalf("swept %d files before any retention date", n)
	}

	e.now = e.now.Add(2 * 24 * time.Hour) // day 8: the first file is past retention
	if n := SweepExpired(t.Context(), e.repo, e.store, e.now); n != 1 {
		t.Fatalf("swept %d files at day 8, want 1", n)
	}
	if e.repo.files[old].Status != "deleted" || !e.store.deleted[e.repo.files[old].ObjectKey] {
		t.Fatal("file past retention was not removed from the bucket")
	}
	if e.repo.files[fresh].Status != "ready" {
		t.Fatal("file inside retention was removed")
	}
}

func TestSweepExpiredRemovesEverythingAfterGrace(t *testing.T) {
	e := newPlanEnv(t)
	e.repo.subs = append(e.repo.subs, plans.Subscription{UserID: "u1", PlanID: "ultra", StartsAt: e.now, ExpiresAt: e.now.Add(5 * 24 * time.Hour)})
	done := e.ready("u1", 10)
	half := e.startUpload("u1", 10)

	// A user who never had a plan keeps the files they uploaded before plans existed.
	e.repo.files["legacy"] = File{ID: "legacy", OwnerID: "u9", ObjectKey: "files/legacy", SizeBytes: 10, Status: "ready"}

	e.now = e.now.Add(5*24*time.Hour + plans.Grace - time.Hour)
	if n := SweepExpired(t.Context(), e.repo, e.store, e.now); n != 0 {
		t.Fatalf("swept %d files inside the grace days", n)
	}
	e.now = e.now.Add(2 * time.Hour)
	if n := SweepExpired(t.Context(), e.repo, e.store, e.now); n != 2 {
		t.Fatalf("swept %d files after grace, want 2", n)
	}
	if !e.store.deleted[e.repo.files[done].ObjectKey] || !e.store.aborted[e.repo.files[half].UploadID] {
		t.Fatal("lapsed files were not removed from the bucket")
	}
	if e.repo.files["legacy"].Status != "ready" {
		t.Fatal("file of a user without any plan was removed")
	}
}

func TestPlansOffKeepsOldBehaviour(t *testing.T) {
	e := newEnv(t)
	id := e.ready("u1", 10)
	if !e.repo.files[id].ExpiresAt.IsZero() {
		t.Fatal("file got a retention date with plans off")
	}
	token, _ := e.share("u1", id, ``)
	if dl := e.do("", "GET", "/api/v1/shares/"+token+"/download", ""); dl.Code != http.StatusFound {
		t.Fatalf("download with plans off = %d", dl.Code)
	}
	if len(e.repo.downloads) != 0 {
		t.Fatal("traffic recorded with plans off")
	}
}

func TestPlanCatalogueIsPublic(t *testing.T) {
	e := newPlanEnv(t)
	out := e.json(e.do("", "GET", "/api/v1/plans", ""))
	if len(out["plans"].([]any)) != 3 || out["periodDays"].(float64) != 30 || out["graceDays"].(float64) != 3 {
		t.Fatalf("catalogue: %v", out)
	}
}
