// Package api is Easy Plan's JSON HTTP API, mounted under /api/.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"easyplan/internal/auth"
	"easyplan/internal/store"
)

const (
	cookieName = "ep_session"
	sessionTTL = 30 * 24 * time.Hour
)

type Config struct {
	AllowRegistration bool
	// SecureCookies forces the Secure flag. When false it is still set
	// automatically for requests that arrive over HTTPS.
	SecureCookies bool
}

type Server struct {
	db      *store.Store
	cfg     Config
	limiter *auth.Limiter
	mux     *http.ServeMux
}

func NewServer(db *store.Store, cfg Config) *Server {
	s := &Server{db: db, cfg: cfg, limiter: auth.NewLimiter(10, time.Minute), mux: http.NewServeMux()}

	s.mux.HandleFunc("GET /api/config", s.handleConfig)
	s.mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("GET /api/me", s.requireUser(s.handleMe))

	s.mux.HandleFunc("GET /api/calendars", s.requireUser(s.handleListCalendars))
	s.mux.HandleFunc("POST /api/calendars", s.requireUser(s.handleCreateCalendar))
	s.mux.HandleFunc("GET /api/calendars/{id}", s.requireUser(s.withCalendar(s.handleGetCalendar)))
	s.mux.HandleFunc("PATCH /api/calendars/{id}", s.requireUser(s.withCalendar(s.handleRenameCalendar)))
	s.mux.HandleFunc("DELETE /api/calendars/{id}", s.requireUser(s.withCalendar(s.handleDeleteCalendar)))
	s.mux.HandleFunc("PUT /api/calendars/{id}/event-types", s.requireUser(s.withCalendar(s.handleReplaceTypes)))
	s.mux.HandleFunc("PUT /api/calendars/{id}/days/{day}", s.requireUser(s.withCalendar(s.handleSetDay)))

	s.mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, http.StatusNotFound, "Not found.")
	})
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	// CSRF guard: browsers can't send a cross-site request with a JSON body
	// without a CORS preflight, which this server never approves.
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			writeErr(w, http.StatusUnsupportedMediaType, "Requests must be sent as JSON.")
			return
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	s.mux.ServeHTTP(w, r)
}

// ---------- Context helpers ----------

type ctxKey int

const (
	userKey ctxKey = iota
	calKey
)

func userFrom(r *http.Request) *store.User { return r.Context().Value(userKey).(*store.User) }
func calendarFrom(r *http.Request) *store.Calendar {
	return r.Context().Value(calKey).(*store.Calendar)
}

func (s *Server) requireUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil || c.Value == "" {
			writeErr(w, http.StatusUnauthorized, "Please log in.")
			return
		}
		u, err := s.db.SessionUser(r.Context(), auth.HashToken(c.Value))
		if err != nil {
			if !errors.Is(err, store.ErrNotFound) {
				serverErr(w, err)
				return
			}
			writeErr(w, http.StatusUnauthorized, "Your session has ended. Please log in again.")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	}
}

func (s *Server) withCalendar(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeErr(w, http.StatusNotFound, "Calendar not found.")
			return
		}
		cal, err := s.db.GetCalendar(r.Context(), userFrom(r).ID, id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeErr(w, http.StatusNotFound, "Calendar not found.")
				return
			}
			serverErr(w, err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), calKey, cal)))
	}
}

// ---------- Auth ----------

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"registration": s.cfg.AllowRegistration})
}

type credentials struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.AllowRegistration {
		writeErr(w, http.StatusForbidden, "Registration is turned off on this server.")
		return
	}
	if !s.limiter.Allow("reg:" + clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "Too many attempts. Wait a minute and try again.")
		return
	}
	var in credentials
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	switch {
	case in.Name == "" || utf8.RuneCountInString(in.Name) > 80:
		writeErr(w, http.StatusBadRequest, "Enter your name (up to 80 characters).")
		return
	case !validEmail(in.Email):
		writeErr(w, http.StatusBadRequest, "Enter a valid email address.")
		return
	case utf8.RuneCountInString(in.Password) < 8:
		writeErr(w, http.StatusBadRequest, "Use a password of at least 8 characters.")
		return
	case len(in.Password) > 256:
		writeErr(w, http.StatusBadRequest, "Use a password of at most 256 characters.")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		serverErr(w, err)
		return
	}
	u, err := s.db.CreateUser(r.Context(), in.Name, in.Email, hash)
	if errors.Is(err, store.ErrEmailTaken) {
		writeErr(w, http.StatusConflict, "That email already has an account. Log in instead.")
		return
	}
	if err != nil {
		serverErr(w, err)
		return
	}
	// Every new account starts with one calendar so the first screen isn't empty.
	if _, err := s.db.CreateCalendar(r.Context(), u.ID, "My calendar", StarterTypes); err != nil {
		serverErr(w, err)
		return
	}
	if !s.startSession(w, r, u.ID) {
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.Allow("login:" + clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "Too many attempts. Wait a minute and try again.")
		return
	}
	var in credentials
	if !decode(w, r, &in) {
		return
	}
	u, err := s.db.UserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(in.Email)))
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		serverErr(w, err)
		return
	}
	if u == nil {
		auth.CheckDummy(in.Password)
		writeErr(w, http.StatusUnauthorized, "That email and password don't match an account.")
		return
	}
	if !auth.CheckPassword(u.PasswordHash, in.Password) {
		writeErr(w, http.StatusUnauthorized, "That email and password don't match an account.")
		return
	}
	if !s.startSession(w, r, u.ID) {
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID int64) bool {
	token, hash := auth.NewToken()
	expires := time.Now().Add(sessionTTL)
	if err := s.db.CreateSession(r.Context(), hash, userID, expires); err != nil {
		serverErr(w, err)
		return false
	}
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: token, Path: "/", Expires: expires,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secure(r),
	})
	return true
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		_ = s.db.DeleteSession(r.Context(), auth.HashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secure(r),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) secure(r *http.Request) bool {
	return s.cfg.SecureCookies || r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, userFrom(r))
}

