package handlers

import (
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/vicecatcher/aigateway/internal/middleware"
	"github.com/vicecatcher/aigateway/internal/proxy"
)

func RegisterAnthropic(g *gin.Engine, p *proxy.Proxy) {
	g.POST("/v1/messages",
		middleware.APIKeyAuth(),
		anthropicMessagesHandler(p),
	)
}

func anthropicMessagesHandler(p *proxy.Proxy) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx, ok := middleware.GetUserContext(c)
		if !ok {
			return
		}

		// Check quota before processing
		if !checkQuota(c, userCtx.UserID) {
			return
		}

		// Read request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(400, gin.H{"error": "failed to read body"})
			return
		}

		// Parse to extract model
		var req map[string]interface{}
		if err := json.Unmarshal(body, &req); err != nil {
			c.JSON(400, gin.H{"error": "invalid JSON"})
			return
		}

		model, ok := req["model"].(string)
		if !ok || model == "" {
			c.JSON(400, gin.H{"error": "model is required"})
			return
		}

		// Resolve upstream
		upstream, err := proxy.ResolveUpstream(model)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		upstream.Path = "/v1/messages"
		upstream.Body = body

		// Check if streaming
		if stream, _ := req["stream"].(bool); stream {
			p.HandleOpenAIStream(c, userCtx, *upstream)
			return
		}

		p.HandleAnthropic(c, userCtx, *upstream)
	}
}
