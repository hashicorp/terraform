# "Try" and "can" functions

This Go package contains two `cty` functions intended for use in an
`hcl.EvalContext` when evaluating HCL native syntax expressions.

The first function `try` attempts to evaluate each of its argument expressions
in order until one produces a result without any errors.

```hcl
try(non_existent_variable, 2) # returns 2
```

If none of the expressions succeed, the function call fails with all of the
errors it encountered.

The second function `can` is similar except that it ignores the result of
the given expression altogether and simply returns `true` if the expression
produced a successful result or `false` if it produced errors.

Both of these are primarily intended for working with deep data structures
which might not have a dependable shape. For example, we can use `try` to
attempt to fetch a value from deep inside a data structure but produce a
default value if any step of the traversal fails:

```hcl
result = try(foo.deep[0].lots.of["traversals"], null)
```

The final result to `try` should generally be some sort of constant value that
will always evaluate successfully.

## Using these functions

Languages built on HCL can make `try` and `can` available to user code by
exporting them in the `hcl.EvalContext` used for expression evaluation:

```go
ctx := &hcl.EvalContext{
    Functions: map[string]function.Function{
        "try": tryfunc.TryFunc,
        "can": tryfunc.CanFunc,
    },
}
```
