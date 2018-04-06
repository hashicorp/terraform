name 'base'
description 'Common Env'
default_attributes(
  authorization: {
    sudo: {
      users: [ "centos" ]
    }
  }
)
