### Configure NAT instance Example

In the Virtual Private Cloud（VPC） environment, to enable multiple back-end intranet hosts to provide services externally with a limited number of EIPs, map the ports on the EIP-bound host to the back-end intranet hosts.

### Get up and running

* Planning phase

		terraform plan 

* Apply phase

		terraform apply 
		
		Get the outputs:
		+ nat_instance_eip_address = 123.56.19.238
		+ nat_instance_private_ip = 10.1.1.57
		+ worker_instance_private_ip = 10.1.1.56

* Apply phase

        + login the vm: ssh root@123.56.19.238|Test123456
        + Run the "iptables -t nat -nvL" command to check the result
        
          | prot | in |   source    |  destination   |                          |
          | ---- | -- | ----------- | -------------- | ------------------------ |
          | tcp  | *  | 0.0.0.0/0   |  10.1.1.57     |  tcp dpt:80 to:10.1.1.56
          | all  | *  | 10.1.1.0/24 |  0.0.0.0/0     |  to:10.1.1.57
        

* Destroy 

		terraform destroy