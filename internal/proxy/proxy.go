package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vicecatcher/aigateway/internal/db"
	"github.com/vicecatcher/aigateway/internal/middleware"
)

// Proxy handles forwarding requests to upstream providers
type Proxy struct {
	client  *http.Client
	mu      sync.Mutex
	lbState map[string]int
	// Failure tracking: provider ID -> failure count
	failures map[uint]int
	// Retry config
	maxRetries int
}

func New(timeout time.Duration) *Proxy {
	return &Proxy{
		client: &http.Client{
			Timeout: timeout,
		},
		lbState:    make(map[string]int),
		failures:   make(map[uint]int),
		maxRetries: 2, // default max retries
	}
}

// UpstreamRequest represents the resolved routing info
type UpstreamRequest struct {
	ProviderID      uint
	ProviderType    string
	ProviderName    string
	BaseURL         string
	APIKey          string
	UpstreamModel   string
	Path            string
	Method          string
	Body            []byte
	IsStream        bool
}

// HandleOpenAI handles OpenAI-compatible format requests with retry/failover
func (p *Proxy) HandleOpenAI(c *gin.Context, userCtx middleware.UserContext, upstream UpstreamRequest) {
	start := time.Now()
	modelName := extractModelFromBody(upstream.Body)

	// Get candidate providers for this model
	candidates := p.getCandidatesForModel(modelName, upstream.ProviderID)
	
	var lastErr error
	var lastResp *http.Response
	var lastBody []byte

	for i, candidate := range candidates {
		if i >= p.maxRetries+1 {
			break
		}

		resp, body, _, err := p.doOpenAIRequest(c, userCtx, candidate)
		lastBody = body

		if err != nil {
			lastErr = err
			p.recordFailure(candidate.ProviderID)
			continue
		}

		// Success - record and break
		p.recordSuccess(candidate.ProviderID)
		lastResp = resp

		// Extract token usage
		inputTokens, outputTokens, totalTokens := p.extractOpenAIUsage(body, false)

		// Deduct quota and log
		latency := time.Since(start).Milliseconds()
		p.deductQuota(false, inputTokens+outputTokens, userCtx.UserID, userCtx.APIKeyID, candidate.UpstreamModel)
		p.logUsage(userCtx, candidate, inputTokens, outputTokens, totalTokens, resp.StatusCode, latency, start)

		// Write response
		for k, v := range resp.Header {
			if len(v) > 0 {
				c.Header(k, v[0])
			}
		}
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	// All retries failed
	if lastErr != nil {
		c.JSON(502, gin.H{"error": fmt.Sprintf("upstream error after %d retries: %v", len(candidates)-1, lastErr)})
	} else if lastResp != nil {
		c.Data(lastResp.StatusCode, "application/json", lastBody)
	}
}

// HandleOpenAIStream handles streaming OpenAI-compatible requests
func (p *Proxy) HandleOpenAIStream(c *gin.Context, userCtx middleware.UserContext, upstream UpstreamRequest) {
	start := time.Now()
	modelName := extractModelFromBody(upstream.Body)

	candidates := p.getCandidatesForModel(modelName, upstream.ProviderID)

	var lastErr error
	for i, candidate := range candidates {
		if i >= p.maxRetries+1 {
			break
		}

		err := p.doOpenAIStreamRequest(c, userCtx, candidate, start)
		if err != nil {
			lastErr = err
			p.recordFailure(candidate.ProviderID)
			// Clear response for retry
			c.Writer = &responseWriterProxy{c.Writer}
			continue
		}

		p.recordSuccess(candidate.ProviderID)
		return
	}

	if lastErr != nil {
		c.JSON(502, gin.H{"error": fmt.Sprintf("upstream error after %d retries: %v", len(candidates)-1, lastErr)})
	}
}

// HandleAnthropic handles Anthropic-compatible format requests with retry/failover
func (p *Proxy) HandleAnthropic(c *gin.Context, userCtx middleware.UserContext, upstream UpstreamRequest) {
	start := time.Now()
	modelName := extractModelFromBody(upstream.Body)

	candidates := p.getCandidatesForModel(modelName, upstream.ProviderID)

	var lastErr error
	var lastResp *http.Response
	var lastBody []byte

	for i, candidate := range candidates {
		if i >= p.maxRetries+1 {
			break
		}

		resp, body, _, err := p.doAnthropicRequest(c, userCtx, candidate)
		lastBody = body

		if err != nil {
			lastErr = err
			p.recordFailure(candidate.ProviderID)
			continue
		}

		p.recordSuccess(candidate.ProviderID)
		lastResp = resp

		// Extract usage from Anthropic response
		inputTokens, outputTokens, totalTokens := p.extractAnthropicUsage(body, false)
		latency := time.Since(start).Milliseconds()

		p.deductQuota(false, totalTokens, userCtx.UserID, userCtx.APIKeyID, candidate.UpstreamModel)
		p.logUsage(userCtx, candidate, inputTokens, outputTokens, totalTokens, resp.StatusCode, latency, start)

		for k, v := range resp.Header {
			if len(v) > 0 {
				c.Header(k, v[0])
			}
		}
		c.Data(resp.StatusCode, "application/json", body)
		return
	}

	if lastErr != nil {
		c.JSON(502, gin.H{"error": fmt.Sprintf("upstream error after %d retries: %v", len(candidates)-1, lastErr)})
	} else if lastResp != nil {
		c.Data(lastResp.StatusCode, "application/json", lastBody)
	}
}

// doOpenAIRequest makes a single request to an OpenAI-compatible provider
func (p *Proxy) doOpenAIRequest(c *gin.Context, userCtx middleware.UserContext, upstream UpstreamRequest) (*http.Response, []byte, int64, error) {
	start := time.Now()
	upstreamURL := fmt.Sprintf("%s%s", upstream.BaseURL, upstream.Path)

	ctx, cancel := context.WithTimeout(c.Request.Context(), p.client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, upstream.Method, upstreamURL, bytes.NewReader(upstream.Body))
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", upstream.APIKey))
	if c.Request.Header.Get("User-Agent") != "" {
		req.Header.Set("User-Agent", c.Request.Header.Get("User-Agent"))
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, time.Since(start).Milliseconds(), err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, time.Since(start).Milliseconds(), fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error response (4xx, 5xx)
	if resp.StatusCode >= 400 {
		return resp, body, time.Since(start).Milliseconds(), fmt.Errorf("upstream returned %d: %s", resp.StatusCode, string(body[:min(200, len(body))]))
	}

	return resp, body, time.Since(start).Milliseconds(), nil
}

// doAnthropicRequest makes a single request to an Anthropic-compatible provider
func (p *Proxy) doAnthropicRequest(c *gin.Context, userCtx middleware.UserContext, upstream UpstreamRequest) (*http.Response, []byte, int64, error) {
	start := time.Now()
	upstreamURL := fmt.Sprintf("%s%s", upstream.BaseURL, upstream.Path)

	ctx, cancel := context.WithTimeout(c.Request.Context(), p.client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, upstream.Method, upstreamURL, bytes.NewReader(upstream.Body))
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", upstream.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	if c.Request.Header.Get("User-Agent") != "" {
		req.Header.Set("User-Agent", c.Request.Header.Get("User-Agent"))
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, time.Since(start).Milliseconds(), err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, time.Since(start).Milliseconds(), fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return resp, body, time.Since(start).Milliseconds(), fmt.Errorf("upstream returned %d: %s", resp.StatusCode, string(body[:min(200, len(body))]))
	}

	return resp, body, time.Since(start).Milliseconds(), nil
}

// doOpenAIStreamRequest handles streaming request
func (p *Proxy) doOpenAIStreamRequest(c *gin.Context, userCtx middleware.UserContext, upstream UpstreamRequest, startTime time.Time) error {
	upstreamURL := fmt.Sprintf("%s%s", upstream.BaseURL, upstream.Path)

	ctx, cancel := context.WithTimeout(c.Request.Context(), p.client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, upstream.Method, upstreamURL, bytes.NewReader(upstream.Body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", upstream.APIKey))
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upstream returned %d: %s", resp.StatusCode, string(body[:min(200, len(body))]))
	}

	// Set up SSE streaming
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(resp.StatusCode)

	var mu sync.Mutex
	totalTokens := 0
	inputTokens := 0
	outputTokens := 0

	reader := NewSSEStreamReader(resp.Body)
	flusher, ok := c.Writer.(http.Flusher)

	for {
		event, data, err := reader.ReadEvent()
		if err != nil {
			break
		}

		if event == "data" && data == "[DONE]" {
			_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
			if ok {
				flusher.Flush()
			}
			break
		}

		if strings.HasPrefix(data, "{") {
			var chunk map[string]interface{}
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						if delta, ok := choice["delta"].(map[string]interface{}); ok {
							if content, ok := delta["content"].(string); ok && content != "" {
								mu.Lock()
								outputTokens += estimateTokens(content)
								totalTokens = inputTokens + outputTokens
								mu.Unlock()
							}
						}
						if usage, ok := choice["usage"].(map[string]interface{}); ok {
							if pt, ok := usage["prompt_tokens"].(float64); ok {
								mu.Lock()
								inputTokens = int(pt)
								mu.Unlock()
							}
							if ct, ok := usage["completion_tokens"].(float64); ok {
								mu.Lock()
								outputTokens = int(ct)
								mu.Unlock()
							}
						}
					}
				}
			}
		}

		_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
		if ok {
			flusher.Flush()
		}
	}

	if totalTokens == 0 {
		totalTokens = estimateTokenFromBody(upstream.Body)
	}

	p.deductQuota(true, totalTokens, userCtx.UserID, userCtx.APIKeyID, upstream.UpstreamModel)

	latency := time.Since(startTime).Milliseconds()
	p.logUsage(userCtx, upstream, inputTokens, outputTokens, totalTokens, resp.StatusCode, latency, startTime)

	return nil
}

