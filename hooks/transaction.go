package hooks

import (
	"fmt"
)

type Transaction interface {
	Commit() error
	Rollback() error
}

// Rollback performs transaction auto-rollback when returning error is not nil
func Rollback(tx Transaction, ret *error) {
	caller := getCaller()

	if *ret != nil {
		if err := tx.Rollback(); err != nil {
			*ret = fmt.Errorf("%s: failed to rollback transaction: %w: %w", caller, err, *ret)
		}
		return
	}

	if err := tx.Commit(); err != nil {
		*ret = fmt.Errorf("%s: failed to commit transaction: %w", caller, err)
	}
}

// RollbackHook performs rollback hook when returning error is not nil
func RollbackHook(name string, ret *error, f func() error) {
	caller := getCaller()

	if *ret != nil {
		if err := f(); err != nil {
			*ret = fmt.Errorf("%s: failed to rollback %s: %w: %w", caller, name, err, *ret)
		}
	}
}
