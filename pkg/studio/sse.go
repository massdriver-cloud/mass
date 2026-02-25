package studio

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

// SSEEvent represents an event to broadcast to connected clients
type SSEEvent struct {
	Type     string `json:"type"`               // Event type: "file_changed", "rescan_complete", "item_added", "item_removed"
	Path     string `json:"path,omitempty"`     // Path to the affected item
	ItemType string `json:"itemType,omitempty"` // Type of item: "bundle" or "artifact-definition"
	Count    int    `json:"count,omitempty"`    // Item count for rescan_complete events
}

// SSENotifier manages Server-Sent Events connections for file change notifications
type SSENotifier struct {
	subscribers map[chan []byte]bool
	mu          sync.RWMutex
}

// NewSSENotifier creates a new SSE notifier
func NewSSENotifier() *SSENotifier {
	return &SSENotifier{
		subscribers: make(map[chan []byte]bool),
	}
}

// ServeHTTP handles SSE connections
func (n *SSENotifier) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Check if flushing is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Create subscriber channel
	ch := n.Subscribe()
	defer n.Unsubscribe(ch)

	// Send initial connection event
	_, err := fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	if err != nil {
		slog.Debug("Failed to send initial SSE event", "error", err)
		return
	}
	flusher.Flush()

	// Listen for events or client disconnect
	for {
		select {
		case event := <-ch:
			_, err := fmt.Fprintf(w, "data: %s\n\n", event)
			if err != nil {
				slog.Debug("Failed to send SSE event", "error", err)
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			slog.Debug("SSE client disconnected")
			return
		}
	}
}

// Subscribe creates a new subscriber channel
func (n *SSENotifier) Subscribe() chan []byte {
	n.mu.Lock()
	defer n.mu.Unlock()

	ch := make(chan []byte, 10) // Buffered to prevent blocking
	n.subscribers[ch] = true
	slog.Debug("SSE subscriber added", "total", len(n.subscribers))
	return ch
}

// Unsubscribe removes a subscriber channel
func (n *SSENotifier) Unsubscribe(ch chan []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, ok := n.subscribers[ch]; ok {
		delete(n.subscribers, ch)
		close(ch)
		slog.Debug("SSE subscriber removed", "total", len(n.subscribers))
	}
}

// Broadcast sends an event to all connected subscribers
func (n *SSENotifier) Broadcast(event SSEEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("Failed to marshal SSE event", "error", err)
		return
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	for ch := range n.subscribers {
		// Non-blocking send - drop if subscriber is slow
		select {
		case ch <- data:
		default:
			slog.Debug("Dropping SSE event for slow subscriber")
		}
	}
}

// BroadcastFileChanged sends a file changed event
func (n *SSENotifier) BroadcastFileChanged(path string, itemType ItemType) {
	n.Broadcast(SSEEvent{
		Type:     "file_changed",
		Path:     path,
		ItemType: string(itemType),
	})
}

// BroadcastRescanComplete sends a rescan complete event
func (n *SSENotifier) BroadcastRescanComplete(itemCount int) {
	n.Broadcast(SSEEvent{
		Type:  "rescan_complete",
		Count: itemCount,
	})
}

// SubscriberCount returns the number of active subscribers
func (n *SSENotifier) SubscriberCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.subscribers)
}
