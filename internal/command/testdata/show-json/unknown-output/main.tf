output "foo" {
  value = "hello"
}

output "bar" {
  value = tolist([
    "hello",
    timestamp(),
    "world",
  ])
}

output "baz" {
  value = {
    greeting: "hello",
    time: timestamp(),
    subject: "world",
  }
}