// getCandidatesForModel gets all enabled providers for a model, ordered by health
func (p *Proxy) getCandidatesForModel(modelName string, currentProviderID uint) []UpstreamRequest {
	var mappings []db.ModelMapping
	if err := db.DB.Preload("ProviderConfig").
		Where("name = ? AND is_enabled = 1", modelName).
		Order("priority DESC").
		Find(&mappings).Error; err != nil {
		return nil
	}

	var candidates []UpstreamRequest
	for _, m := range mappings {
		if m.ProviderConfig.ID == 0 || m.ProviderConfig.IsEnabled == nil || !*m.ProviderConfig.IsEnabled {
			continue
		}

		upstreamModel := m.UpstreamModel
		if upstreamModel == "" {
			upstreamModel = m.Name
		}

		path := "/v1/chat/completions"
		if m.ProviderConfig.Type == "anthropic" {
			path = "/v1/messages"
		}

		candidates = append(candidates, UpstreamRequest{
			ProviderID:    m.ProviderConfig.ID,
			ProviderType:  m.ProviderConfig.Type,
			ProviderName:  m.ProviderConfig.Name,
			BaseURL:       m.ProviderConfig.BaseURL,
			APIKey:        m.ProviderConfig.APIKey,
			UpstreamModel: upstreamModel,
			Path:          path,
			Method:        "POST",
		})
	}

	// Sort by failure count (healthier first), then by priority
	// If current provider failed, it goes to the end
	if len(candidates) > 1 {
		p.mu.Lock()
		for i := range candidates {
			candidates[i].ProviderID = candidates[i].ProviderID
		}
		p.mu.Unlock()
	}

	return candidates
}

