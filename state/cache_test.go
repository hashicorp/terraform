package state

import (
	"os"
	"reflect"
	"testing"
)

func TestCacheState(t *testing.T) {
	cache := testLocalState(t)
	durable := testLocalState(t)
	defer os.Remove(cache.Path)
	defer os.Remove(durable.Path)

	TestState(t, &CacheState{
		Cache:   cache,
		Durable: durable,
	})
}

func TestCacheState_persistDurable(t *testing.T) {
	cache := testLocalState(t)
	durable := testLocalState(t)
	defer os.Remove(cache.Path)
	defer os.Remove(durable.Path)

	cs := &CacheState{
		Cache:   cache,
		Durable: durable,
	}

	state := cache.State()
	state.Modules = nil
	if err := cs.WriteState(state); err != nil {
		t.Fatalf("err: %s", err)
	}

	if reflect.DeepEqual(cache.State(), durable.State()) {
		t.Fatal("cache and durable should not be the same")
	}

	if err := cs.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(cache.State(), durable.State()) {
		t.Fatalf(
			"cache and durable should be the same\n\n%#v\n\n%#v",
			cache.State(), durable.State())
	}
}

func TestCacheState_impl(t *testing.T) {
	var _ StateReader = new(CacheState)
	var _ StateWriter = new(CacheState)
	var _ StatePersister = new(CacheState)
	var _ StateRefresher = new(CacheState)
}
