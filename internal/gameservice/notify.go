package gameservice

import "sync"

// notifier fans out "game changed" signals to subscribers, independent of the transport (the
// HTTP server turns them into Server-Sent Events). Each subscriber has a channel with a buffer of
// one that always holds the latest version: a slow reader skips intermediate versions instead of
// blocking the game, and it never misses that something changed.
type notifier struct {
	mu       sync.Mutex
	subs     map[string]map[chan uint64]struct{} // game ID -> subscriber channels
	versions map[string]uint64                   // game ID -> change counter since process start
}

func newNotifier() *notifier {
	return &notifier{subs: map[string]map[chan uint64]struct{}{}, versions: map[string]uint64{}}
}

// subscribe registers a listener for gameID. The returned func unsubscribes (idempotent).
func (n *notifier) subscribe(gameID string) (<-chan uint64, func()) {
	ch := make(chan uint64, 1)

	n.mu.Lock()
	if n.subs[gameID] == nil {
		n.subs[gameID] = map[chan uint64]struct{}{}
	}

	n.subs[gameID][ch] = struct{}{}
	n.mu.Unlock()

	var once sync.Once

	return ch, func() {
		once.Do(func() {
			n.mu.Lock()
			defer n.mu.Unlock()

			delete(n.subs[gameID], ch)

			if len(n.subs[gameID]) == 0 {
				delete(n.subs, gameID)
			}
		})
	}
}

// notify bumps the game's version and hands it to every subscriber without blocking.
func (n *notifier) notify(gameID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.versions[gameID]++
	v := n.versions[gameID]

	for ch := range n.subs[gameID] {
		// Replace a version the reader has not picked up yet. Only notify sends (under n.mu), so
		// after the drain the buffer is empty and the send cannot block.
		select {
		case <-ch:
		default:
		}

		ch <- v
	}
}

// Subscribe returns a channel that receives the game's change version after every saved change
// (lobby join, start, decisions incl. the bot moves that follow), plus a func to unsubscribe.
// token must belong to a player of the game, as for View.
func (s *Service) Subscribe(gameID, token string) (<-chan uint64, func(), error) {
	_, _, err := s.authorize(gameID, token)
	if err != nil {
		return nil, nil, err
	}

	ch, unsubscribe := s.events.subscribe(gameID)

	return ch, unsubscribe, nil
}
