// Package agent implements agent-to-agent calling functionality for markdownlang.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// CallManager manages agent-to-agent calls and tracks metrics.
type CallManager struct {
	mu            sync.RWMutex
	registry      *Registry
	activeCalls   map[string]*ActiveCall
	callHistory   []CallRecord
	maxHistory    int
	enableTracing bool
}

// ActiveCall represents an in-progress agent call.
type ActiveCall struct {
	ID         string
	ImportPath string
	AgentName  string
	Input      json.RawMessage
	StartTime  time.Time
	Context    context.Context
}

// CallRecord represents a completed agent call.
type CallRecord struct {
	ID         string          `json:"id"`
	ImportPath string          `json:"import_path"`
	AgentName  string          `json:"agent_name"`
	Input      json.RawMessage `json:"input"`
	Output     json.RawMessage `json:"output"`
	Error      string          `json:"error,omitempty"`
	StartTime  time.Time       `json:"start_time"`
	EndTime    time.Time       `json:"end_time"`
	Duration   time.Duration   `json:"duration"`
	Metrics    Metrics         `json:"metrics"`
}

// CallManagerConfig configures a CallManager.
type CallManagerConfig struct {
	// Registry is the agent registry to use for resolving imports.
	Registry *Registry

	// MaxHistory is the maximum number of call records to keep.
	MaxHistory int

	// EnableTracing enables detailed tracing of agent calls.
	EnableTracing bool
}

// NewCallManager creates a new CallManager.
func NewCallManager(config *CallManagerConfig) *CallManager {
	if config == nil {
		config = &CallManagerConfig{}
	}

	maxHistory := config.MaxHistory
	if maxHistory <= 0 {
		maxHistory = 100 // Default to 100 records
	}

	return &CallManager{
		registry:      config.Registry,
		activeCalls:   make(map[string]*ActiveCall),
		callHistory:   make([]CallRecord, 0, maxHistory),
		maxHistory:    maxHistory,
		enableTracing: config.EnableTracing,
	}
}

// Call invokes an imported agent with the given input.
func (cm *CallManager) Call(ctx context.Context, importPath string, input json.RawMessage) (*AgentCallResult, error) {
	callID := generateCallID()

	if cm.enableTracing {
		slog.Info("starting agent call",
			"call_id", callID,
			"import", importPath,
			"input_length", len(input))
	}

	// Record the active call
	activeCall := &ActiveCall{
		ID:         callID,
		ImportPath: importPath,
		Input:      input,
		StartTime:  time.Now(),
		Context:    ctx,
	}

	// Get agent name if available
	if agent, ok := cm.registry.GetAgent(importPath); ok {
		if prog, ok := agent.(*Program); ok {
			activeCall.AgentName = prog.Name
		}
	}

	cm.mu.Lock()
	cm.activeCalls[callID] = activeCall
	cm.mu.Unlock()

	// Ensure we clean up the active call
	defer func() {
		cm.mu.Lock()
		delete(cm.activeCalls, callID)
		cm.mu.Unlock()
	}()

	// Execute the call using the registry
	result, err := cm.registry.CallAgent(ctx, importPath, input)

	// Record the call in history
	record := CallRecord{
		ID:         callID,
		ImportPath: importPath,
		AgentName:  activeCall.AgentName,
		Input:      input,
		StartTime:  activeCall.StartTime,
		EndTime:    time.Now(),
		Duration:   time.Since(activeCall.StartTime),
	}

	if result != nil {
		record.Output = result.Output
		record.Metrics = result.Metrics
		if result.Error != "" {
			record.Error = result.Error
		}
	}

	if err != nil {
		record.Error = err.Error()
	}

	cm.addToHistory(record)

	if cm.enableTracing {
		slog.Info("completed agent call",
			"call_id", callID,
			"import", importPath,
			"duration", record.Duration,
			"error", record.Error)
	}

	return result, err
}

// addToHistory adds a call record to history, enforcing max history limit.
func (cm *CallManager) addToHistory(record CallRecord) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.callHistory = append(cm.callHistory, record)

	// Trim history if needed
	if len(cm.callHistory) > cm.maxHistory {
		// Remove oldest records
		cm.callHistory = cm.callHistory[len(cm.callHistory)-cm.maxHistory:]
	}
}

