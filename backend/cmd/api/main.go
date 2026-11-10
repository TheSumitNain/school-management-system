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
	"sort"
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
	optionalHandlers ...any,
) (*gin.Engine, error) {
	var staffHandler *handlers.StaffHandler
	var academicHandler *handlers.AcademicHandler
	var guardianHandler *handlers.GuardianHandler
	var attendanceHandler *handlers.AttendanceHandler
	var parentAuthHandler *handlers.ParentAuthHandler
	var platformAdminLifecycleHandler *handlers.PlatformAdminLifecycleHandler
	var sessionActivityTracker middleware.AdminSessionActivityTracker
	for _, optional := range optionalHandlers {
		switch handler := optional.(type) {
		case *handlers.StaffHandler:
			staffHandler = handler
		case *handlers.AcademicHandler:
			academicHandler = handler
		case *handlers.GuardianHandler:
			guardianHandler = handler
		case *handlers.AttendanceHandler:
			attendanceHandler = handler
		case *handlers.ParentAuthHandler:
			parentAuthHandler = handler
		case *handlers.PlatformAdminLifecycleHandler:
			platformAdminLifecycleHandler = handler
		case middleware.AdminSessionActivityTracker:
			sessionActivityTracker = handler
		}
	}
	router := gin.Default()
	authMiddleware := func() gin.HandlerFunc {
		return middleware.AuthMiddleware(tokenService, principalLoader, permissionLoader, sessionActivityTracker)
	}

	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}

	if platformAdminLifecycleHandler != nil {
		platformAdminAuth := authMiddleware()
		router.GET("/platform/admins", platformAdminAuth, middleware.RequirePermission("platform.admins.manage"), platformAdminLifecycleHandler.List)
		router.POST("/platform/admins", platformAdminAuth, middleware.RequirePermission("platform.admins.manage"), platformAdminLifecycleHandler.Create)
		router.POST("/platform/admins/:principalID/deactivate", platformAdminAuth, middleware.RequirePermission("platform.admins.manage"), platformAdminLifecycleHandler.Deactivate)
	}

	if schoolOnboardingHandler != nil {
		router.POST("/platform/schools",
			authMiddleware(),
			middleware.RequirePermission("platform.schools.create"),
			middleware.RequirePermission("platform.school_admins.create"),
			schoolOnboardingHandler.Create,
		)
	}

	if staffHandler != nil {
		router.GET("/school/staff",
			authMiddleware(),
			middleware.RequirePermission("users.read"),
			staffHandler.List,
		)
		router.POST("/school/staff",
			authMiddleware(),
			middleware.RequirePermission("users.create"),
			staffHandler.Create,
		)
		router.DELETE("/school/staff/:principalID",
			authMiddleware(),
			middleware.RequirePermission("users.disable"),
			staffHandler.Disable,
		)
		router.PATCH("/school/staff/:principalID",
			authMiddleware(),
			middleware.RequirePermission("users.update"),
			staffHandler.Update,
		)
		router.POST("/school/staff/:principalID/enable",
			authMiddleware(),
			middleware.RequirePermission("users.update"),
			staffHandler.Enable,
		)
	}

	if roleHandler != nil {
		platformAuth := authMiddleware()
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

		roleAuth := authMiddleware()
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
		router.GET("/school/roles/:roleID/permissions",
			roleAuth,
			middleware.RequirePermission("roles.read"),
			roleHandler.ListCustomRolePermissions,
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

	if academicHandler != nil {
		academic := academicHandler
		academicAuth := authMiddleware()
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
		router.GET("/school/eligible-teachers",
			academicAuth,
			middleware.RequireAnyPermission("teacher_assignments.read", "class_incharge_assignments.read"),
			academic.ListEligibleTeachers,
		)
		router.GET("/school/teacher-assignments", academicAuth, middleware.RequirePermission("teacher_assignments.read"), academic.ListTeacherSubjectAssignments)
		router.POST("/school/teacher-assignments", academicAuth, middleware.RequirePermission("teacher_assignments.manage"), academic.CreateTeacherSubjectAssignment)
		router.DELETE("/school/teacher-assignments/:assignmentID", academicAuth, middleware.RequirePermission("teacher_assignments.manage"), academic.RevokeTeacherSubjectAssignment)
		router.GET("/school/class-incharge-assignments", academicAuth, middleware.RequirePermission("class_incharge_assignments.read"), academic.ListClassInchargeAssignments)
		router.POST("/school/class-incharge-assignments", academicAuth, middleware.RequirePermission("class_incharge_assignments.manage"), academic.CreateClassInchargeAssignment)
		router.DELETE("/school/class-incharge-assignments/:assignmentID", academicAuth, middleware.RequirePermission("class_incharge_assignments.manage"), academic.RevokeClassInchargeAssignment)
		router.GET("/school/students", academicAuth, middleware.RequirePermission("students.read_sensitive"), academic.ListStudents)
		router.POST("/school/students", academicAuth, middleware.RequirePermission("students.create"), academic.CreateStudent)
		router.GET("/school/students/:studentID", academicAuth, middleware.RequirePermission("students.read_sensitive"), academic.GetStudent)
		router.PATCH("/school/students/:studentID", academicAuth, middleware.RequirePermission("students.update"), academic.UpdateStudent)
		router.POST("/school/students/:studentID/archive", academicAuth, middleware.RequirePermission("students.archive"), academic.ArchiveStudent)
		router.GET("/school/enrollments", academicAuth, middleware.RequirePermission("students.read_basic"), academic.ListStudentEnrollments)
		router.POST("/school/enrollments", academicAuth, middleware.RequirePermission("students.create"), academic.CreateStudentEnrollment)
		router.GET("/school/students/:studentID/enrollments", academicAuth, middleware.RequirePermission("students.read_basic"), academic.ListStudentEnrollmentHistory)
		router.POST("/school/enrollments/:enrollmentID/promote", academicAuth, middleware.RequirePermission("students.update"), academic.PromoteStudentEnrollment)
	}

	if attendanceHandler != nil {
		attendanceAuth := authMiddleware()
		router.GET("/school/attendance", attendanceAuth, middleware.RequirePermission("attendance.read"), attendanceHandler.List)
		router.POST("/school/attendance", attendanceAuth, middleware.RequireAnyPermission("attendance.mark", "attendance.correct_past"), attendanceHandler.Mark)
		router.PATCH("/school/attendance/:attendanceID", attendanceAuth, middleware.RequireAnyPermission("attendance.update_same_day", "attendance.correct_past"), attendanceHandler.Update)
		router.PATCH("/school/attendance/:attendanceID/correct", attendanceAuth, middleware.RequireAnyPermission("attendance.update_same_day", "attendance.correct_past"), attendanceHandler.Update)
	}

	if guardianHandler != nil {
		guardianAuth := authMiddleware()
		router.GET("/school/guardians", guardianAuth, middleware.RequirePermission("guardians.read"), guardianHandler.ListGuardians)
		router.POST("/school/guardians", guardianAuth, middleware.RequirePermission("guardians.manage"), guardianHandler.SaveGuardian)
		router.GET("/school/guardians/:guardianID", guardianAuth, middleware.RequirePermission("guardians.read"), guardianHandler.GetGuardian)
		router.PATCH("/school/guardians/:guardianID", guardianAuth, middleware.RequirePermission("guardians.manage"), guardianHandler.SaveGuardian)
		router.GET("/school/students/:studentID/guardians", guardianAuth, middleware.RequirePermission("guardians.read"), guardianHandler.ListStudentGuardians)
		router.POST("/school/students/:studentID/guardians", guardianAuth, middleware.RequirePermission("guardians.manage"), guardianHandler.LinkStudentGuardian)
		router.POST("/school/student-guardian-links/:linkID/revoke", guardianAuth, middleware.RequirePermission("guardians.manage"), guardianHandler.RevokeStudentGuardianLink)
	}

	if authHandler != nil {
		router.POST("/auth/login", authHandler.Login)
		router.POST("/auth/platform-admin/login", authHandler.PlatformAdminLogin)
		router.POST("/auth/mfa/enroll/verify", authHandler.CompleteMFAEnrollment)
		router.POST("/auth/mfa/verify", authHandler.VerifyMFA)
		router.POST("/auth/mfa/recovery-codes/regenerate/start",
			authMiddleware(),
			authHandler.BeginMFARecoveryCodeRegeneration)
		router.POST("/auth/mfa/recovery-codes/regenerate", authMiddleware(), authHandler.RegenerateMFARecoveryCodes)
		router.POST("/auth/refresh", authHandler.Refresh)
		router.POST("/auth/logout", authHandler.Logout)
		router.POST("/auth/password/change",
			authMiddleware(),
			authHandler.ChangeStaffPassword)
		router.POST("/auth/platform-admin/password/change",
			authMiddleware(),
			authHandler.ChangePlatformAdminPassword)
		router.POST("/school/staff/:principalID/password/reset",
			authMiddleware(),
			middleware.RequirePermission("users.password_reset"),
			authHandler.ResetStaffPassword)
	}

	if parentAuthHandler != nil {
		router.POST("/auth/parent/otp/request", parentAuthHandler.RequestOTP)
		router.POST("/auth/parent/otp/verify", parentAuthHandler.VerifyOTP)
	}

	router.GET("/auth/me", authMiddleware(), func(c *gin.Context) {
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
		// Return only the caller's current effective permissions. These values
		// help the frontend render role-aware navigation; every protected API
		// request still enforces authorization independently on the server.
		if principalType == "staff" || principalType == "platform_admin" {
			permissionValues := make([]string, 0)
			if value, exists := c.Get(middleware.ContextPermissionsKey); exists {
				if granted, ok := value.(map[string]struct{}); ok {
					for permission := range granted {
						permissionValues = append(permissionValues, permission)
					}
				}
			}
			sort.Strings(permissionValues)
			response["permissions"] = permissionValues
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
	previousTOTPKeys := make(map[int][]byte, len(cfg.TOTPEncryptionPreviousKeys))
	for _, previousKey := range cfg.TOTPEncryptionPreviousKeys {
		previousTOTPKeys[previousKey.Version] = previousKey.Key
	}
	mfaService, err := services.NewMFAServiceWithKeyRing(
		mfaRepo,
		mfaChallengeRepo,
		cfg.TOTPEncryptionKey,
		cfg.TOTPEncryptionKeyVersion,
		previousTOTPKeys,
		cfg.JWTIssuer,
	)
	if err != nil {
		return fmt.Errorf("initialize MFA service: %w", err)
	}

	sessionService, err := services.NewSessionService(refreshSessionRepo)
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
	authHandler.SetRefreshCookiePolicy(cfg.AppEnv == "production")
	authHandler.SetMFAService(mfaService, principalAuthorizationRepo)
	authHandler.SetPlatformAdminAuthService(platformAdminAuthService)
	staffPasswordService, err := services.NewStaffPasswordService(userRepo)
	if err != nil {
		return fmt.Errorf("initialize staff password service: %w", err)
	}
	authHandler.SetStaffPasswordService(staffPasswordService)
	platformAdminPasswordService, err := services.NewPlatformAdminPasswordService(platformAdminRepo)
	if err != nil {
		return fmt.Errorf("initialize Platform Admin password service: %w", err)
	}
	authHandler.SetPlatformAdminPasswordService(platformAdminPasswordService)

	schoolOnboardingRepo := repository.NewSchoolOnboardingRepository(db)
	schoolOnboardingService := services.NewSchoolOnboardingService(schoolOnboardingRepo)
	schoolOnboardingHandler := handlers.NewSchoolOnboardingHandler(schoolOnboardingService)

	platformAdminLifecycleRepo := repository.NewPlatformAdminLifecycleRepository(db)
	platformAdminLifecycleService := services.NewPlatformAdminLifecycleService(platformAdminLifecycleRepo)
	platformAdminLifecycleHandler := handlers.NewPlatformAdminLifecycleHandler(platformAdminLifecycleService)

	roleRepo := repository.NewRoleRepository(db)
	roleService := services.NewRoleService(roleRepo)
	roleHandler := handlers.NewRoleHandler(roleService)
	academicRepo := repository.NewAcademicRepository(db)
	academicService := services.NewAcademicService(academicRepo)
	academicHandler := handlers.NewAcademicHandler(academicService)
	guardianService := services.NewGuardianService(academicRepo)
	guardianHandler := handlers.NewGuardianHandler(guardianService)
	attendanceService := services.NewAttendanceService(academicRepo)
	attendanceHandler := handlers.NewAttendanceHandler(attendanceService)

	staffAccountRepo := repository.NewUserRepository(db)
	staffAccountService := services.NewStaffAccountService(staffAccountRepo)
	staffHandler := handlers.NewStaffHandler(staffAccountService)

	router, err := setupRouterWithHandlers(db, authHandler, tokenService, principalAuthorizationRepo, principalAuthorizationRepo, schoolOnboardingHandler, roleHandler, staffHandler, academicHandler, guardianHandler, attendanceHandler, platformAdminLifecycleHandler, principalAuthorizationRepo)

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
