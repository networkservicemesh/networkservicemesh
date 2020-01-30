variable "auth_token" {}
variable "project_id" {}
variable "public_key" {}
variable "master1_hostname" {}
variable "worker1_1_hostname" {}
variable "master2_hostname" {}
variable "worker2_1_hostname" {}
variable "public_key_name" {}

provider "packet" {
  auth_token = "${var.auth_token}"
}

resource "packet_ssh_key" "key" {
  name       = "${var.public_key_name}"
  public_key = "${file(var.public_key)}"
}

resource "packet_device" "master1" {
  hostname         = "${var.master1_hostname}"
  facilities       = ["sjc1", "ewr1", "ams1"]
  plan             = "t1.small.x86"
  operating_system = "ubuntu_16_04"
  billing-cycle    = "hourly"
  project_id       = "${var.project_id}"
  depends_on       = ["packet_ssh_key.key"]
}

resource "packet_device" "worker1_1" {
  hostname         = "${var.worker1_1_hostname}"
  facilities       = ["sjc1", "ewr1", "ams1"]
  plan             = "t1.small.x86"
  operating_system = "ubuntu_16_04"
  billing-cycle    = "hourly"
  project_id       = "${var.project_id}"
  depends_on       = ["packet_ssh_key.key"]
}

output "master1.public_ip" {
  value = "${packet_device.master1.access_public_ipv4}"
}

output "worker1_1.public_ip" {
  value = "${packet_device.worker1_1.access_public_ipv4}"
}

resource "packet_device" "master2" {
  hostname         = "${var.master2_hostname}"
  facilities       = ["sjc1", "ewr1", "ams1"]
  plan             = "t1.small.x86"
  operating_system = "ubuntu_16_04"
  billing-cycle    = "hourly"
  project_id       = "${var.project_id}"
  depends_on       = ["packet_ssh_key.key"]
}

resource "packet_device" "worker2_1" {
  hostname         = "${var.worker2_1_hostname}"
  facilities       = ["sjc1", "ewr1", "ams1"]
  plan             = "t1.small.x86"
  operating_system = "ubuntu_16_04"
  billing-cycle    = "hourly"
  project_id       = "${var.project_id}"
  depends_on       = ["packet_ssh_key.key"]
}

output "master2.public_ip" {
  value = "${packet_device.master2.access_public_ipv4}"
}

output "worker2_1.public_ip" {
  value = "${packet_device.worker2_1.access_public_ipv4}"
}
