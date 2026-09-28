package state

import "vio/internal/domain"

// Conversation is the ordered set of provider-independent messages for a
// session. The agent loop appends user, assistant, and tool messages here as a
// turn progresses; the model layer reads them when building a request.
//
// Limit is the size guard (message count / bytes). The zero value means no
// explicit limit, in which case the context builder's budget applies.
type Conversation struct {
	Messages []domain.Message
	Limit    int
}
