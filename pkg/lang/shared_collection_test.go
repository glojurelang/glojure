package lang

import (
	"sync"
	"testing"
)

func TestSharedCollectionHashCachesAndPersistentSnapshots(t *testing.T) {
	values := []any{NewKeyword("a"), NewSymbol("symbol"), int64(1), "text"}
	sources := []IPersistentCollection{NewVector(values...), NewSet(values...)}
	equivalents := []IPersistentCollection{NewVector(values...), NewSet(values...)}
	for i, source := range sources {
		wantHash, wantEq := Hash(equivalents[i]), HashEq(equivalents[i])
		var workers sync.WaitGroup
		start := make(chan struct{})
		for n := 0; n < 16; n++ {
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				for j := 0; j < 20; j++ {
					if Hash(source) != wantHash || HashEq(source) != wantEq {
						t.Error("shared collection changed its hash")
					}
					tagged := source.(IObj).WithMeta(NewMap("tag", true))
					if Hash(tagged) != wantHash || HashEq(tagged) != wantEq {
						t.Error("metadata changed a shared collection hash")
					}
					transient := source.(IEditableCollection).AsTransient()
					transient = transient.Conj("extra").(ITransientCollection)
					grown := transient.Persistent()
					if grown.Count() != source.Count()+1 || !Equals(source, equivalents[i]) {
						t.Error("transient update changed a shared constant")
					}
				}
			}()
		}
		close(start)
		workers.Wait()
	}
}

func TestMapKVReduceOrderAndEarlyTermination(t *testing.T) {
	a, b := NewKeyword("a"), NewKeyword("b")
	for _, source := range []IPersistentMap{NewMap(a, 1, b, 2),
		NewStaticKeywordMap(NewKeywordMapShape("a", "b"), 1, 2),
		NewStaticKeywordMap(NewKeywordMapShape("a", "b"), 0, 2).Assoc(a, 1).(IPersistentMap)} {
		keys := []any{}
		result := source.(IKVReduce).KVReduce(FnFunc3(func(acc, key, value any) any {
			keys = append(keys, key)
			return NewReduced(value)
		}), nil)
		if result != 1 || len(keys) != 1 || keys[0] != a {
			t.Fatal("KVReduce changed order or continued after reduced")
		}
	}
}

func TestSharedSequenceHashCachesAndMetadata(t *testing.T) {
	values := []any{NewKeyword("key"), "text", int64(1)}
	expected := NewList(values...)
	wantHash, wantEq := Hash(expected), HashEq(expected)
	sources := []ISeq{NewList(values...), NewConsSequence(values),
		NewLazySeq(func() any { return NewList(values...) }),
		NewVector(values...).Seq(), newAPVSeq(NewVector(values...), 0)}
	for _, source := range sources {
		var workers sync.WaitGroup
		start := make(chan struct{})
		for n := 0; n < 16; n++ {
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				for i := 0; i < 10; i++ {
					if Hash(source) != wantHash || HashEq(source) != wantEq {
						t.Error("shared sequence hash changed")
					}
					tagged := source.(IObj).WithMeta(NewMap("tag", true))
					if Hash(tagged) != wantHash || HashEq(tagged) != wantEq {
						t.Error("sequence metadata changed hashing")
					}
				}
			}()
		}
		close(start)
		workers.Wait()
	}
}
