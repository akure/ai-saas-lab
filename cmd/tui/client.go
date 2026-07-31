package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aisaaslab/internal/modules/completion"
)

type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &APIClient{
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type UsageResponse struct {
	APIKey string `json:"api_key"`
	Tokens int    `json:"tokens"`
}

type PersonasResponse struct {
	Personas []completion.Persona `json:"personas"`
}

type SessionsResponse struct {
	Sessions []*completion.Session `json:"sessions"`
}

type CreateKeyResponse struct {
	APIKey string `json:"api_key"`
	Plan   string `json:"plan"`
}

func (c *APIClient) FetchUsage(apiKey string) (*UsageResponse, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/v1/usage/" + apiKey)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	var res UsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *APIClient) FetchPersonas() ([]completion.Persona, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/v1/chat/personas")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res PersonasResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Personas, nil
}

func (c *APIClient) FetchSessions(apiKey string) ([]*completion.Session, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/v1/chat/sessions?api_key=" + apiKey)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res SessionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Sessions, nil
}

func (c *APIClient) CreateAPIKey(plan string) (*CreateKeyResponse, error) {
	payload, _ := json.Marshal(map[string]string{"plan": plan})
	resp, err := c.HTTPClient.Post(c.BaseURL+"/admin/api-keys", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	var res CreateKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *APIClient) DeleteSession(sessionID string) error {
	req, err := http.NewRequest(http.MethodDelete, c.BaseURL+"/v1/chat/sessions/"+sessionID, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *APIClient) StreamCompletion(ctx context.Context, req completion.ChatRequest, tokenChan chan<- string, errChan chan<- error) {
	req.Stream = true
	payload, err := json.Marshal(req)
	if err != nil {
		errChan <- err
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		errChan <- err
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		errChan <- err
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		errChan <- fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				close(tokenChan)
				return
			}
			tokenChan <- data
		}
	}

	if err := scanner.Err(); err != nil {
		errChan <- err
	} else {
		close(tokenChan)
	}
}
