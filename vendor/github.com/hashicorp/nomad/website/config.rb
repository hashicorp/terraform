set :base_url, "https://www.nomadproject.io/"

activate :hashicorp do |h|
  h.name        = "nomad"
  h.version     = "0.4.1"
  h.github_slug = "hashicorp/nomad"
end
