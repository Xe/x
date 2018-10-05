package tun2

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func MockStorage() *mockStorage {
	return &mockStorage{
		tokens:  make(map[string]mockToken),
		domains: make(map[string]string),
	}
}

type mockToken struct {
	user   string
	scopes []string
}

// mockStorage is a simple mock of the Storage interface suitable for testing.
type mockStorage struct {
	sync.Mutex
	tokens  map[string]mockToken
	domains map[string]string
}

func (ms *mockStorage) AddToken(token, user string, scopes []string) {
	ms.Lock()
	defer ms.Unlock()

	ms.tokens[token] = mockToken{user: user, scopes: scopes}
}

func (ms *mockStorage) AddRoute(domain, user string) {
	ms.Lock()
	defer ms.Unlock()

	ms.domains[domain] = user
}

func (ms *mockStorage) HasToken(ctx context.Context, token string) (string, []string, error) {
	ms.Lock()
	defer ms.Unlock()

	tok, ok := ms.tokens[token]
	if !ok {
		return "", nil, errors.New("no such token")
	}

	return tok.user, tok.scopes, nil
}

func (ms *mockStorage) HasRoute(ctx context.Context, domain string) (string, error) {
	ms.Lock()
	defer ms.Unlock()

	user, ok := ms.domains[domain]
	if !ok {
		return "", errors.New("no such route")
	}

	return user, nil
}

func TestMockStorage(t *testing.T) {
	ms := MockStorage()

	t.Run("token", func(t *testing.T) {
		ms.AddToken(token, user, []string{"connect"})

		us, sc, err := ms.HasToken(nil, token)
		if err != nil {
			t.Fatal(err)
		}

		if us != user {
			t.Fatalf("username was %q, expected %q", us, user)
		}

		if sc[0] != "connect" {
			t.Fatalf("token expected to only have one scope, connect")
		}
	})

	t.Run("domain", func(t *testing.T) {
		ms.AddRoute(domain, user)

		us, err := ms.HasRoute(nil, domain)
		if err != nil {
			t.Fatal(err)
		}

		if us != user {
			t.Fatalf("username was %q, expected %q", us, user)
		}
	})

}
