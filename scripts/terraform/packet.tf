variable "auth_token" {}
variable "project_id" {}
variable "public_key" {}
variable "master_hostname" {}
variable "worker1_hostname" {}
variable "public_key_name" {}

provider "packet"{
    auth_token = "${var.auth_token}"
}

resource "packet_ssh_key" "key" {
    name = "${var.public_key_name}"
    public_key = "${file(var.public_key)}"
}

resource "packet_device" "master"{
    hostname = "${var.master_hostname}"
    facility = "sjc1"
    plan = "baremetal_0"
    operating_system = "ubuntu_16_04"
    billing_cycle = "hourly"
    project_id = "${var.project_id}"
    depends_on = ["packet_ssh_key.key"]
}
resource "packet_device" "worker1"{
    hostname = "${var.worker1_hostname}"
    facility = "sjc1"
    plan = "baremetal_0"
    operating_system = "ubuntu_16_04"
    billing_cycle = "hourly"
    project_id = "${var.project_id}"
    depends_on = ["packet_ssh_key.key"]
}

output "master.public_ip" {
    value = "${packet_device.master.network.0.address}"
}
output "worker1.public_ip" {
    value = "${packet_device.worker1.network.0.address}"
}
