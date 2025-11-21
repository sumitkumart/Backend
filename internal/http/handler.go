package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/stocky/backend/internal/service"
)

// Handler wires all REST endpoints.
type Handler struct {
	rewardSvc    *service.RewardService
	statsSvc     *service.StatsService
	portfolioSvc *service.PortfolioService
}

func NewHandler(reward *service.RewardService, stats *service.StatsService, portfolio *service.PortfolioService) *Handler {
	return &Handler{
		rewardSvc:    reward,
		statsSvc:     stats,
		portfolioSvc: portfolio,
	}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, map[string]string{"status": "ok"})
	})

	r.Route("/", func(r chi.Router) {
		r.Post("/reward", h.handleReward)
		r.Get("/today-stocks/{userId}", h.handleTodayRewards)
		r.Get("/historical-inr/{userId}", h.handleHistoricalINR)
		r.Get("/stats/{userId}", h.handleStats)
		r.Get("/portfolio/{userId}", h.handlePortfolio)
	})

	return r
}

func (h *Handler) handleReward(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req rewardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse("invalid payload"))
		return
	}
	if err := req.validate(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse(err.Error()))
		return
	}

	result, err := h.rewardSvc.RewardUser(ctx, req.toInput())
	if err != nil {
		render.Status(r, statusCodeForErr(err))
		render.JSON(w, r, errorResponse(err.Error()))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, result)
}

func (h *Handler) handleTodayRewards(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse("invalid user id"))
		return
	}

	records, err := h.statsSvc.TodayRewards(ctx, userID)
	if err != nil {
		render.Status(r, statusCodeForErr(err))
		render.JSON(w, r, errorResponse(err.Error()))
		return
	}
	render.JSON(w, r, records)
}

func (h *Handler) handleHistoricalINR(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse("invalid user id"))
		return
	}

	data, err := h.statsSvc.HistoricalINR(ctx, userID)
	if err != nil {
		render.Status(r, statusCodeForErr(err))
		render.JSON(w, r, errorResponse(err.Error()))
		return
	}
	render.JSON(w, r, data)
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse("invalid user id"))
		return
	}

	resp, err := h.statsSvc.UserStats(ctx, userID)
	if err != nil {
		render.Status(r, statusCodeForErr(err))
		render.JSON(w, r, errorResponse(err.Error()))
		return
	}
	render.JSON(w, r, resp)
}

func (h *Handler) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse("invalid user id"))
		return
	}

	resp, err := h.portfolioSvc.GetPortfolio(ctx, userID)
	if err != nil {
		render.Status(r, statusCodeForErr(err))
		render.JSON(w, r, errorResponse(err.Error()))
		return
	}
	render.JSON(w, r, resp)
}

type rewardRequest struct {
	UserID     string          `json:"userId"`
	Symbol     string          `json:"symbol"`
	Shares     decimal.Decimal `json:"shares"`
	EventID    string          `json:"eventId"`
	RewardedAt time.Time       `json:"rewardedAt"`
}

func (r rewardRequest) validate() error {
	if r.UserID == "" || r.Symbol == "" || r.EventID == "" {
		return errors.New("userId, symbol and eventId are required")
	}
	if !r.Shares.GreaterThan(decimal.Zero) {
		return errors.New("shares must be greater than zero")
	}
	if _, err := uuid.Parse(r.UserID); err != nil {
		return errors.New("userId must be a valid UUID")
	}
	return nil
}

func (r rewardRequest) toInput() service.RewardInput {
	userID, _ := uuid.Parse(r.UserID)
	return service.RewardInput{
		UserID:     userID,
		Symbol:     r.Symbol,
		Shares:     r.Shares,
		EventID:    r.EventID,
		RewardedAt: r.RewardedAt,
	}
}

func errorResponse(msg string) map[string]string {
	return map[string]string{"error": msg}
}

func statusCodeForErr(err error) int {
	switch {
	case errors.Is(err, service.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
