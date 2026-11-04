package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"school-management-system/internal/config"
	"school-management-system/internal/database"

	"school-management-system/internal/handlers"
	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

type databasePinger interface {
	Ping(ctx context.Context) error
}

func setupRouter(
	db databasePinger,
	authHandler *handlers.AuthHandler,
	tokenService middleware.AccessTokenValidator,
	principalLoader middleware.PrincipalLoader,
	permissionLoader middleware.PermissionLoader,
	schoolOnboardingHandlers ...*handlers.SchoolOnboardingHandler,
) (*gin.Engine, error) {
	var schoolOnboardingHandler *handlers.SchoolOnboardingHandler
	if len(schoolOnboardingHandlers) > 0 {
		schoolOnboardingHandler = schoolOnboardingHandlers[0]
	}
	return setupRouterWithHandlers(db, authHandler, tokenService, principalLoader, permissionLoader, schoolOnboardingHandler, nil)
}

func setupRouterWithHandlers(
	db databasePinger,
	authHandler *handlers.AuthHandler,
	tokenService middleware.AccessTokenValidator,
	principalLoader middleware.PrincipalLoader,
	permissionLoader middleware.PermissionLoader,
	schoolOnboardingHandler *handlers.SchoolOnboardingHandler,
	roleHandler *handlers.RoleHandler,
	staffHandlers ...*handlers.StaffHandler,
) (*gin.Engine, error) {
	router := gin.Default()

	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}

	if schoolOnboardingHandler != nil {
		router.POST("/platform/schools",
			middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader),
			middleware.RequirePermission("platform.schools.create"),
			middleware.RequirePermission("platform.school_admins.create"),
			schoolOnboardingHandler.Create,
		)
	}

	if len(staffHandlers) > 0 && staffHandlers[0] != nil {
		router.GET("/school/staff",
			middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader),
			middleware.RequirePermission("users.read"),
			staffHandlers[0].List,
		)
		router.POST("/school/staff",
			middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader),
			middleware.RequirePermission("users.create"),
			staffHandlers[0].Create,
		)
		router.DELETE("/school/staff/:principalID",
			middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader),
			middleware.RequirePermission("users.disable"),
			staffHandlers[0].Disable,
		)
		router.PATCH("/school/staff/:principalID",
			middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader),
			middleware.RequirePermission("users.update"),
			staffHandlers[0].Update,
		)
		router.POST("/school/staff/:principalID/enable",
			middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader),
			middleware.RequirePermission("users.update"),
			staffHandlers[0].Enable,
		)
	}

	if roleHandler != nil {
		platformAuth := middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader)
		router.POST("/platform/schools/:schoolID/permission-delegations",
			platformAuth,
			middleware.RequirePermission("platform.permissions.delegate"),
			roleHandler.GrantSchoolPermissionDelegation,
		)
		router.POST("/platform/permission-delegations/:delegationID/revoke",
			platformAuth,
			middleware.RequirePermission("platform.permissions.delegate"),
			roleHandler.RevokeSchoolPermissionDelegation,
		)

		roleAuth := middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader)
		router.GET("/school/roles",
			roleAuth,
			middleware.RequirePermission("roles.read"),
			roleHandler.ListRoles,
		)
		router.POST("/school/roles",
			roleAuth,
			middleware.RequirePermission("roles.create_custom"),
			roleHandler.CreateRole,
		)
		router.GET("/school/permissions",
			roleAuth,
			middleware.RequirePermission("permissions.read"),
			roleHandler.ListPermissions,
		)
		router.GET("/school/permission-delegations",
			roleAuth,
			middleware.RequirePermission("permissions.assign"),
			roleHandler.ListSchoolPermissionDelegations,
		)
		router.POST("/school/roles/:roleID/permissions",
			roleAuth,
			middleware.RequirePermission("permissions.assign"),
			middleware.RequirePermission("roles.update_custom"),
			roleHandler.GrantCustomRolePermission,
		)
		router.DELETE("/school/roles/:roleID/permissions/:permissionKey",
			roleAuth,
			middleware.RequirePermission("permissions.assign"),
			middleware.RequirePermission("roles.update_custom"),
			roleHandler.RevokeCustomRolePermission,
		)
		router.GET("/school/staff/:principalID/permission-overrides",
			roleAuth,
			middleware.RequirePermission("permissions.override"),
			roleHandler.ListStaffPermissionOverrides,
		)
		router.POST("/school/staff/:principalID/permission-overrides",
			roleAuth,
			middleware.RequirePermission("permissions.override"),
			roleHandler.SetStaffPermissionOverride,
		)
		router.DELETE("/school/staff/:principalID/permission-overrides/:permissionKey",
			roleAuth,
			middleware.RequirePermission("permissions.override"),
			roleHandler.DeleteStaffPermissionOverride,
		)
		router.GET("/school/staff/:principalID/roles",
			roleAuth,
			middleware.RequirePermission("roles.read"),
			roleHandler.ListStaffRoleAssignments,
		)
		router.POST("/school/staff/:principalID/roles/:roleID",
			roleAuth,
			middleware.RequirePermission("roles.assign"),
			roleHandler.AssignStaffRole,
		)
		router.DELETE("/school/staff/:principalID/roles/:roleID",
			roleAuth,
			middleware.RequirePermission("roles.assign"),
			roleHandler.RevokeStaffRole,
		)
	}

	if len(academicHandlers) > 0 && academicHandlers[0] != nil {
		academic := academicHandlers[0]
		academicAuth := middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader)
		router.GET("/school/academic-years", academicAuth, middleware.RequirePermission("academic_years.read"), academic.ListAcademicYears)
		router.POST("/school/academic-years", academicAuth, middleware.RequirePermission("academic_years.manage"), academic.SaveAcademicYear)
		router.PATCH("/school/academic-years/:yearID", academicAuth, middleware.RequirePermission("academic_years.manage"), academic.SaveAcademicYear)
		router.GET("/school/classes", academicAuth, middleware.RequirePermission("classes.read"), academic.ListClasses)
		router.POST("/school/classes", academicAuth, middleware.RequirePermission("classes.manage"), academic.SaveClass)
		router.PATCH("/school/classes/:classID", academicAuth, middleware.RequirePermission("classes.manage"), academic.SaveClass)
		router.GET("/school/sections", academicAuth, middleware.RequirePermission("classes.read"), academic.ListSections)
		router.POST("/school/sections", academicAuth, middleware.RequirePermission("classes.manage"), academic.SaveSection)
		router.PATCH("/school/sections/:sectionID", academicAuth, middleware.RequirePermission("classes.manage"), academic.SaveSection)
		router.GET("/school/subjects", academicAuth, middleware.RequirePermission("subjects.read"), academic.ListSubjects)
		router.POST("/school/subjects", academicAuth, middleware.RequirePermission("subjects.manage"), academic.SaveSubject)
		router.PATCH("/school/subjects/:subjectID", academicAuth, middleware.RequirePermission("subjects.manage"), academic.SaveSubject)
	}

	if authHandler != nil {
		router.POST("/auth/login", authHandler.Login)
		router.POST("/auth/platform-admin/login", authHandler.PlatformAdminLogin)
		router.POST("/auth/mfa/enroll/verify", authHandler.CompleteMFAEnrollment)
		router.POST("/auth/mfa/verify", authHandler.VerifyMFA)
		router.POST("/auth/mfa/recovery-codes/regenerate/start",
			middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader),
			authHandler.BeginMFARecoveryCodeRegeneration)
		router.POST("/auth/mfa/recovery-codes/regenerate", authHandler.RegenerateMFARecoveryCodes)
		router.POST("/auth/refresh", authHandler.Refresh)
		router.POST("/auth/logout", authHandler.Logout)
		router.POST("/auth/password/change",
			middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader),
			authHandler.ChangeStaffPassword)
		router.POST("/school/staff/:principalID/password/reset",
			middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader),
			middleware.RequirePermission("users.password_reset"),
			authHandler.ResetStaffPassword)
	}

	router.GET("/auth/me", middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader), func(c *gin.Context) {
		principalID, _ := c.Get(middleware.ContextPrincipalIDKey)
		principalType, _ := c.Get(middleware.ContextPrincipalTypeKey)
		schoolID, _ := c.Get(middleware.ContextSchoolIDKey)
		username, _ := c.Get(middleware.ContextUsernameKey)
		mustChangePassword, _ := c.Get(middleware.ContextMustChangePasswordKey)

		response := gin.H{
			"status":         "success",
			"principal_id":   principalID,
			"principal_type": principalType,
		}
		// Parent and Platform Admin identities are not school-scoped staff
		// accounts. Do not serialize empty/zero staff-only fields for them.
		if principalType == "staff" {
			response["school_id"] = schoolID
			response["username"] = username
			response["must_change_password"] = mustChangePassword
		}
		c.JSON(http.StatusOK, response)
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "School Management System API is running",
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"message": "Database is not ready",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "API and database are ready",
		})
	})

	return router, nil
}

