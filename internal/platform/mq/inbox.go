package mq

import "context"

// FirstTime records, inside tx, that consumer handled the message and
// reports whether this is the first time. A redelivered message returns
// false, and the handler should do nothing.
func FirstTime(ctx context.Context, tx Execer, consumer, messageID string) (bool, error) {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO inbox (consumer, message_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, consumer, messageID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}
