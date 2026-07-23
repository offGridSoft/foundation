package exchange

import (
	"net/http"
	"testing"
)

func mustTestClient(t testing.TB, httpClient *http.Client) Client {
	t.Helper()
	client, err := NewClient(httpClient)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}
	return client
}

func mustTestClientRuntime(t testing.TB, httpClient *http.Client, clock clock, jitter jitterSource, waiter waiter) Client {
	t.Helper()
	client, err := newClient(httpClient, clock, jitter, waiter)
	if err != nil {
		t.Fatalf("newClient() error = %v, want nil", err)
	}
	return client
}
