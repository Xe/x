// Package pocketid provides a Go client for interacting with the PocketID API.
//
// PocketID is an identity provider for your homelab. This client allows you to
// manage users, groups, OIDC clients, API keys, and more.
//
// The client uses a standard HTTP client for all API interactions, allowing for
// customization of timeouts, transport, and other settings.
//
// This was generated from the Swagger docs with Google Gemini.
package pocketid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client is the main client for interacting with the PocketID API.
//
// Use NewClient to create a new instance.
type Client struct {
	// BaseURL is the base URL for the PocketID API.
	//
	// For example: "https://pocket-id.example.com".
	BaseURL string

	// HTTPClient is the underlying HTTP client used for API requests.
	//
	// This allows for customization of timeouts, transport, and other settings.
	HTTPClient *http.Client

	// apiKey is the API key used for authentication.
	apiKey string
}

// NewClient creates a new PocketID client.
//
// baseURL is the base URL for the PocketID API (e.g., "https://pocket-id.example.com").
// apiKey is the API key to use for authentication.
// httpClient is an optional *http.Client to use for requests. If nil, http.DefaultClient will be used.
func NewClient(baseURL string, apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
		apiKey:     apiKey,
	}
}

// ErrorResponse represents a generic error response from the PocketID API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// setAPIKeyHeader sets the X-API-KEY header on the given request.
func (c *Client) setAPIKeyHeader(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("X-API-KEY", c.apiKey)
	}
}

// Helper to decode to a Paginated type
func decodePaginated[T any](resp *http.Response, paginated *Paginated[T]) error {
	return json.NewDecoder(resp.Body).Decode(paginated)
}

