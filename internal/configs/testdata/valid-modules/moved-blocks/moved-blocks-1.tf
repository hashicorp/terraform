moved {
  from = test.foo
  to   = test.bar
}

moved {
  from = test.foo
  to   = test.bar["bloop"]
}

moved {
  from = module.a
  to   = module.b
}

moved {
  from = module.a
  to   = module.a["foo"]
}

moved {
  from = test.foo
  to   = module.a.test.foo
}
