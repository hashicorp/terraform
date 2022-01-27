
component "iam" {
  module       = "./kubernetes-cluster/iam"
  display_name = "IAM"

  variables = {
    load_balancer_controller_policy_name = "EKSLoadBalancerController"
  }
}

component "regions" {
  module       = "./kubernetes-cluster/regions"
  for_each     = { for i, r in var.aws.regions : r.name => r }
  display_name = each.key

  variables = {
    global = component.kubernetes_global_aws
  }
}

component "global" {
  module       = "./kubernetes-cluster/global"
  display_name = "Global"
}
