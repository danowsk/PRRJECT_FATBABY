package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
)

type Config struct {
	Port            string
	ConversationDir string
	Model           string
	ValidatorModel  string
	APIKey          string
	GitCommit       bool
	RateLimitRPM    int
	MaxToolIters    int
	FatbabyRoot     string
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadConfig() Config {
	gitCommit := true
	if v := os.Getenv("GIT_COMMIT"); v == "false" {
		gitCommit = false
	}
	rpm, _ := strconv.Atoi(envOr("RATE_LIMIT_RPM", "20"))
	if rpm <= 0 {
		rpm = 20
	}
	maxIters, _ := strconv.Atoi(envOr("MAX_TOOL_ITERS", "10"))
	if maxIters <= 0 {
		maxIters = 10
	}
	return Config{
		Port:            envOr("PORT", "8080"),
		ConversationDir: envOr("CONVERSATION_DIR", "./conversations"),
		Model:           envOr("MODEL", "gpt-4o-mini"),
		ValidatorModel:  envOr("VALIDATOR_MODEL", "gpt-4o-mini"),
		APIKey:          os.Getenv("OPENAI_API_KEY"),
		GitCommit:       gitCommit,
		RateLimitRPM:    rpm,
		MaxToolIters:    maxIters,
		FatbabyRoot:     envOr("FATBABY_ROOT", "."),
	}
}

type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  ToolParameters `json:"parameters"`
}

