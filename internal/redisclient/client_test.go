package redisclient

import (
	"testing"
)

func TestGetClient_NotNil(t *testing.T) {
	client := GetClient()
	if client == nil {
		t.Fatal("expected non-nil redis client")
	}
}

func TestGetClient_Singleton(t *testing.T) {
	c1 := GetClient()
	c2 := GetClient()
	if c1 != c2 {
		t.Fatal("expected GetClient to return the same instance (singleton)")
	}
}
