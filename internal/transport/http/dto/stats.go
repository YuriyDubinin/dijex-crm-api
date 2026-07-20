package dto

import (
	"github.com/YuriyDubinin/bms-api/internal/domain"
	"github.com/YuriyDubinin/bms-api/internal/service"
)

// StatBucketResponse — универсальная корзина разбивки. amount осмыслен только
// для разбивок по сделкам (в остальных = 0); label заполнен для ссылок вроде
// исполнителя (иначе "").
type StatBucketResponse struct {
	Key    string  `json:"key"`
	Label  string  `json:"label"`
	Count  int     `json:"count"`
	Amount float64 `json:"amount"`
}

func fromBuckets(bs []domain.StatBucket) []StatBucketResponse {
	out := make([]StatBucketResponse, 0, len(bs))
	for _, b := range bs {
		out = append(out, StatBucketResponse{Key: b.Key, Label: b.Label, Count: b.Count, Amount: b.Amount})
	}
	return out
}

// FinanceResponse — финансы по одной валюте (+ производные net/win_rate).
type FinanceResponse struct {
	Currency         string  `json:"currency"`
	Revenue          float64 `json:"revenue"`
	Expenses         float64 `json:"expenses"`
	Net              float64 `json:"net"`
	Pipeline         float64 `json:"pipeline"`
	WeightedPipeline float64 `json:"weighted_pipeline"`
	WonCount         int     `json:"won_count"`
	LostCount        int     `json:"lost_count"`
	WinRate          float64 `json:"win_rate"` // доля 0..1 = won / (won + lost)
}

func fromFinance(fs []domain.FinanceByCurrency) []FinanceResponse {
	out := make([]FinanceResponse, 0, len(fs))
	for _, f := range fs {
		var winRate float64
		if closed := f.WonCount + f.LostCount; closed > 0 {
			winRate = float64(f.WonCount) / float64(closed)
		}
		out = append(out, FinanceResponse{
			Currency:         f.Currency,
			Revenue:          f.Revenue,
			Expenses:         f.Expenses,
			Net:              f.Revenue - f.Expenses,
			Pipeline:         f.Pipeline,
			WeightedPipeline: f.WeightedPipeline,
			WonCount:         f.WonCount,
			LostCount:        f.LostCount,
			WinRate:          winRate,
		})
	}
	return out
}

// ───────────────────────── Summary ─────────────────────────

type SummaryStatsResponse struct {
	Clients struct {
		Total  int `json:"total"`
		Active int `json:"active"`
		Leads  int `json:"leads"`
	} `json:"clients"`
	Deals struct {
		Total int `json:"total"`
		Open  int `json:"open"`
		Won   int `json:"won"`
		Lost  int `json:"lost"`
	} `json:"deals"`
	Projects struct {
		Total  int `json:"total"`
		Active int `json:"active"`
	} `json:"projects"`
	Services struct {
		Total  int `json:"total"`
		Active int `json:"active"`
	} `json:"services"`
	Tasks struct {
		Total   int `json:"total"`
		Open    int `json:"open"`
		Overdue int `json:"overdue"`
		Done    int `json:"done"`
	} `json:"tasks"`
	Processes struct {
		Total     int `json:"total"`
		Active    int `json:"active"`
		Completed int `json:"completed"`
		Failed    int `json:"failed"`
	} `json:"processes"`
	Users struct {
		Total int `json:"total"`
	} `json:"users"`
	Finance []FinanceResponse `json:"finance"`
}

func FromSummaryStats(s *domain.SummaryStats) SummaryStatsResponse {
	var r SummaryStatsResponse
	r.Clients.Total, r.Clients.Active, r.Clients.Leads = s.ClientsTotal, s.ClientsActive, s.ClientsLeads
	r.Deals.Total, r.Deals.Open, r.Deals.Won, r.Deals.Lost = s.DealsTotal, s.DealsOpen, s.DealsWon, s.DealsLost
	r.Projects.Total, r.Projects.Active = s.ProjectsTotal, s.ProjectsActive
	r.Services.Total, r.Services.Active = s.ServicesTotal, s.ServicesActive
	r.Tasks.Total, r.Tasks.Open, r.Tasks.Overdue, r.Tasks.Done = s.TasksTotal, s.TasksOpen, s.TasksOverdue, s.TasksDone
	r.Processes.Total, r.Processes.Active = s.ProcessesTotal, s.ProcessesActive
	r.Processes.Completed, r.Processes.Failed = s.ProcessesCompleted, s.ProcessesFailed
	r.Users.Total = s.UsersTotal
	r.Finance = fromFinance(s.Finance)
	return r
}

