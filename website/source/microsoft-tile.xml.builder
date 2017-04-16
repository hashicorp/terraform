---
layout: false
noindex: true
---

xml.instruct!
xml.browserconfig do
  xml.msapplication do
    xml.tile do
      xml.square150x150logo src: image_path("favicons/mstile-150x150.png")
      xml.TileColor "#603CBA"
    end
  end
end
