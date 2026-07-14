package core

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const OffgridAPIBaseURL = "https://api.offgridsoftware.ca"

type APIEndpoint struct {
	value string
}

func APIEndpointForBaseURL(baseURL, path string) (APIEndpoint, error) {
	base, err := normalizeAPIBaseURL(baseURL)
	if err != nil {
		return APIEndpoint{}, err
	}
	if !strings.HasPrefix(path, "/") || path == "/" {
		return APIEndpoint{}, fmt.Errorf(ErrFmtAPIEndpoint, ErrFoundationContract)
	}
	return ParseAPIEndpoint(base + path)
}

func ParseAPIEndpoint(value string) (APIEndpoint, error) {
	if err := validateAPIEndpoint(value); err != nil {
		return APIEndpoint{}, fmt.Errorf(ErrFmtAPIEndpoint, err)
	}
	return APIEndpoint{value: value}, nil
}

func (e APIEndpoint) String() string { return e.value }

func (e APIEndpoint) Validate() error {
	_, err := ParseAPIEndpoint(e.value)
	return err
}

func (e APIEndpoint) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(e.value)
}

func (e *APIEndpoint) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtAPIEndpoint, ErrFoundationContract)
	}
	parsed, err := ParseAPIEndpoint(value)
	if err != nil {
		return err
	}
	if err := parsed.Validate(); err != nil {
		return err
	}
	previous := *e
	*e = parsed
	if err := e.Validate(); err != nil {
		*e = previous
		return err
	}
	return nil
}

func normalizeAPIBaseURL(value string) (string, error) {
	parts, ok := parseAPIURL(value)
	if !ok || (parts.path != "" && parts.path != "/") {
		return "", fmt.Errorf(ErrFmtAPIEndpoint, ErrFoundationContract)
	}
	return parts.scheme + "://" + parts.authority, nil
}

func validateAPIEndpoint(value string) error {
	parts, ok := parseAPIURL(value)
	if !ok || parts.path == "" || parts.path == "/" {
		return ErrFoundationContract
	}
	return nil
}

type apiURLParts struct {
	scheme    string
	authority string
	path      string
}

func parseAPIURL(value string) (apiURLParts, bool) {
	if err := ValidateOpaqueToken(value, HTTPSURLDefaultMaxRunes); err != nil || !validASCIIAPIURL(value) {
		return apiURLParts{}, false
	}
	scheme, remainder, ok := strings.Cut(value, "://")
	if !ok || strings.ContainsAny(remainder, "@?#") {
		return apiURLParts{}, false
	}
	authority, path := splitAPIAuthorityPath(remainder)
	if !validAPIAuthority(authority) || !validAPIEndpointScheme(scheme, authority) {
		return apiURLParts{}, false
	}
	return apiURLParts{scheme: scheme, authority: authority, path: path}, true
}

func splitAPIAuthorityPath(value string) (string, string) {
	index := strings.IndexByte(value, '/')
	if index < 0 {
		return value, ""
	}
	return value[:index], value[index:]
}

func validAPIEndpointScheme(scheme, authority string) bool {
	if scheme == URLSchemeHTTPS {
		return true
	}
	return scheme == URLSchemeHTTP && loopbackAPIAuthority(authority)
}

func loopbackAPIAuthority(authority string) bool {
	host, ok := apiAuthorityHost(authority)
	if !ok {
		return false
	}
	if host == HostLocalhost || host == "::1" {
		return true
	}
	parts := strings.Split(host, ".")
	return len(parts) == 4 && parts[0] == "127" && validIPv4Parts(parts)
}

func apiAuthorityHost(authority string) (string, bool) {
	if strings.HasPrefix(authority, "[") {
		closeIndex := strings.IndexByte(authority, ']')
		if closeIndex < 2 || !validAPIPortSuffix(authority[closeIndex+1:]) {
			return "", false
		}
		return authority[1:closeIndex], true
	}
	if strings.Count(authority, ":") > 1 {
		return "", false
	}
	host, port, found := strings.Cut(authority, ":")
	if found && !validAPIPort(port) {
		return "", false
	}
	return host, host != ""
}

func validAPIPortSuffix(value string) bool {
	return value == "" || strings.HasPrefix(value, ":") && validAPIPort(value[1:])
}

func validAPIPort(value string) bool {
	port, err := strconv.ParseUint(value, 10, 16)
	return err == nil && port > 0
}

func validIPv4Parts(parts []string) bool {
	for _, part := range parts {
		value, err := strconv.ParseUint(part, 10, 8)
		if err != nil || strconv.FormatUint(value, 10) != part {
			return false
		}
	}
	return true
}

func validAPIAuthority(authority string) bool {
	if !validAPIEndpointHost(authority) {
		return false
	}
	host, ok := apiAuthorityHost(authority)
	return ok && host != ""
}

func validAPIEndpointHost(host string) bool {
	if host == "" {
		return false
	}
	for _, value := range host {
		switch {
		case value >= 'a' && value <= 'z':
		case value >= 'A' && value <= 'Z':
		case value >= '0' && value <= '9':
		case value == '.', value == '-', value == ':', value == '[', value == ']':
		default:
			return false
		}
	}
	return true
}

func validASCIIAPIURL(value string) bool {
	for index := range len(value) {
		if value[index] > 0x7f {
			return false
		}
	}
	return true
}
