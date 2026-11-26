package files

import (
	"context"
	"log"
	"time"

	"filetransfer.ir/api/internal/plans"
	"filetransfer.ir/api/internal/storage"
)

// SweepStale aborts multipart uploads that were never completed. Abandoned
// parts keep consuming (and billing) space until they are aborted.
func SweepStale(ctx context.Context, repo Repo, store storage.Store, olderThan time.Time) int {
	stale, err := repo.StaleUploads(ctx, olderThan, 100)
	if err != nil {
		log.Printf("sweep: list stale uploads: %v", err)
		return 0
	}
	done := 0
	for _, f := range stale {
		if err := store.Abort(ctx, f.ObjectKey, f.UploadID); err != nil {
			log.Printf("sweep: abort %s: %v", f.ID, err)
			continue
		}
		if err := repo.MarkDeleted(ctx, f.ID); err != nil {
			log.Printf("sweep: mark deleted %s: %v", f.ID, err)
			continue
		}
		done++
	}
	return done
}

// SweepExpired deletes files past their plan's retention, and every file of an owner whose
// last plan ended more than the grace period ago. It returns how many files it removed.
func SweepExpired(ctx context.Context, repo Repo, store storage.Store, now time.Time) int {
	expired, err := repo.ExpiredFiles(ctx, now, 200)
	if err != nil {
		log.Printf("sweep: list expired files: %v", err)
	}
	lapsed, err := repo.LapsedFiles(ctx, now.Add(-plans.Grace), 200)
	if err != nil {
		log.Printf("sweep: list lapsed files: %v", err)
	}
	done := 0
	seen := map[string]bool{}
	for _, f := range append(expired, lapsed...) {
		if seen[f.ID] {
			continue
		}
		seen[f.ID] = true
		var err error
		if f.Status == "uploading" {
			err = store.Abort(ctx, f.ObjectKey, f.UploadID)
		} else {
			err = store.Delete(ctx, f.ObjectKey)
		}
		if err != nil {
			log.Printf("sweep: remove %s: %v", f.ID, err)
			continue
		}
		if err := repo.MarkDeleted(ctx, f.ID); err != nil {
			log.Printf("sweep: mark deleted %s: %v", f.ID, err)
			continue
		}
		done++
	}
	return done
}

// RunJanitor sweeps uploads older than maxAge every interval until ctx ends.
// With enforcePlans it also removes files past retention and files of lapsed plans.
func RunJanitor(ctx context.Context, repo Repo, store storage.Store, interval, maxAge time.Duration, enforcePlans bool) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n := SweepStale(ctx, repo, store, time.Now().Add(-maxAge)); n > 0 {
				log.Printf("sweep: aborted %d stale uploads", n)
			}
			if enforcePlans {
				if n := SweepExpired(ctx, repo, store, time.Now()); n > 0 {
					log.Printf("sweep: deleted %d expired files", n)
				}
			}
		}
	}
}
