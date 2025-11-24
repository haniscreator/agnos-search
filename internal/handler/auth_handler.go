package handler

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/haniscreator/agnos-search/internal/service"
)

// DTOs
type createStaffReq struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	HospitalID  string `json:"hospital_id" binding:"required"`
	DisplayName string `json:"display_name"`
}

type loginReq struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	HospitalID string `json:"hospital_id" binding:"required"`
}

type tokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// RegisterAuthRoutes registers staff create/login routes.
func RegisterAuthRoutes(r *gin.Engine, authSvc service.AuthService) {
	r.POST("/staff/create", func(c *gin.Context) {
		var req createStaffReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "detail": err.Error()})
			return
		}

		st, err := authSvc.Register(c.Request.Context(), req.Username, req.Password, req.HospitalID, req.DisplayName)
		if err != nil {
			// map service errors to status codes
			switch err {
			case service.ErrWeakPassword:
				c.JSON(http.StatusBadRequest, gin.H{"error": "weak password"})
			case service.ErrUserExists:
				c.JSON(http.StatusConflict, gin.H{"error": "username already exists for hospital"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			}
			return
		}

		// return basic staff info (no password)
		c.JSON(http.StatusCreated, gin.H{
			"id":           st.ID,
			"username":     st.Username,
			"display_name": st.DisplayName,
			"hospital_id":  st.HospitalID,
			"role":         st.Role,
		})
	})

	r.POST("/staff/login", func(c *gin.Context) {
		var req loginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "detail": err.Error()})
			return
		}

		// read jwt secret and expiry from env (fallbacks provided)
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server misconfigured"})
			return
		}
		expStr := os.Getenv("JWT_EXPIRES_SECONDS")
		var expires time.Duration = time.Hour
		if expStr != "" {
			if secs, err := time.ParseDuration(expStr + "s"); err == nil {
				expires = secs
			}
		}

		token, err := authSvc.Authenticate(c.Request.Context(), req.Username, req.Password, req.HospitalID, jwtSecret, expires)
		if err != nil {
			switch err {
			case service.ErrUserNotFound, service.ErrInvalidCreds:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			}
			return
		}

		c.JSON(http.StatusOK, tokenResp{
			AccessToken: token,
			TokenType:   "Bearer",
			ExpiresIn:   int64(expires.Seconds()),
		})
	})
}
