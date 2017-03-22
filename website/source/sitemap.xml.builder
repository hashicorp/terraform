---
layout: false
---

xml.instruct!
xml.urlset 'xmlns' => "http://www.sitemaps.org/schemas/sitemap/0.9", 'xmlns:tf' => "http://www.terraform.io/ns/sitemap" do
  sitemap
    .resources
    .select { |page| page.path =~ /\.html/ }
    .select { |page| !page.data.noindex }
    .each do |page|
      xml.url do
        xml.loc File.join(base_url, page.url)
        xml.lastmod Date.today.to_time.iso8601
        xml.changefreq page.data.changefreq || "monthly"
        xml.priority page.data.priority || "0.5"
        if page.data.page_title
          xml.tag! 'tf:title', page.data.page_title
        end
      end
  end
end
