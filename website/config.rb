#-------------------------------------------------------------------------
# Configure Middleman
#-------------------------------------------------------------------------

activate :hashicorp do |h|
  h.version      = ENV['TERRAFORM_VERSION']
  h.bintray_repo = 'mitchellh/terraform'
  h.bintray_user = 'mitchellh'
  h.bintray_key  = ENV['BINTRAY_API_KEY']
end
