package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"school-management-system/internal/models"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

var ErrPrincipalNotAuthenticated = models.ErrPrincipalNotAuthenticated

const (
	ContextPrincipalIDKey   = "auth_principal_id"
	ContextPrincipalTypeKey = "auth_principal_type"
	ContextTokenVersionKey = "auth_token_version"
	ContextSchoolIDKey      = "auth_school_id"
	ContextUsernameKey      = "auth_username"
	ContextPermissionsKey   = "auth_permissions"
	ContextMustChangePasswordKey = "auth_must_change_password"
)


var ErrAdminSessionIdleExpired = errors.New("administrative session idle timeout expired")

type AdminSessionActivityTracker interface {
	CheckAndTouchAdminSession(ctx context.Context, principalID int64, familyID string) (isAdmin bool, valid bool, err error)
}

type AccessTokenValidator interface {
	ValidateAccessToken(tokenString string) (*services.TokenClaims, error)
}

// PrincipalLoader must return the current database state, not data copied from
// the JWT. Its implementation must reject inactive principals, inactive staff
// schools, unverified parent mobiles, and unsupported subtype combinations.
type PrincipalLoader interface {
	LoadPrincipalForAuthentication(ctx context.Context, principalID int64) (*models.AuthPrincipal, error)
}

// PermissionLoader resolves effective permissions from current RBAC tables on
// every protected request. It must apply role scope, overrides and delegation
// ceilings; it must not trust permissions from the token or client.
type PermissionLoader interface {
	LoadEffectivePermissions(ctx context.Context, principal *models.AuthPrincipal) ([]string, error)
}

func AuthMiddleware(tokenService AccessTokenValidator, principals PrincipalLoader, permissions PermissionLoader, trackers ...AdminSessionActivityTracker) gin.HandlerFunc {
	var activityTracker AdminSessionActivityTracker
	if len(trackers) > 0 { activityTracker = trackers[0] }
	return func(c *gin.Context) {
		if tokenService == nil || principals == nil || permissions == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Authentication service unavailable"})
			return
		}
		authHeader := strings.Fields(c.GetHeader("Authorization"))
		if len(authHeader) != 2 || !strings.EqualFold(authHeader[0], "Bearer") {
			unauthorized(c)
			return
		}
		claims, err := tokenService.ValidateAccessToken(authHeader[1])
		if err != nil || claims == nil {
			unauthorized(c)
			return
		}
		principalID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil || principalID <= 0 {
			unauthorized(c)
			return
		}

		principal, err := principals.LoadPrincipalForAuthentication(c.Request.Context(), principalID)
		if err != nil {
			if errors.Is(err, ErrPrincipalNotAuthenticated) {
				unauthorized(c)
			} else {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to authenticate request"})
			}
			return
		}
		if !validCurrentPrincipal(principal) || principal.Type != claims.PrincipalType || principal.TokenVersion != claims.TokenVersion {
			unauthorized(c)
			return
		}

		// Admin sessions are bound to one refresh-token family. Each request
		// touches only that device's family; a missing, revoked, or idle family
		// is rejected before the handler runs.
		if activityTracker != nil {
			isAdmin, valid, trackErr := activityTracker.CheckAndTouchAdminSession(c.Request.Context(), principal.ID, claims.SessionID)
			if trackErr != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status":"error","message":"Unable to validate administrative session"})
				return
			}
			if isAdmin && (!valid || claims.SessionID == "") {
				unauthorized(c)
				return
			}
		}

		c.Set(ContextPrincipalIDKey, principal.ID)
		c.Set(ContextPrincipalTypeKey, principal.Type)
		c.Set(ContextTokenVersionKey, principal.TokenVersion)
		c.Set(ContextMustChangePasswordKey, principal.MustChangePassword)

		// Temporary-password staff sessions are deliberately restricted to
		// password replacement and identity inspection. This check runs on
		// every request using the current database state, so refresh-token
		// rotation cannot bypass the requirement.
		if principal.Type == "staff" && principal.MustChangePassword {
			switch c.FullPath() {
			case "/auth/password/change", "/auth/me":
				c.Set(ContextPermissionsKey, map[string]struct{}{})
				if principal.SchoolID > 0 {
					c.Set(ContextSchoolIDKey, principal.SchoolID)
				}
				if principal.Username != "" {
					c.Set(ContextUsernameKey, principal.Username)
				}
				c.Next()
			default:
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"status": "error",
					"code": "password_change_required",
					"message": "Change your temporary password before continuing",
				})
			}
			return
		}

		effectivePermissions, err := permissions.LoadEffectivePermissions(c.Request.Context(), principal)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to resolve authorization"})
			return
		}

		if principal.SchoolID > 0 {
			c.Set(ContextSchoolIDKey, principal.SchoolID)
		}
		if principal.Username != "" {
			c.Set(ContextUsernameKey, principal.Username)
		}
		c.Set(ContextPermissionsKey, permissionSet(effectivePermissions))
		c.Next()
	}
}

// RequirePermission must be added after AuthMiddleware on each protected route.
// Permissions are already freshly resolved for this request by AuthMiddleware.
func RequirePermission(permission string) gin.HandlerFunc {
	permission = strings.TrimSpace(permission)
	return func(c *gin.Context) {
		if permission == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Authorization policy unavailable"})
			return
		}
		value, exists := c.Get(ContextPermissionsKey)
		granted, ok := value.(map[string]struct{})
		if !exists || !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Authorization context unavailable"})
			return
		}
		if _, allowed := granted[permission]; !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "error", "message": "Permission denied"})
			return
		}
		c.Next()
	}
}

func validCurrentPrincipal(p *models.AuthPrincipal) bool {
	if p == nil || p.ID <= 0 || p.TokenVersion <= 0 || p.Status != "active" {
		return false
	}
	switch p.Type {
	case "staff":
		return p.SchoolID > 0
	case "platform_admin":
		return true
	case "parent":
		return p.MobileVerified
	default:
		return false
	}
}

func permissionSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			set[value] = struct{}{}
		}
	}
	return set
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Authentication required"})
}
