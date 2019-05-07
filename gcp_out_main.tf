// Use provider module to use google credentials
// ideally this would be a service account for Soter allowing
// to add/update/remove firewall rules
module "provider" {
    source = "./provider"
}
// ===========================================================================
// INGRESS
// ===========================================================================

// ==================
// GCP_Foundation
// ====================
// ==================
// SFCore.Test.Perimeter_LoadBalancer
// ====================
 module "Policy_from_SFCore-Test-Perimeter_LoadBalancer_to_SFCore-Test-Logging_Monitoring" {
  source = "./firewall/ingress_rule"
  name = "rule-25"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["22","8443"]
  udp_ports = []
  target_tags = ["sfcore-test-logging-monitoring"]
  source_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_ranges= ["20.48.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Perimeter_LoadBalancer_to_SFCore-Test-Management_Truth" {
  source = "./firewall/ingress_rule"
  name = "rule-26"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-truth"]
  source_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_ranges= ["20.48.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Perimeter_LoadBalancer_to_SFCore-Test-Perimeter_Services" {
  source = "./firewall/ingress_rule"
  name = "rule-27"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["53","111","123","500-1024","2049","6501","32765-65535"]
  udp_ports = ["53","111","123","500-1024","2049","6501","32765-65535"]
  target_tags = ["sfcore-test-perimeter-services"]
  source_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_ranges= ["20.48.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Perimeter_LoadBalancer_to_SFCore-Test-Perimeter_Services" {
  source = "./firewall/ingress_rule"
  name = "rule-28"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["53","111","123","500-1024","2049","6501","32765-65535"]
  udp_ports = ["53","111","123","500-1024","2049","6501","32765-65535"]
  target_tags = ["sfcore-test-perimeter-services"]
  source_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_ranges= ["20.48.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Perimeter_LoadBalancer_to_SFCore-Test-Management_Truth" {
  source = "./firewall/ingress_rule"
  name = "rule-29"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-truth"]
  source_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_ranges= ["20.48.0.0/12"]
} 
 module "intra_sg_rule" {
  source = "./firewall/ingress_rule"
  name = "rule-30"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_ranges= ["20.48.0.0/12"]
} 
// ==================
// SFCore.Test.Logging_Monitoring
// ====================
 module "Policy_from_SFCore-Test-Logging_Monitoring_to_SFCore-Test-Management_Truth" {
  source = "./firewall/ingress_rule"
  name = "rule-31"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-truth"]
  source_tags = ["sfcore-test-logging-monitoring"]
  source_ranges= ["20.0.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Logging_Monitoring_to_SFCore-Test-Perimeter_Services" {
  source = "./firewall/ingress_rule"
  name = "rule-32"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["53","80","443","1521","2181","2484","4242","4900","5055","5121","5432","5666-5667","5693","6379","6665","6667","7802","8000-10000","11000","11004","11009","13701","19888","27017","42099","50010","60000","60020"]
  udp_ports = ["53","161"]
  target_tags = ["sfcore-test-perimeter-services"]
  source_tags = ["sfcore-test-logging-monitoring"]
  source_ranges= ["20.0.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Logging_Monitoring_to_SFCore-Test-Management_Gateway" {
  source = "./firewall/ingress_rule"
  name = "rule-33"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["53","80","443","1521","2181","2484","4242","4567","4900","5055","5121","5432","5666-5667","5693","6379","6665","6667","6681-6689","7802","8000-10000","11000","11004","11009","13701","19888","27017","42099","50010","60000","60020"]
  udp_ports = ["53","161"]
  target_tags = ["sfcore-test-management-gateway"]
  source_tags = ["sfcore-test-logging-monitoring"]
  source_ranges= ["20.0.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Logging_Monitoring_to_SFCore-Test-Management_Truth" {
  source = "./firewall/ingress_rule"
  name = "rule-34"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-truth"]
  source_tags = ["sfcore-test-logging-monitoring"]
  source_ranges= ["20.0.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Logging_Monitoring_to_SFCore-Test-Perimeter_Services" {
  source = "./firewall/ingress_rule"
  name = "rule-35"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["53","80","443","1521","2181","2484","4242","4900","5055","5121","5432","5666-5667","5693","6379","6665","6667","7802","8000-10000","11000","11004","11009","13701","19888","27017","42099","50010","60000","60020"]
  udp_ports = ["53","161"]
  target_tags = ["sfcore-test-perimeter-services"]
  source_tags = ["sfcore-test-logging-monitoring"]
  source_ranges= ["20.0.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Logging_Monitoring_to_SFCore-Test-Management_Gateway" {
  source = "./firewall/ingress_rule"
  name = "rule-36"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["53","80","443","1521","2181","2484","4242","4567","4900","5055","5121","5432","5666-5667","5693","6379","6665","6667","6681-6689","7802","8000-10000","11000","11004","11009","13701","19888","27017","42099","50010","60000","60020"]
  udp_ports = ["53","161"]
  target_tags = ["sfcore-test-management-gateway"]
  source_tags = ["sfcore-test-logging-monitoring"]
  source_ranges= ["20.0.0.0/12"]
} 
 module "intra_sg_rule" {
  source = "./firewall/ingress_rule"
  name = "rule-37"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-logging-monitoring"]
  source_tags = ["sfcore-test-logging-monitoring"]
  source_ranges= ["20.0.0.0/12"]
} 
// ==================
// SFCore.Test.Management_Truth
// ====================
 module "Policy_from_SFCore-Test-Management_Truth_to_SFCore-Test-Perimeter_LoadBalancer" {
  source = "./firewall/ingress_rule"
  name = "rule-38"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["4194","10248-10250","10255"]
  udp_ports = []
  target_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_tags = ["sfcore-test-management-truth"]
  source_ranges= ["20.32.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Management_Truth_to_SFCore-Test-Logging_Monitoring" {
  source = "./firewall/ingress_rule"
  name = "rule-39"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["80","443","1521","4194","4200","7014","7442","8080","8443-8444","9998","10248-10250","10255","15372","18443-18444"]
  udp_ports = ["162"]
  target_tags = ["sfcore-test-logging-monitoring"]
  source_tags = ["sfcore-test-management-truth"]
  source_ranges= ["20.32.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Management_Truth_to_SFCore-Test-Management_Gateway" {
  source = "./firewall/ingress_rule"
  name = "rule-40"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["22","80","111","162","443","1521","2001-2009","2049","2181","2484","4194","4567","4900","5666","6681-6689","7802","8026-8027","8080-8081","8443","8980-8981","10248-10250","10255","50052"]
  udp_ports = ["69","111","161-162","2049","2323","25826"]
  target_tags = ["sfcore-test-management-gateway"]
  source_tags = ["sfcore-test-management-truth"]
  source_ranges= ["20.32.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Management_Truth_to_SFCore-Test-Perimeter_LoadBalancer" {
  source = "./firewall/ingress_rule"
  name = "rule-41"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["4194","10248-10250","10255"]
  udp_ports = []
  target_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_tags = ["sfcore-test-management-truth"]
  source_ranges= ["20.32.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Management_Truth_to_SFCore-Test-Logging_Monitoring" {
  source = "./firewall/ingress_rule"
  name = "rule-42"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["80","443","1521","4194","4200","7014","7442","8080","8443-8444","9998","10248-10250","10255","15372","18443-18444"]
  udp_ports = ["162"]
  target_tags = ["sfcore-test-logging-monitoring"]
  source_tags = ["sfcore-test-management-truth"]
  source_ranges= ["20.32.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Management_Truth_to_SFCore-Test-Management_Gateway" {
  source = "./firewall/ingress_rule"
  name = "rule-43"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["22","80","111","162","443","1521","2001-2009","2049","2181","2484","4194","4567","4900","5666","6681-6689","7802","8026-8027","8080-8081","8443","8980-8981","10248-10250","10255","50052"]
  udp_ports = ["69","111","161-162","2049","2323","25826"]
  target_tags = ["sfcore-test-management-gateway"]
  source_tags = ["sfcore-test-management-truth"]
  source_ranges= ["20.32.0.0/12"]
} 
 module "intra_sg_rule" {
  source = "./firewall/ingress_rule"
  name = "rule-44"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-truth"]
  source_tags = ["sfcore-test-management-truth"]
  source_ranges= ["20.32.0.0/12"]
} 
// ==================
// SFCore.Test.Perimeter_Services
// ====================
 module "Policy_from_SFCore-Test-Perimeter_Services_to_SFCore-Test-Logging_Monitoring" {
  source = "./firewall/ingress_rule"
  name = "rule-1"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["443","516","518","5140-5149","8005-8006","8305"]
  udp_ports = []
  target_tags = ["sfcore-test-logging-monitoring"]
  source_tags = ["sfcore-test-perimeter-services"]
  source_ranges= ["20.64.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Perimeter_Services_to_SFCore-Test-Management_Truth" {
  source = "./firewall/ingress_rule"
  name = "rule-2"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-truth"]
  source_tags = ["sfcore-test-perimeter-services"]
  source_ranges= ["20.64.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Perimeter_Services_to_SFCore-Test-Logging_Monitoring" {
  source = "./firewall/ingress_rule"
  name = "rule-3"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["443","516","518","5140-5149","8005-8006","8305"]
  udp_ports = []
  target_tags = ["sfcore-test-logging-monitoring"]
  source_tags = ["sfcore-test-perimeter-services"]
  source_ranges= ["20.64.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Perimeter_Services_to_SFCore-Test-Management_Truth" {
  source = "./firewall/ingress_rule"
  name = "rule-4"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-truth"]
  source_tags = ["sfcore-test-perimeter-services"]
  source_ranges= ["20.64.0.0/12"]
} 
 module "intra_sg_rule" {
  source = "./firewall/ingress_rule"
  name = "rule-5"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-perimeter-services"]
  source_tags = ["sfcore-test-perimeter-services"]
  source_ranges= ["20.64.0.0/12"]
} 
// ==================
// SFCore.Test.Management_Gateway
// ====================
 module "Policy_from_SFCore-Test-Management_Gateway_to_SFCore-Test-Perimeter_LoadBalancer" {
  source = "./firewall/ingress_rule"
  name = "rule-6"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_tags = ["sfcore-test-management-gateway"]
  source_ranges= ["20.16.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Management_Gateway_to_SFCore-Test-Logging_Monitoring" {
  source = "./firewall/ingress_rule"
  name = "rule-7"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-logging-monitoring"]
  source_tags = ["sfcore-test-management-gateway"]
  source_ranges= ["20.16.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Management_Gateway_to_SFCore-Test-Management_Truth" {
  source = "./firewall/ingress_rule"
  name = "rule-8"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-truth"]
  source_tags = ["sfcore-test-management-gateway"]
  source_ranges= ["20.16.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Management_Gateway_to_SFCore-Test-Perimeter_Services" {
  source = "./firewall/ingress_rule"
  name = "rule-9"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["22","53","80","111","123","443","500-1024","2049","6501","8080-8081","8443","32765-65535"]
  udp_ports = ["53","111","123","161","500-1024","2049","6501","32765-65535"]
  target_tags = ["sfcore-test-perimeter-services"]
  source_tags = ["sfcore-test-management-gateway"]
  source_ranges= ["20.16.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Management_Gateway_to_SFCore-Test-Perimeter_LoadBalancer" {
  source = "./firewall/ingress_rule"
  name = "rule-10"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_tags = ["sfcore-test-management-gateway"]
  source_ranges= ["20.16.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Management_Gateway_to_SFCore-Test-Logging_Monitoring" {
  source = "./firewall/ingress_rule"
  name = "rule-11"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-logging-monitoring"]
  source_tags = ["sfcore-test-management-gateway"]
  source_ranges= ["20.16.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Management_Gateway_to_SFCore-Test-Management_Truth" {
  source = "./firewall/ingress_rule"
  name = "rule-12"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-truth"]
  source_tags = ["sfcore-test-management-gateway"]
  source_ranges= ["20.16.0.0/12"]
} 
 module "Policy_from_SFCore-Test-Management_Gateway_to_SFCore-Test-Perimeter_Services" {
  source = "./firewall/ingress_rule"
  name = "rule-13"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["22","53","80","111","123","443","500-1024","2049","6501","8080-8081","8443","32765-65535"]
  udp_ports = ["53","111","123","161","500-1024","2049","6501","32765-65535"]
  target_tags = ["sfcore-test-perimeter-services"]
  source_tags = ["sfcore-test-management-gateway"]
  source_ranges= ["20.16.0.0/12"]
} 
 module "intra_sg_rule" {
  source = "./firewall/ingress_rule"
  name = "rule-14"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-gateway"]
  source_tags = ["sfcore-test-management-gateway"]
  source_ranges= ["20.16.0.0/12"]
} 
// ==================
// any
// ====================
 module "Policy_from_any_to_SFCore-Test-Logging_Monitoring" {
  source = "./firewall/ingress_rule"
  name = "rule-15"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["80","162","426","443","514","516","518","2181","5140-5149","6667","7014","7442","8080","8443-8444","9001-9002","9093-9094","9998","11443","18443-18444","33000-33100"]
  udp_ports = ["53","161-162","514","5140","6343"]
  target_tags = ["sfcore-test-logging-monitoring"]
  source_tags = [""]
  source_ranges= [""]
} 
 module "Policy_from_any_to_SFCore-Test-Perimeter_Services" {
  source = "./firewall/ingress_rule"
  name = "rule-16"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["25","53","111","123","500-1024","2049","6501","8080-8081","8443","32765-65535"]
  udp_ports = ["53","111","123","500-1024","2049","6501","8192","32765-65535"]
  target_tags = ["sfcore-test-perimeter-services"]
  source_tags = [""]
  source_ranges= [""]
} 
 module "Policy_from_any_to_SFCore-Test-Management_Truth" {
  source = "./firewall/ingress_rule"
  name = "rule-17"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-truth"]
  source_tags = [""]
  source_ranges= [""]
} 
 module "Policy_from_any_to_SFCore-Test-Perimeter_LoadBalancer" {
  source = "./firewall/ingress_rule"
  name = "rule-18"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_tags = [""]
  source_ranges= [""]
} 
 module "Policy_from_any_to_SFCore-Test-Management_Gateway" {
  source = "./firewall/ingress_rule"
  name = "rule-19"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-gateway"]
  source_tags = [""]
  source_ranges= [""]
} 
 module "Policy_from_any_to_SFCore-Test-Logging_Monitoring" {
  source = "./firewall/ingress_rule"
  name = "rule-20"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["80","162","426","443","514","516","518","2181","5140-5149","6667","7014","7442","8080","8443-8444","9001-9002","9093-9094","9998","11443","18443-18444","33000-33100"]
  udp_ports = ["53","161-162","514","5140","6343"]
  target_tags = ["sfcore-test-logging-monitoring"]
  source_tags = [""]
  source_ranges= [""]
} 
 module "Policy_from_any_to_SFCore-Test-Perimeter_Services" {
  source = "./firewall/ingress_rule"
  name = "rule-21"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["25","53","111","123","500-1024","2049","6501","8080-8081","8443","32765-65535"]
  udp_ports = ["53","111","123","500-1024","2049","6501","8192","32765-65535"]
  target_tags = ["sfcore-test-perimeter-services"]
  source_tags = [""]
  source_ranges= [""]
} 
 module "Policy_from_any_to_SFCore-Test-Management_Truth" {
  source = "./firewall/ingress_rule"
  name = "rule-22"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-truth"]
  source_tags = [""]
  source_ranges= [""]
} 
 module "Policy_from_any_to_SFCore-Test-Perimeter_LoadBalancer" {
  source = "./firewall/ingress_rule"
  name = "rule-23"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-perimeter-loadbalancer"]
  source_tags = [""]
  source_ranges= [""]
} 
 module "Policy_from_any_to_SFCore-Test-Management_Gateway" {
  source = "./firewall/ingress_rule"
  name = "rule-24"
  network = "${var.vpc}"
  project = "${var.project}"
  priority= 10000
  tcp_ports = ["1-65355"]
  udp_ports = ["1-65355"]
  target_tags = ["sfcore-test-management-gateway"]
  source_tags = [""]
  source_ranges= [""]
} 
