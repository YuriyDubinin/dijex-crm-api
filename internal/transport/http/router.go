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

	HealthHandler        *handler.HealthHandler
	AuthHandler          *handler.AuthHandler
	MeHandler            *handler.MeHandler
	CustomerHandler      *handler.CustomerHandler
	UserHandler          *handler.UserHandler
	ProjectHandler       *handler.ProjectHandler
	ServiceHandler       *handler.ServiceHandler
	ClientHandler        *handler.ClientHandler
	TaskHandler          *handler.TaskHandler
	CalendarEventHandler *handler.CalendarEventHandler
	DealHandler          *handler.DealHandler
	OverviewHandler      *handler.OverviewHandler
	EntityLinkHandler    *handler.EntityLinkHandler
	TagHandler           *handler.TagHandler
	CustomFieldHandler   *handler.CustomFieldHandler
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

			// Tasks
			r.Post("/tasks/create", deps.TaskHandler.Create)
			r.Get("/tasks/list", deps.TaskHandler.List)
			r.Put("/tasks/update", deps.TaskHandler.Update)
			r.Delete("/tasks/delete", deps.TaskHandler.Delete)

			// Calendar events
			r.Post("/calendar-events/create", deps.CalendarEventHandler.Create)
			r.Get("/calendar-events/list", deps.CalendarEventHandler.List)
			r.Put("/calendar-events/update", deps.CalendarEventHandler.Update)
			r.Delete("/calendar-events/delete", deps.CalendarEventHandler.Delete)
			r.Post("/calendar-events/attendees/add", deps.CalendarEventHandler.AddAttendee)
			r.Delete("/calendar-events/attendees/remove", deps.CalendarEventHandler.RemoveAttendee)
			r.Get("/calendar-events/attendees/list", deps.CalendarEventHandler.ListAttendees)

			// Deals (сделки)
			r.Post("/deals/create", deps.DealHandler.Create)
			r.Get("/deals/list", deps.DealHandler.List)
			r.Put("/deals/update", deps.DealHandler.Update)
			r.Delete("/deals/delete", deps.DealHandler.Delete)
			r.Get("/deals/overview", deps.OverviewHandler.DealOverview)

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
		})
	})

	return r
}
