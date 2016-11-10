description "Nomad by HashiCorp"

start on runlevel [2345]
stop on runlevel [!2345]

respawn

script
    CONFIG_DIR=/usr/local/etc/nomad
    mkdir -p $CONFIG_DIR
    exec /usr/local/bin/nomad agent -config $CONFIG_DIR >> /var/log/nomad.log 2>&1
end script
