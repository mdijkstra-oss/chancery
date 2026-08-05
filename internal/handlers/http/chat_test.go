package http

import (
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mdijkstra-oss/chancery/internal/auth"
	"github.com/mdijkstra-oss/chancery/internal/prompts"
	"github.com/mdijkstra-oss/chancery/internal/ratelimit"
	"github.com/mdijkstra-oss/chancery/internal/responses"

	"github.com/go-chi/chi/v5"
)

const completedStream = "event: response.completed\ndata: {\"type\":\"response.completed\"}\n\n"

// streamReply is the shortest answer a Responses backend can give: one event saying
// the response is done.
var streamReply = backendReply{http.StatusOK, "text/event-stream", completedStream}

// forwarded is the request the backend saw. Everything asserted about what chancery
// sends is read off a real backend rather than off the handler's internals.
type forwarded struct {
	path   string
	header http.Header
	body   []byte
}

type backendReply struct {
	status      int
	contentType string
	body        string
}

// chatConfigWith is the directory the registry is loaded from: three routes over three
// aliases that between them set every field an agent can contribute to a body, plus
// whatever extra a test needs — a tools/ tree, say, which every other test is better
// off without.
func chatConfigWith(t *testing.T, extra map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"models.yaml": "models:\n" +
			"  bare:\n    model: openai/upstream-bare\n" +
			"  fast:\n    model: openai/upstream-fast\n    reasoning_effort: low\n" +
			"  deep:\n    model: openai/upstream-deep\n    reasoning_effort: high\n" +
			"    verbosity: low\n    service_tier: priority\n    max_tokens: 4096\n",
		"bare.md":  "---\ndescription: bare agent\nmodel: bare\n---\nYou are bare.",
		"plain.md": "---\ndescription: plain agent\nmodel: fast\n---\nYou are plain.",
		"named/index.md": "---\ndescription: named agent\nmodels:\n" +
			"  quick:\n    model: fast\n  thorough:\n    model: deep\ndefault: quick\n---\n" +
			"You are named.",
	}
	maps.Copy(files, extra)
	for path, content := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("create directory: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return root
}

func chatRegistry(t *testing.T) prompts.Registry {
	return chatRegistryWith(t, nil)
}

func chatRegistryWith(t *testing.T, extra map[string]string) prompts.Registry {
	t.Helper()
	registry, report := prompts.Load(chatConfigWith(t, extra))
	if report.HasErrors() {
		t.Fatalf("load config: %v", report.Diagnostics)
	}
	return registry
}

// chatBackend answers every request with reply and records what it was sent.
func chatBackend(t *testing.T, reply backendReply) (string, *forwarded) {
	t.Helper()
	seen := &forwarded{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read forwarded body: %v", err)
		}
		seen.path = r.URL.Path
		seen.header = r.Header.Clone()
		seen.body = body
		w.Header().Set("Content-Type", reply.contentType)
		w.WriteHeader(reply.status)
		_, _ = w.Write([]byte(reply.body))
	}))
	t.Cleanup(server.Close)
	return server.URL, seen
}

// chatRouter wires the real middleware chain, so the request ID, the JWT subject and
// the header allowlist reach the handler the way they do when serving.
func chatRouter(
	t *testing.T,
	backendURL string,
	validator auth.Validator,
	headers []string,
) *chi.Mux {
	return chatRouterWith(t, backendURL, validator, headers, nil)
}

