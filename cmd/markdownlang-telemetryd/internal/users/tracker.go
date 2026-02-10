// Package users implements user tracking and quota management for markdownlang-telemetryd.
package users

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"within.website/x/store"
)

const (
	// QuotaLimit is the maximum number of executions allowed per user.
	QuotaLimit = 69
	// UsersPrefix is the S3 key prefix for storing user data.
	UsersPrefix = "_meta"
)

// User represents a tracked user with their first seen timestamp.
type User struct {
	Name      string    `json:"name"`       // User's name from git config
	Email     string    `json:"email"`      // User's email from git config
	FirstSeen time.Time `json:"first_seen"` // When this user was first seen
}

// usersData is the structure stored in S3 for user tracking.
type usersData struct {
	SeenUsers map[string]User `json:"seen_users"`
}

// countsData is the structure stored in S3 for execution counts.
type countsData struct {
	ExecutionCounts map[string]int `json:"execution_counts"`
}

// UserTracker tracks users and their execution counts.
// It provides thread-safe access to user data and persists to S3 via the store Interface.
type UserTracker struct {
	seenUsers       map[string]User // key: email, value: User info
	executionCounts map[string]int  // key: email, value: execution count
	mu              sync.RWMutex    // protects seenUsers and executionCounts
	usersStore      *store.JSON[usersData]
	countsStore     *store.JSON[countsData]
}

// New creates a new UserTracker and loads existing data from S3.
// If the data does not exist in S3, empty maps are initialized.
func New(ctx context.Context, st store.Interface) (*UserTracker, error) {
	usersStore := &store.JSON[usersData]{
		Underlying: st,
		Prefix:     UsersPrefix,
	}
	countsStore := &store.JSON[countsData]{
		Underlying: st,
		Prefix:     UsersPrefix,
	}

	ut := &UserTracker{
		seenUsers:       make(map[string]User),
		executionCounts: make(map[string]int),
		usersStore:      usersStore,
		countsStore:     countsStore,
	}

	if err := ut.Load(ctx); err != nil {
		slog.Warn("failed to load user data from S3, starting with empty state", "error", err)
	}

	return ut, nil
}

// IsNewUser returns true if this (name, email) pair hasn't been seen before.
// A user is considered "new" if their email is not in the seen users map.
func (ut *UserTracker) IsNewUser(name, email string) bool {
	if email == "" {
		return false
	}

	ut.mu.RLock()
	defer ut.mu.RUnlock()

	_, exists := ut.seenUsers[email]
	return !exists
}

// RecordUser records a new user with the given name and email.
// If the user already exists, their information is not updated.
// The FirstSeen timestamp is set to the current time.
func (ut *UserTracker) RecordUser(name, email string) {
	if email == "" {
		return
	}

	ut.mu.Lock()
	defer ut.mu.Unlock()

	// Only record if this email hasn't been seen before
	if _, exists := ut.seenUsers[email]; !exists {
		ut.seenUsers[email] = User{
			Name:      name,
			Email:     email,
			FirstSeen: time.Now(),
		}
	}
}

// IncrementExecution increments the execution count for the given email
// and returns the new count.
// If the email is empty, the count is not incremented and 0 is returned.
func (ut *UserTracker) IncrementExecution(email string) int {
	if email == "" {
		return 0
	}

	ut.mu.Lock()
	defer ut.mu.Unlock()

	ut.executionCounts[email]++
	return ut.executionCounts[email]
}

// HasExceededQuota returns true if the execution count for the given email
// exceeds QuotaLimit (69).
// If the email is empty or has no executions, returns false.
func (ut *UserTracker) HasExceededQuota(email string) bool {
	if email == "" {
		return false
	}

	ut.mu.RLock()
	defer ut.mu.RUnlock()

	count, exists := ut.executionCounts[email]
	return exists && count > QuotaLimit
}

// Save persists the tracker data to S3 as JSON.
// Errors are returned but do not affect the in-memory state.
func (ut *UserTracker) Save(ctx context.Context) error {
	ut.mu.RLock()
	defer ut.mu.RUnlock()

	// Save users data
	usersData := usersData{
		SeenUsers: ut.seenUsers,
	}
	if err := ut.usersStore.Set(ctx, "users.json", usersData); err != nil {
		return err
	}

	// Save execution counts
	countsData := countsData{
		ExecutionCounts: ut.executionCounts,
	}
	if err := ut.countsStore.Set(ctx, "counts.json", countsData); err != nil {
		return err
	}

	return nil
}

// Load loads tracker data from S3.
// If the data does not exist or is invalid, the in-memory maps remain unchanged.
// Errors are returned but do not prevent the tracker from functioning.
func (ut *UserTracker) Load(ctx context.Context) error {
	// Load users data
	usersData, err := ut.usersStore.Get(ctx, "users.json")
	if err != nil {
		if err != store.ErrNotFound {
			slog.Warn("error loading users from S3", "error", err)
		}
	} else {
		ut.mu.Lock()
		ut.seenUsers = usersData.SeenUsers
		if ut.seenUsers == nil {
			ut.seenUsers = make(map[string]User)
		}
		ut.mu.Unlock()
	}

	// Load execution counts
	countsData, err := ut.countsStore.Get(ctx, "counts.json")
	if err != nil {
		if err != store.ErrNotFound {
			slog.Warn("error loading counts from S3", "error", err)
		}
		ut.executionCounts = make(map[string]int)
		return nil
	}

	ut.mu.Lock()
	defer ut.mu.Unlock()

	ut.executionCounts = countsData.ExecutionCounts
	if ut.executionCounts == nil {
		ut.executionCounts = make(map[string]int)
	}

	return nil
}
