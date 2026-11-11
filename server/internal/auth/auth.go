// Package auth handles password hashing, session tokens and login rate limiting.
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PBKDF2-SHA256 from the standard library (Go 1.24+), at OWASP's
// recommended iteration count. Stored as: pbkdf2-sha256$<iter>$<salt>$<hash>
const (
	iterations = 600_000
	keyLen     = 32
	prefix     = "pbkdf2-sha256"
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, iterations, keyLen)
	if err != nil {
		return "", err
	}
	enc := base64.RawStdEncoding
	return fmt.Sprintf("%s$%d$%s$%s", prefix, iterations, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

func CheckPassword(stored, password string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != prefix {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 1 {
		return false
	}
	enc := base64.RawStdEncoding
	salt, err1 := enc.DecodeString(parts[2])
	want, err2 := enc.DecodeString(parts[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

// dummyHash is checked when a login email doesn't exist, so a missing
// account takes as long to reject as a wrong password.
var dummyHash, _ = HashPassword("easy-plan-timing-equalizer")

func CheckDummy(password string) { CheckPassword(dummyHash, password) }

// NewToken returns a random session token (sent to the browser) and its
// SHA-256 hash (the only form stored in the database).
func NewToken() (token, hash string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err) // system entropy source is broken
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token)
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Limiter allows `max` attempts per key within `window`.
type Limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	hits   map[string][]time.Time
}

func NewLimiter(max int, window time.Duration) *Limiter {
	return &Limiter{max: max, window: window, hits: map[string][]time.Time{}}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cut := now.Add(-l.window)
	h := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cut) {
			h = append(h, t)
		}
	}
	if len(h) >= l.max {
		l.hits[key] = h
		return false
	}
	l.hits[key] = append(h, now)
	if len(l.hits) > 10_000 { // keep memory bounded under abuse
		for k, v := range l.hits {
			if len(v) == 0 || v[len(v)-1].Before(cut) {
				delete(l.hits, k)
			}
		}
	}
	return true
}
