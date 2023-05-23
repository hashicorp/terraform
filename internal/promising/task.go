package promising

import (
	"context"
	"sync/atomic"
)

// task represents one of a set of collaborating tasks that are communicating
// in terms of promises.
type task struct {
	awaiting    atomic.Pointer[promise]
	responsible promiseSet
}

// MainTask runs the given function as a "main task", which is a task
// that blocks execution of the caller until it is complete and can create
// the promises and other async tasks required to produce its result.
func MainTask[T any](ctx context.Context, impl func(ctx context.Context) (T, error)) (T, error) {
	mainT := &task{
		responsible: make(promiseSet),
	}
	ctx = contextWithTask(ctx, mainT)
	v, err := impl(ctx)

	// The implementation function must have either resolved all of its
	// promises or transferred responsibility for them to another task
	// before it returns.
	for unresolved := range mainT.responsible {
		resolvePromise(unresolved, nil, ErrUnresolved)
		if err == nil {
			// If the task wasn't already returning its own error then we'll
			// make it return ErrUnresolved so the caller can see that the
			// task behaved incorrectly.
			err = ErrUnresolved
		}
	}
	return v, err
}

// AsyncTask runs the given function as a new task, passing responsibility
// for the promises in the given [PromiseContainer] to the new task.
//
// The new task runs concurrently with the caller as a new goroutine. It must
// either resolve all of the given promises or delegate responsibilty for
// them to another task before returning.
//
// The context passed to the implementation function carries the identity of
// the new task, and so the task must use that context for any calls to
// [PromiseGet] functions and for resolving any promises.
//
// A task should typically be a single thread of execution and not spawn
// any new goroutines unless doing so indirectly through another call to
// [AsyncTask]. If a particular task _does_ spawn additional goroutines then
// it's the task implementer's responsibility to prevent concurrent calls to
// any promise getters or resolvers from multiple goroutines. In particular,
// each task is allowed to await only one promise at a time and violating
// this invariant will cause undefined behavior.
func AsyncTask[P PromiseContainer](ctx context.Context, promises P, impl func(ctx context.Context, promises P)) {
	callerT := mustTaskFromContext(ctx)

	newT := &task{
		responsible: make(promiseSet),
	}

	promises.AnnounceContainedPromises(func(apr AnyPromiseResolver) {
		p := apr.promise()
		if p.responsible.Load() != callerT {
			// TODO: a better error message that gives some information
			// about what mismatched?
			panic("promise responsibility mismatch")
		}
		newT.responsible.Add(p)
		callerT.responsible.Remove(p)
		p.responsible.Store(newT)
	})

	go func() {
		ctx := contextWithTask(ctx, newT)
		impl(ctx, promises)

		// The implementation function must have either resolved all of its
		// promises or transferred responsibility for them to another task
		// before it returns.
		for unresolved := range newT.responsible {
			resolvePromise(unresolved, nil, ErrUnresolved)
		}
	}()
}

func mustTaskFromContext(ctx context.Context) *task {
	ret, ok := ctx.Value(taskContextKey).(*task)
	if !ok {
		panic("cannot interact with promises or tasks from non-task context")
	}
	return ret
}

func contextWithTask(ctx context.Context, t *task) context.Context {
	return context.WithValue(ctx, taskContextKey, t)
}

type taskContextKeyType int

const taskContextKey taskContextKeyType = 0
