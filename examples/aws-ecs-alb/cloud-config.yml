#cloud-config
coreos:
  units:
   - name: update-engine.service
     command: stop
   - name: amazon-ecs-agent.service
     command: start
     runtime: true
     content: |
       [Unit]
       Description=AWS ECS Agent
       Documentation=https://docs.aws.amazon.com/AmazonECS/latest/developerguide/
       Requires=docker.socket
       After=docker.socket

       [Service]
       Environment=ECS_CLUSTER=${ecs_cluster_name}
       Environment=ECS_LOGLEVEL=${ecs_log_level}
       Environment=ECS_VERSION=${ecs_agent_version}
       Restart=on-failure
       RestartSec=30
       RestartPreventExitStatus=5
       SyslogIdentifier=ecs-agent
       ExecStartPre=-/bin/mkdir -p /var/log/ecs /var/ecs-data /etc/ecs
       ExecStartPre=-/usr/bin/docker kill ecs-agent
       ExecStartPre=-/usr/bin/docker rm ecs-agent
       ExecStartPre=/usr/bin/docker pull amazon/amazon-ecs-agent:$${ECS_VERSION}
       ExecStart=/usr/bin/docker run --name ecs-agent \
                                     --volume=/var/run/docker.sock:/var/run/docker.sock \
                                     --volume=/var/log/ecs:/log \
                                     --volume=/var/ecs-data:/data \
                                     --volume=/sys/fs/cgroup:/sys/fs/cgroup:ro \
                                     --volume=/run/docker/execdriver/native:/var/lib/docker/execdriver/native:ro \
                                     --publish=127.0.0.1:51678:51678 \
                                     --env=ECS_LOGFILE=/log/ecs-agent.log \
                                     --env=ECS_LOGLEVEL=$${ECS_LOGLEVEL} \
                                     --env=ECS_DATADIR=/data \
                                     --env=ECS_CLUSTER=$${ECS_CLUSTER} \
                                     --env=ECS_AVAILABLE_LOGGING_DRIVERS='["awslogs"]' \
                                     --log-driver=awslogs \
                                     --log-opt awslogs-region=${aws_region} \
                                     --log-opt awslogs-group=${ecs_log_group_name} \
                                     amazon/amazon-ecs-agent:$${ECS_VERSION}