// request performs an HTTP request to the PocketID API.
func (c *Client) request(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.BaseURL, path), body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	c.setAPIKeyHeader(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// --- Well Known ---

// GetJWKS returns the JSON Web Key Set used for token verification.
//
// See https://pocket-id.example.com/.well-known/jwks.json
func (c *Client) GetJWKS() (map[string]interface{}, error) {
	resp, err := c.request("GET", "/.well-known/jwks.json", nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var jwks map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)
	if err != nil {
		return nil, err
	}

	return jwks, nil
}

// GetOpenIDConfiguration returns the OpenID Connect discovery document.
//
// See https://pocket-id.example.com/.well-known/openid-configuration
func (c *Client) GetOpenIDConfiguration() (map[string]interface{}, error) {
	resp, err := c.request("GET", "/.well-known/openid-configuration", nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var config map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// --- API Keys ---

// ApiKeyCreateDto represents the request body for creating a new API key.
type ApiKeyCreateDto struct {
	Name        string    `json:"name" validate:"required,min=3,max=50"`
	Description string    `json:"description"`
	ExpiresAt   time.Time `json:"expiresAt" validate:"required"`
}

// Valid validates the ApiKeyCreateDto fields.
func (t ApiKeyCreateDto) Valid() error {
	var errs []error

	if len(t.Name) < 3 || len(t.Name) > 50 {
		errs = append(errs, fmt.Errorf("name must be between 3 and 50 characters"))
	}

	if t.ExpiresAt.IsZero() {
		errs = append(errs, fmt.Errorf("expiresAt is required"))
	}

	if len(errs) != 0 {
		return fmt.Errorf("validation errors: %v", errs)
	}

	return nil
}

// ApiKeyDto represents an API key.
type ApiKeyDto struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	ExpiresAt   string `json:"expiresAt"`
	LastUsedAt  string `json:"lastUsedAt"`
}

// ApiKeyResponseDto represents the response body when creating a new API key.
type ApiKeyResponseDto struct {
	ApiKey *ApiKeyDto `json:"apiKey"`
	Token  string     `json:"token"`
}

// Paginated represents a paginated response.
type Paginated[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// Pagination represents pagination metadata.
type Pagination struct {
	CurrentPage  int `json:"currentPage"`
	ItemsPerPage int `json:"itemsPerPage"`
	TotalItems   int `json:"totalItems"`
	TotalPages   int `json:"totalPages"`
}

// ListAPIKeys gets a paginated list of API keys belonging to the current user.
//
// See https://pocket-id.example.com/api-keys
func (c *Client) ListAPIKeys(page, limit int, sortColumn, sortDirection string) (*Paginated[ApiKeyDto], error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("limit", strconv.Itoa(limit))
	params.Set("sort_column", sortColumn)
	params.Set("sort_direction", sortDirection)

	resp, err := c.request("GET", fmt.Sprintf("/api-keys?%s", params.Encode()), nil, "")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiKeys Paginated[ApiKeyDto]
	if err := decodePaginated(resp, &apiKeys); err != nil {
		return nil, err
	}

	return &apiKeys, nil
}

// CreateAPIKey creates a new API key for the current user.
//
// See https://pocket-id.example.com/api-keys
func (c *Client) CreateAPIKey(apiKeyCreateDto ApiKeyCreateDto) (*ApiKeyResponseDto, error) {
	if err := apiKeyCreateDto.Valid(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(apiKeyCreateDto)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("POST", "/api-keys", bytes.NewBuffer(body), "application/json")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiKeyResponseDto ApiKeyResponseDto
	err = json.NewDecoder(resp.Body).Decode(&apiKeyResponseDto)
	if err != nil {
		return nil, err
	}

	return &apiKeyResponseDto, nil
}

// RevokeAPIKey revokes (deletes) an existing API key by ID.
//
// See https://pocket-id.example.com/api-keys/{id}
func (c *Client) RevokeAPIKey(id string) error {
	resp, err := c.request("DELETE", fmt.Sprintf("/api-keys/%s", id), nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// --- Application Configuration ---

// AppConfigUpdateDto represents the request body for updating application configuration.
type AppConfigUpdateDto struct {
	AppName                            string `json:"appName" validate:"required,min=1,max=30"`
	AllowOwnAccountEdit                string `json:"allowOwnAccountEdit" validate:"required"`
	EmailLoginNotificationEnabled      string `json:"emailLoginNotificationEnabled" validate:"required"`
	EmailOneTimeAccessEnabled          string `json:"emailOneTimeAccessEnabled" validate:"required"`
	EmailsVerified                     string `json:"emailsVerified" validate:"required"`
	LdapEnabled                        string `json:"ldapEnabled" validate:"required"`
	SessionDuration                    string `json:"sessionDuration" validate:"required"`
	SmtpTls                            string `json:"smtpTls" validate:"required,oneof=none starttls tls"`
	LdapAttributeAdminGroup            string `json:"ldapAttributeAdminGroup"`
	LdapAttributeGroupMember           string `json:"ldapAttributeGroupMember"`
	LdapAttributeGroupName             string `json:"ldapAttributeGroupName"`
	LdapAttributeGroupUniqueIdentifier string `json:"ldapAttributeGroupUniqueIdentifier"`
	LdapAttributeUserEmail             string `json:"ldapAttributeUserEmail"`
	LdapAttributeUserFirstName         string `json:"ldapAttributeUserFirstName"`
	LdapAttributeUserLastName          string `json:"ldapAttributeUserLastName"`
	LdapAttributeUserProfilePicture    string `json:"ldapAttributeUserProfilePicture"`
	LdapAttributeUserUniqueIdentifier  string `json:"ldapAttributeUserUniqueIdentifier"`
	LdapAttributeUserUsername          string `json:"ldapAttributeUserUsername"`
	LdapBase                           string `json:"ldapBase"`
	LdapBindDn                         string `json:"ldapBindDn"`
	LdapBindPassword                   string `json:"ldapBindPassword"`
	LdapSkipCertVerify                 string `json:"ldapSkipCertVerify"`
	LdapUrl                            string `json:"ldapUrl"`
	LdapUserGroupSearchFilter          string `json:"ldapUserGroupSearchFilter"`
	LdapUserSearchFilter               string `json:"ldapUserSearchFilter"`
	SmtpFrom                           string `json:"smtpFrom"`
	SmtpHost                           string `json:"smtpHost"`
	SmtpPassword                       string `json:"smtpPassword"`
	SmtpPort                           string `json:"smtpPort"`
	SmtpSkipCertVerify                 string `json:"smtpSkipCertVerify"`
	SmtpUser                           string `json:"smtpUser"`
}

// AppConfigVariableDto represents an application configuration variable.
type AppConfigVariableDto struct {
	IsPublic bool   `json:"isPublic"`
	Key      string `json:"key"`
	Type     string `json:"type"`
	Value    string `json:"value"`
}

// PublicAppConfigVariableDto represents a public application configuration variable.
type PublicAppConfigVariableDto struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ListPublicAppConfig gets all public application configurations.
//
// See https://pocket-id.example.com/application-configuration
func (c *Client) ListPublicAppConfig() ([]PublicAppConfigVariableDto, error) {
	resp, err := c.request("GET", "/application-configuration", nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var config []PublicAppConfigVariableDto
	err = json.NewDecoder(resp.Body).Decode(&config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// UpdateAppConfig updates application configuration settings.
//
// See https://pocket-id.example.com/application-configuration
func (c *Client) UpdateAppConfig(config AppConfigUpdateDto) ([]AppConfigVariableDto, error) {
	body, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("PUT", "/application-configuration", bytes.NewBuffer(body), "application/json")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var updatedConfig []AppConfigVariableDto
	err = json.NewDecoder(resp.Body).Decode(&updatedConfig)
	if err != nil {
		return nil, err
	}

	return updatedConfig, nil
}

// ListAllAppConfig gets all application configurations, including private ones.
//
// See https://pocket-id.example.com/application-configuration/all
func (c *Client) ListAllAppConfig() ([]AppConfigVariableDto, error) {
	resp, err := c.request("GET", "/application-configuration/all", nil, "")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var config []AppConfigVariableDto
	err = json.NewDecoder(resp.Body).Decode(&config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetBackgroundImage gets the background image for the application.
//
// See https://pocket-id.example.com/application-configuration/background-image
func (c *Client) GetBackgroundImage() ([]byte, error) {
	resp, err := c.request("GET", "/application-configuration/background-image", nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d - %w", resp.StatusCode, err)
	}

	return io.ReadAll(resp.Body)
}

// UpdateBackgroundImage updates the application background image.
//
// See https://pocket-id.example.com/application-configuration/background-image
func (c *Client) UpdateBackgroundImage(file io.Reader) error {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", "background.png") // filename doesn't matter for this endpoint
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, file); err != nil {
		return err
	}
	w.Close()

	resp, err := c.request("PUT", "/application-configuration/background-image", &b, w.FormDataContentType())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

// GetFavicon gets the favicon for the application.
//
// See https://pocket-id.example.com/application-configuration/favicon
func (c *Client) GetFavicon() ([]byte, error) {
	resp, err := c.request("GET", "/application-configuration/favicon", nil, "")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// UpdateFavicon updates the application favicon.
//
// See https://pocket-id.example.com/application-configuration/favicon
func (c *Client) UpdateFavicon(file io.Reader) error {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", "favicon.ico")
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, file); err != nil {
		return err
	}
	w.Close()

	resp, err := c.request("PUT", "/application-configuration/favicon", &b, w.FormDataContentType())

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

// GetLogo gets the logo image for the application.  If isLight is true, the light mode logo is returned.
//
// See https://pocket-id.example.com/application-configuration/logo
func (c *Client) GetLogo(isLight bool) ([]byte, error) {
	params := url.Values{}
	params.Set("light", strconv.FormatBool(isLight))
	resp, err := c.request("GET", fmt.Sprintf("/application-configuration/logo?%s", params.Encode()), nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// UpdateLogo updates the application logo. If isLight is true, the light mode logo is updated.
//
// See https://pocket-id.example.com/application-configuration/logo
func (c *Client) UpdateLogo(file io.Reader, isLight bool) error {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add the 'light' parameter as a form field
	if err := w.WriteField("light", strconv.FormatBool(isLight)); err != nil {
		return err
	}

	fw, err := w.CreateFormFile("file", "logo.png") // filename doesn't matter for this endpoint
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, file); err != nil {
		return err
	}
	w.Close()

	resp, err := c.request("PUT", "/application-configuration/logo", &b, w.FormDataContentType())

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

// SyncLDAP manually triggers LDAP synchronization.
//
// See https://pocket-id.example.com/application-configuration/sync-ldap
func (c *Client) SyncLDAP() error {
	resp, err := c.request("POST", "/application-configuration/sync-ldap", nil, "")

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

// SendTestEmail sends a test email to verify email configuration.
//
// See https://pocket-id.example.com/application-configuration/test-email
func (c *Client) SendTestEmail() error {
	resp, err := c.request("POST", "/application-configuration/test-email", nil, "")

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

// --- Audit Logs ---

// AuditLogDto represents an audit log entry.
type AuditLogDto struct {
	ID        string        `json:"id"`
	UserID    string        `json:"userID"`
	Event     AuditLogEvent `json:"event"`
	IPAddress string        `json:"ipAddress"`
	Device    string        `json:"device"`
	City      string        `json:"city"`
	Country   string        `json:"country"`
	CreatedAt string        `json:"createdAt"`
	Data      AuditLogData  `json:"data"` // Custom type for nested JSON object
}

// AuditLogEvent represents the type of event in an audit log.
type AuditLogEvent string

// Constants for AuditLogEvent.
const (
	AuditLogEventSignIn                   AuditLogEvent = "SIGN_IN"
	AuditLogEventOneTimeAccessTokenSignIn AuditLogEvent = "TOKEN_SIGN_IN"
	AuditLogEventClientAuthorization      AuditLogEvent = "CLIENT_AUTHORIZATION"
	AuditLogEventNewClientAuthorization   AuditLogEvent = "NEW_CLIENT_AUTHORIZATION"
)

// AuditLogData is a custom type representing additional data in the AuditLogDto.
type AuditLogData map[string]string

// ListAuditLogs gets a paginated list of audit logs for the current user.
//
// See https://pocket-id.example.com/audit-logs
func (c *Client) ListAuditLogs(page, limit int, sortColumn, sortDirection string) (*Paginated[AuditLogDto], error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("limit", strconv.Itoa(limit))
	params.Set("sort_column", sortColumn)
	params.Set("sort_direction", sortDirection)

	resp, err := c.request("GET", fmt.Sprintf("/audit-logs?%s", params.Encode()), nil, "")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var auditLogs Paginated[AuditLogDto]
	err = decodePaginated(resp, &auditLogs)
	if err != nil {
		return nil, err
	}

	return &auditLogs, nil
}

// --- Custom Claims ---

// CustomClaimCreateDto represents the request body for creating or updating custom claims.
type CustomClaimDto struct {
	Key   string `json:"key" validate:"required"`
	Value string `json:"value" validate:"required"`
}

// Valid validates the CustomClaimCreateDto fields.
func (t CustomClaimDto) Valid() error {
	var errs []error

	if t.Key == "" {
		errs = append(errs, fmt.Errorf("key is required"))
	}

	if t.Value == "" {
		errs = append(errs, fmt.Errorf("value is required"))
	}

	if len(errs) != 0 {
		return fmt.Errorf("validation errors: %v", errs)
	}

	return nil
}

// GetCustomClaimSuggestions gets a list of suggested custom claim names.
//
// See https://pocket-id.example.com/custom-claims/suggestions
func (c *Client) GetCustomClaimSuggestions() ([]string, error) {
	resp, err := c.request("GET", "/custom-claims/suggestions", nil, "")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var suggestions []string
	err = json.NewDecoder(resp.Body).Decode(&suggestions)
	if err != nil {
		return nil, err
	}

	return suggestions, nil
}

// UpdateCustomClaimsForUserGroup updates or creates custom claims for a specific user group.
//
// See https://pocket-id.example.com/custom-claims/user-group/{userGroupId}
func (c *Client) UpdateCustomClaimsForUserGroup(userGroupID string, claims []CustomClaimDto) ([]CustomClaimDto, error) {
	body, err := json.Marshal(claims)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("PUT", fmt.Sprintf("/custom-claims/user-group/%s", userGroupID), bytes.NewBuffer(body), "application/json")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var updatedClaims []CustomClaimDto
	err = json.NewDecoder(resp.Body).Decode(&updatedClaims)
	if err != nil {
		return nil, err
	}

	return updatedClaims, nil
}

// UpdateCustomClaimsForUser updates or creates custom claims for a specific user.
//
// See https://pocket-id.example.com/custom-claims/user/{userId}
func (c *Client) UpdateCustomClaimsForUser(userID string, claims []CustomClaimDto) ([]CustomClaimDto, error) {
	body, err := json.Marshal(claims)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("PUT", fmt.Sprintf("/custom-claims/user/%s", userID), bytes.NewBuffer(body), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var updatedClaims []CustomClaimDto
	err = json.NewDecoder(resp.Body).Decode(&updatedClaims)
	if err != nil {
		return nil, err
	}

	return updatedClaims, nil
}

// --- OIDC ---

// AuthorizationRequiredDto is used to check if authorization is required.
type AuthorizationRequiredDto struct {
	ClientID string `json:"clientID" validate:"required"`
	Scope    string `json:"scope" validate:"required"`
}

// AuthorizeOidcClientRequestDto represents the request body for authorizing an OIDC client.
type AuthorizeOidcClientRequestDto struct {
	ClientID            string `json:"clientID" validate:"required"`
	Scope               string `json:"scope" validate:"required"`
	CallbackURL         string `json:"callbackURL"`
	CodeChallenge       string `json:"codeChallenge"`
	CodeChallengeMethod string `json:"codeChallengeMethod"`
	Nonce               string `json:"nonce"`
}

// AuthorizeOidcClientResponseDto represents the response body for authorizing an OIDC client.
type AuthorizeOidcClientResponseDto struct {
	CallbackURL string `json:"callbackURL"`
	Code        string `json:"code"`
}

// OidcClientCreateDto represents the request body for creating an OIDC client.
type OidcClientCreateDto struct {
	Name               string   `json:"name" validate:"required,max=50"`
	CallbackURLs       []string `json:"callbackURLs" validate:"required"`
	IsPublic           bool     `json:"isPublic"`
	LogoutCallbackURLs []string `json:"logoutCallbackURLs"`
	PkceEnabled        bool     `json:"pkceEnabled"`
}

// OidcClientDto represents an OIDC client.
type OidcClientDto struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	CallbackURLs       []string `json:"callbackURLs"`
	IsPublic           bool     `json:"isPublic"`
	LogoutCallbackURLs []string `json:"logoutCallbackURLs"`
	PkceEnabled        bool     `json:"pkceEnabled"`
	HasLogo            bool     `json:"hasLogo"`
}

// OidcClientMetaDataDto represents OIDC client metadata.
type OidcClientMetaDataDto struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	HasLogo bool   `json:"hasLogo"`
}

// OidcClientWithAllowedUserGroupsDto represents an OIDC client with allowed user groups.
type OidcClientWithAllowedUserGroupsDto struct {
	ID                 string                      `json:"id"`
	Name               string                      `json:"name"`
	CallbackURLs       []string                    `json:"callbackURLs"`
	IsPublic           bool                        `json:"isPublic"`
	LogoutCallbackURLs []string                    `json:"logoutCallbackURLs"`
	PkceEnabled        bool                        `json:"pkceEnabled"`
	HasLogo            bool                        `json:"hasLogo"`
	AllowedUserGroups  []UserGroupDtoWithUserCount `json:"allowedUserGroups"`
}

// UserGroupDtoWithUserCount represents a user group with a user count.
type UserGroupDtoWithUserCount struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	FriendlyName string           `json:"friendlyName"`
	CreatedAt    string           `json:"createdAt"`
	LdapId       string           `json:"ldapId"`
	CustomClaims []CustomClaimDto `json:"customClaims"`
	UserCount    int              `json:"userCount"`
}

// OidcUpdateAllowedUserGroupsDto represents the request body for updating allowed user groups.
type OidcUpdateAllowedUserGroupsDto struct {
	UserGroupIds []string `json:"userGroupIds" validate:"required"`
}

// CheckAuthorizationRequired checks if the user needs to confirm authorization for the client.
//
// See https://pocket-id.example.com/oidc/authorization-required
func (c *Client) CheckAuthorizationRequired(request AuthorizationRequiredDto) (bool, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return false, err
	}

	resp, err := c.request("POST", "/oidc/authorization-required", bytes.NewBuffer(body), "application/json")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		AuthorizationRequired bool `json:"authorizationRequired"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return false, err
	}

	return response.AuthorizationRequired, nil
}

// AuthorizeOIDCClient starts the OIDC authorization process for a client.
//
// See https://pocket-id.example.com/oidc/authorize
func (c *Client) AuthorizeOIDCClient(request AuthorizeOidcClientRequestDto) (*AuthorizeOidcClientResponseDto, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("POST", "/oidc/authorize", bytes.NewBuffer(body), "application/json")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var responseDto AuthorizeOidcClientResponseDto
	err = json.NewDecoder(resp.Body).Decode(&responseDto)
	if err != nil {
		return nil, err
	}

	return &responseDto, nil
}

// ListOIDCClients gets a paginated list of OIDC clients.
//
// See https://pocket-id.example.com/oidc/clients
func (c *Client) ListOIDCClients(search string, page, limit int, sortColumn, sortDirection string) (*Paginated[OidcClientDto], error) {
	params := url.Values{}
	params.Set("search", search)
	params.Set("page", strconv.Itoa(page))
	params.Set("limit", strconv.Itoa(limit))
	params.Set("sort_column", sortColumn)
	params.Set("sort_direction", sortDirection)

	resp, err := c.request("GET", fmt.Sprintf("/oidc/clients?%s", params.Encode()), nil, "")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var clients Paginated[OidcClientDto]
	err = decodePaginated(resp, &clients)
	if err != nil {
		return nil, err
	}

	return &clients, nil
}

// CreateOIDCClient creates a new OIDC client.
//
// See https://pocket-id.example.com/oidc/clients
func (c *Client) CreateOIDCClient(clientCreateDto OidcClientCreateDto) (*OidcClientWithAllowedUserGroupsDto, error) {
	body, err := json.Marshal(clientCreateDto)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("POST", "/oidc/clients", bytes.NewBuffer(body), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var clientDto OidcClientWithAllowedUserGroupsDto
	err = json.NewDecoder(resp.Body).Decode(&clientDto)
	if err != nil {
		return nil, err
	}

	return &clientDto, nil
}

// DeleteOIDCClient deletes an OIDC client by ID.
//
// See https://pocket-id.example.com/oidc/clients/{id}
func (c *Client) DeleteOIDCClient(id string) error {
	resp, err := c.request("DELETE", fmt.Sprintf("/oidc/clients/%s", id), nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetOIDCClient gets detailed information about an OIDC client.
//
// See https://pocket-id.example.com/oidc/clients/{id}
func (c *Client) GetOIDCClient(id string) (*OidcClientWithAllowedUserGroupsDto, error) {
	resp, err := c.request("GET", fmt.Sprintf("/oidc/clients/%s", id), nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var clientDto OidcClientWithAllowedUserGroupsDto
	err = json.NewDecoder(resp.Body).Decode(&clientDto)
	if err != nil {
		return nil, err
	}

	return &clientDto, nil
}

// UpdateOIDCClient updates an existing OIDC client.
//
// See https://pocket-id.example.com/oidc/clients/{id}
func (c *Client) UpdateOIDCClient(id string, clientCreateDto OidcClientCreateDto) (*OidcClientWithAllowedUserGroupsDto, error) {
	body, err := json.Marshal(clientCreateDto)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("PUT", fmt.Sprintf("/oidc/clients/%s", id), bytes.NewBuffer(body), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var clientDto OidcClientWithAllowedUserGroupsDto
	err = json.NewDecoder(resp.Body).Decode(&clientDto)
	if err != nil {
		return nil, err
	}

	return &clientDto, nil
}

// UpdateOIDCClientAllowedUserGroups updates the user groups allowed to access an OIDC client.
//
// See https://pocket-id.example.com/oidc/clients/{id}/allowed-user-groups
func (c *Client) UpdateOIDCClientAllowedUserGroups(id string, groups OidcUpdateAllowedUserGroupsDto) (*OidcClientDto, error) {
	body, err := json.Marshal(groups)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("PUT", fmt.Sprintf("/oidc/clients/%s/allowed-user-groups", id), bytes.NewBuffer(body), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var clientDto OidcClientDto
	err = json.NewDecoder(resp.Body).Decode(&clientDto)
	if err != nil {
		return nil, err
	}

	return &clientDto, nil
}

// DeleteOIDCClientLogo deletes the logo for an OIDC client.
//
// See https://pocket-id.example.com/oidc/clients/{id}/logo
func (c *Client) DeleteOIDCClientLogo(id string) error {
	resp, err := c.request("DELETE", fmt.Sprintf("/oidc/clients/%s/logo", id), nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetOIDCClientLogo gets the logo image for an OIDC client.
//
// See https://pocket-id.example.com/oidc/clients/{id}/logo
func (c *Client) GetOIDCClientLogo(id string) ([]byte, error) {
	resp, err := c.request("GET", fmt.Sprintf("/oidc/clients/%s/logo", id), nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// UpdateOIDCClientLogo uploads or updates the logo for an OIDC client.
//
// See https://pocket-id.example.com/oidc/clients/{id}/logo
func (c *Client) UpdateOIDCClientLogo(id string, file io.Reader) error {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", "logo.png") // Filename doesn't affect server-side processing
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, file); err != nil {
		return err
	}
	w.Close()

	resp, err := c.request("POST", fmt.Sprintf("/oidc/clients/%s/logo", id), &b, w.FormDataContentType())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetOIDCClientMeta gets OIDC client metadata for discovery and configuration.
//
// See https://pocket-id.example.com/oidc/clients/{id}/meta
func (c *Client) GetOIDCClientMeta(id string) (*OidcClientMetaDataDto, error) {
	resp, err := c.request("GET", fmt.Sprintf("/oidc/clients/%s/meta", id), nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var metaDto OidcClientMetaDataDto
	err = json.NewDecoder(resp.Body).Decode(&metaDto)
	if err != nil {
		return nil, err
	}

	return &metaDto, nil
}

// CreateOIDCClientSecret generates a new secret for an OIDC client.
//
// See https://pocket-id.example.com/oidc/clients/{id}/secret
func (c *Client) CreateOIDCClientSecret(id string) (string, error) {
	resp, err := c.request("POST", fmt.Sprintf("/oidc/clients/%s/secret", id), nil, "")

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Secret string `json:"secret"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	return response.Secret, nil
}

// EndOIDCSession ends the user session and handles OIDC logout.
//
// See https://pocket-id.example.com/oidc/end-session
func (c *Client) EndOIDCSession(idTokenHint, postLogoutRedirectURI, state string) error {
	params := url.Values{}
	if idTokenHint != "" {
		params.Set("id_token_hint", idTokenHint)
	}
	if postLogoutRedirectURI != "" {
		params.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	}
	if state != "" {
		params.Set("state", state)
	}

	resp, err := c.request("GET", fmt.Sprintf("/oidc/end-session?%s", params.Encode()), nil, "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Expect a redirect.
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusOK { // Allow 200 for cases where it might return HTML
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil

}

// EndOIDCSessionPost ends the user session and handles OIDC logout using POST.
//
// See https://pocket-id.example.com/oidc/end-session
func (c *Client) EndOIDCSessionPost(idTokenHint, postLogoutRedirectURI, state string) error {
	data := url.Values{}
	if idTokenHint != "" {
		data.Set("id_token_hint", idTokenHint)
	}
	if postLogoutRedirectURI != "" {
		data.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	}

	if state != "" {
		data.Set("state", state)
	}

	resp, err := c.request("POST", "/oidc/end-session", strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")

	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Expect a redirect.
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusOK { // Allow 200 for cases where it returns HTML
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// CreateOIDCTokens exchanges an authorization code for ID and access tokens.
//
// See https://pocket-id.example.com/oidc/token
func (c *Client) CreateOIDCTokens(clientID, clientSecret, code, grantType, codeVerifier string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("grant_type", grantType)
	data.Set("code", code)
	if clientID != "" {
		data.Set("client_id", clientID)
	}
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}

	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	resp, err := c.request("POST", "/oidc/token", strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tokens map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tokens)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// GetUserInfo gets user information based on the access token.
//
// See https://pocket-id.example.com/oidc/userinfo
func (c *Client) GetUserInfo(accessToken string) (map[string]interface{}, error) {

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/oidc/userinfo", c.BaseURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.HTTPClient.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&userInfo)
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

// GetUserInfoPost gets user information based on the access token using POST.
// See https://pocket-id.example.com/oidc/userinfo
func (c *Client) GetUserInfoPost(accessToken string) (map[string]interface{}, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/oidc/userinfo", c.BaseURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&userInfo)
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

// --- Users ---

// UserCreateDto represents the request body for creating a user.
type UserCreateDto struct {
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"firstName" validate:"required,min=1,max=50"`
	LastName  string `json:"lastName" validate:"required,min=1,max=50"`
	Username  string `json:"username" validate:"required,min=2,max=50"`
	IsAdmin   bool   `json:"isAdmin"`
}

// UserDto represents a user.
type UserDto struct {
	ID           string           `json:"id"`
	Email        string           `json:"email"`
	FirstName    string           `json:"firstName"`
	LastName     string           `json:"lastName"`
	Username     string           `json:"username"`
	IsAdmin      bool             `json:"isAdmin"`
	LdapId       string           `json:"ldapId"`
	CustomClaims []CustomClaimDto `json:"customClaims"`
	UserGroups   []UserGroupDto   `json:"userGroups"`
}

// UserGroupDto represents a user group.
type UserGroupDto struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	FriendlyName string           `json:"friendlyName"`
	CreatedAt    string           `json:"createdAt"`
	LdapId       string           `json:"ldapId"`
	CustomClaims []CustomClaimDto `json:"customClaims"`
}

// OneTimeAccessTokenCreateDto represents options for creating a one-time access token.
type OneTimeAccessTokenCreateDto struct {
	ExpiresAt string `json:"expiresAt" validate:"required"`
	UserId    string `json:"userId"` // UserId is optional here, as per the API spec. It's only required in some contexts
}

// ExchangeOneTimeAccessToken exchanges a one-time access token for a session token.
//
// See https://pocket-id.example.com/one-time-access-token/{token}
func (c *Client) ExchangeOneTimeAccessToken(token string) (*UserDto, error) {
	resp, err := c.request("POST", fmt.Sprintf("/one-time-access-token/%s", token), nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userDto UserDto
	err = json.NewDecoder(resp.Body).Decode(&userDto)
	if err != nil {
		return nil, err
	}

	return &userDto, nil
}

// SetupInitialAdmin generates a setup access token for initial admin user configuration.
//
// See https://pocket-id.example.com/one-time-access-token/setup
func (c *Client) SetupInitialAdmin() (*UserDto, error) {

	resp, err := c.request("POST", "/one-time-access-token/setup", nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userDto UserDto
	err = json.NewDecoder(resp.Body).Decode(&userDto)
	if err != nil {
		return nil, err
	}
	return &userDto, nil
}

// ListUsers gets a paginated list of users.
//
// See https://pocket-id.example.com/users
func (c *Client) ListUsers(search string, page, limit int, sortColumn, sortDirection string) (*Paginated[UserDto], error) {
	params := url.Values{}
	params.Set("search", search)
	params.Set("page", strconv.Itoa(page))
	params.Set("limit", strconv.Itoa(limit))
	params.Set("sort_column", sortColumn)
	params.Set("sort_direction", sortDirection)

	resp, err := c.request("GET", fmt.Sprintf("/users?%s", params.Encode()), nil, "")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var users Paginated[UserDto]
	err = decodePaginated(resp, &users)
	if err != nil {
		return nil, err
	}

	return &users, nil
}

// CreateUser creates a new user.
//
// See https://pocket-id.example.com/users
func (c *Client) CreateUser(userCreateDto UserCreateDto) (*UserDto, error) {
	body, err := json.Marshal(userCreateDto)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("POST", "/users", bytes.NewBuffer(body), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userDto UserDto
	err = json.NewDecoder(resp.Body).Decode(&userDto)
	if err != nil {
		return nil, err
	}

	return &userDto, nil
}

// DeleteUser deletes a specific user by ID.
//
// See https://pocket-id.example.com/users/{id}
func (c *Client) DeleteUser(id string) error {
	resp, err := c.request("DELETE", fmt.Sprintf("/users/%s", id), nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetUserByID retrieves detailed information about a specific user.
//
// See https://pocket-id.example.com/users/{id}
func (c *Client) GetUserByID(id string) (*UserDto, error) {
	resp, err := c.request("GET", fmt.Sprintf("/users/%s", id), nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userDto UserDto
	err = json.NewDecoder(resp.Body).Decode(&userDto)
	if err != nil {
		return nil, err
	}

	return &userDto, nil
}

// UpdateUser updates an existing user by ID.
//
// See https://pocket-id.example.com/users/{id}
func (c *Client) UpdateUser(id string, userCreateDto UserCreateDto) (*UserDto, error) {
	body, err := json.Marshal(userCreateDto)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("PUT", fmt.Sprintf("/users/%s", id), bytes.NewBuffer(body), "application/json")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userDto UserDto
	err = json.NewDecoder(resp.Body).Decode(&userDto)
	if err != nil {
		return nil, err
	}

	return &userDto, nil
}

// GetUserGroups retrieves all groups a specific user belongs to.
//
// See https://pocket-id.example.com/users/{id}/groups
func (c *Client) GetUserGroups(id string) ([]UserGroupDto, error) {
	resp, err := c.request("GET", fmt.Sprintf("/users/%s/groups", id), nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var groups []UserGroupDto
	err = json.NewDecoder(resp.Body).Decode(&groups)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

// CreateOneTimeAccessTokenForCurrentUser generates a one-time access token for the currently authenticated user.
//
// See https://pocket-id.example.com/users/{id}/one-time-access-token
func (c *Client) CreateOneTimeAccessTokenForCurrentUser(id string, tokenOptions OneTimeAccessTokenCreateDto) (string, error) {
	body, err := json.Marshal(tokenOptions)
	if err != nil {
		return "", err
	}
	resp, err := c.request("POST", fmt.Sprintf("/users/%s/one-time-access-token", id), bytes.NewBuffer(body), "application/json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	return response.Token, nil
}

// UpdateUserProfilePicture updates a specific user's profile picture.
//
// See https://pocket-id.example.com/users/{id}/profile-picture
func (c *Client) UpdateUserProfilePicture(id string, file io.Reader) error {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", "profile.png") // Filename doesn't affect server processing
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, file); err != nil {
		return err
	}
	w.Close()

	resp, err := c.request("PUT", fmt.Sprintf("/users/%s/profile-picture", id), &b, w.FormDataContentType())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetUserProfilePicture retrieves a specific user's profile picture.
//
// See https://pocket-id.example.com/users/{id}/profile-picture.png
func (c *Client) GetUserProfilePicture(id string) ([]byte, error) {
	resp, err := c.request("GET", fmt.Sprintf("/users/%s/profile-picture.png", id), nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// UserUpdateUserGroupDto represents the request body for updating user groups for a user.
type UserUpdateUserGroupDto struct {
	UserGroupIds []string `json:"userGroupIds" validate:"required"`
}

// UpdateUserGroups updates the groups a specific user belongs to.
//
// See https://pocket-id.example.com/users/{id}/user-groups
func (c *Client) UpdateUserGroups(id string, groupDto UserUpdateUserGroupDto) (*UserDto, error) {
	body, err := json.Marshal(groupDto)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("PUT", fmt.Sprintf("/users/%s/user-groups", id), bytes.NewBuffer(body), "application/json")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userDto UserDto
	err = json.NewDecoder(resp.Body).Decode(&userDto)
	if err != nil {
		return nil, err
	}

	return &userDto, nil
}

// GetCurrentUser retrieves information about the currently authenticated user.
//
// See https://pocket-id.example.com/users/me
func (c *Client) GetCurrentUser() (*UserDto, error) {
	resp, err := c.request("GET", "/users/me", nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userDto UserDto
	err = json.NewDecoder(resp.Body).Decode(&userDto)
	if err != nil {
		return nil, err
	}

	return &userDto, nil
}

// UpdateCurrentUser updates the currently authenticated user's information.
//
// See https://pocket-id.example.com/users/me
func (c *Client) UpdateCurrentUser(userCreateDto UserCreateDto) (*UserDto, error) {
	body, err := json.Marshal(userCreateDto)
	if err != nil {
		return nil, err
	}
	resp, err := c.request("PUT", "/users/me", bytes.NewBuffer(body), "application/json")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userDto UserDto
	err = json.NewDecoder(resp.Body).Decode(&userDto)
	if err != nil {
		return nil, err
	}

	return &userDto, nil
}

// UpdateCurrentUserProfilePicture updates the currently authenticated user's profile picture.
//
// See https://pocket-id.example.com/users/me/profile-picture
func (c *Client) UpdateCurrentUserProfilePicture(file io.Reader) error {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", "profile.png") // Filename doesn't affect server processing
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, file); err != nil {
		return err
	}
	w.Close()

	resp, err := c.request("PUT", "/users/me/profile-picture", &b, w.FormDataContentType())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetCurrentUserProfilePicture retrieves the currently authenticated user's profile picture.
//
// See https://pocket-id.example.com/users/me/profile-picture.png
func (c *Client) GetCurrentUserProfilePicture() ([]byte, error) {
	resp, err := c.request("GET", "/users/me/profile-picture.png", nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// --- User Groups ---

// UserGroupCreateDto represents the request body for creating a user group.
type UserGroupCreateDto struct {
	Name         string `json:"name" validate:"required,min=2,max=255"`
	FriendlyName string `json:"friendlyName" validate:"required,min=2,max=50"`
}

// UserGroupDtoWithUsers represents a user group with its users.
type UserGroupDtoWithUsers struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	FriendlyName string           `json:"friendlyName"`
	CreatedAt    string           `json:"createdAt"`
	LdapId       string           `json:"ldapId"`
	CustomClaims []CustomClaimDto `json:"customClaims"`
	Users        []UserDto        `json:"users"`
}

// UserGroupUpdateUsersDto represents the request body for updating users in a group.
type UserGroupUpdateUsersDto struct {
	UserIds []string `json:"userIds" validate:"required"`
}

// ListUserGroups gets a paginated list of user groups.
//
// See https://pocket-id.example.com/user-groups
func (c *Client) ListUserGroups(search string, page, limit int, sortColumn, sortDirection string) (*Paginated[UserGroupDtoWithUserCount], error) {
	params := url.Values{}
	params.Set("search", search)
	params.Set("page", strconv.Itoa(page))
	params.Set("limit", strconv.Itoa(limit))
	params.Set("sort_column", sortColumn)
	params.Set("sort_direction", sortDirection)

	resp, err := c.request("GET", fmt.Sprintf("/user-groups?%s", params.Encode()), nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var userGroups Paginated[UserGroupDtoWithUserCount]

	err = decodePaginated(resp, &userGroups)
	if err != nil {
		return nil, err
	}

	return &userGroups, nil
}

// CreateUserGroup creates a new user group.
//
// See https://pocket-id.example.com/user-groups
func (c *Client) CreateUserGroup(userGroupCreateDto UserGroupCreateDto) (*UserGroupDtoWithUsers, error) {
	body, err := json.Marshal(userGroupCreateDto)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("POST", "/user-groups", bytes.NewBuffer(body), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userGroupDto UserGroupDtoWithUsers
	err = json.NewDecoder(resp.Body).Decode(&userGroupDto)
	if err != nil {
		return nil, err
	}

	return &userGroupDto, nil
}

// DeleteUserGroup deletes a specific user group by ID.
//
// See https://pocket-id.example.com/user-groups/{id}
func (c *Client) DeleteUserGroup(id string) error {
	resp, err := c.request("DELETE", fmt.Sprintf("/user-groups/%s", id), nil, "")

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetUserGroupByID retrieves detailed information about a specific user group, including its users.
//
// See https://pocket-id.example.com/user-groups/{id}
func (c *Client) GetUserGroupByID(id string) (*UserGroupDtoWithUsers, error) {
	resp, err := c.request("GET", fmt.Sprintf("/user-groups/%s", id), nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userGroupDto UserGroupDtoWithUsers
	err = json.NewDecoder(resp.Body).Decode(&userGroupDto)
	if err != nil {
		return nil, err
	}

	return &userGroupDto, nil
}

// UpdateUserGroup updates an existing user group by ID.
//
// See https://pocket-id.example.com/user-groups/{id}
func (c *Client) UpdateUserGroup(id string, userGroupCreateDto UserGroupCreateDto) (*UserGroupDtoWithUsers, error) {
	body, err := json.Marshal(userGroupCreateDto)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("PUT", fmt.Sprintf("/user-groups/%s", id), bytes.NewBuffer(body), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userGroupDto UserGroupDtoWithUsers
	err = json.NewDecoder(resp.Body).Decode(&userGroupDto)
	if err != nil {
		return nil, err
	}

	return &userGroupDto, nil
}

// UpdateUsersInGroup updates the list of users belonging to a specific user group.
//
// See https://pocket-id.example.com/user-groups/{id}/users
func (c *Client) UpdateUsersInGroup(id string, users UserGroupUpdateUsersDto) (*UserGroupDtoWithUsers, error) {
	body, err := json.Marshal(users)
	if err != nil {
		return nil, err
	}

	resp, err := c.request("PUT", fmt.Sprintf("/user-groups/%s/users", id), bytes.NewBuffer(body), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userGroupDto UserGroupDtoWithUsers
	err = json.NewDecoder(resp.Body).Decode(&userGroupDto)
	if err != nil {
		return nil, err
	}

	return &userGroupDto, nil
}
