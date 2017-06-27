package storage

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-oracle-terraform/opc"
)

const (
	_ClientTestUser   = "test-user"
	_ClientTestDomain = "test-domain"
)

func newAuthenticatingServer(handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("ORACLE_LOG") != "" {
			log.Printf("[DEBUG] Received request: %s, %s\n", r.Method, r.URL)
		}

		if r.URL.Path == "/authenticate/" {
			http.SetCookie(w, &http.Cookie{Name: "testAuthCookie", Value: "cookie value"})
			//	w.WriteHeader(200)
		} else {
			handler(w, r)
		}
	}))
}

func getStorageTestClient(c *opc.Config) (*StorageClient, error) {
	// Build up config with default values if omitted

	if c.IdentityDomain == nil {
		domain := os.Getenv("OPC_IDENTITY_DOMAIN")
		c.IdentityDomain = &domain
	}

	if c.Username == nil {
		username := os.Getenv("OPC_USERNAME")
		c.Username = &username
	}

	if c.Password == nil {
		password := os.Getenv("OPC_PASSWORD")
		c.Password = &password
	}

	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				TLSHandshakeTimeout: 120 * time.Second},
		}
	}

	return NewStorageClient(c)
}

func getBlankTestClient() (*StorageClient, *httptest.Server, error) {
	server := newAuthenticatingServer(func(w http.ResponseWriter, r *http.Request) {
	})

	endpoint, err := url.Parse(server.URL)
	if err != nil {
		server.Close()
		return nil, nil, err
	}

	client, err := getStorageTestClient(&opc.Config{
		IdentityDomain: opc.String(_ClientTestDomain),
		Username:       opc.String(_ClientTestUser),
		APIEndpoint:    endpoint,
	})
	if err != nil {
		server.Close()
		return nil, nil, err
	}
	return client, server, nil
}

// Returns a stub client with default values, and a custom API Endpoint
func getStubClient(endpoint *url.URL) (*StorageClient, error) {
	domain := "test"
	username := "test"
	password := "test"
	config := &opc.Config{
		IdentityDomain: &domain,
		Username:       &username,
		Password:       &password,
		APIEndpoint:    endpoint,
	}
	return getStorageTestClient(config)
}

func unmarshalRequestBody(t *testing.T, r *http.Request, target interface{}) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	err := json.Unmarshal(buf.Bytes(), target)
	if err != nil {
		t.Fatalf("Error marshalling request: %s", err)
	}
}
