package ports

import "context"

// TransactionManager handles the execution of a unit of work within a database transaction.
type TransactionManager interface {
	// RunInTransaction executes the given function within a transaction context.
	// If the function returns an error, the transaction is rolled back.
	// If the function returns nil, the transaction is committed.
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
