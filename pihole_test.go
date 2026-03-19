package pihole

import "testing"

func TestNewClient(t *testing.T) {
	client, err := NewClient("http://localhost:8080", "test-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.baseURL != "http://localhost:8080" {
		t.Errorf("expected baseURL http://localhost:8080, got %s", client.baseURL)
	}
}

func TestNewClient_TrailingSlash(t *testing.T) {
	client, err := NewClient("http://localhost:8080/", "test-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.baseURL != "http://localhost:8080" {
		t.Errorf("expected trailing slash stripped, got %s", client.baseURL)
	}
}

func TestNewClient_EmptyURL(t *testing.T) {
	_, err := NewClient("", "test-password")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestNewClient_EmptyPassword(t *testing.T) {
	_, err := NewClient("http://localhost:8080", "")
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}