// GetActiveCalls returns all currently active calls.
func (cm *CallManager) GetActiveCalls() []*ActiveCall {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	calls := make([]*ActiveCall, 0, len(cm.activeCalls))
	for _, call := range cm.activeCalls {
		calls = append(calls, call)
	}
	return calls
}

// GetHistory returns all call records.
func (cm *CallManager) GetHistory() []CallRecord {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	history := make([]CallRecord, len(cm.callHistory))
	copy(history, cm.callHistory)
	return history
}

// GetHistoryByImport returns call records for a specific import path.
func (cm *CallManager) GetHistoryByImport(importPath string) []CallRecord {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var results []CallRecord
	for _, record := range cm.callHistory {
		if record.ImportPath == importPath {
			results = append(results, record)
		}
	}
	return results
}

// GetHistoryByAgent returns call records for a specific agent name.
func (cm *CallManager) GetHistoryByAgent(agentName string) []CallRecord {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var results []CallRecord
	for _, record := range cm.callHistory {
		if record.AgentName == agentName {
			results = append(results, record)
		}
	}
	return results
}

// GetCallMetrics returns aggregate metrics for all calls.
func (cm *CallManager) GetCallMetrics() CallMetrics {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	metrics := CallMetrics{
		TotalCalls:    len(cm.callHistory),
		ActiveCalls:   len(cm.activeCalls),
		CallsByImport: make(map[string]int),
		CallsByAgent:  make(map[string]int),
		TotalDuration: 0,
		TotalTokens:   0,
	}

	for _, record := range cm.callHistory {
		metrics.CallsByImport[record.ImportPath]++
		if record.AgentName != "" {
			metrics.CallsByAgent[record.AgentName]++
		}
		metrics.TotalDuration += record.Duration
		metrics.TotalTokens += record.Metrics.TotalTokens
	}

	// Calculate averages
	if metrics.TotalCalls > 0 {
		metrics.AverageDuration = metrics.TotalDuration / time.Duration(metrics.TotalCalls)
		metrics.AverageTokensPerCall = metrics.TotalTokens / metrics.TotalCalls
	}

	return metrics
}

// CallMetrics contains aggregate metrics for agent calls.
type CallMetrics struct {
	TotalCalls           int            `json:"total_calls"`
	ActiveCalls          int            `json:"active_calls"`
	CallsByImport        map[string]int `json:"calls_by_import"`
	CallsByAgent         map[string]int `json:"calls_by_agent"`
	TotalDuration        time.Duration  `json:"total_duration"`
	AverageDuration      time.Duration  `json:"average_duration"`
	TotalTokens          int            `json:"total_tokens"`
	AverageTokensPerCall int            `json:"average_tokens_per_call"`
}

// ClearHistory clears all call history.
func (cm *CallManager) ClearHistory() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.callHistory = make([]CallRecord, 0, cm.maxHistory)
}

// ClearActiveCalls clears all active calls (use with caution).
func (cm *CallManager) ClearActiveCalls() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.activeCalls = make(map[string]*ActiveCall)
}

// SetRegistry sets the agent registry.
func (cm *CallManager) SetRegistry(registry *Registry) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.registry = registry
}

// GetRegistry returns the agent registry.
func (cm *CallManager) GetRegistry() *Registry {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.registry
}

// generateCallID generates a unique call ID.
func generateCallID() string {
	return fmt.Sprintf("call-%d", time.Now().UnixNano())
}

// AgentCallTool wraps the CallManager as a ToolHandler for agent calls.
type AgentCallTool struct {
	manager    *CallManager
	importPath string
}

// NewAgentCallTool creates a new tool for calling a specific imported agent.
func NewAgentCallTool(manager *CallManager, importPath string) *AgentCallTool {
	return &AgentCallTool{
		manager:    manager,
		importPath: importPath,
	}
}

// Execute calls the imported agent with the given input.
func (t *AgentCallTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	result, err := t.manager.Call(ctx, t.importPath, input)
	if err != nil {
		return nil, err
	}

	// Return the output from the agent call
	return result.Output, nil
}

