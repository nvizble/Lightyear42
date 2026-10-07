// Package notify delivers push notifications to the user's devices and
// remembers what was already sent, so repeated checks stay silent.
//
// The package is an outbound adapter: it knows nothing about the 42 API.
// Business rules live in internal/services.
package notify

import "context"

// Priority levels understood by ntfy (1 = min … 5 = max).
const (
	PriorityDefault = 3
	PriorityHigh    = 4
)

// Notification is one message to deliver to the user's phone.
type Notification struct {
	// Title is the bold headline of the push.
	Title string
	// Message is the body text.
	Message string
	// Click is an optional URL opened when the notification is tapped.
	Click string
	// Tags are emoji shortcodes shown next to the title (e.g. "calendar").
	Tags []string
	// Priority ranks the notification; zero means the server default.
	Priority int
}

// Sender delivers notifications to a push service.
//
// Implementations must be safe to reuse across calls and must honor ctx.
type Sender interface {
	Send(ctx context.Context, n Notification) error
}

// TestNotification is the sample push used to verify the setup end to end.
func TestNotification() Notification {
	return Notification{
		Title:    "lightyear",
		Message:  "Notificações configuradas. Você será avisado quando uma nova avaliação for agendada.",
		Tags:     []string{"rocket"},
		Priority: PriorityDefault,
	}
}
