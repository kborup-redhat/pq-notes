package daemon

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNotificationTracking(t *testing.T) {
	tmpDir := t.TempDir()
	tracker := NewTracker(filepath.Join(tmpDir, "notified.json"))

	noteID := "Acme-Corp/2026-05-06-meeting.md.age"

	if tracker.WasNotified(noteID) {
		t.Error("should not be notified yet")
	}

	tracker.MarkNotified(noteID)
	if err := tracker.Save(); err != nil {
		t.Fatal(err)
	}

	loaded := NewTracker(filepath.Join(tmpDir, "notified.json"))
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	if !loaded.WasNotified(noteID) {
		t.Error("should be notified after load")
	}
}

func TestShouldNotify(t *testing.T) {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.Local)

	overdue := time.Date(2026, 5, 4, 17, 0, 0, 0, time.Local)
	if !ShouldNotify(overdue, now) {
		t.Error("overdue note should trigger notification")
	}

	dueNow := time.Date(2026, 5, 6, 10, 0, 0, 0, time.Local)
	if !ShouldNotify(dueNow, now) {
		t.Error("note due now should trigger notification")
	}

	future := time.Date(2026, 5, 10, 17, 0, 0, 0, time.Local)
	if ShouldNotify(future, now) {
		t.Error("future note should not trigger notification")
	}
}
