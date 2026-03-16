package allowlist

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

// WatchAndReload starts a background goroutine that reloads
// profiles when files in the profiles directory change.
func (s *Store) WatchAndReload(profilesDir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf(
			"Warning: could not start file watcher: %v", err,
		)
		return
	}

	if err := watcher.Add(profilesDir); err != nil {
		log.Printf(
			"Warning: could not watch %s: %v",
			profilesDir, err,
		)
		return
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					log.Printf(
						"Profile change detected: %s",
						event.Name,
					)
					if err := s.loadProfiles(profilesDir); err != nil {
						log.Printf(
							"Error reloading profiles: %v", err,
						)
						continue
					}

					// Notify the checker pool to rebuild indices
					s.mu.RLock()
					fn := s.onReload
					s.mu.RUnlock()
					if fn != nil {
						fn()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("File watcher error: %v", err)
			}
		}
	}()

	log.Printf("Watching %s for profile changes", profilesDir)
}
