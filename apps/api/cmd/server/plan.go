package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"filetransfer.ir/api/internal/auth"
	"filetransfer.ir/api/internal/plans"
	"filetransfer.ir/api/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const planUsage = `usage:
  filetransfer plan show <phone>
  filetransfer plan activate <phone> <basic|max|ultra> [days] [note]
  filetransfer plan cancel <phone>

activate starts now, or when the current plan ends if one is still running. days defaults to 30.
cancel ends the current plan now; the files are deleted after the grace period.`

// planCommand lets the owner manage plans by hand until online payment exists.
func planCommand(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, planUsage)
		return 2
	}
	phone, err := auth.NormalizePhone(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid phone number")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "postgres:", err)
		return 1
	}
	defer pool.Close()
	for _, m := range []string{migrations.Users, migrations.Files, migrations.Plans} {
		if _, err := pool.Exec(ctx, m); err != nil {
			fmt.Fprintln(os.Stderr, "migrate:", err)
			return 1
		}
	}

	switch args[0] {
	case "show":
		err = planShow(ctx, pool, phone)
	case "activate":
		err = planActivate(ctx, pool, phone, args[2:])
	case "cancel":
		err = planCancel(ctx, pool, phone)
	default:
		fmt.Fprintln(os.Stderr, planUsage)
		return 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

type subRow struct {
	id, plan        string
	starts, expires time.Time
}

// current returns the user's subscription that has not ended yet, if any.
func current(ctx context.Context, pool *pgxpool.Pool, userID string) (subRow, bool, error) {
	var s subRow
	err := pool.QueryRow(ctx, `
		SELECT id::text, plan, starts_at, expires_at FROM subscriptions
		WHERE user_id = $1::uuid AND expires_at > now()
		ORDER BY expires_at DESC LIMIT 1`, userID).Scan(&s.id, &s.plan, &s.starts, &s.expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, false, nil
	}
	return s, err == nil, err
}

func userID(ctx context.Context, pool *pgxpool.Pool, phone string, create bool) (string, error) {
	var id string
	if create {
		// The owner may activate a plan for an invited user before their first login.
		if _, err := pool.Exec(ctx, `INSERT INTO users (phone) VALUES ($1) ON CONFLICT (phone) DO NOTHING`, phone); err != nil {
			return "", err
		}
	}
	err := pool.QueryRow(ctx, `SELECT id::text FROM users WHERE phone = $1`, phone).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("no user with phone %s", phone)
	}
	return id, err
}

func planShow(ctx context.Context, pool *pgxpool.Pool, phone string) error {
	id, err := userID(ctx, pool, phone, false)
	if err != nil {
		return err
	}
	var storage int64
	if err := pool.QueryRow(ctx, `SELECT coalesce(sum(size_bytes), 0)::bigint FROM files WHERE owner_id = $1::uuid AND status IN ('uploading', 'ready')`, id).Scan(&storage); err != nil {
		return err
	}
	fmt.Printf("user     %s\nstorage  %s used\n", phone, gb(storage))
	s, ok, err := current(ctx, pool, id)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("plan     none")
		return nil
	}
	p, _ := plans.Get(s.plan)
	var traffic int64
	if err := pool.QueryRow(ctx, `SELECT coalesce(sum(size_bytes), 0)::bigint FROM downloads WHERE owner_id = $1::uuid AND created_at >= $2`, id, s.starts).Scan(&traffic); err != nil {
		return err
	}
	fmt.Printf("plan     %s\nstarts   %s\nexpires  %s\nquota    %s storage, %s traffic\ntraffic  %s used\n",
		s.plan, s.starts.Format(time.RFC3339), s.expires.Format(time.RFC3339), gb(p.StorageBytes), gb(p.DownloadBytes), gb(traffic))
	return nil
}

func planActivate(ctx context.Context, pool *pgxpool.Pool, phone string, args []string) error {
	if len(args) < 1 {
		return errors.New("which plan? basic, max or ultra")
	}
	p, ok := plans.Get(args[0])
	if !ok {
		return fmt.Errorf("unknown plan %q", args[0])
	}
	days := int(plans.Period.Hours() / 24)
	note := "activated by hand"
	if len(args) > 1 {
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 || n > 366 {
			return errors.New("days must be a number from 1 to 366")
		}
		days = n
	}
	if len(args) > 2 {
		note = args[2]
	}
	id, err := userID(ctx, pool, phone, true)
	if err != nil {
		return err
	}
	starts := time.Now()
	if cur, ok, err := current(ctx, pool, id); err != nil {
		return err
	} else if ok {
		if cur.plan != p.ID {
			return fmt.Errorf("user is on %s until %s; cancel it first to switch plan", cur.plan, cur.expires.Format(time.RFC3339))
		}
		starts = cur.expires
	}
	expires := starts.AddDate(0, 0, days)
	if _, err := pool.Exec(ctx, `INSERT INTO subscriptions (user_id, plan, starts_at, expires_at, note) VALUES ($1::uuid, $2, $3, $4, $5)`, id, p.ID, starts, expires, note); err != nil {
		return err
	}
	fmt.Printf("activated %s for %s: %s to %s\n", p.ID, phone, starts.Format(time.RFC3339), expires.Format(time.RFC3339))
	return nil
}

func planCancel(ctx context.Context, pool *pgxpool.Pool, phone string) error {
	id, err := userID(ctx, pool, phone, false)
	if err != nil {
		return err
	}
	// End everything that has not ended, including a renewal queued behind the current plan.
	tag, err := pool.Exec(ctx, `
		UPDATE subscriptions SET cancelled_at = now(), expires_at = now(), starts_at = least(starts_at, now())
		WHERE user_id = $1::uuid AND expires_at > now()`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("no running plan to cancel")
	}
	fmt.Printf("cancelled the plan of %s; files are deleted %d days from now\n", phone, int(plans.Grace.Hours()/24))
	return nil
}

func gb(n int64) string { return fmt.Sprintf("%.2f GB", float64(n)/float64(1<<30)) }
