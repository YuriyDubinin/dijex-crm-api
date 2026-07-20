package domain

import "time"

// StatsFilter — общий фильтр периода для статистики (по created_at сущности).
// Пустые границы = без ограничения (за всё время).
type StatsFilter struct {
	From *time.Time
	To   *time.Time
}

// StatBucket — универсальная «корзина» разбивки: Key — значение измерения
// (статус/тип/источник/uuid), Label — человекочитаемая метка (для ссылок вроде
// исполнителя), Count — количество, Amount — сумма (используется для сделок).
type StatBucket struct {
	Key    string
	Label  string
	Count  int
	Amount float64
}

// FinanceByCurrency — финансовые агрегаты по сделкам в разрезе одной валюты.
type FinanceByCurrency struct {
	Currency         string
	Revenue          float64 // сумма выигранных сделок типа INCOME
	Expenses         float64 // сумма выигранных сделок типа EXPENSE
	Pipeline         float64 // сумма открытых сделок (NEW/PENDING)
	WeightedPipeline float64 // сумма открытых * probability / 100
	WonCount         int
	LostCount        int
}

// TimePoint — точка временного ряда (period — начало интервала).
type TimePoint struct {
	Period time.Time
	Value  float64
}

// SummaryStats — сводные KPI по тенанту (текущее состояние в пределах периода).
type SummaryStats struct {
	ClientsTotal, ClientsActive, ClientsLeads      int
	DealsTotal, DealsOpen, DealsWon, DealsLost     int
	ProjectsTotal, ProjectsActive                  int
	ServicesTotal, ServicesActive                  int
	TasksTotal, TasksOpen, TasksOverdue, TasksDone int
	ProcessesTotal, ProcessesActive                int
	ProcessesCompleted, ProcessesFailed            int
	UsersTotal                                     int
	Finance                                        []FinanceByCurrency
}

// DealStats — всё для графиков по сделкам.
type DealStats struct {
	ByStatus   []StatBucket
	ByType     []StatBucket
	ByAssignee []StatBucket
	Finance    []FinanceByCurrency
}

// ClientStats — всё для графиков по клиентам.
type ClientStats struct {
	Total         int
	ByStatus      []StatBucket
	BySubjectType []StatBucket
	BySource      []StatBucket
}

// TaskStats — всё для графиков по задачам.
type TaskStats struct {
	Total, Open, Overdue, Done int
	ByStatus                   []StatBucket
	ByPriority                 []StatBucket
	ByAssignee                 []StatBucket
}

// ProjectStats — всё для графиков по проектам.
type ProjectStats struct {
	Total    int
	ByStatus []StatBucket
}

// ServiceStats — всё для графиков по услугам.
type ServiceStats struct {
	Total      int
	ByStatus   []StatBucket
	ByCategory []StatBucket
}

// ProcessStats — всё для графиков по процессам.
type ProcessStats struct {
	Total    int
	ByStatus []StatBucket
}
