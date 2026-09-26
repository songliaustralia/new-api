/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package threexui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DefaultTimeout bounds every call to the panel so a slow/unreachable VPS
// never hangs a Stripe webhook handler or an admin request indefinitely.
const DefaultTimeout = 15 * time.Second

// Client is a small REST client for one 3x-ui panel instance.
type Client struct {
	BaseURL    string // e.g. "https://108.61.161.5:2053/4iV2FcjbFgILregyHc/" (trailing slash optional)
	Token      string
	HTTPClient *http.Client
}

// NewClient builds a client for a 3x-ui panel authenticated with a scoped
// Bearer API token (Settings -> API Tokens in the panel UI).
func NewClient(baseURL string, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Token:   strings.TrimSpace(token),
		HTTPClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// APIError is returned for both HTTP-level failures (status >= 400) and
// panel-level failures (HTTP 200 but the envelope's "success" is false).
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("3x-ui panel error (status %d): %s", e.StatusCode, e.Message)
}

func (c *Client) doRequest(method, path string, body any) (*apiResponse, error) {
	if c.BaseURL == "" {
		return nil, fmt.Errorf("3x-ui panel base url is not configured")
	}
	if c.Token == "" {
		return nil, fmt.Errorf("3x-ui panel api token is not configured")
	}

	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal 3x-ui request body: %w", err)
		}
	}

	fullURL := c.BaseURL + path
	httpReq, err := http.NewRequest(method, fullURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to build 3x-ui request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("3x-ui request failed: %w", err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read 3x-ui response: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg := buf.String()
		if len(msg) > 500 {
			msg = msg[:500]
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Message: msg}
	}

	var parsed apiResponse
	if buf.Len() > 0 {
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			return nil, fmt.Errorf("failed to parse 3x-ui response: %w", err)
		}
	}
	if !parsed.Success {
		msg := parsed.Msg
		if msg == "" {
			msg = "3x-ui panel reported failure with no message"
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Message: msg}
	}
	return &parsed, nil
}

// pathSegment defensively escapes a client email for use inside a URL path.
// Emails passed by this feature are always generated internally
// (service.BuildVpnClientEmail, "vpn-u<id>") so this never actually needs to
// do anything, but a raw string still shouldn't be concatenated into a URL
// path unescaped.
func pathSegment(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "/", "%2F"), " ", "%20")
}

// AddClient provisions a brand-new client on the given inbound.
func (c *Client) AddClient(inboundId int, client ClientConfig) error {
	req := addClientRequest{Client: client, InboundIds: []int{inboundId}}
	_, err := c.doRequest(http.MethodPost, "/panel/api/clients/add", req)
	return err
}

// UpdateClient updates (renews, disables, or edits) an existing client,
// looked up by its current email.
func (c *Client) UpdateClient(currentEmail string, inboundId int, client ClientConfig) error {
	path := fmt.Sprintf("/panel/api/clients/update/%s", pathSegment(currentEmail))
	req := addClientRequest{Client: client, InboundIds: []int{inboundId}}
	_, err := c.doRequest(http.MethodPost, path, req)
	return err
}

// DeleteClient removes a client by email.
func (c *Client) DeleteClient(email string) error {
	path := fmt.Sprintf("/panel/api/clients/del/%s", pathSegment(email))
	_, err := c.doRequest(http.MethodPost, path, nil)
	return err
}

// GetClient looks up a client by email. Returns (nil, nil) if the panel
// reports the client as not found, so callers don't have to special-case a
// sentinel error for the common "not provisioned yet" path.
func (c *Client) GetClient(email string) (*ClientConfig, error) {
	path := fmt.Sprintf("/panel/api/clients/get/%s", pathSegment(email))
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	if resp.Obj == nil {
		return nil, nil
	}
	raw, err := json.Marshal(resp.Obj)
	if err != nil {
		return nil, err
	}
	var client ClientConfig
	if err := json.Unmarshal(raw, &client); err != nil {
		return nil, err
	}
	return &client, nil
}

// ListInbounds is used only by the admin "test connection" action, to
// confirm the configured base URL/token/inbound id actually work together,
// without touching any client.
func (c *Client) ListInbounds() ([]InboundSummary, error) {
	resp, err := c.doRequest(http.MethodGet, "/panel/api/inbounds/list", nil)
	if err != nil {
		return nil, err
	}
	if resp.Obj == nil {
		return nil, nil
	}
	raw, err := json.Marshal(resp.Obj)
	if err != nil {
		return nil, err
	}
	var inbounds []InboundSummary
	if err := json.Unmarshal(raw, &inbounds); err != nil {
		return nil, err
	}
	return inbounds, nil
}
