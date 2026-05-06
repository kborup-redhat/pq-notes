package daemon

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"

	"filippo.io/age"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

type Tracker struct {
	path     string
	Notified map[string]time.Time `json:"notified"`
}

func NewTracker(path string) *Tracker {
	return &Tracker{
		path:     path,
		Notified: make(map[string]time.Time),
	}
}

func (t *Tracker) Load() error {
	data, err := os.ReadFile(t.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &t.Notified)
}

func (t *Tracker) Save() error {
	data, err := json.MarshalIndent(t.Notified, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.path, data, 0600)
}

func (t *Tracker) WasNotified(noteID string) bool {
	_, ok := t.Notified[noteID]
	return ok
}

func (t *Tracker) MarkNotified(noteID string) {
	t.Notified[noteID] = time.Now()
}

func ShouldNotify(due, now time.Time) bool {
	return !due.IsZero() && !due.After(now)
}

func Run(cfg *config.Config, identity *age.HybridIdentity, notesDir, configDir string) {
	store := notes.NewNoteStore(notesDir, identity, cfg.DateFormat)
	tracker := NewTracker(filepath.Join(configDir, "notified.json"))
	if err := tracker.Load(); err != nil {
		log.Printf("daemon: load tracker: %v", err)
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	check(store, tracker)

	for range ticker.C {
		check(store, tracker)
	}
}

func check(store *notes.NoteStore, tracker *Tracker) {
	allNotes, err := store.List()
	if err != nil {
		log.Printf("daemon: list notes: %v", err)
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for _, n := range allNotes {
		if n.Status == notes.StatusDone {
			continue
		}
		if !ShouldNotify(n.Due, now) {
			continue
		}

		noteID := n.FilePath
		lastNotified, ok := tracker.Notified[noteID]
		if ok && lastNotified.After(today) {
			continue
		}

		if err := SendNotification(n.Title, n.Customer, n.Due); err != nil {
			log.Printf("daemon: notify: %v", err)
			continue
		}
		tracker.MarkNotified(noteID)
	}

	if err := tracker.Save(); err != nil {
		log.Printf("daemon: save tracker: %v", err)
	}
}