// Schema returns the JSON schema for this tool's input.
func (t *AgentCallTool) Schema() json.RawMessage {
	// Get the schema from the imported agent
	registry := t.manager.GetRegistry()
	if registry == nil {
		return json.RawMessage(`{"type": "object", "properties": {}}`)
	}

	agent, ok := registry.GetAgent(t.importPath)
	if !ok {
		return json.RawMessage(`{"type": "object", "properties": {}}`)
	}

	if prog, ok := agent.(*Program); ok {
		return prog.InputSchema
	}

	return json.RawMessage(`{"type": "object", "properties": {}}`)
}

// BatchCallResult represents the result of a batch agent call.
type BatchCallResult struct {
	ImportPath string          `json:"import_path"`
	Input      json.RawMessage `json:"input"`
	Output     json.RawMessage `json:"output"`
	Error      error           `json:"error,omitempty"`
	Duration   time.Duration   `json:"duration"`
	Metrics    Metrics         `json:"metrics"`
}

// BatchCall calls multiple agents in parallel.
func (cm *CallManager) BatchCall(ctx context.Context, calls map[string]json.RawMessage) map[string]*BatchCallResult {
	results := make(map[string]*BatchCallResult)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for importPath, input := range calls {
		wg.Add(1)
		go func(importPath string, input json.RawMessage) {
			defer wg.Done()

			startTime := time.Now()
			result := &BatchCallResult{
				ImportPath: importPath,
				Input:      input,
			}

			callResult, err := cm.Call(ctx, importPath, input)
			result.Duration = time.Since(startTime)

			if err != nil {
				result.Error = err
			} else {
				result.Output = callResult.Output
				result.Metrics = callResult.Metrics
			}

			mu.Lock()
			results[importPath] = result
			mu.Unlock()
		}(importPath, input)
	}

	wg.Wait()
	return results
}

// StreamCall executes an agent call and streams updates to a channel.
func (cm *CallManager) StreamCall(ctx context.Context, importPath string, input json.RawMessage, updates chan<- CallUpdate) (*AgentCallResult, error) {
	if updates != nil {
		defer close(updates)
	}

	callID := generateCallID()

	if updates != nil {
		updates <- CallUpdate{
			CallID:     callID,
			ImportPath: importPath,
			Status:     "starting",
			Timestamp:  time.Now(),
		}
	}

	// Execute the call
	result, err := cm.Call(ctx, importPath, input)

	if updates != nil {
		status := "completed"
		if err != nil {
			status = "failed"
		}

		updates <- CallUpdate{
			CallID:     callID,
			ImportPath: importPath,
			Status:     status,
			Timestamp:  time.Now(),
			Result:     result,
			Error:      err,
		}
	}

	return result, err
}

// CallUpdate represents an update during a streaming call.
type CallUpdate struct {
	CallID     string           `json:"call_id"`
	ImportPath string           `json:"import_path"`
	Status     string           `json:"status"`
	Timestamp  time.Time        `json:"timestamp"`
	Result     *AgentCallResult `json:"result,omitempty"`
	Error      error            `json:"error,omitempty"`
}

// GetRecentErrors returns recent error records from the call history.
func (cm *CallManager) GetRecentErrors(limit int) []CallRecord {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var errors []CallRecord
	for i := len(cm.callHistory) - 1; i >= 0 && len(errors) < limit; i-- {
		record := cm.callHistory[i]
		if record.Error != "" {
			errors = append(errors, record)
		}
	}

	return errors
}

// GetSuccessRate returns the success rate as a percentage.
func (cm *CallManager) GetSuccessRate() float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if len(cm.callHistory) == 0 {
		return 100.0
	}

	successCount := 0
	for _, record := range cm.callHistory {
		if record.Error == "" {
			successCount++
		}
	}

	return float64(successCount) / float64(len(cm.callHistory)) * 100.0
}

// GetSlowestCalls returns the slowest N calls by duration.
func (cm *CallManager) GetSlowestCalls(n int) []CallRecord {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Create a copy and sort by duration
	copies := make([]CallRecord, len(cm.callHistory))
	copy(copies, cm.callHistory)

	// Simple sort (for small lists, this is fine)
	for i := 0; i < len(copies)-1; i++ {
		for j := i + 1; j < len(copies); j++ {
			if copies[i].Duration < copies[j].Duration {
				copies[i], copies[j] = copies[j], copies[i]
			}
		}
	}

	if len(copies) > n {
		copies = copies[:n]
	}

	return copies
}
