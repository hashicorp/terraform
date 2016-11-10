module CommandHelpers
  # Returns the markdown text for the general options usage.
  def general_options_usage()
    <<EOF
* `-address=<addr>`: The address of the Nomad server. Overrides the `NOMAD_ADDR`
  environment variable if set. Defaults to `http://127.0.0.1:4646`.

* `-region=<region>`: The region of the Nomad server to forward commands to.
  Overrides the `NOMAD_REGION` environment variable if set. Defaults to the
  Agent's local region.

* `-no-color`: Disables colored command output.
EOF
  end
end
