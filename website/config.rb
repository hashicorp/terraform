#-------------------------------------------------------------------------
# Configure Middleman
#-------------------------------------------------------------------------

set :css_dir, 'stylesheets'
set :js_dir, 'javascripts'
set :images_dir, 'images'

# Use the RedCarpet Markdown engine
set :markdown_engine, :redcarpet
set :markdown,
    :fenced_code_blocks => true,
    :with_toc_data => true

# Build-specific configuration
configure :build do
  activate :asset_hash
  activate :minify_html
  activate :minify_javascript
end
