package middleware

import (
	appsecurity "FinancialPlannerGo/app/security"
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type authErrorResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func RequireAPIKey(cfg appsecurity.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		provided := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(cfg.APIKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, authErrorResponse{
				Success: false,
				Code:    "AUTH_401",
				Message: "invalid or missing API key",
			})
			return
		}

		c.Next()
	}
}

func RestrictToAllowedIPs(cfg appsecurity.Config) gin.HandlerFunc {
	allowed := compileAllowedNetworks(cfg.AllowedIPs)

	return func(c *gin.Context) {
		if !cfg.Enabled || !cfg.IPWhitelistEnabled {
			c.Next()
			return
		}

		clientIP := net.ParseIP(c.ClientIP())
		if clientIP == nil || !ipAllowed(clientIP, allowed) {
			c.AbortWithStatusJSON(http.StatusForbidden, authErrorResponse{
				Success: false,
				Code:    "AUTH_403",
				Message: "client IP is not allowed",
			})
			return
		}

		c.Next()
	}
}

func RequireSignedReportURL(signer *appsecurity.ReportSigner) gin.HandlerFunc {
	return func(c *gin.Context) {
		if signer == nil || !signer.Enabled() {
			c.Next()
			return
		}

		err := signer.Validate(
			c.Request.URL.Path,
			c.Query("expires"),
			c.Query("signature"),
		)
		if err != nil {
			code := "REPORT_LINK_INVALID_401"
			if errors.Is(err, appsecurity.ErrReportURLExpired) {
				code = "REPORT_LINK_EXPIRED_401"
			} else if errors.Is(err, appsecurity.ErrSignedReportURLRequired) {
				code = "REPORT_LINK_REQUIRED_401"
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, authErrorResponse{
				Success: false,
				Code:    code,
				Message: err.Error(),
			})
			return
		}

		c.Next()
	}
}

func compileAllowedNetworks(entries []string) []*net.IPNet {
	result := make([]*net.IPNet, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if ip := net.ParseIP(entry); ip != nil {
			bits := 128
			if ip.To4() != nil {
				bits = 32
			}
			result = append(result, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}
		if _, network, err := net.ParseCIDR(entry); err == nil {
			result = append(result, network)
		}
	}
	return result
}

func ipAllowed(ip net.IP, allowed []*net.IPNet) bool {
	for _, network := range allowed {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
