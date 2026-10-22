package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrCooldown        = errors.New("a code was sent recently")
	ErrRateLimited     = errors.New("too many code requests")
	ErrInvalidCode     = errors.New("invalid or expired code")
	ErrTooManyAttempts = errors.New("too many wrong attempts")
)

const (
	codeTTL        = 2 * time.Minute
	resendCooldown = 60 * time.Second
	phoneWindow    = 10 * time.Minute
	phoneMaxPerWin = 3
	ipWindow       = time.Hour
	ipMaxPerWin    = 20
	maxVerifyTries = 5
)

// OTP issues and verifies short-lived, single-use login codes. Codes are
// stored only as an HMAC so a Redis dump does not reveal live codes.
type OTP struct {
	rdb    *redis.Client
	sender Sender
	secret []byte
}

func NewOTP(rdb *redis.Client, sender Sender, secret []byte) *OTP {
	return &OTP{rdb: rdb, sender: sender, secret: secret}
}

// Request generates a code for phone (E.164) and sends it.
func (o *OTP) Request(ctx context.Context, phone, ip string) error {
	if ok, err := o.rdb.SetNX(ctx, "otp:cool:"+phone, 1, resendCooldown).Result(); err != nil {
		return err
	} else if !ok {
		return ErrCooldown
	}
	if err := o.hit(ctx, "otp:rl:phone:"+phone, phoneWindow, phoneMaxPerWin); err != nil {
		return err
	}
	if err := o.hit(ctx, "otp:rl:ip:"+ip, ipWindow, ipMaxPerWin); err != nil {
		return err
	}

	code, err := newCode()
	if err != nil {
		return err
	}
	key := "otp:code:" + phone
	pipe := o.rdb.TxPipeline()
	pipe.Del(ctx, key)
	pipe.HSet(ctx, key, "h", o.mac(phone, code), "a", 0)
	pipe.Expire(ctx, key, codeTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	if err := o.sender.SendCode(ctx, phone, code); err != nil {
		o.rdb.Del(ctx, key)
		return fmt.Errorf("send code: %w", err)
	}
	return nil
}

// Verify checks code for phone. A correct code is consumed exactly once.
func (o *OTP) Verify(ctx context.Context, phone, code string) error {
	key := "otp:code:" + phone
	attempts, err := o.rdb.HIncrBy(ctx, key, "a", 1).Result()
	if err != nil {
		return err
	}
	stored, err := o.rdb.HGet(ctx, key, "h").Result()
	if errors.Is(err, redis.Nil) {
		o.rdb.Del(ctx, key) // HIncrBy may have created a bare hash
		return ErrInvalidCode
	}
	if err != nil {
		return err
	}
	if attempts > maxVerifyTries {
		o.rdb.Del(ctx, key)
		return ErrTooManyAttempts
	}
	if !hmac.Equal([]byte(stored), []byte(o.mac(phone, code))) {
		if attempts >= maxVerifyTries {
			o.rdb.Del(ctx, key)
			return ErrTooManyAttempts
		}
		return ErrInvalidCode
	}
	// DEL returns 1 for exactly one concurrent caller.
	if n, err := o.rdb.Del(ctx, key).Result(); err != nil {
		return err
	} else if n != 1 {
		return ErrInvalidCode
	}
	return nil
}

func (o *OTP) hit(ctx context.Context, key string, window time.Duration, max int64) error {
	n, err := o.rdb.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	if n == 1 {
		o.rdb.Expire(ctx, key, window)
	}
	if n > max {
		return ErrRateLimited
	}
	return nil
}

func (o *OTP) mac(phone, code string) string {
	m := hmac.New(sha256.New, o.secret)
	m.Write([]byte(phone + ":" + code))
	return hex.EncodeToString(m.Sum(nil))
}

func newCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
