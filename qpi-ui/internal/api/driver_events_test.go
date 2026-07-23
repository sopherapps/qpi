package api

import (
	"testing"

	"qpi/internal/drivers"
)

// TestCatalogEventsAreKnownTypes guards the one seam between the driver catalog
// and the wire protocol: the catalog names the events each kind participates in
// as plain strings (it is a leaf package and cannot import the api event
// types), so this asserts every one of those strings is an EventType QPI-UI
// actually has a handler for. It fails loudly if the two ever drift.
func TestCatalogEventsAreKnownTypes(t *testing.T) {
	for _, kind := range drivers.Default.Kinds() {
		for _, event := range drivers.Default.Events(kind) {
			if !isKnownEventType(EventType(event)) {
				t.Errorf("catalog kind %q lists unknown event type %q", kind, event)
			}
		}
	}
}
