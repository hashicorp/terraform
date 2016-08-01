set :base_url, "https://www.terraform.io/"

configure :development do
  set :releases_enabled, false
end

activate :hashicorp do |h|
  h.name             = "terraform"
  h.version          = "0.6.16"
  h.github_slug      = "hashicorp/terraform"
  h.releases_enabled = config[:releases_enabled]
end