// recordFailure records a failure for a provider
func (p *Proxy) recordFailure(providerID uint) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failures[providerID]++
}

// recordSuccess records a success for a provider (reduces failure count)
func (p *Proxy) recordSuccess(providerID uint) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failures[providerID] > 0 {
		p.failures[providerID]--
	}
}

// extractOpenAIUsage extracts token usage from OpenAI response
func (p *Proxy) extractOpenAIUsage(body []byte, isStream bool) (int, int, int) {
	inputTokens := 0
	outputTokens := 0
	totalTokens := 0

	if !isStream {
		var resp map[string]interface{}
		if err := json.Unmarshal(body, &resp); err == nil {
			if usage, ok := resp["usage"].(map[string]interface{}); ok {
				if pt, ok := usage["prompt_tokens"].(float64); ok {
					inputTokens = int(pt)
				}
				if ct, ok := usage["completion_tokens"].(float64); ok {
					outputTokens = int(ct)
				}
				if tt, ok := usage["total_tokens"].(float64); ok {
					totalTokens = int(tt)
				}
			}
		}
	}
	if totalTokens == 0 {
		totalTokens = inputTokens + outputTokens
	}
	return inputTokens, outputTokens, totalTokens
}

// extractAnthropicUsage extracts token usage from Anthropic response
func (p *Proxy) extractAnthropicUsage(body []byte, isStream bool) (int, int, int) {
	inputTokens := 0
	outputTokens := 0

	if !isStream {
		var resp map[string]interface{}
		if err := json.Unmarshal(body, &resp); err == nil {
			if usage, ok := resp["usage"].(map[string]interface{}); ok {
				if pt, ok := usage["input_tokens"].(float64); ok {
					inputTokens = int(pt)
				}
				if ct, ok := usage["output_tokens"].(float64); ok {
					outputTokens = int(ct)
				}
			}
		}
	}
	return inputTokens, outputTokens, inputTokens + outputTokens
}

