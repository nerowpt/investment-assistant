package mcp

import "testing"

func TestRegistry_ReadOnlyOnly(t *testing.T) {
	if len(ToolNames()) != 9 {
		t.Fatalf("tool count=%d want 9", len(ToolNames()))
	}
	for _, forbidden := range ForbiddenWriteTools() {
		for _, name := range ToolNames() {
			if name == forbidden {
				t.Fatalf("forbidden tool registered: %s", forbidden)
			}
		}
	}
}

func TestForbiddenWriteToolsNotEmpty(t *testing.T) {
	if len(ForbiddenWriteTools()) == 0 {
		t.Fatal("expected forbidden list")
	}
}
