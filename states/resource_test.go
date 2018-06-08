package states

import (
	"testing"
)

func TestResourceInstanceDeposeCurrentObject(t *testing.T) {
	obj := &ResourceInstanceObject{
	// Empty for the sake of this test, because we're just going to
	// compare by pointer below anyway.
	}

	is := NewResourceInstance()
	is.Current = obj
	var dk DeposedKey

	t.Run("first depose", func(t *testing.T) {
		dk = is.DeposeCurrentObject() // dk is randomly-generated but should be eight characters long
		t.Logf("deposedKey is %q", dk)

		if got := is.Current; got != nil {
			t.Errorf("current is %#v; want nil", got)
		}
		if got, want := is.Deposed[dk], obj; got != want {
			t.Errorf("deposed object pointer is %#v; want %#v", got, want)
		}
		if got, want := len(is.Deposed), 1; got != want {
			t.Errorf("wrong len(is.Deposed) %d; want %d", got, want)
		}
		if got, want := len(dk), 8; got != want {
			t.Errorf("wrong len(deposedkey) %d; want %d", got, want)
		}
	})

	t.Run("second depose", func(t *testing.T) {
		notDK := is.DeposeCurrentObject()
		if notDK != NotDeposed {
			t.Errorf("got deposedKey %q; want NotDeposed", notDK)
		}

		// Make sure we really did abort early, and haven't corrupted the
		// state somehow.
		if got := is.Current; got != nil {
			t.Errorf("current is %#v; want nil", got)
		}
		if got, want := is.Deposed[dk], obj; got != want {
			t.Errorf("deposed object pointer is %#v; want %#v", got, want)
		}
		if got, want := len(is.Deposed), 1; got != want {
			t.Errorf("wrong len(is.Deposed) %d; want %d", got, want)
		}
		if got, want := len(dk), 8; got != want {
			t.Errorf("wrong len(deposedkey) %d; want %d", got, want)
		}
	})
}