func chatRouterWith(
	t *testing.T,
	backendURL string,
	validator auth.Validator,
	headers []string,
	extra map[string]string,
) *chi.Mux {
	t.Helper()
	client, err := responses.NewClient(responses.Config{BaseURL: backendURL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	router := chi.NewRouter()
	handler := NewChatHandler(chatRegistryWith(t, extra), client, ratelimit.NewLimiter(), headers)
	SetupRoutes(router, handler, JWTAuthentication(validator), []string{"*"}, headers)
	return router
}

func disabledValidator(t *testing.T) auth.Validator {
	t.Helper()
	validator, err := auth.NewValidator(context.Background(), auth.Config{})
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	return validator
}

func decodeBody(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	fields := map[string]any{}
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("decode body %s: %v", raw, err)
	}
	return fields
}

// The done criterion in its own words: with no overrides anywhere, the two bodies
// differ by model and instructions and by nothing else.
func TestChatForwardsBodyUntouchedApartFromModelAndInstructions(t *testing.T) {
	backendURL, seen := chatBackend(t, streamReply)
	router := chatRouter(t, backendURL, disabledValidator(t), nil)

	sent := `{"input":[{"role":"user","content":[{"type":"input_text","text":"hi"}]}],` +
		`"stream":true,"store":false,"metadata":{"run":"7"},"tools":[{"type":"function",` +
		`"name":"lookup","parameters":{"type":"object","properties":{"q":{"type":"string"}}}}]}`
	request := httptest.NewRequest(http.MethodPost, "/bare", strings.NewReader(sent))
	router.ServeHTTP(httptest.NewRecorder(), request)

	if seen.path != "/responses" {
		t.Fatalf("backend path = %q, want /responses", seen.path)
	}
	got := decodeBody(t, seen.body)
	for _, key := range []string{"model", "instructions"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("forwarded body has no %q: %s", key, seen.body)
		}
		delete(got, key)
	}
	if !reflect.DeepEqual(got, decodeBody(t, []byte(sent))) {
		t.Fatalf("forwarded body differs beyond model and instructions:\ngot  %s\nsent %s",
			seen.body, sent)
	}
}

func TestChatComposesAgentFields(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{{
		name: "a single-model route writes its alias",
		path: "/plain",
		want: `{"input":[],"model":"openai/upstream-fast","instructions":"You are plain.",
			"reasoning":{"effort":"low"}}`,
	}, {
		name: "a named route defaults to the named entry",
		path: "/named",
		want: `{"input":[],"model":"openai/upstream-fast","instructions":"You are named.",
			"reasoning":{"effort":"low"}}`,
	}, {
		name: "a named route reached by name writes that entry",
		path: "/named.thorough",
		want: `{"input":[],"model":"openai/upstream-deep","instructions":"You are named.",
			"reasoning":{"effort":"high"},"text":{"verbosity":"low"},
			"service_tier":"priority","max_output_tokens":4096}`,
	}, {
		name: "a query string names nothing the agent resolves from",
		path: "/plain?tool_choice=required&reasoning_summary=concise",
		want: `{"input":[],"model":"openai/upstream-fast","instructions":"You are plain.",
			"reasoning":{"effort":"low"}}`,
	}}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			backendURL, seen := chatBackend(t, streamReply)
			router := chatRouter(t, backendURL, disabledValidator(t), nil)

			request := httptest.NewRequest(http.MethodPost, testCase.path,
				strings.NewReader(`{"input":[]}`))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", response.Code, response.Body)
			}
			got := decodeBody(t, seen.body)
			want := decodeBody(t, []byte(testCase.want))
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("forwarded body:\ngot  %s\nwant %s", seen.body, testCase.want)
			}
		})
	}
}

