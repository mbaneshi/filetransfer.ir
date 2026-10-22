package files

import (
	"context"
	"log"
	"time"

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

// RunJanitor sweeps uploads older than maxAge every interval until ctx ends.
func RunJanitor(ctx context.Context, repo Repo, store storage.Store, interval, maxAge time.Duration) {
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
		}
	}
}
