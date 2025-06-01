package notify

import (
	"fmt"
	"log"
)

// NotificationType represents the kind of notification to send
// e.g., "admin", "user", "webhook", etc.
type NotificationType string

const (
	NotifyAdmin NotificationType = "admin"
	NotifyUser  NotificationType = "user"
)

// Notification holds the data for a notification event
// You can extend this struct as needed for your adapters/UIs
// (e.g., add Email, WebhookURL, etc.)
type Notification struct {
	TxID        string
	Reason      string
	Attempt     int
	Type        NotificationType
	Recipient   string // e.g., user email, admin username, etc.
}

// Notify is a stub for sending notifications (log, email, webhook, etc.)
func Notify(n Notification) {
	// For now, just log the notification. Replace with actual sending logic as needed.
	log.Printf("[NOTIFY] To: %s | Type: %s | TxID: %s | Attempt: %d | Reason: %s", n.Recipient, n.Type, n.TxID, n.Attempt, n.Reason)
	// Example for email: sendEmail(n.Recipient, n.Reason)
	// Example for webhook: postWebhook(n.WebhookURL, n)
}

// Example usage (call this from your expiry worker or error handler):
func ExampleNotifyAdmin(txID, reason string, attempt int) {
	n := Notification{
		TxID:      txID,
		Reason:    reason,
		Attempt:   attempt,
		Type:      NotifyAdmin,
		Recipient: "admin@yourdomain.com",
	}
	Notify(n)
}

// You can add more helper functions for different channels/types as needed.
