package api

import (
	"strings"
	"testing"
)

func TestIsKnownDriverKind(t *testing.T) {
	known := []DriverKind{DriverKindMock, DriverKindQiskitAer, DriverKindQuantify, DriverKindQblox, DriverKindPresto, DriverKindCustom}
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
		DriverKindQblox:     "qpi-driver[cli,qblox]",
		DriverKindQuantify:  "qpi-driver[cli,quantify]",
		DriverKindQiskitAer: "qpi-driver[cli,aer]",
		DriverKindMock:      baseCliExtra,
		DriverKindPresto:    baseCliExtra,
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