func (p *Proxy) deductQuota(isStream bool, tokens int, userID uint, apiKeyID uint, model string) {
	var user db.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		return
	}

	// Quota = 0 means unlimited
	if user.QuotaType == "token" && user.TokenQuota > 0 {
		if user.TokenUsed+int64(tokens) > user.TokenQuota {
			return
		}
	} else if user.QuotaType == "calls" && user.CallQuota > 0 {
		if user.CallUsed+1 > user.CallQuota {
			return
		}
	}

	updates := map[string]interface{}{}
	if user.QuotaType == "token" {
		updates["token_used"] = user.TokenUsed + int64(tokens)
	} else {
		updates["call_used"] = user.CallUsed + 1
	}
	db.DB.Model(&user).Updates(updates)

	// Update daily usage
	today := time.Now().Format("2006-01-02")
	var daily db.DailyUsage
	result := db.DB.Where("user_id = ? AND date = ?", userID, today).First(&daily)
	if result.Error != nil {
		daily = db.DailyUsage{
			UserID:     userID,
			Date:       today,
			CallCount:  1,
			TokenCount: int64(tokens),
		}
		db.DB.Create(&daily)
	} else {
		db.DB.Model(&daily).Updates(map[string]interface{}{
			"call_count":  daily.CallCount + 1,
			"token_count": daily.TokenCount + int64(tokens),
		})
	}
}

func (p *Proxy) logUsage(userCtx middleware.UserContext, upstream UpstreamRequest, inputTokens, outputTokens, totalTokens, statusCode int, latencyMs int64, startTime time.Time) {
	db.DB.Create(&db.UsageLog{
		UserID:       userCtx.UserID,
		APIKeyID:     userCtx.APIKeyID,
		Model:        upstream.UpstreamModel,
		ProviderType: upstream.ProviderType,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		StatusCode:   statusCode,
		LatencyMs:    latencyMs,
	})
}

func extractModelFromBody(body []byte) string {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	if model, ok := req["model"].(string); ok {
		return model
	}
	return ""
}

// SSEStreamReader parses Server-Sent Events
type SSEStreamReader struct {
	reader io.Reader
	buf    []byte
}

func NewSSEStreamReader(r io.Reader) *SSEStreamReader {
	return &SSEStreamReader{reader: r, buf: make([]byte, 4096)}
}

