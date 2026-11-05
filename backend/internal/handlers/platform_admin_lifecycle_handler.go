package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"
)

type PlatformAdminLifecycleService interface {
	List(context.Context,int64) ([]repository.PlatformAdminAccount,error)
	Create(context.Context,int64,string,string) (*repository.PlatformAdminAccount,error)
	Deactivate(context.Context,int64,int64) error
}

type PlatformAdminLifecycleHandler struct { service PlatformAdminLifecycleService }

func NewPlatformAdminLifecycleHandler(service PlatformAdminLifecycleService) *PlatformAdminLifecycleHandler {
	return &PlatformAdminLifecycleHandler{service:service}
}

type createPlatformAdminRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

func (h *PlatformAdminLifecycleHandler) actorID(c *gin.Context) (int64,bool) {
	value,exists:=c.Get(middleware.ContextPrincipalIDKey)
	id,ok:=value.(int64)
	return id,exists && ok && id>0
}

func (h *PlatformAdminLifecycleHandler) List(c *gin.Context) {
	actorID,ok:=h.actorID(c)
	if !ok { c.JSON(http.StatusUnauthorized,gin.H{"status":"error","message":"Authentication required"}); return }
	accounts,err:=h.service.List(c.Request.Context(),actorID)
	if err!=nil { c.JSON(http.StatusForbidden,gin.H{"status":"error","message":"Platform Admin lifecycle access denied"}); return }
	c.JSON(http.StatusOK,gin.H{"status":"success","admins":accounts})
}

func (h *PlatformAdminLifecycleHandler) Create(c *gin.Context) {
	actorID,ok:=h.actorID(c)
	if !ok { c.JSON(http.StatusUnauthorized,gin.H{"status":"error","message":"Authentication required"}); return }
	var req createPlatformAdminRequest
	if err:=c.ShouldBindJSON(&req); err!=nil { c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid request body"}); return }
	account,err:=h.service.Create(c.Request.Context(),actorID,req.Email,req.Password)
	if err!=nil {
		switch {
		case errors.Is(err,services.ErrInvalidPlatformAdminAccount),errors.Is(err,services.ErrInvalidPassword):
			c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Valid email and password meeting the password policy are required"})
		case errors.Is(err,services.ErrCompromisedPassword):
			c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Password has appeared in a known data breach"})
		case errors.Is(err,services.ErrPasswordBreachCheckUnavailable):
			c.JSON(http.StatusServiceUnavailable,gin.H{"status":"error","message":"Password security check is unavailable"})
		case errors.Is(err,services.ErrPlatformAdminEmailTaken):
			c.JSON(http.StatusConflict,gin.H{"status":"error","message":"Platform Admin email already exists"})
		default:
			c.JSON(http.StatusForbidden,gin.H{"status":"error","message":"Platform Admin creation was not authorized"})
		}
		return
	}
	c.JSON(http.StatusCreated,gin.H{"status":"success","admin":account,"mfa_enrollment_required":true})
}

func (h *PlatformAdminLifecycleHandler) Deactivate(c *gin.Context) {
	actorID,ok:=h.actorID(c)
	if !ok { c.JSON(http.StatusUnauthorized,gin.H{"status":"error","message":"Authentication required"}); return }
	targetID,err:=strconv.ParseInt(strings.TrimSpace(c.Param("principalID")),10,64)
	if err!=nil || targetID<=0 { c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid Platform Admin principal ID"}); return }
	if err:=h.service.Deactivate(c.Request.Context(),actorID,targetID); err!=nil {
		switch {
		case errors.Is(err,services.ErrPlatformAdminSelfDeactivation):
			c.JSON(http.StatusConflict,gin.H{"status":"error","message":"You cannot deactivate your own Platform Admin account"})
		case errors.Is(err,services.ErrPlatformAdminLastAccount):
			c.JSON(http.StatusConflict,gin.H{"status":"error","message":"The last active Platform Admin cannot be deactivated"})
		case errors.Is(err,services.ErrPlatformAdminTargetNotFound):
			c.JSON(http.StatusNotFound,gin.H{"status":"error","message":"Active Platform Admin not found"})
		default:
			c.JSON(http.StatusForbidden,gin.H{"status":"error","message":"Platform Admin deactivation was not authorized"})
		}
		return
	}
	c.JSON(http.StatusOK,gin.H{"status":"success","message":"Platform Admin deactivated; all refresh sessions were revoked"})
}
