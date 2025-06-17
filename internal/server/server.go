package server

import (
	"fmt"

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

// Server encapsula todas as dependências da nossa API.
type Server struct {
	logger *zerolog.Logger
	config config.Config
	db     *sqlx.DB
	router *gin.Engine
}

// NewServer cria e configura uma nova instância do servidor da API.
func NewServer(cfg config.Config, db *sqlx.DB, logger *zerolog.Logger) *Server {
	server := &Server{
		logger: logger,
		config: cfg,
		db:     db,
	}

	router := gin.Default()
	server.setupRouter(router, logger)
	server.router = router

	return server
}

// Start inicia o servidor HTTP na porta configurada.
func (s *Server) Start() error {
	serverPort := fmt.Sprintf(":%s", s.config.ServerPort)
	return s.router.Run(serverPort)
}

// setupRouter configura todos os middlewares e rotas da API.
func (s *Server) setupRouter(router *gin.Engine, logger *zerolog.Logger) {
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
	router.Use(middleware.LoggerMiddleware(*logger))
	router.Use(cors.New(s.buildCORSConfig()))

	// --- Registro de Rotas ---
	// Rota do Swagger
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Agrupamento de rotas v1
	v1 := router.Group("/v1")
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

// buildCORSConfig constrói a configuração de CORS a partir do config.
func (s *Server) buildCORSConfig() cors.Config {
	config := cors.DefaultConfig()
	// Você pode buscar as origins do s.config.CORS.AllowOrigins
	config.AllowAllOrigins = true // Simplificado por enquanto
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	return config
}

func CORSConfig(allowOrigins []string) *cors.Config {
	// --- CONFIGURAÇÃO DO CORS ---
	// Esta configuração deve vir ANTES do registro das suas rotas.
	config := cors.DefaultConfig()

	// Especifique os domínios que podem acessar sua API.
	// Para desenvolvimento, você pode usar o endereço do seu frontend local.
	config.AllowOrigins = allowOrigins

	// Em produção, você colocaria o domínio do seu frontend.
	// Ex: config.AllowOrigins = []string{"https://www.meusistema.com"}

	// Para permitir qualquer origem (NÃO RECOMENDADO PARA PRODUÇÃO):
	// config.AllowAllOrigins = true

	// Métodos HTTP permitidos
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}

	// Cabeçalhos HTTP permitidos
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}

	return &config
}
