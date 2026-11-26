package files

import (
	"context"
	"os"
	"testing"
	"time"

	"filetransfer.ir/api/internal/plans"
	"filetransfer.ir/api/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPGPlanQueries runs the plan SQL against a real Postgres. It is skipped unless
// TEST_DATABASE_URL points at a throwaway database.
func TestPGPlanQueries(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	// Applying every migration twice proves they are safe to re-run at each start.
	for range 2 {
		for _, m := range []string{migrations.Users, migrations.Files, migrations.Plans} {
			if _, err := pool.Exec(ctx, m); err != nil {
				t.Fatalf("migrate: %v", err)
			}
		}
	}
	repo := PG{Pool: pool}
	now := time.Now().UTC().Truncate(time.Second)
	day := 24 * time.Hour

	user := func(phone string) string {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO users (phone) VALUES ($1) RETURNING id::text`, phone).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	sub := func(uid, plan string, starts, expires time.Time) {
		if _, err := pool.Exec(ctx, `INSERT INTO subscriptions (user_id, plan, starts_at, expires_at) VALUES ($1::uuid, $2, $3, $4)`, uid, plan, starts, expires); err != nil {
			t.Fatal(err)
		}
	}
	file := func(uid string, size int64, expires time.Time, ready bool) File {
		f, err := repo.Create(ctx, File{OwnerID: uid, ObjectKey: "files/" + randomHex(8), Name: "a", SizeBytes: size, ContentType: "x/y", UploadID: "up", PartSize: 1, ExpiresAt: expires})
		if err != nil {
			t.Fatal(err)
		}
		if ready {
			if err := repo.MarkReady(ctx, f.ID); err != nil {
				t.Fatal(err)
			}
		}
		return f
	}
	ids := func(fs []File) map[string]bool {
		m := map[string]bool{}
		for _, f := range fs {
			m[f.ID] = true
		}
		return m
	}
	tag := randomHex(4) // phones must be unique across runs

	// Active user: an old plan, then the current one.
	active := user("+98a" + tag)
	sub(active, "basic", now.Add(-40*day), now.Add(-10*day))
	sub(active, "max", now.Add(-day), now.Add(29*day))
	sub(active, "ultra", now.Add(29*day), now.Add(59*day)) // queued renewal, not started
	got, err := repo.LatestSubscription(ctx, active, now)
	if err != nil || got.PlanID != "max" {
		t.Fatalf("latest subscription = %+v, %v; want the max plan", got, err)
	}
	if _, err := repo.LatestSubscription(ctx, user("+98n"+tag), now); err != ErrNotFound {
		t.Fatalf("user without a plan: %v", err)
	}

	kept := file(active, 100, now.Add(5*day), true)
	expired := file(active, 30, now.Add(-time.Hour), true)
	file(active, 7, time.Time{}, false) // upload in progress, no retention date
	if n, err := repo.StorageUsed(ctx, active); err != nil || n != 137 {
		t.Fatalf("storage used = %d, %v; want 137", n, err)
	}
	if back, err := repo.Get(ctx, kept.ID); err != nil || !back.ExpiresAt.Equal(now.Add(5*day)) {
		t.Fatalf("retention date did not round-trip: %v %v", back.ExpiresAt, err)
	}

	for range 3 {
		if err := repo.RecordDownload(ctx, Download{FileID: kept.ID, OwnerID: active, ShareToken: "tok", SizeBytes: 100}); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := repo.TrafficUsed(ctx, active, now.Add(-day)); err != nil || n != 300 {
		t.Fatalf("traffic used = %d, %v; want 300", n, err)
	}
	if n, _ := repo.TrafficUsed(ctx, active, now.Add(time.Hour)); n != 0 {
		t.Fatalf("traffic counted from before the period: %d", n)
	}

	// Lapsed user: the only plan ended 4 days ago, past the 3-day grace.
	lapsed := user("+98l" + tag)
	sub(lapsed, "basic", now.Add(-34*day), now.Add(-4*day))
	gone := file(lapsed, 5, now.Add(20*day), true)
	// Grace user: plan ended 2 days ago.
	grace := user("+98g" + tag)
	sub(grace, "basic", now.Add(-32*day), now.Add(-2*day))
	inGrace := file(grace, 5, now.Add(20*day), true)
	// Legacy user: files but never a plan.
	legacy := file(user("+98o"+tag), 5, time.Time{}, true)

	exp, err := repo.ExpiredFiles(ctx, now, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if e := ids(exp); !e[expired.ID] || e[kept.ID] || e[legacy.ID] {
		t.Fatalf("expired files wrong: %v", e)
	}
	lap, err := repo.LapsedFiles(ctx, now.Add(-plans.Grace), 1000)
	if err != nil {
		t.Fatal(err)
	}
	if l := ids(lap); !l[gone.ID] || l[inGrace.ID] || l[legacy.ID] || l[kept.ID] {
		t.Fatalf("lapsed files wrong: %v", l)
	}
}