// ───────────────────────── Per-entity ─────────────────────────

type DealStatsResponse struct {
	ByStatus   []StatBucketResponse `json:"by_status"`
	ByType     []StatBucketResponse `json:"by_type"`
	ByAssignee []StatBucketResponse `json:"by_assignee"`
	Finance    []FinanceResponse    `json:"finance"`
}

func FromDealStats(s *domain.DealStats) DealStatsResponse {
	return DealStatsResponse{
		ByStatus:   fromBuckets(s.ByStatus),
		ByType:     fromBuckets(s.ByType),
		ByAssignee: fromBuckets(s.ByAssignee),
		Finance:    fromFinance(s.Finance),
	}
}

type ClientStatsResponse struct {
	Total         int                  `json:"total"`
	ByStatus      []StatBucketResponse `json:"by_status"`
	BySubjectType []StatBucketResponse `json:"by_subject_type"`
	BySource      []StatBucketResponse `json:"by_source"`
}

func FromClientStats(s *domain.ClientStats) ClientStatsResponse {
	return ClientStatsResponse{
		Total:         s.Total,
		ByStatus:      fromBuckets(s.ByStatus),
		BySubjectType: fromBuckets(s.BySubjectType),
		BySource:      fromBuckets(s.BySource),
	}
}

type TaskStatsResponse struct {
	Total      int                  `json:"total"`
	Open       int                  `json:"open"`
	Overdue    int                  `json:"overdue"`
	Done       int                  `json:"done"`
	ByStatus   []StatBucketResponse `json:"by_status"`
	ByPriority []StatBucketResponse `json:"by_priority"`
	ByAssignee []StatBucketResponse `json:"by_assignee"`
}

func FromTaskStats(s *domain.TaskStats) TaskStatsResponse {
	return TaskStatsResponse{
		Total:      s.Total,
		Open:       s.Open,
		Overdue:    s.Overdue,
		Done:       s.Done,
		ByStatus:   fromBuckets(s.ByStatus),
		ByPriority: fromBuckets(s.ByPriority),
		ByAssignee: fromBuckets(s.ByAssignee),
	}
}

type ProjectStatsResponse struct {
	Total    int                  `json:"total"`
	ByStatus []StatBucketResponse `json:"by_status"`
}

func FromProjectStats(s *domain.ProjectStats) ProjectStatsResponse {
	return ProjectStatsResponse{Total: s.Total, ByStatus: fromBuckets(s.ByStatus)}
}

type ServiceStatsResponse struct {
	Total      int                  `json:"total"`
	ByStatus   []StatBucketResponse `json:"by_status"`
	ByCategory []StatBucketResponse `json:"by_category"`
}

func FromServiceStats(s *domain.ServiceStats) ServiceStatsResponse {
	return ServiceStatsResponse{Total: s.Total, ByStatus: fromBuckets(s.ByStatus), ByCategory: fromBuckets(s.ByCategory)}
}

type ProcessStatsResponse struct {
	Total    int                  `json:"total"`
	ByStatus []StatBucketResponse `json:"by_status"`
}

func FromProcessStats(s *domain.ProcessStats) ProcessStatsResponse {
	return ProcessStatsResponse{Total: s.Total, ByStatus: fromBuckets(s.ByStatus)}
}

// ───────────────────────── Time series ─────────────────────────

type TimePointResponse struct {
	Period string  `json:"period"` // YYYY-MM-DD (начало интервала)
	Value  float64 `json:"value"`
}

type TimeSeriesResponse struct {
	Metric   string              `json:"metric"`
	Interval string              `json:"interval"`
	Currency string              `json:"currency"`
	Points   []TimePointResponse `json:"points"`
}

func FromTimeSeries(res *service.TimeSeriesResult) TimeSeriesResponse {
	pts := make([]TimePointResponse, 0, len(res.Points))
	for _, p := range res.Points {
		pts = append(pts, TimePointResponse{Period: p.Period.Format(dateLayout), Value: p.Value})
	}
	return TimeSeriesResponse{Metric: res.Metric, Interval: res.Interval, Currency: res.Currency, Points: pts}
}
