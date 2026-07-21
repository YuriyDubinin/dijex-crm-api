package transporthttp

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/YuriyDubinin/bms-api/internal/domain"
	"github.com/YuriyDubinin/bms-api/internal/transport/http/handler"
	mw "github.com/YuriyDubinin/bms-api/internal/transport/http/middleware"
)

// Deps — зависимости роутера. Хендлеры добавляются сюда по мере роста API.
type Deps struct {
	Logger        *slog.Logger
	Authenticator domain.Authenticator

	HealthHandler      *handler.HealthHandler
	AuthHandler        *handler.AuthHandler
	MeHandler          *handler.MeHandler
	CustomerHandler    *handler.CustomerHandler
	UserHandler        *handler.UserHandler
	ProjectHandler     *handler.ProjectHandler
	ServiceHandler     *handler.ServiceHandler
	ClientHandler      *handler.ClientHandler
	TaskHandler        *handler.TaskHandler
	DealHandler        *handler.DealHandler
	ProcessHandler     *handler.ProcessHandler
	OverviewHandler    *handler.OverviewHandler
	EntityLinkHandler  *handler.EntityLinkHandler
	TagHandler         *handler.TagHandler
	CustomFieldHandler *handler.CustomFieldHandler
	StatsHandler       *handler.StatsHandler
	AgendaHandler      *handler.AgendaHandler
}

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	// Глобальные мидлвари.
	r.Use(mw.RequestID)
	r.Use(mw.Logger(deps.Logger))
	r.Use(mw.Recover(deps.Logger))
	r.Use(mw.CORS)

	r.Route("/api", func(r chi.Router) {
		// ───────────────────────── Публичные ─────────────────────────
		r.Get("/ping", deps.HealthHandler.Ping)
		r.Post("/auth/login", deps.AuthHandler.Login)
		r.Post("/auth/register", deps.AuthHandler.Register)

		// ──────────────────────── Защищённые ─────────────────────────
		// Внутри группы действует Auth: каждый запрос обязан принести
		// `Authorization: Bearer <token>`. Мидлварь кладёт Principal в context.
		r.Group(func(r chi.Router) {
			r.Use(mw.Auth(deps.Authenticator, deps.Logger))

			r.Get("/me", deps.MeHandler.Get)
			r.Post("/auth/logout", deps.AuthHandler.Logout)

			// Тенант (текущий) и его операторы
			r.Get("/customer", deps.CustomerHandler.Get)
			r.Put("/customer/update", deps.CustomerHandler.Update)
			r.Post("/users/create", deps.UserHandler.Create)
			r.Get("/users/list", deps.UserHandler.List)
			r.Put("/users/update", deps.UserHandler.Update)
			r.Delete("/users/delete", deps.UserHandler.Delete)

			// Projects
			r.Post("/projects/create", deps.ProjectHandler.Create)
			r.Get("/projects/list", deps.ProjectHandler.List)
			r.Put("/projects/update", deps.ProjectHandler.Update)
			r.Delete("/projects/delete", deps.ProjectHandler.Delete)

			// Services
			r.Post("/services/create", deps.ServiceHandler.Create)
			r.Get("/services/list", deps.ServiceHandler.List)
			r.Put("/services/update", deps.ServiceHandler.Update)
			r.Delete("/services/delete", deps.ServiceHandler.Delete)

			// Clients
			r.Post("/clients/create", deps.ClientHandler.Create)
			r.Get("/clients/list", deps.ClientHandler.List)
			r.Put("/clients/update", deps.ClientHandler.Update)
			r.Delete("/clients/delete", deps.ClientHandler.Delete)
			r.Get("/clients/overview", deps.OverviewHandler.ClientOverview)
			// Членство клиента в проектах (M:N)
			r.Post("/clients/projects/attach", deps.ClientHandler.AttachProject)
			r.Delete("/clients/projects/detach", deps.ClientHandler.DetachProject)
			r.Get("/clients/projects/list", deps.ClientHandler.ListProjects)

			// Tasks
			r.Post("/tasks/create", deps.TaskHandler.Create)
			r.Get("/tasks/list", deps.TaskHandler.List)
			r.Put("/tasks/update", deps.TaskHandler.Update)
			r.Delete("/tasks/delete", deps.TaskHandler.Delete)

			// Календарь: единый агрегатор всех датированных сущностей окна
			r.Get("/calendar/agenda", deps.AgendaHandler.Agenda)

			// Deals (сделки)
			r.Post("/deals/create", deps.DealHandler.Create)
			r.Get("/deals/list", deps.DealHandler.List)
			r.Put("/deals/update", deps.DealHandler.Update)
			r.Delete("/deals/delete", deps.DealHandler.Delete)
			r.Get("/deals/overview", deps.OverviewHandler.DealOverview)

			// Processes (процессы) и их пользовательские этапы
			r.Post("/processes/create", deps.ProcessHandler.Create)
			r.Get("/processes/list", deps.ProcessHandler.List)
			r.Put("/processes/update", deps.ProcessHandler.Update)
			r.Delete("/processes/delete", deps.ProcessHandler.Delete)
			r.Get("/processes/overview", deps.OverviewHandler.ProcessOverview)
			r.Post("/processes/stages/create", deps.ProcessHandler.CreateStage)
			r.Get("/processes/stages/list", deps.ProcessHandler.ListStages)
			r.Put("/processes/stages/update", deps.ProcessHandler.UpdateStage)
			r.Delete("/processes/stages/delete", deps.ProcessHandler.DeleteStage)

			// Entity links (универсальные связи "всё со всем")
			r.Post("/links/create", deps.EntityLinkHandler.Create)
			r.Get("/links/list", deps.EntityLinkHandler.List)
			r.Delete("/links/delete", deps.EntityLinkHandler.Delete)

			// Tags
			r.Post("/tags/create", deps.TagHandler.Create)
			r.Get("/tags/list", deps.TagHandler.List)
			r.Put("/tags/update", deps.TagHandler.Update)
			r.Delete("/tags/delete", deps.TagHandler.Delete)
			r.Post("/tags/attach", deps.TagHandler.Attach)
			r.Delete("/tags/detach", deps.TagHandler.Detach)

			// Custom fields
			r.Post("/custom-fields/definitions/create", deps.CustomFieldHandler.CreateDefinition)
			r.Get("/custom-fields/definitions/list", deps.CustomFieldHandler.ListDefinitions)
			r.Put("/custom-fields/definitions/update", deps.CustomFieldHandler.UpdateDefinition)
			r.Delete("/custom-fields/definitions/delete", deps.CustomFieldHandler.DeleteDefinition)
			r.Post("/custom-fields/values/set", deps.CustomFieldHandler.SetValue)
			r.Get("/custom-fields/values/list", deps.CustomFieldHandler.ListValues)
			r.Delete("/custom-fields/values/unset", deps.CustomFieldHandler.UnsetValue)

			// Dashboard / статистика (read-only, для дашборда и графиков)
			r.Get("/stats/summary", deps.StatsHandler.Summary)
			r.Get("/stats/deals", deps.StatsHandler.Deals)
			r.Get("/stats/clients", deps.StatsHandler.Clients)
			r.Get("/stats/tasks", deps.StatsHandler.Tasks)
			r.Get("/stats/projects", deps.StatsHandler.Projects)
			r.Get("/stats/services", deps.StatsHandler.Services)
			r.Get("/stats/processes", deps.StatsHandler.Processes)
			r.Get("/stats/timeseries", deps.StatsHandler.TimeSeries)
		})
	})

	return r
}
