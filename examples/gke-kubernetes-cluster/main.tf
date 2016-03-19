provider "google" {
    region = "${var.region}"
    project = "${var.project_name}"
    account_file = "${file(var.account_file_path)}"
}

resource "google_container_cluster" "primary" {
    name = "marcellus-wallace"
    zone = "us-central1-a"
    initial_node_count = 3

    master_auth {
        username = "mr.yoda"
        password = "adoy.rm"
    }
}

output "kubernetes_endpoint" {
  value = "${google_container_cluster.primary.endpoint}"
}

provider "kubernetes" {
	endpoint = "${google_container_cluster.primary.endpoint}"
	username = "mr.yoda"
	password = "adoy.rm"
	insecure = true
}

resource "kubernetes_namespace" "default" {
    name = "myns"
    labels {
        name = "development"
    }
}

resource "kubernetes_persistent_volume" "wp" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
capacity:
  storage: 10Gi
accessModes:
  - ReadWriteOnce
hostPath:
  path: "/tmp/data01"
SPEC
}

resource "kubernetes_pod" "wp" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
containers:
  - image: wordpress
    name: wordpress
    env:
      - name: WORDPRESS_DB_PASSWORD
        # change this - must match mysql.yaml password
        value: yourpassword
    ports:
      - containerPort: 80
        name: wordpress
    volumeMounts:
        # name must match the volume name below
      - name: wordpress-persistent-storage
        # mount path within the container
        mountPath: /var/www/html
volumes:
  - name: wordpress-persistent-storage
    gcePersistentDisk:
      # This GCE PD must already exist.
      pdName: wordpress-disk
      fsType: ext4
SPEC
}

resource "kubernetes_replication_controller" "wp" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
replicas: 2
selector:
  app: nginx
template:
  metadata:
    labels:
      app: nginx
  spec:
    containers:
    - name: nginx
      image: nginx
      ports:
      - containerPort: 80
SPEC
}

resource "kubernetes_service" "wp" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
ports:
- port: 8000
  targetPort: 80
  protocol: TCP
selector:
  app: nginx
SPEC
}
