package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/ridopark/solo-adeventure/backend/internal/app"
	"github.com/ridopark/solo-adeventure/backend/internal/domain"
)

type Router struct {
	svc *app.Service
	log zerolog.Logger
}

func NewRouter(svc *app.Service, log zerolog.Logger, corsOrigin string) http.Handler {
	r := &Router{svc: svc, log: log.With().Str("component", "http.router").Logger()}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", r.health)
	mux.HandleFunc("POST /stories", r.startStory)
	mux.HandleFunc("GET /stories/{id}", r.getStory)
	mux.HandleFunc("POST /stories/{id}/choose", r.chooseStory)
	mux.HandleFunc("POST /images", r.generateImage)
	mux.HandleFunc("POST /visit", r.visit)

	var h http.Handler = mux
	h = cors(corsOrigin, h)
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

func (r *Router) visit(w http.ResponseWriter, req *http.Request) {
	var in domain.VisitInput
	_ = json.NewDecoder(req.Body).Decode(&in)
	if in.UserAgent == "" {
		in.UserAgent = req.Header.Get("User-Agent")
	}
	r.svc.RecordVisit(req.Context(), in)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
