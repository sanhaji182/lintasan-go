package experimental

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// acp_test.go — ACP integration layer tests (Foundation Phase 4).
//
// These drive the broker against a REAL scripted ACP agent subprocess (the
// "acp-agent" mode in TestMain), exercising the full lifecycle over the E1
// transport: initialize → session/new → session/prompt (with a tool round-trip)
// → shutdown. The load-bearing assertion is IDENTIFIER FIDELITY: the toolCallId
// the agent emits must round-trip verbatim through the host handler and back,
// proving the loop is wired correctly end-to-end (the M3 call_id lesson applied
// to ACP).

func newACPTestClient(t *testing.T) *ACPClient {
	t.Helper()
	cfg := childConfig(t, "acp-agent", 5*time.Second, 2*time.Second)
	return NewACPClient(New(cfg))
}

// TestACP_FullLifecycleWithToolRoundTrip is the Phase-4 acceptance-shaped test:
// the whole loop completes and the toolCallId survives verbatim.
func TestACP_FullLifecycleWithToolRoundTrip(t *testing.T) {
	c := newACPTestClient(t)
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// initialize
	init, err := c.Initialize(ctx, InitializeParams{ProtocolVersion: "1.0",
		ClientInfo: map[string]any{"name": "lintasan"}})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if init.ProtocolVersion != "1.0" {
		t.Fatalf("unexpected protocol version: %q", init.ProtocolVersion)
	}

	// session/new
	sid, err := c.NewSession(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}
	if sid != "sess-42" {
		t.Fatalf("unexpected session id: %q", sid)
	}

	// session/prompt with a tool handler. The agent emits a tool-call
	// (toolCallId="tc-1"); the handler must receive that verbatim and its result
	// must round-trip back so the agent echoes it in the final prompt result.
	var seenToolCallID string
	var seenName string
	handler := func(_ context.Context, call ToolCall) (ToolResult, error) {
		seenToolCallID = call.ToolCallID
		seenName = call.Name
		return ToolResult{Content: "2026-05-31T00:00:00Z"}, nil // ToolCallID set by broker
	}

	res, err := c.Prompt(ctx, PromptParams{SessionID: sid, Prompt: "what time is it?"}, handler)
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	// The handler must have seen the agent's verbatim toolCallId + name.
	if seenToolCallID != "tc-1" {
		t.Fatalf("handler saw wrong toolCallId: got %q want tc-1", seenToolCallID)
	}
	if seenName != "get_time" {
		t.Fatalf("handler saw wrong tool name: got %q want get_time", seenName)
	}

	// The agent echoed back the toolCallId it received in its tool result — this
	// is the END-TO-END fidelity proof: tc-1 made the full round trip.
	var content struct {
		EchoedToolCallID string `json:"echoedToolCallId"`
	}
	if err := json.Unmarshal(res.Content, &content); err != nil {
		t.Fatalf("decode prompt result content: %v (raw=%s)", err, res.Content)
	}
	if content.EchoedToolCallID != "tc-1" {
		t.Fatalf("IDENTIFIER FIDELITY BROKEN: agent echoed %q, want tc-1", content.EchoedToolCallID)
	}
	if res.StopReason != "end_turn" {
		t.Fatalf("unexpected stop reason: %q", res.StopReason)
	}

	// shutdown is graceful.
	if err := c.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

// TestACP_BrokerEnforcesToolCallIDVerbatim proves the broker overrides whatever
// ToolCallID a handler sets, forcing the originating id. Even a buggy handler
// cannot break identifier fidelity.
func TestACP_BrokerEnforcesToolCallIDVerbatim(t *testing.T) {
	c := newACPTestClient(t)
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer c.Close()
	ctx := context.Background()
	if _, err := c.Initialize(ctx, InitializeParams{ProtocolVersion: "1.0"}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	sid, err := c.NewSession(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	// A misbehaving handler that tries to set a WRONG toolCallId. The broker must
	// overwrite it with the originating id, so the agent still echoes "tc-1".
	handler := func(_ context.Context, call ToolCall) (ToolResult, error) {
		return ToolResult{ToolCallID: "WRONG-ID", Content: "x"}, nil
	}
	res, err := c.Prompt(ctx, PromptParams{SessionID: sid, Prompt: "go"}, handler)
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}
	var content struct {
		EchoedToolCallID string `json:"echoedToolCallId"`
	}
	json.Unmarshal(res.Content, &content)
	if content.EchoedToolCallID != "tc-1" {
		t.Fatalf("broker failed to enforce verbatim toolCallId: agent saw %q, want tc-1", content.EchoedToolCallID)
	}
}

// TestACP_NilToolHandlerIsContained proves that with no tool handler, an
// agent tool request is answered with an error ToolResult (so the agent can
// terminate) rather than hanging the broker. The fake agent still completes.
func TestACP_NilToolHandlerIsContained(t *testing.T) {
	c := newACPTestClient(t)
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer c.Close()
	ctx := context.Background()
	if _, err := c.Initialize(ctx, InitializeParams{ProtocolVersion: "1.0"}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	sid, _ := c.NewSession(ctx, map[string]any{})

	// nil handler → broker replies with an error tool result; agent still echoes
	// tc-1 (it just receives an error payload). The loop must NOT hang.
	res, err := c.Prompt(ctx, PromptParams{SessionID: sid, Prompt: "go"}, nil)
	if err != nil {
		t.Fatalf("prompt with nil handler should still complete: %v", err)
	}
	var content struct {
		EchoedToolCallID string `json:"echoedToolCallId"`
	}
	json.Unmarshal(res.Content, &content)
	if content.EchoedToolCallID != "tc-1" {
		t.Fatalf("nil-handler path broke the loop, echoed %q", content.EchoedToolCallID)
	}
}

// TestACP_ClosedClientRejects proves operations on a closed client are contained.
func TestACP_ClosedClientRejects(t *testing.T) {
	c := newACPTestClient(t)
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := c.Initialize(context.Background(), InitializeParams{}); err == nil {
		t.Fatal("expected error from a closed client")
	}
}

// TestACP_UnknownMethodReturnsRPCError proves a JSON-RPC error from the agent is
// surfaced as a typed error (not swallowed). We send shutdown-like unknown via a
// direct call to a method the fake agent rejects.
func TestACP_UnknownMethodReturnsRPCError(t *testing.T) {
	c := newACPTestClient(t)
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer c.Close()
	// "session/new" is known; "bogus/method" is not → agent returns rpc error.
	err := c.call(context.Background(), "bogus/method", nil, nil)
	if err == nil {
		t.Fatal("expected an rpc error for an unknown method")
	}
	if !strings.Contains(err.Error(), "method not found") {
		t.Fatalf("expected method-not-found rpc error, got %v", err)
	}
}
