package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/handlers"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/middleware"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/config"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/service"
	"github.com/rs/zerolog"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server encapsulates all dependencies of our API.
// It now includes the http.Server instance for lifecycle management.
type Server struct {
	config     config.Config
	db         *sqlx.DB
	router     *gin.Engine
	httpServer *http.Server
}

// NewServer creates and configures a new instance of the API server.
func NewServer(cfg config.Config, db *sqlx.DB, logger *zerolog.Logger) *Server {
	router := gin.New()

	server := &Server{
		config: cfg,
		db:     db,
		router: router,
	}

	server.setupRouter(logger)

	// Create the http.Server instance
	server.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: server.router,
	}

	return server
}

// Start runs the HTTP server. This call is blocking.
func (s *Server) Start() error {
	// ListenAndServe blocks until an error occurs or the server is shut down.
	// We check for ErrServerClosed to know if it was a graceful shutdown.
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server with a timeout.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// setupRouter configura todos os middlewares e rotas da API.
func (s *Server) setupRouter(logger *zerolog.Logger) {
	// --- Injeção de Dependências ---
	// Repositórios
	userRepo := repository.NewUserRepository(s.db)
	accountRepo := repository.NewAccountRepository(s.db)
	categoryRepo := repository.NewCategoryRepository(s.db)
	transactionRepo := repository.NewTransactionRepository(s.db)
	budgetRepo := repository.NewBudgetRepository(s.db)

	// Serviços
	authService := service.NewAuthService(userRepo, s.config.JWTSecretKey)
	userService := service.NewUserService(userRepo, categoryRepo)
	accountService := service.NewAccountService(accountRepo, transactionRepo)
	categoryService := service.NewCategoryService(categoryRepo, transactionRepo)
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
	budgetService := service.NewBudgetService(budgetRepo, categoryRepo, transactionRepo)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	accountHandler := handlers.NewAccountHandler(accountService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	transactionHandler := handlers.NewTransactionHandler(transactionService)
	budgetHandler := handlers.NewBudgetHandler(budgetService)

	// --- Middlewares Globais ---
	s.router.Use(middleware.LoggerMiddleware(*logger))
	s.router.Use(cors.New(s.buildCORSConfig()))

	// --- Registro de Rotas ---
	// Rota do Swagger
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Agrupamento de rotas v1
	v1 := s.router.Group("/v1")
	{
		// Rotas Públicas
		authRoutes := v1.Group("/auth")
		{
			authRoutes.POST("/login", authHandler.Login)
		}
		usersPublicRoutes := v1.Group("/users")
		{
			usersPublicRoutes.POST("", userHandler.CreateUser)
		}

		// Rotas Protegidas
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(s.config.JWTSecretKey, *logger))
		{
			userRoutes := protected.Group("/users")
			{
				userRoutes.GET("/me", userHandler.GetProfile)
			}

			accounts := protected.Group("/accounts")
			{
				accounts.POST("", accountHandler.CreateAccount)
				accounts.GET("", accountHandler.ListAccounts)
				accounts.GET("/:id", accountHandler.GetAccount)
				accounts.PUT("/:id", accountHandler.UpdateAccount)
				accounts.DELETE("/:id", accountHandler.DeleteAccount)
			}

			budgetRoutes := protected.Group("/budgets")
			{
				budgetRoutes.GET("", budgetHandler.ListBudgets)
				budgetRoutes.POST("", budgetHandler.CreateBudget)
				budgetRoutes.PUT("/:id", budgetHandler.UpdateBudget)
				budgetRoutes.DELETE("/:id", budgetHandler.DeleteBudget)
			}

			categories := protected.Group("/categories")
			{
				categories.POST("", categoryHandler.CreateCategory)
				categories.GET("", categoryHandler.ListCategories)
				categories.GET("/:id", categoryHandler.GetCategory)
				categories.PUT("/:id", categoryHandler.UpdateCategory)
				categories.DELETE("/:id", categoryHandler.DeleteCategory)
			}

			transactions := protected.Group("/transactions")
			{
				transactions.POST("", transactionHandler.CreateTransaction)
				transactions.GET("", transactionHandler.ListTransactions)
				transactions.GET("/:id", transactionHandler.GetTransaction)
				transactions.PUT("/:id", transactionHandler.UpdateTransaction)
				transactions.PATCH("/:id", transactionHandler.PatchTransaction)
				transactions.DELETE("/:id", transactionHandler.DeleteTransaction)
			}
		}
	}
}

// buildCORSConfig builds the CORS configuration.
func (s *Server) buildCORSConfig() cors.Config {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	return config
}
