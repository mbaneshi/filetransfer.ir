package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/redis/go-redis/v9"
	"time"
)

const SessionTTL = 30 * 24 * time.Hour

var ErrNoSession = errors.New("no valid session")

// Sessions maps opaque random tokens to user IDs. Only a hash of the token is
// stored, so a Redis dump cannot be replayed as cookies.
type Sessions struct{ rdb *redis.Client }

func NewSessions(rdb *redis.Client) *Sessions { return &Sessions{rdb: rdb} }

func (s *Sessions) Create(ctx context.Context, userID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	if err := s.rdb.Set(ctx, sessionKey(token), userID, SessionTTL).Err(); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Sessions) Get(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrNoSession
	}
	id, err := s.rdb.Get(ctx, sessionKey(token)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNoSession
	}
	return id, err
}

func (s *Sessions) Delete(ctx context.Context, token string) error {
	return s.rdb.Del(ctx, sessionKey(token)).Err()
}

func sessionKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "sess:" + hex.EncodeToString(sum[:])
}
