variable "region" {
  default = "us-central1"
}

variable "region_zone" {
  default = "us-central1-f"
}

variable "project_name" {
  description = "The ID of the Google Cloud project"
}

variable "credentials_file_path" {
  description = "Path to the JSON file used to describe your account credentials"
  default     = "~/.gcloud/Terraform.json"
}

variable "public_key_path" {
  description = "Path to file containing public key"
  default     = "~/.ssh/gcloud_id_rsa.pub"
}

variable "private_key_path" {
  description = "Path to file containing private key"
  default     = "~/.ssh/gcloud_id_rsa"
}

variable "www_install_script_src_path" {
  description = "Path to www install script within this repository"
  default     = "scripts/install-www.sh"
}

variable "www_install_script_dest_path" {
  description = "Path to put the www install script on each destination resource"
  default     = "/tmp/install-www.sh"
}

variable "video_install_script_src_path" {
  description = "Path to video install script within this repository"
  default     = "scripts/install-video.sh"
}

variable "video_install_script_dest_path" {
  description = "Path to put the video install script on each destination resource"
  default     = "/tmp/install-video.sh"
}
