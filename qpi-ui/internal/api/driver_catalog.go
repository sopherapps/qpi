package api

import (
	"fmt"
	"strings"
)

// DriverKind identifies one of the known official driver backends, or "custom"
// for a driver QPI-UI has no built-in knowledge of (RFC 0001 §3).
type DriverKind string

const (
	DriverKindMock      DriverKind = "mock"
	DriverKindQiskitAer DriverKind = "qiskit_aer"
	DriverKindQuantify  DriverKind = "quantify"
	DriverKindQblox     DriverKind = "qblox"
	DriverKindPresto    DriverKind = "presto"
	DriverKindBlueforsGen1 DriverKind = "bluefors_gen1"
	DriverKindCustom       DriverKind = "custom"
)

// DriverLanguage identifies one of the SDK languages a driver can be written
// in (RFC 0001 §2).
type DriverLanguage string

const (
	DriverLanguagePython     DriverLanguage = "python"
	DriverLanguageTypeScript DriverLanguage = "typescript"
	DriverLanguageGo         DriverLanguage = "go"
)

// driverKindExtras maps an official kind to the qpi-driver Python extra that
// ships a ready-to-run driver for it, mirroring install-systemd.sh's EXECUTOR
// branching (RFC 0001 §3). A kind absent here (mock, presto) installs the
// base cli extra; custom installs the bare SDK.
var driverKindExtras = map[DriverKind]string{
	DriverKindQblox:        "qpi-driver[cli,qblox]",
	DriverKindQuantify:     "qpi-driver[cli,quantify]",
	DriverKindQiskitAer:    "qpi-driver[cli,aer]",
	DriverKindBlueforsGen1: "qpi-driver[cli,bluefors_gen1]",
}

// baseCliExtra is installed for official kinds with no dedicated extra.
const baseCliExtra = "qpi-driver[cli]"

// baseSdkPackage is installed for a custom driver, which depends on the SDK
// alone and inherits the abstraction (RFC 0001 §5).
const baseSdkPackage = "qpi-driver"

// driverKindEvents lists the event types a QPU-shaped official driver
// participates in. A custom driver instead has its events chosen at
// registration (RFC 0001 §7).
var driverKindEvents = map[DriverKind][]EventType{
	DriverKindMock:         {EventJobDispatch, EventJobResult},
	DriverKindQiskitAer:    {EventJobDispatch, EventJobResult},
	DriverKindQuantify:     {EventJobDispatch, EventJobResult},
	DriverKindQblox:        {EventJobDispatch, EventJobResult},
	DriverKindPresto:       {EventJobDispatch, EventJobResult},
	DriverKindBlueforsGen1: {EventCryostatReading},
}

// isKnownDriverKind reports whether kind is one this QPI-UI version recognises.
func isKnownDriverKind(kind DriverKind) bool {
	if kind == DriverKindCustom {
		return true
	}
	_, ok := driverKindEvents[kind]
	return ok
}

// isKnownDriverLanguage reports whether language is one of the SDK languages.
func isKnownDriverLanguage(language DriverLanguage) bool {
	switch language {
	case DriverLanguagePython, DriverLanguageTypeScript, DriverLanguageGo:
		return true
	default:
		return false
	}
}

// isKnownEventType reports whether eventType is one QPI-UI has a handler for.
func isKnownEventType(eventType EventType) bool {
	for _, known := range AllEventTypes {
		if known == eventType {
			return true
		}
	}
	return false
}

// driverExtraFor resolves the pip extra to install for an official kind.
func driverExtraFor(kind DriverKind) string {
	if extra, ok := driverKindExtras[kind]; ok {
		return extra
	}
	return baseCliExtra
}

// eventsForKind resolves the fixed event set for an official kind. Empty for
// custom, whose events are chosen at registration instead.
func eventsForKind(kind DriverKind) []string {
	types, ok := driverKindEvents[kind]
	if !ok {
		return nil
	}
	out := make([]string, len(types))
	for i, t := range types {
		out[i] = string(t)
	}
	return out
}

// hasOfficialBuild reports whether QPI-UI ships a ready-to-run driver for the
// given kind×language pair. Today every known kind ships in Python only; the
// other SDK languages get the base SDK plus a stub to fill in (RFC 0001 §3, §11).
func hasOfficialBuild(kind DriverKind, language DriverLanguage) bool {
	return language == DriverLanguagePython && kind != DriverKindCustom
}

// buildDriverSnippets resolves the kind×language setup snippets shown once at
// registration (RFC 0001 §3). Official builds get systemd/manual-CLI/install-
// and-run snippets prefilled with the token, address, CA fingerprint and
// name; everything else gets a base install plus a stub with the handlers to
// fill in.
func buildDriverSnippets(kind DriverKind, language DriverLanguage, name, token, qpiAddr, caFingerprint string) DriverSnippets {
	// bluefors_gen1 does not run jobs, so it does not go through the executor
	// CLI (`qpi-driver start --executor ...`) the snippets below assume; it
	// runs through its own `qpi-driver monitor --kind` subcommand instead,
	// with its own required config (RFC 0001 §7, Phase 3).
	if kind == DriverKindBlueforsGen1 && language == DriverLanguagePython {
		return blueforsGen1Snippets(name, token, qpiAddr, caFingerprint)
	}

	// FIXME: Expand this to be more general and not only tailored to builtin QPU drivers
	//.  the name 'hasOfficialBuild' is misleading
	if hasOfficialBuild(kind, language) {
		extra := driverExtraFor(kind)
		quotedName := shellQuote(name)
		return DriverSnippets{
			Systemd: fmt.Sprintf(
				"curl -fsSL https://raw.githubusercontent.com/sopherapps/qpi/main/qpi-driver/install-systemd.sh | \\\n"+
					"  QPI_TOKEN=%s QPI_ADDR=%s CA_FINGERPRINT=%s QPU_NAME=%s EXECUTOR=%s sudo -E bash",
				token, qpiAddr, caFingerprint, quotedName, kind,
			),
			ManualCLI: fmt.Sprintf(
				"uv tool install --python 3.12 %q && \\\n"+
					"qpi-driver start --token %s --qpi-addr %s --ca-fingerprint %s --name %s --executor %s",
				extra, token, qpiAddr, caFingerprint, quotedName, kind,
			),
			InstallAndRun: fmt.Sprintf(
				"pip install %q && \\\n"+
					"qpi-driver start --token %s --qpi-addr %s --ca-fingerprint %s --name %s --executor %s",
				extra, token, qpiAddr, caFingerprint, quotedName, kind,
			),
		}
	}

	return DriverSnippets{
		Install: installSnippetFor(language),
		Stub:    stubSnippetFor(language),
	}
}

