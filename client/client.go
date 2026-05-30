package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type APIResponse struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj"`
}

func (c *Client) doRequest(method, path string, body interface{}) (*APIResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d for %s %s: %s", resp.StatusCode, method, path, string(bodyBytes))
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response for %s %s: %w", method, path, err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("api error for %s %s: %s", method, path, apiResp.Msg)
	}

	return &apiResp, nil
}

func (c *Client) Get(path string) (*APIResponse, error) {
	return c.doRequest(http.MethodGet, path, nil)
}

func (c *Client) Post(path string, body interface{}) (*APIResponse, error) {
	return c.doRequest(http.MethodPost, path, body)
}

type ClientStat struct {
	ID         int    `json:"id"`
	InboundID  int    `json:"inboundId"`
	Email      string `json:"email"`
	Enable     bool   `json:"enable"`
	Up         int64  `json:"up"`
	Down       int64  `json:"down"`
	Total      int64  `json:"total"`
	ExpiryTime int64  `json:"expiryTime"`
}

type Inbound struct {
	ID             int          `json:"id"`
	UserID         int          `json:"userId"`
	Up             int64        `json:"up"`
	Down           int64        `json:"down"`
	Total          int64        `json:"total"`
	Remark         string       `json:"remark"`
	Enable         bool         `json:"enable"`
	ExpiryTime     int64        `json:"expiryTime"`
	Listen         string       `json:"listen"`
	Port           int          `json:"port"`
	Protocol       string       `json:"protocol"`
	Tag            string       `json:"tag"`
	ClientStats    []ClientStat `json:"clientStats"`
}

func (c *Client) GetInbounds() ([]Inbound, error) {
	resp, err := c.Get("/panel/api/inbounds/list")
	if err != nil {
		return nil, err
	}
	var inbounds []Inbound
	if err := json.Unmarshal(resp.Obj, &inbounds); err != nil {
		return nil, fmt.Errorf("unmarshal inbounds: %w", err)
	}
	return inbounds, nil
}

func (c *Client) GetOnlineClients() ([]string, error) {
	resp, err := c.Post("/panel/api/clients/onlines", nil)
	if err != nil {
		return nil, err
	}
	var emails []string
	if err := json.Unmarshal(resp.Obj, &emails); err != nil {
		return nil, fmt.Errorf("unmarshal online clients: %w", err)
	}
	return emails, nil
}

type ServerStatus struct {
	CPU       float64        `json:"cpu"`
	Mem       MemStats       `json:"mem"`
	Swap      MemStats       `json:"swap"`
	Disk      MemStats       `json:"disk"`
	NetIO     NetIOStats     `json:"netIO"`
	Xray      XrayState      `json:"xray"`
	TCPCount  int            `json:"tcpCount"`
	Load      LoadStats      `json:"load"`
	Uptime    int64          `json:"uptime"`
}

type MemStats struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
}

type NetIOStats struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

type XrayState struct {
	State   string `json:"state"`
	Version string `json:"version"`
}

type LoadStats struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

func (c *Client) GetServerStatus() (*ServerStatus, error) {
	resp, err := c.Get("/panel/api/server/status")
	if err != nil {
		return nil, err
	}
	var status ServerStatus
	if err := json.Unmarshal(resp.Obj, &status); err != nil {
		return nil, fmt.Errorf("unmarshal server status: %w", err)
	}
	return &status, nil
}

type Node struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Remark          string  `json:"remark"`
	Address         string  `json:"address"`
	Port            int     `json:"port"`
	Enable          bool    `json:"enable"`
	Status          string  `json:"status"`
	LatencyMs       int     `json:"latencyMs"`
	XrayVersion     string  `json:"xrayVersion"`
	PanelVersion    string  `json:"panelVersion"`
	CPUPct          float64 `json:"cpuPct"`
	MemPct          float64 `json:"memPct"`
	UptimeSecs      int64   `json:"uptimeSecs"`
	InboundCount    int     `json:"inboundCount"`
	ClientCount     int     `json:"clientCount"`
	OnlineCount     int     `json:"onlineCount"`
	DepletedCount   int     `json:"depletedCount"`
	LastError       string  `json:"lastError"`
}

func (c *Client) GetNodes() ([]Node, error) {
	resp, err := c.Get("/panel/api/nodes/list")
	if err != nil {
		return nil, err
	}
	var nodes []Node
	if err := json.Unmarshal(resp.Obj, &nodes); err != nil {
		return nil, fmt.Errorf("unmarshal nodes: %w", err)
	}
	return nodes, nil
}
