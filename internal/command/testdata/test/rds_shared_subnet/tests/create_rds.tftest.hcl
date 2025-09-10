test {
  parallel = true
}

provider "test" {
}

run "setup_tests" {
  # This will create a DB subnet group that can be passed db_subnet_group_name
  # input of the module under test
  module {
    source = "./tests/setup"
  }
}

run "rds_without_dns_records" {
  command = apply
  state_key = "rds_without_dns_records"
  variables {
    environment          = "${run.setup_tests.name}0"
    password             = run.setup_tests.password
    db_subnet_group_name = run.setup_tests.subnet_group.name
    vpc_id               = run.setup_tests.vpc_id
    destroy_wait_seconds = 0
  }
}

run "rds_with_replica" {
  command = apply
  providers = {
    test = test
  }
  state_key = "rds_with_replica"
  variables {
    environment          = "${run.setup_tests.name}1"
    password             = run.setup_tests.password
    db_subnet_group_name = run.setup_tests.subnet_group.name
    vpc_id               = run.setup_tests.vpc_id
    destroy_wait_seconds = 1
  }
}

run "rds_instance_three" {
  command = apply
  state_key = "rds_instance_three"

  providers = {
    test = test
  }

  variables {
    environment          = "${run.setup_tests.name}3"
    password             = run.setup_tests.password
    db_subnet_group_name = run.setup_tests.subnet_group.name
    vpc_id               = run.setup_tests.vpc_id
    destroy_wait_seconds = 1
  }
}

run "rds_instance_four" {
  command = apply
  state_key = "rds_instance_four"

  providers = {
    test = test
  }

  variables {
    environment          = "${run.setup_tests.name}4"
    password             = run.setup_tests.password
    db_subnet_group_name = run.setup_tests.subnet_group.name
    vpc_id               = run.setup_tests.vpc_id
    destroy_wait_seconds = 1
  }
}

run "rds_instance_five" {
  command = apply
  state_key = "rds_instance_five"

  providers = {
    test = test
  }

  variables {
    environment          = "${run.setup_tests.name}5"
    password             = run.setup_tests.password
    db_subnet_group_name = run.setup_tests.subnet_group.name
    vpc_id               = run.setup_tests.vpc_id
    destroy_wait_seconds = 1
  }
}

run "rds_instance_six" {
  command = apply
  state_key = "rds_instance_six"

  providers = {
    test = test
  }

  variables {
    environment          = "${run.setup_tests.name}6"
    password             = run.setup_tests.password
    db_subnet_group_name = run.setup_tests.subnet_group.name
    vpc_id               = run.setup_tests.vpc_id
    destroy_wait_seconds = 1
  }
}
