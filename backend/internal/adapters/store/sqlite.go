package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
)

type SQLite struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	google_sub TEXT UNIQUE NOT NULL,
	email TEXT,
	name TEXT,
	avatar_url TEXT,
	created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	created_at DATETIME NOT NULL,
	expires_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS stories (
	id TEXT PRIMARY KEY,
	user_id TEXT REFERENCES users(id),
	topic TEXT NOT NULL,
	title TEXT,
	style_prefix TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_stories_user ON stories(user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS pages (
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	idx INTEGER NOT NULL,
	narrative TEXT NOT NULL,
	image_prompt TEXT,
	image_url TEXT,
	image_provider TEXT,
	audio_url TEXT,
	depth_url TEXT,
	choices TEXT NOT NULL,
	is_ending INTEGER NOT NULL DEFAULT 0,
	ending_type TEXT,
	running_summary TEXT,
	created_at DATETIME NOT NULL,
	PRIMARY KEY (story_id, idx)
);
`

func NewSQLite(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("sqlite: pragma: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("sqlite: migrate: %w", err)
	}
	if _, err := db.Exec(`ALTER TABLE pages ADD COLUMN audio_url TEXT`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, fmt.Errorf("sqlite: migrate audio_url: %w", err)
	}
	if _, err := db.Exec(`ALTER TABLE pages ADD COLUMN depth_url TEXT`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, fmt.Errorf("sqlite: migrate depth_url: %w", err)
	}
	if _, err := db.Exec(`ALTER TABLE pages ADD COLUMN language TEXT`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, fmt.Errorf("sqlite: migrate language: %w", err)
	}
	if _, err := db.Exec(`ALTER TABLE stories ADD COLUMN title TEXT`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, fmt.Errorf("sqlite: migrate stories.title: %w", err)
	}
	return &SQLite{db: db}, nil
}

func (s *SQLite) Close() error { return s.db.Close() }

// --- StoryStore ---

func (s *SQLite) Create(ctx context.Context, st domain.Story) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO stories (id, user_id, topic, style_prefix, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		st.ID, nullable(st.UserID), st.Topic, string(st.StylePrefix), st.CreatedAt, st.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("sqlite: story create: %w", err)
	}
	return nil
}

func (s *SQLite) Get(ctx context.Context, id string) (domain.Story, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, COALESCE(user_id,''), topic, COALESCE(title,''), style_prefix, created_at, updated_at FROM stories WHERE id = ?`, id)
	var st domain.Story
	var style string
	if err := row.Scan(&st.ID, &st.UserID, &st.Topic, &st.Title, &style, &st.CreatedAt, &st.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Story{}, domain.ErrStoryNotFound
		}
		return domain.Story{}, fmt.Errorf("sqlite: story get: %w", err)
	}
	st.StylePrefix = domain.StylePrefix(style)
	pages, err := s.loadPages(ctx, id)
	if err != nil {
		return domain.Story{}, err
	}
	st.Pages = pages
	return st, nil
}

func (s *SQLite) loadPages(ctx context.Context, storyID string) ([]domain.Page, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT idx, narrative, COALESCE(image_prompt,''), COALESCE(image_url,''), COALESCE(image_provider,''), COALESCE(audio_url,''), COALESCE(depth_url,''), COALESCE(language,''), choices, is_ending, COALESCE(ending_type,''), COALESCE(running_summary,''), created_at
		 FROM pages WHERE story_id = ? ORDER BY idx ASC`, storyID)
	if err != nil {
		return nil, fmt.Errorf("sqlite: pages load: %w", err)
	}
	defer rows.Close()
	var pages []domain.Page
	for rows.Next() {
		var p domain.Page
		var choicesJSON, endingType string
		var isEndingI int
		if err := rows.Scan(&p.Index, &p.Narrative, &p.ImagePrompt, &p.ImageURL, &p.ImageProvider, &p.AudioURL, &p.DepthURL, &p.Language, &choicesJSON, &isEndingI, &endingType, &p.RunningSummary, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("sqlite: pages scan: %w", err)
		}
		p.IsEnding = isEndingI != 0
		p.EndingType = domain.EndingType(endingType)
		if err := json.Unmarshal([]byte(choicesJSON), &p.Choices); err != nil {
			return nil, fmt.Errorf("sqlite: pages choices json: %w", err)
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

func (s *SQLite) AppendPage(ctx context.Context, storyID string, p domain.Page) error {
	choicesJSON, err := json.Marshal(p.Choices)
	if err != nil {
		return fmt.Errorf("sqlite: choices marshal: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin: %w", err)
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM stories WHERE id = ?`, storyID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrStoryNotFound
		}
		return fmt.Errorf("sqlite: story check: %w", err)
	}
	isEndingI := 0
	if p.IsEnding {
		isEndingI = 1
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO pages (story_id, idx, narrative, image_prompt, image_url, image_provider, language, choices, is_ending, ending_type, running_summary, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		storyID, p.Index, p.Narrative, p.ImagePrompt, p.ImageURL, p.ImageProvider, p.Language, string(choicesJSON), isEndingI, string(p.EndingType), p.RunningSummary, p.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("sqlite: page insert: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE stories SET updated_at = ? WHERE id = ?`, time.Now().UTC(), storyID); err != nil {
		return fmt.Errorf("sqlite: story touch: %w", err)
	}
	return tx.Commit()
}

func (s *SQLite) ListByUser(ctx context.Context, userID string, limit int) ([]domain.Story, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT s.id, s.user_id, s.topic, COALESCE(s.title,''), s.style_prefix, s.created_at, s.updated_at
		 FROM stories s WHERE s.user_id = ? ORDER BY s.updated_at DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list stories: %w", err)
	}
	defer rows.Close()
	var out []domain.Story
	for rows.Next() {
		var st domain.Story
		var style string
		if err := rows.Scan(&st.ID, &st.UserID, &st.Topic, &st.Title, &style, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, fmt.Errorf("sqlite: list scan: %w", err)
		}
		st.StylePrefix = domain.StylePrefix(style)
		out = append(out, st)
	}
	return out, rows.Err()
}

