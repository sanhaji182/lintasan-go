package expprovider

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/sanhaji182/lintasan-go/internal/experimental"
	"github.com/sanhaji182/lintasan-go/internal/provider"
)

// e2e_test.go — end-to-end proof that the G1 adapter (ACPProvider.Run) actually
// drives the ACP loop through a REAL E1 subprocess, with credential injection
// applied. The test binary re-execs itself as a scripted ACP agent (the same
// pattern the experimental package uses). This closes the loop the M5 principle
// requires: a prompt turn that triggers a tool call and COMPLETES, with the
// toolCallId round-tripping verbatim.

const childModeEnv = "LINTASAN_EXPPROV_TEST_CHILD"

// TestMain dispatches to the scripted ACP agent when launched as a child;
// otherwise it runs the normal suite.
func TestMain(m *testing.M) {
	switch os.Getenv(childModeEnv) {
	case "acp-agent":
		runScriptedACPAgent()
		return
	case "acp-agent-assert-secret":
		// Same agent, but FIRST assert the injected secret is visible to the
		// child (proves credential injection reached the process env) and that
		// no foreign secret leaked in.
		if os.Getenv("OPENAI_API_KEY") != "sk-e2e-injected" {
			os.Exit(11) // injected secret missing → contained as child-exit error
		}
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			os.Exit(12) // foreign secret leaked → fail
		}
		runScriptedACPAgent()
		return
	}
	os.Exit(m.Run())
}

// runScriptedACPAgent speaks the ACP JSON-RPC lifecycle over stdio, one line in
// → one line out, including a tool-call round-trip that echoes the toolCallId.
func runScriptedACPAgent() {
	r := bufio.NewReader(os.Stdin)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	writeLine := func(v any) {
		b, _ := json.Marshal(v)
		w.Write(b)
		w.WriteByte('\n')
		w.Flush()
	}
	for {
		line, err := r.ReadString('\n')
		if len(line) > 0 {
			var msg struct {
				ID     any    `json:"id"`
				Method string `json:"method"`
			}
			json.Unmarshal([]byte(line), &msg)
			switch msg.Method {
			case "initialize":
				writeLine(map[string]any{"jsonrpc": "2.0", "id": msg.ID,
					"result": map[string]any{"protocolVersion": "0.1", "agentInfo": map[string]any{"name": "scripted"}}})
			case "session/new":
				writeLine(map[string]any{"jsonrpc": "2.0", "id": msg.ID,
					"result": map[string]any{"sessionId": "sess-e2e"}})
			case "session/prompt":
				// Emit an agent→host tool-call, then read the host's result,
				// then emit the terminal prompt result echoing the toolCallId.
				writeLine(map[string]any{"jsonrpc": "2.0", "id": "agent-req-1",
					"method": "session/requestToolCall",
					"params": map[string]any{"toolCallId": "tc-e2e-1", "name": "ping",
						"arguments": json.RawMessage(`{}`)}})
				resultLine, _ := r.ReadString('\n')
				var tr struct {
					Result struct {
						ToolCallID string `json:"toolCallId"`
					} `json:"result"`
				}
				json.Unmarshal([]byte(resultLine), &tr)
				writeLine(map[string]any{"jsonrpc": "2.0", "id": "prompt-done",
					"result": map[string]any{"stopReason": "end_turn",
						"content": json.RawMessage(`{"echoedToolCallId":"` + tr.Result.ToolCallID + `"}`)}})
			case "shutdown":
				writeLine(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": map[string]any{}})
			}
		}
		if err != nil {
			return
		}
	}
}

// e2eSpec returns a spec that re-execs THIS test binary as the scripted agent.
func e2eSpec(t *testing.T, mode string) LaunchSpec {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return LaunchSpec{
		Name:       "codex",
		Protocol:   ProtocolACP,
		Path:       exe,
		Args:       nil,
		AuthMode:   AuthAPIKey,
		AuthEnvVar: "OPENAI_API_KEY",
		// BaseEnv re-execs the agent mode; the secret is injected by G4 on top.
		BaseEnv:        append(os.Environ(), childModeEnv+"="+mode),
		RequestTimeout: 5 * time.Second,
		StopTimeout:    2 * time.Second,
	}
}

// TestE2E_AgentRun_ClosesToolLoopWithVerbatimID is the acceptance-shaped proof:
// Run launches the agent, drives the full lifecycle, the host tool handler
// fires, and the toolCallId round-trips verbatim — the loop closes.
func TestE2E_AgentRun_ClosesToolLoopWithVerbatimID(t *testing.T) {
	src := CredentialSourceFunc(func(p string) (string, bool) {
		if p == "codex" {
			return "sk-e2e-injected", true
		}
		return "", false
	})
	p := NewACPProvider(e2eSpec(t, "acp-agent"), provider.NewCapabilitySet(provider.CapCoding), NewInjector(src))
	defer p.StopAgent()

	var sawToolCall string
	turn := AgentTurn{
		Prompt: map[string]any{"text": "ping please"},
		OnTool: func(ctx context.Context, call experimental.ToolCall) (experimental.ToolResult, error) {
			sawToolCall = call.ToolCallID
			// Host returns a result; the broker copies ToolCallID through verbatim.
			return experimental.ToolResult{Content: map[string]any{"pong": true}}, nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	res, err := p.Run(ctx, turn)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sawToolCall != "tc-e2e-1" {
		t.Fatalf("host saw toolCallId %q, want tc-e2e-1", sawToolCall)
	}
	// The agent echoed the toolCallId it received in the terminal result —
	// proving the id survived host→agent verbatim (the M3/ACP fidelity bar).
	var content struct {
		Echoed string `json:"echoedToolCallId"`
	}
	json.Unmarshal(res.Content, &content)
	if content.Echoed != "tc-e2e-1" {
		t.Fatalf("agent echoed toolCallId %q, want tc-e2e-1 — fidelity broken", content.Echoed)
	}
	if res.StopReason != "end_turn" {
		t.Fatalf("stopReason = %q, want end_turn", res.StopReason)
	}
}

// TestE2E_CredentialInjectionReachesChild proves G4: the injected secret is in
// the child's process env, and a foreign provider's secret is NOT. The child
// exits non-zero if either condition fails, which Run surfaces as a contained
// error (so a PASS here means both assertions held inside the real subprocess).
func TestE2E_CredentialInjectionReachesChild(t *testing.T) {
	src := CredentialSourceFunc(func(p string) (string, bool) {
		if p == "codex" {
			return "sk-e2e-injected", true
		}
		return "", false
	})
	p := NewACPProvider(e2eSpec(t, "acp-agent-assert-secret"), nil, NewInjector(src))
	defer p.StopAgent()

	turn := AgentTurn{
		Prompt: map[string]any{"text": "ping"},
		OnTool: func(ctx context.Context, call experimental.ToolCall) (experimental.ToolResult, error) {
			return experimental.ToolResult{Content: "ok"}, nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if _, err := p.Run(ctx, turn); err != nil {
		t.Fatalf("Run failed — credential injection assertion likely failed inside child: %v", err)
	}
}
