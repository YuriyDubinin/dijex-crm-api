package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/YuriyDubinin/bms-api/internal/config"
	"github.com/YuriyDubinin/bms-api/internal/repository/postgres"
	"github.com/YuriyDubinin/bms-api/internal/service"
	transporthttp "github.com/YuriyDubinin/bms-api/internal/transport/http"
	"github.com/YuriyDubinin/bms-api/internal/transport/http/handler"
	"github.com/YuriyDubinin/bms-api/pkg/crypto"
	"github.com/YuriyDubinin/bms-api/pkg/logger"
	"github.com/YuriyDubinin/bms-api/pkg/validator"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Log.Level, cfg.App.Env)
	log.Info("service starting", "env", cfg.App.Env, "http_port", cfg.HTTP.Port)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.Postgres.DSN(), cfg.Postgres.MaxConns)
	if err != nil {
		log.Error("connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("database connected")

	if err := postgres.RunMigrations(cfg.Postgres.DSN(), "migrations", log); err != nil {
		log.Error("run migrations", "err", err)
		os.Exit(1)
	}

	// ───────── Репозитории ─────────
	authTokenRepo := postgres.NewAuthTokenRepository(pool)
	userRepo := postgres.NewCustomerUserRepository(pool)
	customerRepo := postgres.NewCustomerRepository(pool)
	entityRepo := postgres.NewEntityRepository(pool)
	projectRepo := postgres.NewProjectRepository(pool)
	serviceRepo := postgres.NewServiceRepository(pool)
	clientRepo := postgres.NewClientRepository(pool)
	taskRepo := postgres.NewTaskRepository(pool)
	dealRepo := postgres.NewDealRepository(pool)
	processRepo := postgres.NewProcessRepository(pool)
	entityLinkRepo := postgres.NewEntityLinkRepository(pool)
	tagRepo := postgres.NewTagRepository(pool)
	customFieldRepo := postgres.NewCustomFieldRepository(pool)
	statsRepo := postgres.NewStatsRepository(pool)

	// ───────── Крипто ─────────
	passwordHasher, err := crypto.NewPasswordHasher(cfg.Auth.PasswordHashCost)
	if err != nil {
		log.Error("init password hasher", "err", err)
		os.Exit(1)
	}

	// ───────── Сервисы ─────────
	authService, err := service.NewAuthService(authTokenRepo, userRepo, customerRepo, passwordHasher, cfg.Auth.TokenTTL, log)
	if err != nil {
		log.Error("init auth service", "err", err)
		os.Exit(1)
	}
	customerService := service.NewCustomerService(customerRepo, log)
	userService := service.NewCustomerUserService(userRepo, passwordHasher, log)
	projectService := service.NewProjectService(projectRepo, log)
	serviceService := service.NewServiceService(serviceRepo, projectRepo, log)
	clientService := service.NewClientService(clientRepo, projectRepo, log)
	taskService := service.NewTaskService(taskRepo, projectRepo, clientRepo, dealRepo, serviceRepo, processRepo, processRepo, userRepo, log)
	dealService := service.NewDealService(dealRepo, projectRepo, clientRepo, serviceRepo, processRepo, userRepo, log)
	processService := service.NewProcessService(processRepo, projectRepo, log)
	overviewService := service.NewOverviewService(dealRepo, clientRepo, projectRepo, taskRepo, serviceRepo, processRepo, entityLinkRepo, log)
	entityLinkService := service.NewEntityLinkService(entityLinkRepo, entityRepo, log)
	tagService := service.NewTagService(tagRepo, entityRepo, log)
	customFieldService := service.NewCustomFieldService(customFieldRepo, entityRepo, log)
	statsService := service.NewStatsService(statsRepo, log)
	agendaService := service.NewAgendaService(taskRepo, dealRepo, projectRepo, processRepo, log)

	v := validator.New()

	// ───────── Хендлеры ─────────
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authService, v, log)
	meHandler := handler.NewMeHandler()
	customerHandler := handler.NewCustomerHandler(customerService, v, log)
	userHandler := handler.NewUserHandler(userService, v, log)
	projectHandler := handler.NewProjectHandler(projectService, v, log)
	serviceHandler := handler.NewServiceHandler(serviceService, v, log)
	clientHandler := handler.NewClientHandler(clientService, v, log)
	taskHandler := handler.NewTaskHandler(taskService, v, log)
	dealHandler := handler.NewDealHandler(dealService, v, log)
	processHandler := handler.NewProcessHandler(processService, v, log)
	overviewHandler := handler.NewOverviewHandler(overviewService, log)
	entityLinkHandler := handler.NewEntityLinkHandler(entityLinkService, v, log)
	tagHandler := handler.NewTagHandler(tagService, v, log)
	customFieldHandler := handler.NewCustomFieldHandler(customFieldService, v, log)
	statsHandler := handler.NewStatsHandler(statsService, log)
	agendaHandler := handler.NewAgendaHandler(agendaService, log)

	router := transporthttp.NewRouter(transporthttp.Deps{
		Logger:             log,
		Authenticator:      authService,
		HealthHandler:      healthHandler,
		AuthHandler:        authHandler,
		MeHandler:          meHandler,
		CustomerHandler:    customerHandler,
		UserHandler:        userHandler,
		ProjectHandler:     projectHandler,
		ServiceHandler:     serviceHandler,
		ClientHandler:      clientHandler,
		TaskHandler:        taskHandler,
		DealHandler:        dealHandler,
		ProcessHandler:     processHandler,
		OverviewHandler:    overviewHandler,
		EntityLinkHandler:  entityLinkHandler,
		TagHandler:         tagHandler,
		CustomFieldHandler: customFieldHandler,
		StatsHandler:       statsHandler,
		AgendaHandler:      agendaHandler,
	})
	srv := transporthttp.NewServer(cfg.HTTP, router, log)

	log.Info("http server starting on :" + cfg.HTTP.Port)
	log.Info("ready")

	if err := srv.Run(ctx); err != nil {
		log.Error("http server", "err", err)
		os.Exit(1)
	}

	log.Info("service stopped")
}
