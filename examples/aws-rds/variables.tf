variable "identifier" {
	default = "mydb-rds"
    description = "Idnetifier for your DB"
}

variable "storage" {
	default = "10"
    description = "Storage size in GB"
}

variable "engine" {
	default = "mysql"
    description = "Engine type, supported values mysql"
}

variable "engine_version" {
	default = "5.6.17"
    description = "Engine version"
}

variable "instance_class" {
	default = "db.t1.micro"
    description = "Instance class"
}

variable "db_name" {
	default = "mydb"
    description = "db name"
}

variable "username" {
	default = "user"
    description = "User name"
}

variable "password" {
	default = "abcd1234"
    description = "password"
}
