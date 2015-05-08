module SidebarHelpers
  # This helps by setting the "active" class for sidebar nav elements
  # if the YAML frontmatter matches the expected value.
  def sidebar_current(expected)
    current = current_page.data.sidebar_current || ""
    if current == expected or (expected.is_a?(Regexp) and expected.match(current))
      return " class=\"active\""
    else
      return ""
    end
  end
end
