package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YuriyDubinin/bms-api/internal/domain"
)

// StatsRepository — read-only агрегатные выборки для дашборда. Ничего не пишет.
// Все фрагменты SQL (таблицы/выражения) — хардкод-константы из методов ниже;
// пользовательский ввод попадает только через bind-параметры ($N).
type StatsRepository struct {
	pool *pgxpool.Pool
}

func NewStatsRepository(pool *pgxpool.Pool) *StatsRepository {
	return &StatsRepository{pool: pool}
}

// rangeSQL добавляет ограничение периода по колонке col и наполняет args.
func rangeSQL(f domain.StatsFilter, col string, args *[]any) string {
	var sb strings.Builder
	if f.From != nil {
		*args = append(*args, *f.From)
		fmt.Fprintf(&sb, " AND %s >= $%d", col, len(*args))
	}
	if f.To != nil {
		*args = append(*args, *f.To)
		fmt.Fprintf(&sb, " AND %s <= $%d", col, len(*args))
	}
	return sb.String()
}

// buckets выполняет GROUP BY-запрос вида (key, label, count, amount).
func (r *StatsRepository) buckets(ctx context.Context, query string, args ...any) ([]domain.StatBucket, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: stats buckets: %w", err)
	}
	defer rows.Close()

	out := []domain.StatBucket{}
	for rows.Next() {
		var b domain.StatBucket
		if err := rows.Scan(&b.Key, &b.Label, &b.Count, &b.Amount); err != nil {
			return nil, fmt.Errorf("postgres: scan bucket: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// groupSimple — разбивка по keyExpr без денежных сумм.
func (r *StatsRepository) groupSimple(ctx context.Context, customerID uuid.UUID, f domain.StatsFilter, table, keyExpr string) ([]domain.StatBucket, error) {
	args := []any{customerID}
	rng := rangeSQL(f, "created_at", &args)
	q := fmt.Sprintf(
		"SELECT %s AS k, '' AS lbl, count(*) AS cnt, 0::float8 AS amt FROM %s WHERE customer_id=$1 AND deleted_at IS NULL%s GROUP BY %s ORDER BY cnt DESC",
		keyExpr, table, rng, keyExpr,
	)
	return r.buckets(ctx, q, args...)
}

// dealGroup — разбивка сделок по keyExpr с суммой amount.
func (r *StatsRepository) dealGroup(ctx context.Context, customerID uuid.UUID, f domain.StatsFilter, keyExpr string) ([]domain.StatBucket, error) {
	args := []any{customerID}
	rng := rangeSQL(f, "created_at", &args)
	q := fmt.Sprintf(
		"SELECT %s AS k, '' AS lbl, count(*) AS cnt, COALESCE(sum(amount),0)::float8 AS amt FROM deals WHERE customer_id=$1 AND deleted_at IS NULL%s GROUP BY %s ORDER BY cnt DESC",
		keyExpr, rng, keyExpr,
	)
	return r.buckets(ctx, q, args...)
}

// assigneeGroup — разбивка по исполнителю (assigned_to) с меткой-именем.
// table ∈ {"deals","tasks"}; withAmount — суммировать amount (только сделки).
func (r *StatsRepository) assigneeGroup(ctx context.Context, customerID uuid.UUID, f domain.StatsFilter, table string, withAmount bool) ([]domain.StatBucket, error) {
	amt := "0::float8"
	if withAmount {
		amt = "COALESCE(sum(t.amount),0)::float8"
	}
	args := []any{customerID}
	rng := rangeSQL(f, "t.created_at", &args)
	q := fmt.Sprintf(
		"SELECT COALESCE(t.assigned_to::text,'') AS k, COALESCE(u.full_name,'') AS lbl, count(*) AS cnt, %s AS amt "+
			"FROM %s t LEFT JOIN customer_users u ON u.id = t.assigned_to "+
			"WHERE t.customer_id=$1 AND t.deleted_at IS NULL%s GROUP BY t.assigned_to, u.full_name ORDER BY cnt DESC",
		amt, table, rng,
	)
	return r.buckets(ctx, q, args...)
}

// financeByCurrency — финансовые агрегаты по сделкам в разрезе валюты.
func (r *StatsRepository) financeByCurrency(ctx context.Context, customerID uuid.UUID, f domain.StatsFilter) ([]domain.FinanceByCurrency, error) {
	args := []any{customerID}
	rng := rangeSQL(f, "created_at", &args)
	q := "SELECT currency, " +
		"COALESCE(sum(amount) FILTER (WHERE type='INCOME' AND status='WON'),0)::float8, " +
		"COALESCE(sum(amount) FILTER (WHERE type='EXPENSE' AND status='WON'),0)::float8, " +
		"COALESCE(sum(amount) FILTER (WHERE status IN ('NEW','PENDING')),0)::float8, " +
		"COALESCE(sum(amount*probability/100.0) FILTER (WHERE status IN ('NEW','PENDING')),0)::float8, " +
		"count(*) FILTER (WHERE status='WON'), " +
		"count(*) FILTER (WHERE status='LOST') " +
		"FROM deals WHERE customer_id=$1 AND deleted_at IS NULL" + rng +
		" GROUP BY currency ORDER BY currency"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: stats finance: %w", err)
	}
	defer rows.Close()

	out := []domain.FinanceByCurrency{}
	for rows.Next() {
		var fc domain.FinanceByCurrency
		if err := rows.Scan(&fc.Currency, &fc.Revenue, &fc.Expenses, &fc.Pipeline, &fc.WeightedPipeline, &fc.WonCount, &fc.LostCount); err != nil {
			return nil, fmt.Errorf("postgres: scan finance: %w", err)
		}
		out = append(out, fc)
	}
	return out, rows.Err()
}

// scalarCounts выполняет одиночный запрос агрегатов-счётчиков в dests.
func (r *StatsRepository) scalarCounts(ctx context.Context, customerID uuid.UUID, f domain.StatsFilter, selectExpr, table string, dests ...any) error {
	args := []any{customerID}
	rng := rangeSQL(f, "created_at", &args)
	q := fmt.Sprintf("SELECT %s FROM %s WHERE customer_id=$1 AND deleted_at IS NULL%s", selectExpr, table, rng)
	return r.pool.QueryRow(ctx, q, args...).Scan(dests...)
}

func (r *StatsRepository) Summary(ctx context.Context, customerID uuid.UUID, f domain.StatsFilter) (*domain.SummaryStats, error) {
	s := &domain.SummaryStats{}

	if err := r.scalarCounts(ctx, customerID, f,
		"count(*), count(*) FILTER (WHERE status='ACTIVE'), count(*) FILTER (WHERE status='LEAD')",
		"clients", &s.ClientsTotal, &s.ClientsActive, &s.ClientsLeads); err != nil {
		return nil, fmt.Errorf("postgres: summary clients: %w", err)
	}
	if err := r.scalarCounts(ctx, customerID, f,
		"count(*), count(*) FILTER (WHERE status IN ('NEW','PENDING')), count(*) FILTER (WHERE status='WON'), count(*) FILTER (WHERE status='LOST')",
		"deals", &s.DealsTotal, &s.DealsOpen, &s.DealsWon, &s.DealsLost); err != nil {
		return nil, fmt.Errorf("postgres: summary deals: %w", err)
	}
	if err := r.scalarCounts(ctx, customerID, f,
		"count(*), count(*) FILTER (WHERE status='ACTIVE')",
		"projects", &s.ProjectsTotal, &s.ProjectsActive); err != nil {
		return nil, fmt.Errorf("postgres: summary projects: %w", err)
	}
	if err := r.scalarCounts(ctx, customerID, f,
		"count(*), count(*) FILTER (WHERE status='ACTIVE')",
		"services", &s.ServicesTotal, &s.ServicesActive); err != nil {
		return nil, fmt.Errorf("postgres: summary services: %w", err)
	}
	if err := r.scalarCounts(ctx, customerID, f,
		"count(*), count(*) FILTER (WHERE status NOT IN ('DONE','CANCELLED')), count(*) FILTER (WHERE due_at < NOW() AND status NOT IN ('DONE','CANCELLED')), count(*) FILTER (WHERE status='DONE')",
		"tasks", &s.TasksTotal, &s.TasksOpen, &s.TasksOverdue, &s.TasksDone); err != nil {
		return nil, fmt.Errorf("postgres: summary tasks: %w", err)
	}
	if err := r.scalarCounts(ctx, customerID, f,
		"count(*), count(*) FILTER (WHERE status='ACTIVE'), count(*) FILTER (WHERE status='COMPLETED'), count(*) FILTER (WHERE status='FAILED')",
		"processes", &s.ProcessesTotal, &s.ProcessesActive, &s.ProcessesCompleted, &s.ProcessesFailed); err != nil {
		return nil, fmt.Errorf("postgres: summary processes: %w", err)
	}
	if err := r.scalarCounts(ctx, customerID, f,
		"count(*)", "customer_users", &s.UsersTotal); err != nil {
		return nil, fmt.Errorf("postgres: summary users: %w", err)
	}

	fin, err := r.financeByCurrency(ctx, customerID, f)
	if err != nil {
		return nil, err
	}
	s.Finance = fin
	return s, nil
}

func (r *StatsRepository) DealStats(ctx context.Context, customerID uuid.UUID, f domain.StatsFilter) (*domain.DealStats, error) {
	out := &domain.DealStats{}
	var err error
	if out.ByStatus, err = r.dealGroup(ctx, customerID, f, "status::text"); err != nil {
		return nil, err
	}
	if out.ByType, err = r.dealGroup(ctx, customerID, f, "type::text"); err != nil {
		return nil, err
	}
	if out.ByAssignee, err = r.assigneeGroup(ctx, customerID, f, "deals", true); err != nil {
		return nil, err
	}
	if out.Finance, err = r.financeByCurrency(ctx, customerID, f); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *StatsRepository) ClientStats(ctx context.Context, customerID uuid.UUID, f domain.StatsFilter) (*domain.ClientStats, error) {
	out := &domain.ClientStats{}
	if err := r.scalarCounts(ctx, customerID, f, "count(*)", "clients", &out.Total); err != nil {
		return nil, fmt.Errorf("postgres: client stats total: %w", err)
	}
	var err error
	if out.ByStatus, err = r.groupSimple(ctx, customerID, f, "clients", "status::text"); err != nil {
		return nil, err
	}
	if out.BySubjectType, err = r.groupSimple(ctx, customerID, f, "clients", "subject_type::text"); err != nil {
		return nil, err
	}
	if out.BySource, err = r.groupSimple(ctx, customerID, f, "clients", "COALESCE(source,'')"); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *StatsRepository) TaskStats(ctx context.Context, customerID uuid.UUID, f domain.StatsFilter) (*domain.TaskStats, error) {
	out := &domain.TaskStats{}
	if err := r.scalarCounts(ctx, customerID, f,
		"count(*), count(*) FILTER (WHERE status NOT IN ('DONE','CANCELLED')), count(*) FILTER (WHERE due_at < NOW() AND status NOT IN ('DONE','CANCELLED')), count(*) FILTER (WHERE status='DONE')",
		"tasks", &out.Total, &out.Open, &out.Overdue, &out.Done); err != nil {
		return nil, fmt.Errorf("postgres: task stats: %w", err)
	}
	var err error
	if out.ByStatus, err = r.groupSimple(ctx, customerID, f, "tasks", "status::text"); err != nil {
		return nil, err
	}
	if out.ByPriority, err = r.groupSimple(ctx, customerID, f, "tasks", "priority::text"); err != nil {
		return nil, err
	}
	if out.ByAssignee, err = r.assigneeGroup(ctx, customerID, f, "tasks", false); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *StatsRepository) ProjectStats(ctx context.Context, customerID uuid.UUID, f domain.StatsFilter) (*domain.ProjectStats, error) {
	out := &domain.ProjectStats{}
	if err := r.scalarCounts(ctx, customerID, f, "count(*)", "projects", &out.Total); err != nil {
		return nil, fmt.Errorf("postgres: project stats total: %w", err)
	}
	var err error
	if out.ByStatus, err = r.groupSimple(ctx, customerID, f, "projects", "status::text"); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *StatsRepository) ServiceStats(ctx context.Context, customerID uuid.UUID, f domain.StatsFilter) (*domain.ServiceStats, error) {
	out := &domain.ServiceStats{}
	if err := r.scalarCounts(ctx, customerID, f, "count(*)", "services", &out.Total); err != nil {
		return nil, fmt.Errorf("postgres: service stats total: %w", err)
	}
	var err error
	if out.ByStatus, err = r.groupSimple(ctx, customerID, f, "services", "status::text"); err != nil {
		return nil, err
	}
	if out.ByCategory, err = r.groupSimple(ctx, customerID, f, "services", "COALESCE(category,'')"); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *StatsRepository) ProcessStats(ctx context.Context, customerID uuid.UUID, f domain.StatsFilter) (*domain.ProcessStats, error) {
	out := &domain.ProcessStats{}
	if err := r.scalarCounts(ctx, customerID, f, "count(*)", "processes", &out.Total); err != nil {
		return nil, fmt.Errorf("postgres: process stats total: %w", err)
	}
	var err error
	if out.ByStatus, err = r.groupSimple(ctx, customerID, f, "processes", "status::text"); err != nil {
		return nil, err
	}
	return out, nil
}

// timeSeriesSpec — параметры метрики временного ряда (хардкод-whitelist).
type timeSeriesSpec struct {
	table   string
	dateCol string
	agg     string
	extra   string
}

var timeSeriesSpecs = map[string]timeSeriesSpec{
	"deals_created":     {"deals", "created_at", "count(*)", ""},
	"deals_won":         {"deals", "closed_at", "count(*)", " AND status='WON'"},
	"revenue":           {"deals", "closed_at", "COALESCE(sum(amount),0)::float8", " AND status='WON' AND type='INCOME'"},
	"expenses":          {"deals", "closed_at", "COALESCE(sum(amount),0)::float8", " AND status='WON' AND type='EXPENSE'"},
	"clients_created":   {"clients", "created_at", "count(*)", ""},
	"tasks_created":     {"tasks", "created_at", "count(*)", ""},
	"tasks_completed":   {"tasks", "completed_at", "count(*)", ""},
	"processes_created": {"processes", "created_at", "count(*)", ""},
}

func (r *StatsRepository) TimeSeries(ctx context.Context, customerID uuid.UUID, metric, interval, currency string, f domain.StatsFilter) ([]domain.TimePoint, error) {
	spec, ok := timeSeriesSpecs[metric]
	if !ok {
		return nil, fmt.Errorf("postgres: unknown timeseries metric %q", metric)
	}

	args := []any{customerID}
	extra := spec.extra
	if currency != "" && (metric == "revenue" || metric == "expenses") {
		args = append(args, currency)
		extra += fmt.Sprintf(" AND currency=$%d", len(args))
	}
	rng := rangeSQL(f, spec.dateCol, &args)

	q := fmt.Sprintf(
		"SELECT date_trunc('%s', %s)::date AS period, %s AS value "+
			"FROM %s WHERE customer_id=$1 AND deleted_at IS NULL AND %s IS NOT NULL%s%s "+
			"GROUP BY period ORDER BY period",
		interval, spec.dateCol, spec.agg, spec.table, spec.dateCol, extra, rng,
	)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: timeseries: %w", err)
	}
	defer rows.Close()

	out := []domain.TimePoint{}
	for rows.Next() {
		var p domain.TimePoint
		if err := rows.Scan(&p.Period, &p.Value); err != nil {
			return nil, fmt.Errorf("postgres: scan timepoint: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
