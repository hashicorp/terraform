require "rack"
require "rack/contrib/not_found"
require "rack/contrib/response_headers"
require "rack/contrib/static_cache"
require "rack/contrib/try_static"
require "rack/protection"

# Protect against various bad things
use Rack::Protection::JsonCsrf
use Rack::Protection::RemoteReferrer
use Rack::Protection::HttpOrigin
use Rack::Protection::EscapedParams
use Rack::Protection::XSSHeader
use Rack::Protection::FrameOptions
use Rack::Protection::PathTraversal
use Rack::Protection::IPSpoofing

# Properly compress the output if the client can handle it.
use Rack::Deflater

# Set the "forever expire" cache headers for these static assets. Since
# we hash the contents of the assets to determine filenames, this is safe
# to do.
use Rack::StaticCache,
  :root => "build",
  :urls => ["/images", "/javascripts", "/stylesheets"],
  :duration => 2,
  :versioning => false

# Try to find a static file that matches our request, since Middleman
# statically generates everything.
use Rack::TryStatic,
  :root => "build",
  :urls => ["/"],
  :try  => [".html", "index.html", "/index.html"]

# 404 if we reached this point. Sad times.
run Rack::NotFound.new(File.expand_path("../build/404.html", __FILE__))
