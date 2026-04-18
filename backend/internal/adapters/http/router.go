package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/ridopark/solo-adeventure/backend/internal/app"
	"github.com/ridopark/solo-adeventure/backend/internal/domain"
)

type Router struct {
	svc         *app.Service
	log         zerolog.Logger
	frontendURL string
	cookieDomain string
	secure      bool
}

type RouterConfig struct {
	CORSOrigin   string
	FrontendURL  string
	CookieDomain string
	Secure       bool
	AudioDir     string
}

func NewRouter(svc *app.Service, log zerolog.Logger, cfg RouterConfig) http.Handler {
	r := &Router{
		svc:          svc,
		log:          log.With().Str("component", "http.router").Logger(),
		frontendURL:  cfg.FrontendURL,
		cookieDomain: cfg.CookieDomain,
		secure:       cfg.Secure,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", r.health)
	mux.HandleFunc("POST /stories", r.startStory)
	mux.HandleFunc("GET /stories/{id}", r.getStory)
	mux.HandleFunc("POST /stories/{id}/choose", r.chooseStory)
	mux.HandleFunc("POST /stories/{id}/claim", r.claimStory)
	mux.HandleFunc("POST /stories/{id}/pages/{seq}/speech", r.generateSpeech)
	mux.HandleFunc("GET /stories", r.myStories)
	mux.HandleFunc("POST /images", r.generateImage)
	mux.HandleFunc("POST /visit", r.visit)
	mux.HandleFunc("GET /auth/google/start", r.authStart)
	mux.HandleFunc("GET /auth/google/callback", r.authCallback)
	mux.HandleFunc("POST /auth/logout", r.authLogout)
	mux.HandleFunc("GET /auth/me", r.authMe)

	if cfg.AudioDir != "" {
		mux.Handle("GET /audio/", http.StripPrefix("/audio/", http.FileServer(http.Dir(cfg.AudioDir))))
	}

	var h http.Handler = mux
	h = session(svc, r.log, h)
	h = cors(cfg.CORSOrigin, h)
	h = requestLog(r.log, h)
	h = recovery(r.log, h)
	return h
}

func (r *Router) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (r *Router) startStory(w http.ResponseWriter, req *http.Request) {
	var in domain.StartStoryInput
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if in.Topic == "" {
		writeError(w, http.StatusBadRequest, "topic required")
		return
	}
	out, err := r.svc.StartStory(req.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUnsafeTopic):
			writeError(w, http.StatusBadRequest, "that topic isn't a good fit for this gamebook -- please try another")
		case errors.Is(err, domain.ErrTopicLength):
			writeError(w, http.StatusBadRequest, "topic must be between 3 and 200 characters")
		default:
			r.log.Error().Err(err).Msg("start story")
			writeError(w, http.StatusInternalServerError, "start failed")
		}
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (r *Router) getStory(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	story, err := r.svc.GetStory(req.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrStoryNotFound) {
			writeError(w, http.StatusNotFound, "story not found")
			return
		}
		r.log.Error().Err(err).Msg("get story")
		writeError(w, http.StatusInternalServerError, "get failed")
		return
	}
	writeJSON(w, http.StatusOK, story)
}

