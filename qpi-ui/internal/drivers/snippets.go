package drivers

import (
	"fmt"
	"strings"
)

const (
	// scriptURL is where install-systemd.sh is fetched from in the systemd
	// snippet.
	scriptURL = "https://raw.githubusercontent.com/sopherapps/qpi/main/qpi-driver/install-systemd.sh"
	// baseCliExtra is installed for an official kind with no dedicated extra.
	baseCliExtra = "qpi-driver[cli]"
	// baseSdkPackage is installed for a custom driver, which depends on the SDK
	// alone and supplies its own handlers (RFC 0001 §5).
	baseSdkPackage = "qpi-driver"
	// optionFlag is the CLI flag an operation's per-kind config is passed
	// under, once per option.
	optionFlag = "-o"
)

// Params carries the per-registration values spliced into the setup snippets.
type Params struct {
	Name          string
	Token         string
	QpiAddr       string
	CaFingerprint string
}

// Snippets are the ready-to-use setup commands shown once at registration
// (RFC 0001 §3). An official Python backend fills Systemd and ManualCLI; every
// other case — a custom driver, or a language with no official build yet —
// fills Install and Stub instead.
type Snippets struct {
	Systemd   string `json:"systemd,omitempty"`
	ManualCLI string `json:"manual_cli,omitempty"`
	Install   string `json:"install,omitempty"`
	Stub      string `json:"stub,omitempty"`
}

// Snippets resolves the kind×language setup snippets for a registration. An
// official kind registered in this catalog, written in Python, gets the
// prefilled systemd + manual-CLI commands; anything else gets a bare SDK
// install plus a stub of the driver to fill in.
func (r *Registry) Snippets(kind Kind, language Language, p Params) Snippets {
	if spec, ok := r.Lookup(kind); ok && language == Python {
		return officialSnippets(spec, p)
	}
	return Snippets{
		Install: installCommand(language),
		Stub:    stub(language),
	}
}

// officialSnippets renders the systemd + manual-CLI commands for an official
// Python backend, from the spec's operation, device, and install extra.
func officialSnippets(spec Spec, p Params) Snippets {
	data := snippetData{
		ScriptURL:     scriptURL,
		Token:         p.Token,
		QpiAddr:       p.QpiAddr,
		CaFingerprint: p.CaFingerprint,
		Name:          shellQuote(p.Name),
		Device:        string(spec.Kind),
		Operation:     string(spec.Operation),
		Extra:         shellQuote(spec.extra()),
		Subcommand:    spec.subcommand(),
		OptionsEnv:    optionsEnv(spec.Options),
		Options:       cliOptions(spec.Options),
	}
	return Snippets{
		Systemd:   mustRender(systemdTmpl, data),
		ManualCLI: mustRender(manualCLITmpl, data),
	}
}

// extra resolves the pip extra to install for this backend.
func (s Spec) extra() string {
	if s.Extra != "" {
		return s.Extra
	}
	return baseCliExtra
}

// subcommand is the `qpi-driver` invocation that launches this backend, with
// the device already bound, e.g. "process --device qblox" or
// "monitor --device bluefors_gen1".
func (s Spec) subcommand() string {
	return fmt.Sprintf("%s %s %s", s.Operation, deviceFlag, s.Kind)
}

// cliOptions renders an operation's per-kind config as trailing `-o key=value`
// CLI flags, or "" when there are none.
func cliOptions(opts []Option) string {
	var b strings.Builder
	for _, opt := range opts {
		fmt.Fprintf(&b, " %s %s=%s", optionFlag, opt.Key, opt.Example)
	}
	return b.String()
}

// optionsEnv renders an operation's per-kind config as the DRIVER_OPTIONS
// environment variable install-systemd.sh reads, or "" when there are none.
func optionsEnv(opts []Option) string {
	if len(opts) == 0 {
		return ""
	}
	pairs := make([]string, len(opts))
	for i, opt := range opts {
		pairs[i] = opt.Key + "=" + opt.Example
	}
	return fmt.Sprintf(" DRIVER_OPTIONS='%s'", strings.Join(pairs, ";"))
}

// installCommand is the bare-SDK install for a language. Only the Python SDK
// ships today; the others are named ahead of their Phase 4 release, following
// qpi-client's per-language convention.
func installCommand(language Language) string {
	switch language {
	case TypeScript:
		return "npm install qpi-driver"
	case Go:
		return "go get github.com/sopherapps/qpi/qpi-driver/go"
	default:
		return "pip install " + baseSdkPackage
	}
}

// stub sketches a custom driver in a language: subclass the SDK base, dispatch
// on the event type in handle_event, and emit results. For a language without
// an SDK yet it shows the same shape the Python SDK exposes.
func stub(language Language) string {
	switch language {
	case TypeScript:
		return `import { QpiDriver, Event, EventType } from "qpi-driver";

class MyDriver extends QpiDriver {
  handleEvent(event: Event): void {
    if (event.type === EventType.JobDispatch) {
      const results = runMyBackend(event.payload);
      this.emit(new Event(EventType.JobResult, {
        job_id: event.payload.job_id, status: "completed", results,
      }));
    }
  }
}
`
	case Go:
		return `package main

import qpidriver "github.com/sopherapps/qpi/qpi-driver/go"

type MyDriver struct{ qpidriver.Base }

func (d *MyDriver) HandleEvent(event qpidriver.Event) {
	if event.Type == qpidriver.JobDispatch {
		results := runMyBackend(event.Payload)
		d.Emit(qpidriver.Event{Type: qpidriver.JobResult, Payload: map[string]any{
			"job_id": event.Payload["job_id"], "status": "completed", "results": results,
		}})
	}
}
`
	default:
		return `from qpi_driver import QpiDriver, Event, EventType

class MyDriver(QpiDriver):
    def handle_event(self, event: Event) -> None:
        if event.type == EventType.JOB_DISPATCH:
            results = run_my_backend(event.payload)
            self.emit(Event(
                type=EventType.JOB_RESULT,
                driver=self.name,
                payload={"job_id": event.payload["job_id"], "status": "completed", "results": results},
            ))
`
	}
}
