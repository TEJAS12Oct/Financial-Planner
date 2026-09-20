package routes

import (
	controller "FinancialPlannerGo/app/controllers/FinancialPlanner"
	"FinancialPlannerGo/app/middleware"
	repo "FinancialPlannerGo/app/repositories/FinancialPlanner"
	appsecurity "FinancialPlannerGo/app/security"
	service "FinancialPlannerGo/app/services/FinancialPlanner"
	"FinancialPlannerGo/config"
	"FinancialPlannerGo/docs"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type HealthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func RegisterFinancialPlannerRoutes(
	router *gin.Engine,
) {
	securityConfig, err := appsecurity.LoadConfig()
	if err != nil {
		log.Fatal(
			"invalid security configuration: ",
			err,
		)
	}

	reportSigner :=
		appsecurity.NewReportSigner(securityConfig)

	financialPlannerRepo :=
		repo.NewFinancialPlannerRepository(config.DB)

	financialPlannerService :=
		service.NewFinancialPlannerService(
			financialPlannerRepo,
		)

	ctrl :=
		controller.NewFinancialPlannerController(
			financialPlannerService,
			reportSigner,
		)

	apiKeyAuth :=
		middleware.RequireAPIKey(securityConfig)

	allowedIPOnly :=
		middleware.RestrictToAllowedIPs(
			securityConfig,
		)

	signedReportOnly :=
		middleware.RequireSignedReportURL(
			reportSigner,
		)

	reportRateLimiter :=
		middleware.NewReportRateLimiter(
			securityConfig,
		)

	/*
		Utility and documentation routes:

		GET /financial-planner/
		GET /financial-planner/health
		GET /boltpe-financial-planner/apidocs/
	*/
	registerUtilityRoutes(
		router,
		ctrl,
	)

	registerSwaggerRoutes(router)

	/*
		BoltPe public APIs:

		POST /api/boltpe-financial-planner/report
		POST /api/boltpe-financial-planner/report-pdf-direct
		GET  /api/boltpe-financial-planner/reports
		GET  /api/boltpe-financial-planner/smtp-status
		POST /api/boltpe-financial-planner/test-email
		GET  /api/boltpe-financial-planner/reports/:filename
	*/
	registerBoltPeAPIRoutes(
		router,
		ctrl,
		apiKeyAuth,
		allowedIPOnly,
		signedReportOnly,
		reportRateLimiter.Middleware(),
	)
}

func registerUtilityRoutes(
	router *gin.Engine,
	ctrl *controller.FinancialPlannerController,
) {
	// Primary service paths. Register both forms so clients do not depend on
	// Gin's trailing-slash redirect behaviour.
	router.GET("/boltpe-financial-planner", ctrl.Home)
	router.GET("/boltpe-financial-planner/", ctrl.Home)
	router.GET("/boltpe-financial-planner/health", healthCheck)

	// Proxy-compatible aliases. Some ingress configurations strip the
	// /boltpe-financial-planner prefix before forwarding to this service.
	router.GET("/", ctrl.Home)
	router.GET("/health", healthCheck)
}

// healthCheck godoc
// @Summary Check Financial Planner service health
// @ID get_financial_planner_health
// @Description Returns the current BoltPe Financial Planner service status.
// @Tags Pages
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /boltpe-financial-planner/health [get]
func healthCheck(c *gin.Context) {
	c.JSON(
		http.StatusOK,
		HealthResponse{
			Success: true,
			Message: "Financial Planner service is running",
		},
	)
}

func registerBoltPeAPIRoutes(
	router *gin.Engine,
	ctrl *controller.FinancialPlannerController,
	apiKeyAuth gin.HandlerFunc,
	allowedIPOnly gin.HandlerFunc,
	signedReportOnly gin.HandlerFunc,
	reportRateLimit gin.HandlerFunc,
) {
	api :=
		router.Group("/api/boltpe-financial-planner")

	api.GET(
		"/smtp-status",
		// apiKeyAuth,
		allowedIPOnly,
		ctrl.SMTPStatus,
	)

	api.POST(
		"/test-email",
		// apiKeyAuth,
		allowedIPOnly,
		ctrl.TestEmail,
	)

	api.POST(
		"/report",
		// apiKeyAuth,
		reportRateLimit,
		ctrl.GeneratePublicReport,
	)

	api.POST(
		"/report-json",
		// apiKeyAuth,
		reportRateLimit,
		ctrl.GenerateStructuredJSONReport,
	)

	api.POST(
		"/report-pdf-direct",
		// apiKeyAuth,
		reportRateLimit,
		ctrl.GeneratePDFDirect,
	)

	api.GET(
		"/reports",
		// apiKeyAuth,
		ctrl.ListReports,
	)

	api.GET(
		"/reports/*reportPath",
		signedReportOnly,
		ctrl.ServeReport,
	)
}

func registerSwaggerRoutes(
	router *gin.Engine,
) {
	registerSwaggerSet := func(docsPath, openAPIPath string) {
		swaggerHandler := ginSwagger.WrapHandler(
			swaggerFiles.Handler,
			ginSwagger.URL(openAPIPath),
			ginSwagger.DocExpansion("list"),
			ginSwagger.DeepLinking(true),
			ginSwagger.PersistAuthorization(true),
			ginSwagger.DefaultModelsExpandDepth(-1),
		)

		router.GET(docsPath, func(c *gin.Context) {
			c.Redirect(http.StatusTemporaryRedirect, docsPath+"/index.html")
		})

		router.GET(docsPath+"/*any", func(c *gin.Context) {
			if c.Param("any") == "/" {
				c.Redirect(http.StatusTemporaryRedirect, docsPath+"/index.html")
				return
			}
			swaggerHandler(c)
		})

		router.GET(openAPIPath, func(c *gin.Context) {
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
			c.String(http.StatusOK, docs.SwaggerInfo.ReadDoc())
		})
	}

	// Normal direct-service paths.
	registerSwaggerSet(
		"/boltpe-financial-planner/apidocs",
		"/boltpe-financial-planner/openapi.json",
	)

	// Compatibility paths for ingress/proxies that strip the service prefix.
	registerSwaggerSet("/apidocs", "/openapi.json")
}
