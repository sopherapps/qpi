package api

import (
	"strings"
	"testing"
)

func TestIsKnownDriverKind(t *testing.T) {
	known := []DriverKind{DriverKindMock, DriverKindQiskitAer, DriverKindQuantify, DriverKindQblox, DriverKindPresto, DriverKindBlueforsGen1, DriverKindCustom}
	for _, kind := range known {
		if !isKnownDriverKind(kind) {
			t.Errorf("expected %q to be a known kind", kind)
		}
	}

	if isKnownDriverKind(DriverKind("rigetti")) {
		t.Errorf("expected unknown kind to be reported as unknown")
	}
}

func TestIsKnownDriverLanguage(t *testing.T) {
	known := []DriverLanguage{DriverLanguagePython, DriverLanguageTypeScript, DriverLanguageGo}
	for _, language := range known {
		if !isKnownDriverLanguage(language) {
			t.Errorf("expected %q to be a known language", language)
		}
	}

	if isKnownDriverLanguage(DriverLanguage("rust")) {
		t.Errorf("expected unknown language to be reported as unknown")
	}
}

func TestHasOfficialBuild(t *testing.T) {
	cases := []struct {
		kind     DriverKind
		language DriverLanguage
		want     bool
	}{
		{DriverKindQblox, DriverLanguagePython, true},
		{DriverKindMock, DriverLanguagePython, true},
		{DriverKindCustom, DriverLanguagePython, false},
		{DriverKindQblox, DriverLanguageGo, false},
		{DriverKindQblox, DriverLanguageTypeScript, false},
	}

	for _, c := range cases {
		if got := hasOfficialBuild(c.kind, c.language); got != c.want {
			t.Errorf("hasOfficialBuild(%q, %q) = %v, want %v", c.kind, c.language, got, c.want)
		}
	}
}

func TestDriverExtraFor(t *testing.T) {
	cases := map[DriverKind]string{
		DriverKindQblox:        "qpi-driver[cli,qblox]",
		DriverKindQuantify:     "qpi-driver[cli,quantify]",
		DriverKindQiskitAer:    "qpi-driver[cli,aer]",
		DriverKindBlueforsGen1: "qpi-driver[cli,bluefors_gen1]",
		DriverKindMock:         baseCliExtra,
		DriverKindPresto:       baseCliExtra,
	}

	for kind, want := range cases {
		if got := driverExtraFor(kind); got != want {
			t.Errorf("driverExtraFor(%q) = %q, want %q", kind, got, want)
		}
	}
}

func TestEventsForKind(t *testing.T) {
	events := eventsForKind(DriverKindQblox)
	if len(events) != 2 || events[0] != string(EventJobDispatch) || events[1] != string(EventJobResult) {
		t.Errorf("expected qblox events to be [JobDispatch, JobResult], got %v", events)
	}

	if events := eventsForKind(DriverKindCustom); events != nil {
		t.Errorf("expected custom kind to have no preset events, got %v", events)
	}
}

// TestEventsForKind_BlueforsGen1 proves the monitor is registered with only
// CryostatReading — it never participates in the job flow (RFC 0001 §7,
// Phase 3).
func TestEventsForKind_BlueforsGen1(t *testing.T) {
	events := eventsForKind(DriverKindBlueforsGen1)
	if len(events) != 1 || events[0] != string(EventCryostatReading) {
		t.Errorf("expected bluefors_gen1 events to be [CryostatReading], got %v", events)
	}
}

// TestBuildDriverSnippets_BlueforsGen1UsesMonitorSubcommand proves
// bluefors_gen1 gets the same three-snippet tier as an executor kind, but
// invoked through `qpi-driver monitor --kind` rather than the executor CLI's
// `start --executor` flag, which does not know about monitor drivers.
func TestBuildDriverSnippets_BlueforsGen1UsesMonitorSubcommand(t *testing.T) {
	snippets := buildDriverSnippets(DriverKindBlueforsGen1, DriverLanguagePython, "cryostat-1", "tok123", "http://localhost:8090", "aa:bb")

	if snippets.Install != "" || snippets.Stub != "" {
		t.Errorf("expected no base install/stub for an official build, got %+v", snippets)
	}
	for _, snippet := range []string{snippets.Systemd, snippets.ManualCLI, snippets.InstallAndRun} {
		if !strings.Contains(snippet, "tok123") {
			t.Errorf("expected snippet to contain the token, got %q", snippet)
		}
		if !strings.Contains(snippet, "bluefors_gen1") {
			t.Errorf("expected snippet to reference the bluefors_gen1 kind, got %q", snippet)
		}
	}
	if !strings.Contains(snippets.ManualCLI, "monitor --kind bluefors_gen1") {
		t.Errorf("expected ManualCLI to run the monitor subcommand, got %q", snippets.ManualCLI)
	}
	if strings.Contains(snippets.ManualCLI, "start --executor") {
		t.Errorf("expected ManualCLI not to use the executor CLI shape, got %q", snippets.ManualCLI)
	}
	if !strings.Contains(snippets.ManualCLI, "qpi-driver[cli,bluefors_gen1]") {
		t.Errorf("expected ManualCLI to use the bluefors_gen1 extra, got %q", snippets.ManualCLI)
	}
}

func TestBuildDriverSnippets_OfficialBuild(t *testing.T) {
	snippets := buildDriverSnippets(DriverKindQblox, DriverLanguagePython, "qpu_1", "tok_abc", "https://qpi.example.com", "ca_hash")

	if snippets.Install != "" || snippets.Stub != "" {
		t.Errorf("expected no base install/stub for an official build, got %+v", snippets)
	}
	for _, snippet := range []string{snippets.Systemd, snippets.ManualCLI, snippets.InstallAndRun} {
		if !strings.Contains(snippet, "tok_abc") {
			t.Errorf("expected snippet to contain the token, got %q", snippet)
		}
		if !strings.Contains(snippet, "qpu_1") {
			t.Errorf("expected snippet to contain the driver name, got %q", snippet)
		}
	}
	if !strings.Contains(snippets.ManualCLI, "qpi-driver[cli,qblox]") {
		t.Errorf("expected manual CLI snippet to use the qblox extra, got %q", snippets.ManualCLI)
	}
}

func TestBuildDriverSnippets_CustomHasNoOfficialBuild(t *testing.T) {
	snippets := buildDriverSnippets(DriverKindCustom, DriverLanguagePython, "qpu_1", "tok_abc", "https://qpi.example.com", "ca_hash")

	if snippets.Systemd != "" || snippets.ManualCLI != "" || snippets.InstallAndRun != "" {
		t.Errorf("expected no official-build snippets for a custom driver, got %+v", snippets)
	}
	if snippets.Install == "" || snippets.Stub == "" {
		t.Errorf("expected an install command and a stub for a custom driver, got %+v", snippets)
	}
}

func TestBuildDriverSnippets_NonPythonLanguageHasNoOfficialBuild(t *testing.T) {
	snippets := buildDriverSnippets(DriverKindQblox, DriverLanguageGo, "qpu_1", "tok_abc", "https://qpi.example.com", "ca_hash")

	if snippets.Systemd != "" || snippets.ManualCLI != "" || snippets.InstallAndRun != "" {
		t.Errorf("expected no official-build snippets for a go driver (no official build yet), got %+v", snippets)
	}
	if snippets.Install == "" || snippets.Stub == "" {
		t.Errorf("expected an install command and a stub for a go driver, got %+v", snippets)
	}
}
