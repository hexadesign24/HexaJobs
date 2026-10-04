package ui

import (
	"os"
	"testing"
)

func TestPreferencesRoundTripAndInvalidFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	prefs, err := LoadPreferences()
	if err != nil || prefs != DefaultPreferences() {
		t.Fatalf("defaults %+v %v", prefs, err)
	}
	prefs.Language = "jp"
	prefs.Region = "asia"
	prefs.ScamShield = false
	if err := SavePreferences(prefs); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPreferences()
	if err != nil || got != prefs {
		t.Fatalf("roundtrip %+v %v", got, err)
	}
	path, _ := preferencesPath()
	if err := os.WriteFile(path, []byte(`{"language":"bad"}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = LoadPreferences()
	if err == nil || got != DefaultPreferences() {
		t.Fatal("invalid preferences not reported")
	}
}
