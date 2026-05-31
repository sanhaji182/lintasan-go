package experimental

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// acp.go — ACP (Agent Client Protocol) integration layer (Foundation Phase 4).
//
// ACP lets one host drive many agents over a single official protocol: JSON-RPC
// 2.0 over the agent's stdio. Lintasan is the ACP CLIENT/HOST; an agent CLI
// (launched as an E1 Subprocess) is the ACP AGENT. This is "Shape 2" — official
// orchestration of an official CLI/SDK, ZERO reverse-engineering.
//
// This layer is built ON TOP of the Phase-3 E1 byte transport (Subprocess): the
// Subprocess gives us isolation (timeout/crash/panic containment); this file
// adds the JSON-RPC framing + the ACP lifecycle broker. The JSON-RPC envelope
// matches the repo's existing MCP convention (internal/mcp) for consistency.
//
// SCOPE LOCK (Phase 4): provider-agnostic protocol broker ONLY. It brokers the
// lifecycle (initialize → session/new → session/prompt → tool round-trip →
// shutdown) and carries identifiers VERBATIM (the M3 call_id-fidelity lesson
// applies to ACP toolCallId too). It implements NO specific provider (Codex,
// Claude Code, Gemini CLI, Copilot are later, separately-approved onboarding),
// performs NO actual tool execution (the host decides; this transports the
// request/result), and is NOT wired into the production router (the membrane
// keeps it off the Official path).

// --- JSON-RPC 2.0 envelope (matches internal/mcp conventions) ----------------

// jsonrpcRequest is a JSON-RPC 2.0 request/notification sent to the agent.
type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"` // omitted → notification
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// jsonrpcError is the JSON-RPC 2.0 error object.
type jsonrpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *jsonrpcError) Error() string {
	return fmt.Sprintf("acp: rpc error %d: %s", e.Code, e.Message)
}

// jsonrpcResponse is a JSON-RPC 2.0 response from the agent.
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

// --- ACP message payloads (the subset the broker needs) ----------------------

// InitializeParams negotiates protocol version + client capabilities.
type InitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	ClientInfo      map[string]any `json:"clientInfo,omitempty"`
	Capabilities    map[string]any `json:"capabilities,omitempty"`
}

// InitializeResult is the agent's handshake reply (version + its capabilities).
type InitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	AgentInfo       map[string]any  `json:"agentInfo,omitempty"`
	Capabilities    json.RawMessage `json:"capabilities,omitempty"`
}

// NewSessionResult carries the session id the agent allocated.
type NewSessionResult struct {
	SessionID string `json:"sessionId"`
}

// PromptParams sends one turn to a session.
type PromptParams struct {
	SessionID string `json:"sessionId"`
	Prompt    any    `json:"prompt"`
}

// ToolCall is an agent-initiated tool request surfaced to the host. The host
// executes it (or declines) and returns a ToolResult with the SAME ToolCallID.
type ToolCall struct {
	ToolCallID string          `json:"toolCallId"`
	Name       string          `json:"name"`
	Arguments  json.RawMessage `json:"arguments,omitempty"`
}

// ToolResult is the host's reply to a ToolCall. ToolCallID MUST equal the
// originating ToolCall.ToolCallID VERBATIM — identifier fidelity is the loop's
// make-or-break property (the M3 call_id lesson, applied to ACP).
type ToolResult struct {
	ToolCallID string `json:"toolCallId"`
	Content    any    `json:"content"`
	IsError    bool   `json:"isError,omitempty"`
}

// PromptResult is the terminal result of a prompt turn (after any tool calls).
type PromptResult struct {
	StopReason string          `json:"stopReason,omitempty"`
	Content    json.RawMessage `json:"content,omitempty"`
}

var (
	// ErrACPClosed is returned when an operation is attempted on a closed client.
	ErrACPClosed = errors.New("acp: client closed")
	// ErrACPProtocol indicates a malformed/unexpected message from the agent.
	ErrACPProtocol = errors.New("acp: protocol error")
)

// ToolHandler is the host-side callback invoked when the agent requests a tool.
// The host decides what to do and returns a ToolResult (the broker copies the
// ToolCallID through verbatim — handlers MUST NOT change it). Returning an error
// is surfaced to the agent as an error ToolResult. THE BROKER NEVER EXECUTES A
// TOOL ITSELF — that is the host's responsibility (and a later provider's).
type ToolHandler func(ctx context.Context, call ToolCall) (ToolResult, error)

// ACPClient drives an ACP agent over an E1 Subprocess. It is the protocol broker
// only: it frames JSON-RPC, sequences the lifecycle, and round-trips tool calls
// to a host-supplied ToolHandler. It owns request-id allocation.
type ACPClient struct {
	proc *Subprocess

	mu     sync.Mutex
	nextID int
	closed bool
}

// NewACPClient wraps a (not-yet-started) Subprocess as an ACP client. Start the
// underlying process via the returned client's Start.
func NewACPClient(proc *Subprocess) *ACPClient {
	return &ACPClient{proc: proc, nextID: 1}
}

// Start launches the underlying agent subprocess.
func (c *ACPClient) Start(ctx context.Context) error {
	if c.proc == nil {
		return errors.New("acp: nil subprocess")
	}
	return c.proc.Start(ctx)
}

// Close shuts the agent down (graceful → force-kill via the E1 harness).
func (c *ACPClient) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	return c.proc.Stop()
}

// allocID returns the next JSON-RPC request id.
func (c *ACPClient) allocID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.nextID
	c.nextID++
	return id
}