func (r *Router) chooseStory(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	var in domain.ProgressInput
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	in.StoryID = id
	out, err := r.svc.ProgressStory(req.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUnauthorized):
			writeError(w, http.StatusUnauthorized, "sign in to continue this story")
		case errors.Is(err, domain.ErrForbidden):
			writeError(w, http.StatusForbidden, "this story belongs to another reader")
		case errors.Is(err, domain.ErrStoryNotFound):
			writeError(w, http.StatusNotFound, "story not found")
		case errors.Is(err, domain.ErrInvalidChoice):
			writeError(w, http.StatusBadRequest, "invalid choice index")
		case errors.Is(err, domain.ErrStoryEnded):
			writeError(w, http.StatusConflict, "story has ended")
		default:
			r.log.Error().Err(err).Msg("progress story")
			writeError(w, http.StatusInternalServerError, "progress failed")
		}
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (r *Router) claimStory(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	if err := r.svc.ClaimStory(req.Context(), id); err != nil {
		switch {
		case errors.Is(err, domain.ErrUnauthorized):
			writeError(w, http.StatusUnauthorized, "sign in to claim this story")
		case errors.Is(err, domain.ErrForbidden):
			writeError(w, http.StatusForbidden, "story already owned")
		case errors.Is(err, domain.ErrStoryNotFound):
			writeError(w, http.StatusNotFound, "story not found")
		default:
			r.log.Error().Err(err).Msg("claim story")
			writeError(w, http.StatusInternalServerError, "claim failed")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (r *Router) myStories(w http.ResponseWriter, req *http.Request) {
	stories, err := r.svc.MyStories(req.Context())
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "sign in to see your stories")
			return
		}
		r.log.Error().Err(err).Msg("my stories")
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stories": stories})
}

func (r *Router) generateImage(w http.ResponseWriter, req *http.Request) {
	var in domain.GenerateImageInput
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if in.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt required")
		return
	}
	out, err := r.svc.GenerateImage(req.Context(), in)
	if err != nil {
		r.log.Error().Err(err).Msg("generate image")
		writeError(w, http.StatusBadGateway, "image providers unavailable")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (r *Router) generateSpeech(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	seqStr := req.PathValue("seq")
	seq, err := strconv.Atoi(seqStr)
	if err != nil || seq < 0 {
		writeError(w, http.StatusBadRequest, "invalid page index")
		return
	}
	url, err := r.svc.GenerateSpeech(req.Context(), id, seq)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrStoryNotFound), errors.Is(err, domain.ErrPageNotFound):
			writeError(w, http.StatusNotFound, "page not found")
		case errors.Is(err, domain.ErrTTSUnavailable):
			writeError(w, http.StatusServiceUnavailable, "narration unavailable")
		default:
			r.log.Error().Err(err).Msg("generate speech")
			writeError(w, http.StatusBadGateway, "narration failed")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"audioUrl": url})
}

func (r *Router) visit(w http.ResponseWriter, req *http.Request) {
	var in domain.VisitInput
	_ = json.NewDecoder(req.Body).Decode(&in)
	if in.UserAgent == "" {
		in.UserAgent = req.Header.Get("User-Agent")
	}
	r.svc.RecordVisit(req.Context(), in)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

const oauthStateCookieName = "solo_adv_oauth_state"

func (r *Router) authStart(w http.ResponseWriter, req *http.Request) {
	authURL, state, err := r.svc.BeginLogin()
	if err != nil {
		r.log.Error().Err(err).Msg("begin login")
		writeError(w, http.StatusServiceUnavailable, "sign-in unavailable")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   r.secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, req, authURL, http.StatusFound)
}

func (r *Router) authCallback(w http.ResponseWriter, req *http.Request) {
	code := req.URL.Query().Get("code")
	state := req.URL.Query().Get("state")
	c, err := req.Cookie(oauthStateCookieName)
	if err != nil || c.Value == "" || c.Value != state {
		writeError(w, http.StatusBadRequest, "invalid oauth state")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oauthStateCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: r.secure, SameSite: http.SameSiteLaxMode})

	sess, _, err := r.svc.CompleteLogin(req.Context(), code)
	if err != nil {
		r.log.Error().Err(err).Msg("complete login")
		writeError(w, http.StatusBadGateway, "sign-in failed")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		Domain:   r.cookieDomain,
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   r.secure,
		SameSite: http.SameSiteLaxMode,
	})
	returnTo := r.frontendURL
	if rt := req.URL.Query().Get("return_to"); rt != "" {
		returnTo = rt
	}
	http.Redirect(w, req, returnTo, http.StatusFound)
}

func (r *Router) authLogout(w http.ResponseWriter, req *http.Request) {
	c, _ := req.Cookie(SessionCookieName)
	if c != nil {
		_ = r.svc.Logout(req.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   r.cookieDomain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.secure,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (r *Router) authMe(w http.ResponseWriter, req *http.Request) {
	uid := app.CurrentUserID(req.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "not signed in")
		return
	}
	c, err := req.Cookie(SessionCookieName)
	if err != nil || c == nil {
		writeError(w, http.StatusUnauthorized, "not signed in")
		return
	}
	user, err := r.svc.UserBySession(req.Context(), c.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "not signed in")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