func (s *SQLite) UpdatePageAudio(ctx context.Context, storyID string, idx int, audioURL string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE pages SET audio_url = ? WHERE story_id = ? AND idx = ?`,
		audioURL, storyID, idx)
	if err != nil {
		return fmt.Errorf("sqlite: update audio: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrStoryNotFound
	}
	return nil
}

func (s *SQLite) UpdateStoryTitle(ctx context.Context, storyID, title string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE stories SET title = ?, updated_at = ? WHERE id = ?`,
		title, time.Now().UTC(), storyID)
	if err != nil {
		return fmt.Errorf("sqlite: update title: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrStoryNotFound
	}
	return nil
}

func (s *SQLite) UpdatePageDepth(ctx context.Context, storyID string, idx int, depthURL string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE pages SET depth_url = ? WHERE story_id = ? AND idx = ?`,
		depthURL, storyID, idx)
	if err != nil {
		return fmt.Errorf("sqlite: update depth: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrStoryNotFound
	}
	return nil
}

func (s *SQLite) AttachUser(ctx context.Context, storyID, userID string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE stories SET user_id = ?, updated_at = ? WHERE id = ? AND user_id IS NULL`,
		userID, time.Now().UTC(), storyID)
	if err != nil {
		return fmt.Errorf("sqlite: attach user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrForbidden
	}
	return nil
}

// --- UserStore ---

func (s *SQLite) GetByGoogleSub(ctx context.Context, sub string) (domain.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, google_sub, COALESCE(email,''), COALESCE(name,''), COALESCE(avatar_url,''), created_at FROM users WHERE google_sub = ?`, sub)
	var u domain.User
	err := row.Scan(&u.ID, &u.GoogleSub, &u.Email, &u.Name, &u.AvatarURL, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, sql.ErrNoRows
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("sqlite: user by sub: %w", err)
	}
	return u, nil
}

func (s *SQLite) CreateUser(ctx context.Context, u domain.User) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, google_sub, email, name, avatar_url, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		u.ID, u.GoogleSub, u.Email, u.Name, u.AvatarURL, u.CreatedAt)
	if err != nil {
		return fmt.Errorf("sqlite: user create: %w", err)
	}
	return nil
}

func (s *SQLite) GetUser(ctx context.Context, id string) (domain.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, google_sub, COALESCE(email,''), COALESCE(name,''), COALESCE(avatar_url,''), created_at FROM users WHERE id = ?`, id)
	var u domain.User
	err := row.Scan(&u.ID, &u.GoogleSub, &u.Email, &u.Name, &u.AvatarURL, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrUnauthorized
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("sqlite: user get: %w", err)
	}
	return u, nil
}

// --- SessionStore ---

func (s *SQLite) CreateSession(ctx context.Context, sess domain.Session) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		sess.ID, sess.UserID, sess.CreatedAt, sess.ExpiresAt)
	if err != nil {
		return fmt.Errorf("sqlite: session create: %w", err)
	}
	return nil
}

func (s *SQLite) GetSession(ctx context.Context, id string) (domain.Session, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, created_at, expires_at FROM sessions WHERE id = ?`, id)
	var sess domain.Session
	err := row.Scan(&sess.ID, &sess.UserID, &sess.CreatedAt, &sess.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Session{}, domain.ErrSessionInvalid
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("sqlite: session get: %w", err)
	}
	if time.Now().After(sess.ExpiresAt) {
		return domain.Session{}, domain.ErrSessionInvalid
	}
	return sess, nil
}

func (s *SQLite) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("sqlite: session delete: %w", err)
	}
	return nil
}

// --- Wrapper types so the same SQLite struct satisfies multiple ports ---
// Users returns a ports.UserStore-compatible view.
func (s *SQLite) Users() *UserView { return &UserView{s: s} }

// Sessions returns a ports.SessionStore-compatible view.
func (s *SQLite) Sessions() *SessionView { return &SessionView{s: s} }

type UserView struct{ s *SQLite }

func (u *UserView) GetByGoogleSub(ctx context.Context, sub string) (domain.User, error) {
	return u.s.GetByGoogleSub(ctx, sub)
}
func (u *UserView) Create(ctx context.Context, user domain.User) error { return u.s.CreateUser(ctx, user) }
func (u *UserView) Get(ctx context.Context, id string) (domain.User, error) {
	return u.s.GetUser(ctx, id)
}

type SessionView struct{ s *SQLite }

func (s *SessionView) Create(ctx context.Context, sess domain.Session) error {
	return s.s.CreateSession(ctx, sess)
}
func (s *SessionView) Get(ctx context.Context, id string) (domain.Session, error) {
	return s.s.GetSession(ctx, id)
}
func (s *SessionView) Delete(ctx context.Context, id string) error { return s.s.DeleteSession(ctx, id) }

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
