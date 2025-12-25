package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const jwtPrefix = "Bearer "

func (h *Handler) WithAuthCheck() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var jwtStr string
		
		jwtStr = ctx.GetHeader("Authorization")
		if !strings.HasPrefix(jwtStr, jwtPrefix) {
			cookieToken, err := ctx.Cookie("session_token")
			if err == nil && cookieToken != "" {
				jwtStr = jwtPrefix + cookieToken
			}
		}

		if !strings.HasPrefix(jwtStr, jwtPrefix) {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		jwtStr = jwtStr[len(jwtPrefix):]

		err := h.RedisClient.CheckJWTInBlacklist(ctx.Request.Context(), jwtStr)
		if err != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, err := h.JWTService.ParseToken(jwtStr)
		if err != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Println(err)
			return
		}

		ctx.Set("userID", claims.UserID)
		ctx.Set("username", claims.Username)
		ctx.Set("isModerator", claims.IsModerator)

		ctx.Next()
	}
}

func (h *Handler) WithModeratorCheck() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		h.WithAuthCheck()(ctx)

		if ctx.IsAborted() {
			return
		}

		isModerator, exists := ctx.Get("isModerator")
		if !exists || !isModerator.(bool) {
			ctx.AbortWithStatus(http.StatusForbidden)
			log.Printf("user is not a moderator")
			return
		}

		ctx.Next()
	}
}

