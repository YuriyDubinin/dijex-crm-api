package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// StatsService — тонкая обёртка над StatsRepository: валидирует ввод для
// временных рядов и передаёт период в репозиторий. Ничего не пишет.
type StatsService struct {
	repo   domain.StatsRepository
	logger *slog.Logger
}

func NewStatsService(repo domain.StatsRepository, logger *slog.Logger) *StatsService {
	return &StatsService{repo: repo, logger: logger}
}

// StatsPeriod — период выборки (границы по created_at), уже распарсенный
// хендлером. Пустые границы = за всё время.
type StatsPeriod struct {
	From *time.Time
	To   *time.Time
}

func (p StatsPeriod) filter() domain.StatsFilter {
	return domain.StatsFilter{From: p.From, To: p.To}
}

func (s *StatsService) Summary(ctx context.Context, customerID uuid.UUID, p StatsPeriod) (*domain.SummaryStats, error) {
	return s.repo.Summary(ctx, customerID, p.filter())
}

func (s *StatsService) DealStats(ctx context.Context, customerID uuid.UUID, p StatsPeriod) (*domain.DealStats, error) {
	return s.repo.DealStats(ctx, customerID, p.filter())
}

func (s *StatsService) ClientStats(ctx context.Context, customerID uuid.UUID, p StatsPeriod) (*domain.ClientStats, error) {
	return s.repo.ClientStats(ctx, customerID, p.filter())
}

func (s *StatsService) TaskStats(ctx context.Context, customerID uuid.UUID, p StatsPeriod) (*domain.TaskStats, error) {
	return s.repo.TaskStats(ctx, customerID, p.filter())
}

func (s *StatsService) ProjectStats(ctx context.Context, customerID uuid.UUID, p StatsPeriod) (*domain.ProjectStats, error) {
	return s.repo.ProjectStats(ctx, customerID, p.filter())
}

func (s *StatsService) ServiceStats(ctx context.Context, customerID uuid.UUID, p StatsPeriod) (*domain.ServiceStats, error) {
	return s.repo.ServiceStats(ctx, customerID, p.filter())
}

func (s *StatsService) ProcessStats(ctx context.Context, customerID uuid.UUID, p StatsPeriod) (*domain.ProcessStats, error) {
	return s.repo.ProcessStats(ctx, customerID, p.filter())
}

// ───────────────────────── Time series ─────────────────────────

// TimeSeriesResult — временной ряд + эхо параметров запроса.
type TimeSeriesResult struct {
	Metric   string
	Interval string
	Currency string
	Points   []domain.TimePoint
}

var validTimeSeriesMetrics = map[string]bool{
	"deals_created":     true,
	"deals_won":         true,
	"revenue":           true,
	"expenses":          true,
	"clients_created":   true,
	"tasks_created":     true,
	"tasks_completed":   true,
	"processes_created": true,
}

var validTimeSeriesIntervals = map[string]bool{
	"day":   true,
	"week":  true,
	"month": true,
}

func (s *StatsService) TimeSeries(ctx context.Context, customerID uuid.UUID, metric, interval, currency string, p StatsPeriod) (*TimeSeriesResult, error) {
	metric = strings.ToLower(strings.TrimSpace(metric))
	interval = strings.ToLower(strings.TrimSpace(interval))
	if interval == "" {
		interval = "day"
	}

	var errs domain.ValidationErrors
	if !validTimeSeriesMetrics[metric] {
		errs = append(errs, &domain.ValidationError{Field: "metric", Message: "must be one of deals_created, deals_won, revenue, expenses, clients_created, tasks_created, tasks_completed, processes_created"})
	}
	if !validTimeSeriesIntervals[interval] {
		errs = append(errs, &domain.ValidationError{Field: "interval", Message: "must be one of day, week, month"})
	}
	if len(errs) > 0 {
		return nil, errs
	}

	currency = strings.ToUpper(strings.TrimSpace(currency))
	points, err := s.repo.TimeSeries(ctx, customerID, metric, interval, currency, p.filter())
	if err != nil {
		return nil, err
	}
	return &TimeSeriesResult{Metric: metric, Interval: interval, Currency: currency, Points: points}, nil
}
