variable "auth_token" {}
variable "project_id" {}
variable "public_key" {}
variable "master_hostname" {}
variable "worker1_hostname" {}
variable "public_key_name" {}

provider "packet" {
  auth_token = "${var.auth_token}"
}

resource "packet_ssh_key" "key" {
  name       = "${var.public_key_name}"
  public_key = "${file(var.public_key)}"
}

resource "packet_device" "master" {
  hostname         = "${var.master_hostname}"
  facilities       = ["sjc1"]
  plan             = "t1.small.x86"
  operating_system = "ubuntu_16_04"
  billing_cycle    = "hourly"
  project_id       = "${var.project_id}"
  depends_on       = ["packet_ssh_key.key"]

  timeouts {
    create = "5m"
    delete = "5m"
  }
}

resource "packet_device" "worker1" {
  hostname         = "${var.worker1_hostname}"
  facilities       = ["sjc1"]
  plan             = "t1.small.x86"
  operating_system = "ubuntu_16_04"
  billing_cycle    = "hourly"
  project_id       = "${var.project_id}"
  depends_on       = ["packet_ssh_key.key"]

  timeouts {
    create = "5m"
    delete = "5m"
  }
}

output "master.public_ip" {
  value = "${packet_device.master.access_public_ipv4}"
}

output "worker1.public_ip" {
  value = "${packet_device.worker1.access_public_ipv4}"
}
