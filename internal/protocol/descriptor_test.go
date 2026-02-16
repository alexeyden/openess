package protocol

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testDataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "data", name)
}

func TestLoadSubModelDefs(t *testing.T) {
	path := testDataPath("0925.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("test data file not found: %s", path)
	}

	desc, err := LoadProtocolDescriptor(path)
	if err != nil {
		t.Fatalf("failed to load descriptor: %s", err)
	}

	appShows, ok := desc.SubModelDefs["APPShows"]
	if !ok {
		t.Fatal("SubModelDefs missing APPShows")
	}

	if len(appShows) != 4 {
		t.Fatalf("expected 4 APPShows fields, got %d", len(appShows))
	}

	expected := []struct {
		address int
		length  int
		title   string
	}{
		{0, 2, "Battery show"},
		{2, 2, "PV show"},
		{4, 2, "Mains supply show"},
		{6, 2, "Loader"},
	}

	for i, exp := range expected {
		field := appShows[i]
		if field.Address != exp.address {
			t.Errorf("field %d: address = %d, want %d", i, field.Address, exp.address)
		}
		if field.Length != exp.length {
			t.Errorf("field %d: length = %d, want %d", i, field.Length, exp.length)
		}
		if field.Title["base"] != exp.title {
			t.Errorf("field %d: title = %q, want %q", i, field.Title["base"], exp.title)
		}
		if field.EnumerationStrings == nil {
			t.Errorf("field %d: missing EnumerationStrings", i)
		}
	}
}

func TestFindSubRegister(t *testing.T) {
	path := testDataPath("0925.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("test data file not found: %s", path)
	}

	desc, err := LoadProtocolDescriptor(path)
	if err != nil {
		t.Fatalf("failed to load descriptor: %s", err)
	}

	seg, reg := desc.FindRegister("APP show.Battery show")
	if seg == nil || reg == nil {
		t.Fatal("FindRegister returned nil for APP show.Battery show")
	}

	if reg.Address != 4553 {
		t.Errorf("address = %d, want 4553", reg.Address)
	}
	if reg.BitShift != 0 {
		t.Errorf("BitShift = %d, want 0", reg.BitShift)
	}
	if reg.BitMask != 0x3 {
		t.Errorf("BitMask = 0x%x, want 0x3", reg.BitMask)
	}

	seg, reg = desc.FindRegister("APP show.PV show")
	if seg == nil || reg == nil {
		t.Fatal("FindRegister returned nil for APP show.PV show")
	}

	if reg.BitShift != 2 {
		t.Errorf("BitShift = %d, want 2", reg.BitShift)
	}

	seg, reg = desc.FindRegister("APP show.Mains supply show")
	if seg == nil || reg == nil {
		t.Fatal("FindRegister returned nil for APP show.Mains supply show")
	}

	if reg.BitShift != 4 {
		t.Errorf("BitShift = %d, want 4", reg.BitShift)
	}

	seg, reg = desc.FindRegister("APP show.Loader")
	if seg == nil || reg == nil {
		t.Fatal("FindRegister returned nil for APP show.Loader")
	}

	if reg.BitShift != 6 {
		t.Errorf("BitShift = %d, want 6", reg.BitShift)
	}

	// Verify enum was parsed
	if reg.EnumerationStrings == nil {
		t.Fatal("EnumerationStrings is nil")
	}
}

func TestFindRegisterPlain(t *testing.T) {
	path := testDataPath("0925.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("test data file not found: %s", path)
	}

	desc, err := LoadProtocolDescriptor(path)
	if err != nil {
		t.Fatalf("failed to load descriptor: %s", err)
	}

	// Non-dotted names still work
	seg, reg := desc.FindRegister("Working State")
	if seg == nil || reg == nil {
		t.Fatal("FindRegister returned nil for Working State")
	}
	if reg.BitMask != 0 {
		t.Errorf("plain register should have BitMask=0, got 0x%x", reg.BitMask)
	}
}