// ---------- Calendars ----------

// StarterTypes seed a calendar when the user asks for the example events.
var StarterTypes = []store.EventType{
	{Name: "Work", Emoji: "💼", Count: "month"},
	{Name: "Holiday", Emoji: "🏖️", Count: "year"},
	{Name: "National holiday", Emoji: "🎌", Count: "year"},
	{Name: "Weekend", Emoji: "🛋️", Count: "none"},
	{Name: "New job", Emoji: "🎉", Count: "none"},
}

func (s *Server) handleListCalendars(w http.ResponseWriter, r *http.Request) {
	cals, err := s.db.ListCalendars(r.Context(), userFrom(r).ID)
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cals)
}

func (s *Server) handleCreateCalendar(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string `json:"name"`
		Starter bool   `json:"starter"`
	}
	if !decode(w, r, &in) {
		return
	}
	name, ok := cleanCalendarName(w, in.Name)
	if !ok {
		return
	}
	var starter []store.EventType
	if in.Starter {
		starter = StarterTypes
	}
	cal, err := s.db.CreateCalendar(r.Context(), userFrom(r).ID, name, starter)
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, cal)
}

func cleanCalendarName(w http.ResponseWriter, name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 60 {
		writeErr(w, http.StatusBadRequest, "Give the calendar a name (up to 60 characters).")
		return "", false
	}
	return name, true
}

type calendarDetail struct {
	store.Calendar
	EventTypes []store.EventType  `json:"eventTypes"`
	Days       map[string][]int64 `json:"days"`
}

func (s *Server) calendarDetail(r *http.Request, cal *store.Calendar) (*calendarDetail, error) {
	types, err := s.db.ListEventTypes(r.Context(), cal.ID)
	if err != nil {
		return nil, err
	}
	days, err := s.db.Days(r.Context(), cal.ID)
	if err != nil {
		return nil, err
	}
	return &calendarDetail{Calendar: *cal, EventTypes: types, Days: days}, nil
}

func (s *Server) handleGetCalendar(w http.ResponseWriter, r *http.Request) {
	d, err := s.calendarDetail(r, calendarFrom(r))
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleRenameCalendar(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	name, ok := cleanCalendarName(w, in.Name)
	if !ok {
		return
	}
	cal := calendarFrom(r)
	if err := s.db.RenameCalendar(r.Context(), userFrom(r).ID, cal.ID, name); err != nil {
		serverErr(w, err)
		return
	}
	cal.Name = name
	writeJSON(w, http.StatusOK, cal)
}

func (s *Server) handleDeleteCalendar(w http.ResponseWriter, r *http.Request) {
	if err := s.db.DeleteCalendar(r.Context(), userFrom(r).ID, calendarFrom(r).ID); err != nil {
		serverErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleReplaceTypes(w http.ResponseWriter, r *http.Request) {
	var in []store.EventType
	if !decode(w, r, &in) {
		return
	}
	if len(in) > 100 {
		writeErr(w, http.StatusBadRequest, "A calendar can have up to 100 events.")
		return
	}
	for i := range in {
		t := &in[i]
		t.Name = strings.TrimSpace(t.Name)
		t.Emoji = strings.TrimSpace(t.Emoji)
		if t.Name == "" || utf8.RuneCountInString(t.Name) > 40 {
			writeErr(w, http.StatusBadRequest, "Every event needs a name (up to 40 characters).")
			return
		}
		if t.Emoji == "" {
			t.Emoji = "•"
		}
		if utf8.RuneCountInString(t.Emoji) > 8 {
			writeErr(w, http.StatusBadRequest, "Use a single emoji for each event.")
			return
		}
		if t.Count != "month" && t.Count != "year" {
			t.Count = "none"
		}
	}
	cal := calendarFrom(r)
	if err := s.db.ReplaceEventTypes(r.Context(), cal.ID, in); err != nil {
		serverErr(w, err)
		return
	}
	d, err := s.calendarDetail(r, cal)
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

var dayRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func (s *Server) handleSetDay(w http.ResponseWriter, r *http.Request) {
	day := r.PathValue("day")
	if _, err := time.Parse("2006-01-02", day); err != nil || !dayRe.MatchString(day) {
		writeErr(w, http.StatusBadRequest, "Use a date like 2026-11-11.")
		return
	}
	var in struct {
		EventTypeIDs []int64 `json:"eventTypeIds"`
	}
	if !decode(w, r, &in) {
		return
	}
	stored, err := s.db.SetDay(r.Context(), calendarFrom(r).ID, day, in.EventTypeIDs)
	if err != nil {
		serverErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"day": day, "eventTypeIds": stored})
}

// ---------- Helpers ----------

func validEmail(e string) bool {
	if len(e) > 254 || !strings.Contains(e, "@") {
		return false
	}
	a, err := mail.ParseAddress(e)
	return err == nil && a.Address == e && strings.Contains(e[strings.LastIndex(e, "@"):], ".")
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "The request body isn't valid JSON.")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func serverErr(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeErr(w, http.StatusInternalServerError, "Something went wrong on the server. Try again.")
}