// A tool prompt is selected by the name the request gives the tool, so the file under
// tools/ reaches the instructions only when the caller can actually call the thing it
// describes. The tools array itself travels untouched.
func TestChatComposesToolPrompts(t *testing.T) {
	toolFiles := map[string]string{
		"tools/preamble.md":            "You have tools.",
		"tools/search.web_search.md":   "Search sparingly.",
		"tools/shell/run.run_shell.md": "Shell is dangerous.",
	}
	cases := []struct {
		name  string
		tools string
		want  string
	}{{
		name:  "a request naming no tool gets the unconditional prompt only",
		tools: `[]`,
		want:  "You are plain.\n\nYou have tools.",
	}, {
		name:  "a named tool pulls its own prompt in",
		tools: `[{"type":"function","name":"web_search"}]`,
		want:  "You are plain.\n\nYou have tools.\n\nSearch sparingly.",
	}, {
		name:  "a tool prompt nested under a directory is reached the same way",
		tools: `[{"type":"function","name":"run_shell"}]`,
		want:  "You are plain.\n\nYou have tools.\n\nShell is dangerous.",
	}, {
		name:  "a built-in tool carrying no name asks for no prompt",
		tools: `[{"type":"web_search_preview"}]`,
		want:  "You are plain.\n\nYou have tools.",
	}}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			backendURL, seen := chatBackend(t, streamReply)
			router := chatRouterWith(t, backendURL, disabledValidator(t), nil, toolFiles)

			body := `{"input":[],"tools":` + testCase.tools + `}`
			request := httptest.NewRequest(http.MethodPost, "/plain", strings.NewReader(body))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", response.Code, response.Body)
			}
			forwardedBody := decodeBody(t, seen.body)
			if got, _ := forwardedBody["instructions"].(string); got != testCase.want {
				t.Errorf("instructions =\n%q\nwant\n%q", got, testCase.want)
			}
			gotTools := decodeBody(t, []byte(`{"tools":`+testCase.tools+`}`))["tools"]
			if !reflect.DeepEqual(forwardedBody["tools"], gotTools) {
				t.Errorf("tools = %v, want %v", forwardedBody["tools"], gotTools)
			}
		})
	}
}

// The caller's own instructions survive with the agent's in front of them.
func TestChatPrependsAgentInstructions(t *testing.T) {
	backendURL, seen := chatBackend(t, streamReply)
	router := chatRouter(t, backendURL, disabledValidator(t), nil)

	request := httptest.NewRequest(http.MethodPost, "/plain",
		strings.NewReader(`{"input":[],"instructions":"Answer in French."}`))
	router.ServeHTTP(httptest.NewRecorder(), request)

	got, _ := decodeBody(t, seen.body)["instructions"].(string)
	if want := "You are plain.\n\nAnswer in French."; got != want {
		t.Fatalf("instructions = %q, want %q", got, want)
	}
}

func TestChatOutboundIdentity(t *testing.T) {
	key := middlewareRSAKey(t)
	validator := middlewareValidator(t, key)
	token := middlewareToken(t, key, time.Now().Add(time.Hour), "user-7")

	backendURL, seen := chatBackend(t, streamReply)
	router := chatRouter(t, backendURL, validator, []string{"X-Session-ID", "X-Project-ID"})

	request := httptest.NewRequest(http.MethodPost, "/named.thorough",
		strings.NewReader(`{"input":[]}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-Session-ID", "session-1")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body)
	}
	cases := []struct {
		header string
		want   string
	}{
		{header: "X-Agent", want: "named.thorough"},
		{header: "X-Subject", want: "user-7"},
		{header: "X-Session-Id", want: "session-1"},
		{header: "X-Project-Id", want: ""},
		{header: "Authorization", want: ""},
	}
	for _, testCase := range cases {
		t.Run(testCase.header, func(t *testing.T) {
			if got := seen.header.Get(testCase.header); got != testCase.want {
				t.Fatalf("%s = %q, want %q", testCase.header, got, testCase.want)
			}
		})
	}
	if seen.header.Get("X-Request-Id") == "" {
		t.Fatal("X-Request-Id is empty")
	}
}

func TestChatRelaysBackendStream(t *testing.T) {
	stream := "event: response.created\ndata: {\"type\":\"response.created\"}\n\n" +
		"event: response.output_text.delta\ndata: {\"delta\":\"hi\"}\n\n" + completedStream
	backendURL, _ := chatBackend(t, backendReply{http.StatusOK, "text/event-stream", stream})
	router := chatRouter(t, backendURL, disabledValidator(t), nil)

	request := httptest.NewRequest(http.MethodPost, "/plain", strings.NewReader(`{"input":[]}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %q", got)
	}
	if got := response.Body.String(); got != stream {
		t.Fatalf("relayed %q, want %q", got, stream)
	}
}

func TestChatRejectsUnknownAgent(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{name: "no such route", path: "/missing"},
		{name: "no such named model", path: "/named.missing"},
		{name: "a named model on a single-model route", path: "/plain.fast"},
		{name: "a fragment is not a route", path: "/models.yaml"},
	}

	backendURL, seen := chatBackend(t, streamReply)
	router := chatRouter(t, backendURL, disabledValidator(t), nil)
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, testCase.path,
				strings.NewReader(`{"input":[]}`))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
			if seen.body != nil {
				t.Fatalf("an unresolved route reached the backend: %s", seen.body)
			}
		})
	}
}

