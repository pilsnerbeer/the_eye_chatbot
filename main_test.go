package main

import (
	"context"
	"github.com/google/generative-ai-go/genai"
	"os"
	"testing"
)

// Function to read API key from file
func readAPIKey() (string, error) {
	data, err := os.ReadFile("key.txt")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Function to create a client with the API key
func createClient(apiKey string) (*Client, error) {
	c, err := NewClient(apiKey, context.Background())
	if err != nil {
		return nil, err
	}
	return &Client{APIKey: apiKey, genai: c}, nil
}

// Mock function to get a response from the API (replace with actual API interaction in a real test)
func (c *Client) getResponse() (string, error) {
	// Simulate an API response (replace with actual API call)
	return "API response", nil
}

// Test for reading API key
func TestReadAPIKey(t *testing.T) {
	apiKey, err := readAPIKey()
	if err != nil {
		t.Fatalf("Failed to read API key: %v", err)
	}
	if apiKey == "" {
		t.Fatal("API key is empty")
	}
}

// Test for creating client with API key
func TestCreateClient(t *testing.T) {
	apiKey, err := readAPIKey()
	if err != nil {
		t.Fatalf("Failed to read API key: %v", err)
	}
	client, err := createClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if client.APIKey != apiKey {
		t.Fatal("Client API key does not match")
	}
}

func setupClient(t *testing.T) (*Client, error) {
	apiKey, err := readAPIKey()
	if err != nil {
		return nil, err // Propagate error to test function
	}
	client, err := createClient(apiKey)
	if err != nil {
		return nil, err // Propagate error to test function
	}
	return client, nil
}

// Test for getting a response from the API
func TestGetResponse(t *testing.T) {
	client, err := setupClient(t)
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}
	c := NewModel(client.genai, "gemini-1.5-flash")
	cs := c.StartChat()
	_, err = cs.SendMessage(context.Background(), genai.Text("Hello"))
	if err != nil {
		t.Fatalf("Failed to get response: %v", err)
	}
}

type Client struct {
	APIKey string
	genai  *genai.Client
}