// blueforsGen1Snippets builds the setup snippets for the officially
// maintained Bluefors Gen. 1 monitor driver (RFC 0001 §7, Phase 3) — the same
// three-snippet tier as an executor kind like qblox, just launched through
// `qpi-driver monitor --kind` instead of `start --executor` since it does not
// run jobs. BLUEFORS_BASE_URL/BLUEFORS_CHANNELS are placeholders for the
// operator's own Control API address and value-tree channel paths (e.g.
// "mapper.bf.tmc"), which are system-specific.
func blueforsGen1Snippets(name, token, qpiAddr, caFingerprint string) DriverSnippets {
	extra := driverExtraFor(DriverKindBlueforsGen1)
	quotedName := shellQuote(name)
	const blueforsEnv = "BLUEFORS_BASE_URL=http://localhost:49099 BLUEFORS_CHANNELS=mapper.bf.tmc,mapper.bf.tstill"

	return DriverSnippets{
		Systemd: fmt.Sprintf(
			"curl -fsSL https://raw.githubusercontent.com/sopherapps/qpi/main/qpi-driver/install-systemd.sh | \\\n"+
				"  QPI_TOKEN=%s QPI_ADDR=%s CA_FINGERPRINT=%s QPU_NAME=%s EXECUTOR=bluefors_gen1 \\\n"+
				"  %s sudo -E bash",
			token, qpiAddr, caFingerprint, quotedName, blueforsEnv,
		),
		ManualCLI: fmt.Sprintf(
			"uv tool install --python 3.12 %q && \\\n"+
				"%s qpi-driver monitor --kind bluefors_gen1 \\\n"+
				"  --token %s --qpi-addr %s --ca-fingerprint %s --name %s",
			extra, blueforsEnv, token, qpiAddr, caFingerprint, quotedName,
		),
		InstallAndRun: fmt.Sprintf(
			"pip install %q && \\\n"+
				"%s qpi-driver monitor --kind bluefors_gen1 \\\n"+
				"  --token %s --qpi-addr %s --ca-fingerprint %s --name %s",
			extra, blueforsEnv, token, qpiAddr, caFingerprint, quotedName,
		),
	}
}

// shellQuote wraps a value in single quotes for safe interpolation into the
// shell snippets above, escaping any embedded single quotes. A driver's name
// is superuser-supplied free text, so it must not be splice-able into a shell
// command the same superuser later pastes and runs.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// installSnippetFor gives the bare-SDK install command for a language. Only
// the Python SDK ships today; the others are named ahead of their Phase 4
// release, following the qpi-client per-language naming convention.
func installSnippetFor(language DriverLanguage) string {
	switch language {
	case DriverLanguageTypeScript:
		return "npm install qpi-driver"
	case DriverLanguageGo:
		return "go get github.com/sopherapps/qpi/qpi-driver/go"
	default:
		return fmt.Sprintf("pip install %s", baseSdkPackage)
	}
}

// stubSnippetFor sketches the driver shape for a language (RFC 0001 §4). For
// languages without an SDK yet, it illustrates the same shape the Python SDK
// exposes today.
func stubSnippetFor(language DriverLanguage) string {
	switch language {
	case DriverLanguageTypeScript:
		return "import { QpiDriver, JobResult } from \"qpi-driver\";\n\n" +
			"class MyDriver extends QpiDriver {\n" +
			"  onJobDispatch(event) {\n" +
			"    const result = this.backend.execute(event.payload);\n" +
			"    this.emit(new JobResult({ jobId: event.payload.job_id, status: \"completed\", results: result }));\n" +
			"  }\n" +
			"}\n"
	case DriverLanguageGo:
		return "package main\n\n" +
			"import qpidriver \"github.com/sopherapps/qpi/qpi-driver/go\"\n\n" +
			"type MyDriver struct {\n" +
			"\tqpidriver.Base\n" +
			"}\n\n" +
			"func (d *MyDriver) OnJobDispatch(event qpidriver.Event) {\n" +
			"\tresult := d.Backend.Execute(event.Payload)\n" +
			"\td.Emit(qpidriver.JobResult{JobID: event.Payload[\"job_id\"], Status: \"completed\", Results: result})\n" +
			"}\n"
	default:
		return "class MyDriver(QpiDriver):\n" +
			"    def on_job_dispatch(self, event):\n" +
			"        result = self.backend.execute(event.payload)\n" +
			"        self.emit(JobResult(job_id=event.payload[\"job_id\"], status=\"completed\", results=result))\n"
	}
}
