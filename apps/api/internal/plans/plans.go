// Package plans holds the three fixed prepaid plans. The numbers come from
// docs/superpowers/specs/2026-11-25-prepaid-plans-and-legal-proposal.md, section 3.1.
// The web app shows the same table (apps/web/app/lib/plans.ts); change both together.
package plans

import "time"

const (
	gib = int64(1) << 30
	day = 24 * time.Hour

	// Period is how long one purchase lasts.
	Period = 30 * day
	// Grace is how long links keep working after a plan ends, before the files are deleted.
	Grace = 3 * day
)

type Plan struct {
	ID            string `json:"id"`
	StorageBytes  int64  `json:"storageBytes"`
	DownloadBytes int64  `json:"downloadBytes"`
	MaxFileBytes  int64  `json:"maxFileBytes"`
	RetentionDays int    `json:"retentionDays"`
	PriceToman    int64  `json:"priceToman"`
}

func (p Plan) Retention() time.Duration { return time.Duration(p.RetentionDays) * day }

var catalog = []Plan{
	{ID: "basic", StorageBytes: 10 * gib, DownloadBytes: 20 * gib, MaxFileBytes: 5 * gib, RetentionDays: 7, PriceToman: 65_000},
	{ID: "max", StorageBytes: 50 * gib, DownloadBytes: 100 * gib, MaxFileBytes: 20 * gib, RetentionDays: 14, PriceToman: 315_000},
	{ID: "ultra", StorageBytes: 200 * gib, DownloadBytes: 400 * gib, MaxFileBytes: 50 * gib, RetentionDays: 30, PriceToman: 1_255_000},
}

func All() []Plan { return append([]Plan(nil), catalog...) }

func Get(id string) (Plan, bool) {
	for _, p := range catalog {
		if p.ID == id {
			return p, true
		}
	}
	return Plan{}, false
}

// Subscription is one purchased (or hand-activated) period of a plan.
// Cancelling a subscription moves ExpiresAt to the moment of cancellation.
type Subscription struct {
	ID        string
	UserID    string
	PlanID    string
	StartsAt  time.Time
	ExpiresAt time.Time
}

// Active reports whether uploads are allowed at t.
func (s Subscription) Active(t time.Time) bool {
	return !t.Before(s.StartsAt) && t.Before(s.ExpiresAt)
}

// Serving reports whether the owner's links still work at t: the plan is active or in its grace days.
func (s Subscription) Serving(t time.Time) bool {
	return !t.Before(s.StartsAt) && t.Before(s.ExpiresAt.Add(Grace))
}