// call sends a JSON-RPC request and reads exactly one response, decoding Result
// into out. It does NOT handle interleaved server→client requests; for the
// prompt turn (which CAN interleave tool calls) use Prompt, which loops.
func (c *ACPClient) call(ctx context.Context, method string, params any, out any) error {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return ErrACPClosed
	}

	id := c.allocID()
	reqBytes, err := json.Marshal(jsonrpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params})
	if err != nil {
		return fmt.Errorf("acp: marshal %s: %w", method, err)
	}
	respBytes, err := c.proc.Request(ctx, reqBytes)
	if err != nil {
		return err // already a contained E1 error (timeout/exit/etc.)
	}
	var resp jsonrpcResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return fmt.Errorf("%w: bad response json: %v", ErrACPProtocol, err)
	}
	if resp.Error != nil {
		return resp.Error
	}
	if out != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, out); err != nil {
			return fmt.Errorf("%w: bad result json: %v", ErrACPProtocol, err)
		}
	}
	return nil
}

// Initialize performs the ACP handshake (protocol version + capabilities).
func (c *ACPClient) Initialize(ctx context.Context, params InitializeParams) (*InitializeResult, error) {
	var res InitializeResult
	if err := c.call(ctx, "initialize", params, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// NewSession opens a new agent session and returns its id.
func (c *ACPClient) NewSession(ctx context.Context, params any) (string, error) {
	var res NewSessionResult
	if err := c.call(ctx, "session/new", params, &res); err != nil {
		return "", err
	}
	if res.SessionID == "" {
		return "", fmt.Errorf("%w: session/new returned empty sessionId", ErrACPProtocol)
	}
	return res.SessionID, nil
}

// Prompt sends one turn and drives the agent loop to a terminal PromptResult.
// The agent may interleave server→client tool-call requests; for each, the
// broker invokes onTool and replies with a ToolResult carrying the SAME
// toolCallId VERBATIM. The loop ends when the agent returns the prompt result
// (a JSON-RPC response to the original prompt id) rather than another tool call.
//
// onTool may be nil; if the agent then requests a tool, the broker replies with
// an error ToolResult ("no tool handler") so the agent can terminate cleanly
// instead of hanging.
//
// IMPORTANT: this brokers the loop; it NEVER executes a tool itself. Tool
// execution semantics are the host's (onTool) responsibility.
func (c *ACPClient) Prompt(ctx context.Context, params PromptParams, onTool ToolHandler) (*PromptResult, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, ErrACPClosed
	}
	c.mu.Unlock()

	promptID := c.allocID()
	reqBytes, err := json.Marshal(jsonrpcRequest{JSONRPC: "2.0", ID: promptID, Method: "session/prompt", Params: params})
	if err != nil {
		return nil, fmt.Errorf("acp: marshal prompt: %w", err)
	}

	// Send the prompt, then read frames until we see the response to promptID.
	respBytes, err := c.proc.Request(ctx, reqBytes)
	if err != nil {
		return nil, err
	}

	for {
		// A frame is either (a) the response to our prompt id, or (b) a
		// server→client tool-call request (has a Method). Decode generically.
		var probe struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      any             `json:"id,omitempty"`
			Method  string          `json:"method,omitempty"`
			Params  json.RawMessage `json:"params,omitempty"`
			Result  json.RawMessage `json:"result,omitempty"`
			Error   *jsonrpcError   `json:"error,omitempty"`
		}
		if err := json.Unmarshal(respBytes, &probe); err != nil {
			return nil, fmt.Errorf("%w: bad frame json: %v", ErrACPProtocol, err)
		}

		// Case A: terminal response to the prompt.
		if probe.Method == "" {
			if probe.Error != nil {
				return nil, probe.Error
			}
			var res PromptResult
			if len(probe.Result) > 0 {
				if err := json.Unmarshal(probe.Result, &res); err != nil {
					return nil, fmt.Errorf("%w: bad prompt result: %v", ErrACPProtocol, err)
				}
			}
			return &res, nil
		}

		// Case B: a server→client tool-call request. Broker it.
		var call ToolCall
		if len(probe.Params) > 0 {
			if err := json.Unmarshal(probe.Params, &call); err != nil {
				return nil, fmt.Errorf("%w: bad tool-call params: %v", ErrACPProtocol, err)
			}
		}

		var result ToolResult
		if onTool == nil {
			result = ToolResult{ToolCallID: call.ToolCallID, IsError: true, Content: "no tool handler registered"}
		} else {
			r, herr := onTool(ctx, call)
			if herr != nil {
				result = ToolResult{ToolCallID: call.ToolCallID, IsError: true, Content: herr.Error()}
			} else {
				result = r
			}
		}
		// IDENTIFIER FIDELITY: the result MUST carry the originating toolCallId
		// verbatim. The broker enforces it regardless of what the handler set.
		result.ToolCallID = call.ToolCallID

		// Reply to the agent's tool-call request (using the agent's request id),
		// then continue the loop reading the next frame.
		replyBytes, merr := json.Marshal(jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      probe.ID,
			Result:  mustJSON(result),
		})
		if merr != nil {
			return nil, fmt.Errorf("acp: marshal tool result: %w", merr)
		}
		respBytes, err = c.proc.Request(ctx, replyBytes)
		if err != nil {
			return nil, err
		}
	}
}

// Shutdown asks the agent to terminate the protocol session gracefully. Errors
// are non-fatal (Close still force-stops the process).
func (c *ACPClient) Shutdown(ctx context.Context) error {
	return c.call(ctx, "shutdown", nil, nil)
}

// mustJSON marshals v to json.RawMessage, returning null on error (never panics
// — a marshal failure becomes a JSON null the agent can handle/ignore).
func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("null")
	}
	return b
}