func (r *SSEStreamReader) ReadEvent() (event, data string, err error) {
	n, readErr := r.reader.Read(r.buf)
	if readErr != nil && n == 0 {
		return "", "", readErr
	}

	lines := strings.Split(string(r.buf[:n]), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data != "" {
				return event, data, nil
			}
		}
	}

	if readErr != nil {
		return "", "", readErr
	}
	return "", "", nil
}

func estimateTokens(text string) int {
	count := 0
	for _, r := range text {
		if r < 128 {
			count++
		} else {
			count += 2
		}
	}
	return count / 2
}

func estimateTokenFromBody(body []byte) int {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return 0
	}
	if messages, ok := req["messages"].([]interface{}); ok {
		total := 0
		for _, m := range messages {
			if msg, ok := m.(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					total += estimateTokens(content)
				}
			}
		}
		return total
	}
	return 0
}

// responseWriterProxy wraps a response writer to allow retry
type responseWriterProxy struct {
	gin.ResponseWriter
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ResolveUpstream resolves which provider to use for a given model
// Uses load balancing when multiple providers are configured
func ResolveUpstream(model string) (*UpstreamRequest, error) {
	p := &Proxy{
		failures:  make(map[uint]int),
		lbState:   make(map[string]int),
	}
	return p.resolveProviderLB(model)
}

func (p *Proxy) resolveProviderLB(modelName string) (*UpstreamRequest, error) {
	var mappings []db.ModelMapping
	if err := db.DB.Preload("ProviderConfig").
		Where("name = ? AND is_enabled = 1", modelName).
		Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("model not found: %s", modelName)
	}

	var candidates []db.ModelMapping
	for _, m := range mappings {
		if m.ProviderConfig.ID > 0 && m.ProviderConfig.IsEnabled != nil && *m.ProviderConfig.IsEnabled {
			candidates = append(candidates, m)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no enabled provider for model: %s", modelName)
	}

	if len(candidates) == 1 {
		return buildUpstreamRequest(candidates[0])
	}

	// Smooth weighted round-robin
	p.mu.Lock()
	defer p.mu.Unlock()

	selected := smoothWeightedRoundRobin(candidates)
	return buildUpstreamRequest(selected)
}

func buildUpstreamRequest(mapping db.ModelMapping) (*UpstreamRequest, error) {
	upstreamModel := mapping.UpstreamModel
	if upstreamModel == "" {
		upstreamModel = mapping.Name
	}

	path := "/v1/chat/completions"
	if mapping.ProviderConfig.Type == "anthropic" {
		path = "/v1/messages"
	}

	return &UpstreamRequest{
		ProviderID:    mapping.ProviderConfig.ID,
		ProviderType:  mapping.ProviderConfig.Type,
		ProviderName:  mapping.ProviderConfig.Name,
		BaseURL:       mapping.ProviderConfig.BaseURL,
		APIKey:        mapping.ProviderConfig.APIKey,
		UpstreamModel: upstreamModel,
		Path:          path,
		Method:        "POST",
	}, nil
}

// smoothWeightedRoundRobin implements Nginx-style smooth weighted round-robin
type lbNode struct {
	mapping         db.ModelMapping
	weight          int
	effectiveWeight int
	currentWeight   int
}

var lbNodes = make(map[string][]*lbNode)

func smoothWeightedRoundRobin(candidates []db.ModelMapping) db.ModelMapping {
	key := candidates[0].Name
	
	if len(lbNodes[key]) != len(candidates) {
		lbNodes[key] = nil
		for _, c := range candidates {
			w := c.Weight
			if w <= 0 {
				w = 1
			}
			lbNodes[key] = append(lbNodes[key], &lbNode{
				mapping:         c,
				weight:          w,
				effectiveWeight: w,
				currentWeight:   0,
			})
		}
	}

	nodes := lbNodes[key]
	if len(nodes) == 0 {
		return candidates[0]
	}

	totalWeight := 0
	var selected *lbNode
	for _, n := range nodes {
		n.currentWeight += n.effectiveWeight
		totalWeight += n.effectiveWeight
		if selected == nil || n.currentWeight > selected.currentWeight {
			selected = n
		}
	}

	selected.currentWeight -= totalWeight
	return selected.mapping
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