type ToolParameters struct {
	Type       string                    `json:"type"`
	Properties map[string]ToolPropSchema `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

type ToolPropSchema struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolFunc func(args map[string]any) (string, error)

type ToolDispatcher struct {
	defs     []ToolDef
	handlers map[string]ToolFunc
}

func NewToolDispatcher() *ToolDispatcher { return &ToolDispatcher{handlers: map[string]ToolFunc{}} }
func (d *ToolDispatcher) Register(def ToolDef, fn ToolFunc) {
	d.defs = append(d.defs, def)
	d.handlers[def.Name] = fn
}
func (d *ToolDispatcher) Defs() []ToolDef { return d.defs }

func registerGitTools(d *ToolDispatcher, repoDir string) {}

func absStorePath(root, store string) string {
	return filepath.Join(root, "var", store)
}

func openStoreOrMessage(root, storeName string) (*eventstore.FileStore, string, error) {
	dir := absStorePath(root, storeName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, "store not initialised — has the process been run?", nil
	}
	fs, err := eventstore.NewFileStore(dir)
	if err != nil {
		return nil, "", err
	}
	return fs, "", nil
}

func runCommandWithTimeout(args []string, workDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if len(text) > 4000 {
		text = text[:4000]
	}
	return text, err
}

func registerFatbabyTools(d *ToolDispatcher, fatbabyRoot string) {
	storeProp := map[string]ToolPropSchema{"store_name": {Type: "string", Description: `"secwatch"|"prwatch"|"prwatch-body"`}}

	d.Register(ToolDef{Name: "fatbaby_store_status", Description: "Latest sequence and recent records from a store.", Parameters: ToolParameters{Type: "object", Properties: map[string]ToolPropSchema{"store_name": storeProp["store_name"], "limit": {Type: "number", Description: "default 5"}}, Required: []string{"store_name"}}}, func(args map[string]any) (string, error) {
		storeName, _ := args["store_name"].(string)
		if storeName == "" {
			return "", errors.New("store_name is required")
		}
		limit := 5
		if v, ok := args["limit"].(float64); ok && int(v) > 0 {
			limit = int(v)
		}
		store, msg, err := openStoreOrMessage(fatbabyRoot, storeName)
		if msg != "" || err != nil {
			return msg, err
		}
		defer store.Close()
		latest, _ := store.LatestSequence(context.Background())
		start := int64(latest) - int64(limit) + 1
		if start < 1 {
			start = 1
		}
		recs, _ := store.ReadFrom(context.Background(), uint64(start), limit)
		resp := map[string]any{"latest_seq": latest, "record_count_shown": len(recs), "records": recs}
		b, _ := json.MarshalIndent(resp, "", "  ")
		return string(b), nil
	})

	d.Register(ToolDef{Name: "fatbaby_process_status", Description: "Check if fatbaby processes are running.", Parameters: ToolParameters{Type: "object", Properties: map[string]ToolPropSchema{}}}, func(args map[string]any) (string, error) {
		names := []string{"cmd/secwatch", "cmd/prwatch", "cmd/prwatch-body", "cmd/processor", "cmd/dashboard"}
		type ps struct {
			Name    string `json:"name"`
			Running bool   `json:"running"`
			PID     string `json:"pid,omitempty"`
		}
		var out []ps
		for _, n := range names {
			res, _ := runCommandWithTimeout([]string{"pgrep", "-af", n}, fatbabyRoot)
			p := ps{Name: n, Running: strings.TrimSpace(res) != ""}
			if p.Running {
				p.PID = strings.Fields(res)[0]
			}
			out = append(out, p)
		}
		dirs := map[string]bool{}
		for _, s := range []string{"secwatch", "prwatch", "prwatch-body"} {
			_, err := os.Stat(absStorePath(fatbabyRoot, s))
			dirs[s] = err == nil
		}
		b, _ := json.MarshalIndent(map[string]any{"processes": out, "store_dirs": dirs}, "", "  ")
		return string(b), nil
	})

	d.Register(ToolDef{Name: "fatbaby_run_secwatch_dryryn", Description: "Run one dry-run SEC discovery poll.", Parameters: ToolParameters{Type: "object", Properties: map[string]ToolPropSchema{}}}, func(args map[string]any) (string, error) {
		return runCommandWithTimeout([]string{"go", "run", "./cmd/secwatch", "-watchlist", "./config/watchlist.json", "-store", "./var/secwatch", "-dry-run"}, fatbabyRoot)
	})

	d.Register(ToolDef{Name: "fatbaby_tail_store", Description: "Tail last N events from a store.", Parameters: ToolParameters{Type: "object", Properties: map[string]ToolPropSchema{"store_name": storeProp["store_name"], "event_type": {Type: "string", Description: "optional"}, "limit": {Type: "number", Description: "default 10"}}, Required: []string{"store_name"}}}, func(args map[string]any) (string, error) {
		storeName, _ := args["store_name"].(string)
		eventType, _ := args["event_type"].(string)
		limit := 10
		if v, ok := args["limit"].(float64); ok && int(v) > 0 {
			limit = int(v)
		}
		store, msg, err := openStoreOrMessage(fatbabyRoot, storeName)
		if msg != "" || err != nil {
			return msg, err
		}
		defer store.Close()
		latest, _ := store.LatestSequence(context.Background())
		start := int64(latest) - int64(limit) + 1
		if start < 1 {
			start = 1
		}
		recs, _ := store.ReadFrom(context.Background(), uint64(start), limit)
		var lines []string
		for _, r := range recs {
			if eventType != "" && r.Event.Type != eventType {
				continue
			}
			preview := string(r.Event.Data)
			if len(preview) > 200 {
				preview = preview[:200]
			}
			line, _ := json.Marshal(map[string]any{"seq": r.Sequence, "type": r.Event.Type, "aggregate_key": r.Event.AggregateKey, "occurred_at": r.Event.OccurredAt, "data_preview": preview})
			lines = append(lines, string(line))
		}
		return strings.Join(lines, "\n"), nil
	})

	d.Register(ToolDef{Name: "fatbaby_check_env", Description: "Check processor env and prerequisites.", Parameters: ToolParameters{Type: "object", Properties: map[string]ToolPropSchema{}}}, func(args map[string]any) (string, error) {
		type check struct{ Name, Status, Value string }
		checks := []check{}
		ant := os.Getenv("ANTHROPIC_API_KEY")
		op := os.Getenv("OPENAI_API_KEY")
		if ant != "" || op != "" {
			checks = append(checks, check{"llm_key", "ok", "present"})
		} else {
			checks = append(checks, check{"llm_key", "missing", ""})
		}
		if _, err := exec.LookPath("go"); err == nil {
			checks = append(checks, check{"go_path", "ok", "go found"})
		} else {
			checks = append(checks, check{"go_path", "missing", ""})
		}
		watch := filepath.Join(fatbabyRoot, "config", "watchlist.json")
		b, err := os.ReadFile(watch)
		if err != nil {
			checks = append(checks, check{"watchlist", "missing", ""})
		} else if json.Valid(b) {
			checks = append(checks, check{"watchlist", "ok", "valid json"})
		} else {
			checks = append(checks, check{"watchlist", "invalid", "invalid json"})
		}
		resp, _ := json.MarshalIndent(map[string]any{"checks": checks}, "", "  ")
		return string(resp), nil
	})

	d.Register(ToolDef{Name: "fatbaby_count_signals", Description: "Count signal events in secwatch store.", Parameters: ToolParameters{Type: "object", Properties: map[string]ToolPropSchema{}}}, func(args map[string]any) (string, error) {
		store, msg, err := openStoreOrMessage(fatbabyRoot, "secwatch")
		if msg != "" || err != nil {
			return msg, err
		}
		defer store.Close()
		latest, _ := store.LatestSequence(context.Background())
		recs, _ := store.ReadFrom(context.Background(), 1, int(latest))
		generated, failed, discovered := 0, 0, 0
		var latestSignal map[string]any
		for _, r := range recs {
			switch r.Event.Type {
			case "signal_generated":
				generated++
				_ = json.Unmarshal(r.Event.Data, &latestSignal)
			case "signal_failed":
				failed++
			case "filing_discovered":
				discovered++
			}
		}
		resp := map[string]any{"total_records": len(recs), "signal_generated_count": generated, "signal_failed_count": failed, "filing_discovered_count": discovered, "latest_signal": latestSignal}
		b, _ := json.MarshalIndent(resp, "", "  ")
		return string(b), nil
	})
}

const emilySystemPrompt = `You are Emily, an AI agent with a dual role.

FATBABY OPERATIONS:
You manage the prrject-fatbaby intelligence pipeline. It has five processes and three event stores.
Processes: cmd/secwatch (SEC poller), cmd/prwatch (PR Newswire poller),
           cmd/prwatch-body (PR body fetcher), cmd/processor (LLM signal generator),
           cmd/dashboard (web UI on :8080).
Event stores: var/secwatch (SEC filings + signals), var/prwatch (PR discoveries),
              var/prwatch-body (PR full bodies).
The dashboard only reads var/secwatch. The processor reads var/secwatch and writes signals back to it.
Use your fatbaby tools proactively to answer operational questions — do not guess.`

func NewServer(cfg Config) {
	d := NewToolDispatcher()
	registerGitTools(d, cfg.ConversationDir)
	registerFatbabyTools(d, cfg.FatbabyRoot)
	log.Printf("tools registered: %d", len(d.Defs()))
}

func main() { NewServer(loadConfig()); fmt.Println("emily-agent fork initialized") }