func main() {
	if err := run(); err != nil {
		log.Printf("Application failed: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	if err := cfg.ValidateDatabase(); err != nil {
		return fmt.Errorf("invalid database configuration: %w", err)
	}

	if err := cfg.ValidateJWT(); err != nil {
		return fmt.Errorf("invalid JWT configuration: %w", err)
	}
	if err := cfg.ValidateTOTPEncryption(); err != nil {
		return fmt.Errorf("invalid TOTP encryption configuration: %w", err)
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := database.Connect(ctx, cfg)
	cancel()

	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()

	log.Println("PostgreSQL connected successfully")

	userRepo := repository.NewUserRepository(db)
	schoolRepo := repository.NewSchoolRepository(db)

	loginSecurityRepo := repository.NewStaffLoginSecurityRepository(db)
	authService := services.NewAuthService(userRepo, schoolRepo, loginSecurityRepo)
	platformAdminRepo := repository.NewPlatformAdminRepository(db)
	platformAdminAuthService, err := services.NewPlatformAdminAuthService(platformAdminRepo, loginSecurityRepo)
	if err != nil {
		return fmt.Errorf("initialize Platform Admin authentication: %w", err)
	}
	refreshSessionRepo := repository.NewPrincipalRefreshSessionRepository(db)
	principalAuthorizationRepo := repository.NewPrincipalAuthorizationRepository(db)
	mfaChallengeRepo := repository.NewMFAChallengeRepository(db)
	mfaRepo := repository.NewMFARepository(db)
	mfaService, err := services.NewMFAService(
		mfaRepo,
		mfaChallengeRepo,
		cfg.TOTPEncryptionKey,
		cfg.TOTPEncryptionKeyVersion,
		cfg.JWTIssuer,
	)
	if err != nil {
		return fmt.Errorf("initialize MFA service: %w", err)
	}

	sessionService, err := services.NewSessionService(
		refreshSessionRepo,
		cfg.RefreshTokenExpiryDays,
	)
	if err != nil {
		return fmt.Errorf("initialize refresh session service: %w", err)
	}

	tokenService, err := services.NewTokenService(
		cfg.JWTSecret,
		cfg.JWTIssuer,
		cfg.JWTAccessExpiryMinutes,
	)
	if err != nil {
		return fmt.Errorf("initialize token service: %w", err)
	}

	authHandler := handlers.NewAuthHandler(
		authService,
		tokenService,
		sessionService,
		loginSecurityRepo,
	)
	authHandler.SetMFAService(mfaService, principalAuthorizationRepo)
	authHandler.SetPlatformAdminAuthService(platformAdminAuthService)
	staffPasswordService, err := services.NewStaffPasswordService(userRepo)
	if err != nil {
		return fmt.Errorf("initialize staff password service: %w", err)
	}
	authHandler.SetStaffPasswordService(staffPasswordService)

	schoolOnboardingRepo := repository.NewSchoolOnboardingRepository(db)
	schoolOnboardingService := services.NewSchoolOnboardingService(schoolOnboardingRepo)
	schoolOnboardingHandler := handlers.NewSchoolOnboardingHandler(schoolOnboardingService)

	roleRepo := repository.NewRoleRepository(db)
	roleService := services.NewRoleService(roleRepo)
	roleHandler := handlers.NewRoleHandler(roleService)
	academicRepo := repository.NewAcademicRepository(db)
	academicService := services.NewAcademicService(academicRepo)
	academicHandler := handlers.NewAcademicHandler(academicService)

	staffAccountRepo := repository.NewUserRepository(db)
	staffAccountService := services.NewStaffAccountService(staffAccountRepo)
	staffHandler := handlers.NewStaffHandler(staffAccountService)

	router, err := setupRouterWithHandlers(db, authHandler, tokenService, principalAuthorizationRepo, principalAuthorizationRepo, schoolOnboardingHandler, roleHandler, staffHandler, academicHandler)

	if err != nil {
		return fmt.Errorf("setup router: %w", err)
	}

	server := &http.Server{
		Addr:              net.JoinHostPort("", cfg.AppPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", server.Addr, err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	log.Printf("Starting API in %s mode on port %s", cfg.AppEnv, cfg.AppPort)

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.Serve(listener)
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server failed: %w", err)
		}
		return nil

	case <-stop:
		log.Println("Shutting down API server...")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)

		if closeErr := server.Close(); closeErr != nil &&
			!errors.Is(closeErr, net.ErrClosed) {
			log.Printf("Failed to force close server: %v", closeErr)
		}
	}

	err = <-serverErr
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server stopped unexpectedly: %w", err)
	}

	log.Println("API server stopped")
	return nil
}
