package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxRequestBodyBytes int64 = 1 << 20 // 1 MB

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		isDocumentationRoute :=
			strings.HasPrefix(path, "/boltpe-financial-planner/apidocs")

		c.Header(
			"X-Content-Type-Options",
			"nosniff",
		)

		c.Header(
			"X-XSS-Protection",
			"0",
		)

		c.Header(
			"Referrer-Policy",
			"no-referrer",
		)

		if isDocumentationRoute {
			/*
				The BoltPe API documentation uses embedded HTML,
				inline styles and inline JavaScript. Allow only
				the resources required by the documentation page.
			*/
			c.Header(
				"X-Frame-Options",
				"SAMEORIGIN",
			)

			c.Header(
				"Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline'; "+
					"style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data:; "+
					"font-src 'self' data:; "+
					"connect-src 'self'; "+
					"frame-ancestors 'self';",
			)
		} else {
			c.Header(
				"X-Frame-Options",
				"DENY",
			)

			c.Header(
				"Content-Security-Policy",
				"default-src 'none'; "+
					"frame-ancestors 'none'",
			)
		}

		c.Next()
	}
}

func LimitRequestBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		/*
			Do not limit GET requests because Swagger UI
			and OpenAPI JSON are GET endpoints.
		*/
		switch c.Request.Method {
		case http.MethodPost,
			http.MethodPut,
			http.MethodPatch:

			c.Request.Body = http.MaxBytesReader(
				c.Writer,
				c.Request.Body,
				maxRequestBodyBytes,
			)
		}

		c.Next()
	}
}

// RecoveryJSON converts unexpected panics into a stable JSON API response.
// It prevents Gin's default recovery path from returning an empty/plain response.
func RecoveryJSON() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"code":    "FP_INTERNAL_500",
			"message": "An unexpected internal error occurred",
		})
	})
}
