// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package promising_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform/internal/promising"
)

func TestMainTaskNoOp(t *testing.T) {
	wantVal := "hello"
	wantErr := errors.New("hello")

	ctx := context.Background()
	gotVal, gotErr := promising.MainTask(ctx, func(ctx context.Context) (string, error) {
		return wantVal, wantErr
	})

	if gotVal != wantVal {
		t.Errorf("wrong result value\ngot:  %q\nwant: %q", gotVal, wantVal)
	}
	if gotErr != wantErr {
		t.Errorf("wrong error\ngot:  %q\nwant: %q", gotErr, wantErr)
	}
}

func TestPromiseResolveSimple(t *testing.T) {
	wantVal := "hello"

	ctx := context.Background()
	gotVal, err := promising.MainTask(ctx, func(ctx context.Context) (string, error) {
		resolver, get := promising.NewPromise[string](ctx)

		promising.AsyncTask(
			ctx, resolver,
			func(ctx context.Context, resolver promising.PromiseResolver[string]) {
				resolver.Resolve(ctx, wantVal, nil)
			},
		)
		return get(ctx)
	})

	if gotVal != wantVal {
		t.Errorf("wrong result value\ngot:  %q\nwant: %q", gotVal, wantVal)
	}
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestPromiseUnresolvedMainWithoutGet(t *testing.T) {
	ctx := context.Background()
	var promiseID promising.PromiseID
	gotVal, err := promising.MainTask(ctx, func(ctx context.Context) (string, error) {
		resolver, _ := promising.NewPromise[string](ctx)
		promiseID = resolver.PromiseID()
		// Call to PromiseResolver.Resolve intentionally omitted to cause error
		// Also not calling the getter to prevent this from being classified as
		// a self-dependency.
		return "", nil
	})

	if wantVal := ""; gotVal != wantVal {
		// When unresolved the return value should be the zero value of the type.
		t.Errorf("wrong result value\ngot:  %q\nwant: %q", gotVal, wantVal)
	}
	if promiseIDs, ok := err.(promising.ErrUnresolved); !ok {
		t.Errorf("wrong error\ngot:  %s\nwant: an ErrUnresolved value", err)
	} else if got, want := len(promiseIDs), 1; got != want {
		t.Errorf("wrong number of unresolved promises %d; want %d", got, want)
	} else if promiseIDs[0] != promiseID {
		t.Error("errored promise ID does not match the one returned during the task")
	}
}

func TestPromiseUnresolvedMainWithGet(t *testing.T) {
	ctx := context.Background()
	var promiseID promising.PromiseID
	gotVal, gotErr := promising.MainTask(ctx, func(ctx context.Context) (string, error) {
		resolver, get := promising.NewPromise[string](ctx)
		promiseID = resolver.PromiseID()
		// Call to PromiseResolver.Resolve intentionally omitted to cause error
		return get(ctx)
	})

	if wantVal := ""; gotVal != wantVal {
		// When unresolved the return value should be the zero value of the type.
		t.Errorf("wrong result value\ngot:  %q\nwant: %q", gotVal, wantVal)
	}

	// Since the main task was both the one responsible for the promise and
	// the one trying to read it, this is classified as a self-dependency
	// rather than an "unresolved".
	if err, ok := gotErr.(promising.ErrSelfDependent); ok {
		if got, want := len(err), 1; got != want {
			t.Fatalf("wrong number of promise IDs in error %d; want %d", got, want)
		}
		if got, want := err[0], promiseID; got != want {
			t.Errorf("wrong promise ID in error\ngot:  %#v\nwant: %#v", got, want)
		}
	} else {
		t.Errorf("wrong error\ngot:  %s\nwant: an ErrSelfDependent value", gotErr)
	}
}

func TestPromiseUnresolvedAsync(t *testing.T) {
	ctx := context.Background()
	var promiseID promising.PromiseID
	gotVal, err := promising.MainTask(ctx, func(ctx context.Context) (string, error) {
		resolver, get := promising.NewPromise[string](ctx)
		promiseID = resolver.PromiseID()

		promising.AsyncTask(
			ctx, resolver,
			func(ctx context.Context, resolver promising.PromiseResolver[string]) {
				// Call to resolver.Resolve intentionally omitted to cause error
			},
		)
		return get(ctx)
	})

	if wantVal := ""; gotVal != wantVal {
		// When unresolved the return value should be the zero value of the type.
		t.Errorf("wrong result value\ngot:  %q\nwant: %q", gotVal, wantVal)
	}
	if promiseIDs, ok := err.(promising.ErrUnresolved); !ok {
		t.Errorf("wrong error\ngot:  %s\nwant: an ErrUnresolved value", err)
	} else if got, want := len(promiseIDs), 1; got != want {
		t.Errorf("wrong number of unresolved promises %d; want %d", got, want)
	} else if promiseIDs[0] != promiseID {
		t.Error("errored promise ID does not match the one returned during the task")
	}
}

func TestPromiseSelfDependentSibling(t *testing.T) {
	ctx := context.Background()
	var err1, err2 error
	promising.MainTask(ctx, func(ctx context.Context) (string, error) {
		resolver1, get1 := promising.NewPromise[string](ctx)
		resolver2, get2 := promising.NewPromise[string](ctx)

		// The following is an intentional self-dependency, though its
		// unpredictable which of the two tasks will actually detect the error,
		// since it'll be whichever one reaches its getter second.
		promising.AsyncTask(
			ctx, resolver1,
			func(ctx context.Context, resolver1 promising.PromiseResolver[string]) {
				v, err := get2(ctx)
				resolver1.Resolve(ctx, v, err)
			},
		)
		promising.AsyncTask(
			ctx, resolver2,
			func(ctx context.Context, resolver1 promising.PromiseResolver[string]) {
				v, err := get1(ctx)
				resolver2.Resolve(ctx, v, err)
			},
		)

		_, err1 = get1(ctx)
		_, err2 = get2(ctx)
		return "", nil
	})

	switch {
	case err1 == nil && err2 == nil:
		t.Fatalf("both promises succeeded; expected both to fail")
	case err1 == nil:
		t.Fatalf("first promise succeeded; expected both to fail")
	case err2 == nil:
		t.Fatalf("second promise succeeded; expected both to fail")
	}

	if err, ok := err1.(promising.ErrSelfDependent); ok {
		if got, want := len(err), 2; got != want {
			t.Fatalf("wrong number of promise IDs in err1 %d; want %d", got, want)
		}
	} else {
		t.Errorf("wrong err1\ngot:  %s\nwant: an ErrSelfDependent value", err1)
	}
	if err, ok := err2.(promising.ErrSelfDependent); ok {
		if got, want := len(err), 2; got != want {
			t.Fatalf("wrong number of promise IDs in err2 %d; want %d", got, want)
		}
	} else {
		t.Errorf("wrong err2\ngot:  %s\nwant: an ErrSelfDependent value", err2)
	}

}

func TestPromiseSelfDependentNested(t *testing.T) {
	ctx := context.Background()
	var err1, err2 error
	promising.MainTask(ctx, func(ctx context.Context) (string, error) {
		resolver1, get1 := promising.NewPromise[string](ctx)
		resolver2, get2 := promising.NewPromise[string](ctx)
		pair := promising.PromiseResolverPair[string, string]{A: resolver1, B: resolver2}

		// The following is an intentional self-dependency. Both calls should
		// fail here, since a self-dependency problem causes all affected
		// promises to immediately emit an error.
		promising.AsyncTask(
			ctx, pair,
			func(ctx context.Context, pair promising.PromiseResolverPair[string, string]) {
				resolver1 := pair.A
				resolver2 := pair.B

				promising.AsyncTask(
					ctx, resolver2,
					func(ctx context.Context, resolver1 promising.PromiseResolver[string]) {
						v, err := get1(ctx)
						resolver2.Resolve(ctx, v, err)
					},
				)

				v, err := get2(ctx)
				resolver1.Resolve(ctx, v, err)
			},
		)

		_, err1 = get1(ctx)
		_, err2 = get2(ctx)
		return "", nil
	})

	switch {
	case err1 == nil && err2 == nil:
		t.Fatalf("both promises succeeded; expected both to fail")
	case err1 == nil:
		t.Fatalf("first promise succeeded; expected both to fail")
	case err2 == nil:
		t.Fatalf("second promise succeeded; expected both to fail")
	}

	if err, ok := err1.(promising.ErrSelfDependent); ok {
		if got, want := len(err), 2; got != want {
			t.Fatalf("wrong number of promise IDs in err1 %d; want %d", got, want)
		}
	} else {
		t.Errorf("wrong err1\ngot:  %s\nwant: an ErrSelfDependent value", err1)
	}
	if err, ok := err2.(promising.ErrSelfDependent); ok {
		if got, want := len(err), 2; got != want {
			t.Fatalf("wrong number of promise IDs in err2 %d; want %d", got, want)
		}
	} else {
		t.Errorf("wrong err2\ngot:  %s\nwant: an ErrSelfDependent value", err2)
	}

}