func TestChatRejectsUncomposableBody(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "not JSON", body: "not json"},
		{name: "empty", body: ""},
		{name: "instructions is not a string", body: `{"input":[],"instructions":42}`},
		{name: "reasoning is not an object", body: `{"input":[],"reasoning":"high"}`},
	}

	backendURL, seen := chatBackend(t, streamReply)
	router := chatRouter(t, backendURL, disabledValidator(t), nil)
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/plain", strings.NewReader(testCase.body))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", response.Code)
			}
			if seen.body != nil {
				t.Fatalf("an uncomposable body reached the backend: %s", seen.body)
			}
		})
	}
}

func TestChatRejectsOversizedBody(t *testing.T) {
	backendURL, seen := chatBackend(t, streamReply)
	router := chatRouter(t, backendURL, disabledValidator(t), nil)

	oversized := `{"input":[{"role":"user","content":"` +
		strings.Repeat("x", maxRequestBodyBytes+1) + `"}]}`
	request := httptest.NewRequest(http.MethodPost, "/plain", strings.NewReader(oversized))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
	if seen.body != nil {
		t.Fatalf("an oversized body reached the backend: %s", seen.body)
	}
}

// Where chancery writes the body itself it writes the status and nothing else: the URL
// it forwards to and the address that resolved to are the deployment's business.
func TestChatBackendFailureStatus(t *testing.T) {
	cases := []struct {
		name       string
		reply      backendReply
		unreached  bool
		wantStatus int
		wantBody   string
	}{{
		name:       "the backend's own error travels with its status",
		reply:      backendReply{http.StatusBadRequest, "application/json", `{"error":"bad effort"}`},
		wantStatus: http.StatusBadRequest,
		wantBody:   `{"error":"bad effort"}`,
	}, {
		name:       "a backend fault travels with its status",
		reply:      backendReply{http.StatusInternalServerError, "application/json", `{"error":"boom"}`},
		wantStatus: http.StatusInternalServerError,
		wantBody:   `{"error":"boom"}`,
	}, {
		name:       "a backend that never answered is unavailable",
		unreached:  true,
		wantStatus: http.StatusServiceUnavailable,
		wantBody:   "Service Unavailable\n",
	}}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			backendURL, _ := chatBackend(t, testCase.reply)
			if testCase.unreached {
				closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
				backendURL = closed.URL
				closed.Close()
			}
			router := chatRouter(t, backendURL, disabledValidator(t), nil)
			request := httptest.NewRequest(http.MethodPost, "/plain", strings.NewReader(`{"input":[]}`))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, testCase.wantStatus)
			}
			if response.Body.String() != testCase.wantBody {
				t.Fatalf("body = %q, want %q", response.Body.String(), testCase.wantBody)
			}
			host := strings.TrimPrefix(backendURL, "http://")
			if strings.Contains(response.Body.String(), host) {
				t.Fatalf("body names the backend %q: %q", host, response.Body.String())
			}
		})
	}
}

// Every route the registry holds is reachable under the wildcard, including the ones
// a named model adds beside their agent's own path.
func TestChatServesEveryConfiguredRoute(t *testing.T) {
	registry := chatRegistry(t)
	routes := slices.Collect(maps.Keys(registry.Agents))
	for path, named := range registry.NamedConfigs {
		for name := range named {
			routes = append(routes, path+"."+name)
		}
	}
	slices.Sort(routes)
	want := []string{"bare", "named", "named.quick", "named.thorough", "plain"}
	if !reflect.DeepEqual(routes, want) {
		t.Fatalf("routes = %v, want %v", routes, want)
	}

	backendURL, _ := chatBackend(t, streamReply)
	router := chatRouter(t, backendURL, disabledValidator(t), nil)
	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/"+route, strings.NewReader(`{"input":[]}`))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", response.Code, response.Body)
			}
		})
	}
}
