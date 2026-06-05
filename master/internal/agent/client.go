package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AgentClient communicates with the lightweight agent running on a managed server.
type AgentClient struct {
	agentURL   string
	agentToken string
	httpClient *http.Client
}

// NewAgentClient creates a new AgentClient targeting the given agent URL and using
// the provided bearer token for authentication.
func NewAgentClient(agentURL, agentToken string) *AgentClient {
	return &AgentClient{
		agentURL:   agentURL,
		agentToken: agentToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *AgentClient) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := c.agentURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.agentToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *AgentClient) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("agent request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return data, resp.StatusCode, fmt.Errorf("agent returned status %d: %s", resp.StatusCode, string(data))
	}

	return data, resp.StatusCode, nil
}

// Get performs a GET request to the agent at the given path and returns the response body.
func (c *AgentClient) Get(ctx context.Context, path string) ([]byte, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	data, _, err := c.do(req)
	return data, err
}

// Post serialises body as JSON and performs a POST request to the agent.
func (c *AgentClient) Post(ctx context.Context, path string, body any) ([]byte, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, &buf)
	if err != nil {
		return nil, err
	}
	data, _, err := c.do(req)
	return data, err
}

// Delete performs a DELETE request to the agent at the given path.
func (c *AgentClient) Delete(ctx context.Context, path string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	_, _, err = c.do(req)
	return err
}

// PostForm serialises body as JSON and performs a POST request, returning the response
// body, HTTP status code, and any error. Unlike Post it always returns the status code.
func (c *AgentClient) PostForm(ctx context.Context, path string, body any) ([]byte, int, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, 0, fmt.Errorf("encode request body: %w", err)
		}
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, &buf)
	if err != nil {
		return nil, 0, err
	}
	return c.do(req)
}

// Put serialises body as JSON and performs a PUT request to the agent.
func (c *AgentClient) Put(ctx context.Context, path string, body any) ([]byte, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}
	req, err := c.newRequest(ctx, http.MethodPut, path, &buf)
	if err != nil {
		return nil, err
	}
	data, _, err := c.do(req)
	return data, err
}
