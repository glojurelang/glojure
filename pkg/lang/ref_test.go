package lang

import "testing"

func TestRefAlterUpdatesTransactionValue(t *testing.T) {
	ref := NewRef(int64(1))
	result := LockingTransaction.RunInTransaction(FnFunc0(func() any {
		return ref.Alter(FnFunc1(func(value any) any {
			return value.(int64) + 1
		}), nil)
	}))
	if result != int64(2) || ref.Deref() != int64(2) {
		t.Fatalf("Alter result = %v, ref = %v", result, ref.Deref())
	}
}
