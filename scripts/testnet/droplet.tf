# auth to GCP
terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.24.0"
    }
  }
}

provider "google" {
  credentials = file("~/.gcp/terraform.json")

  project = "stride-nodes"
  region  = "us-central1"
  zone    = "us-central1-b"
}

data "google_compute_default_service_account" "default" {
}
resource "google_compute_address" "node1" {
  name   = "node1"
  region = "us-central1"
}
resource "google_compute_address" "node2" {
  name   = "node2"
  region = "europe-west6"
}
resource "google_compute_address" "node3" {
  name   = "node3"
  region = "us-east4"
}
resource "google_compute_address" "seed" {
  name   = "seed"
  region = "us-west1"
}
resource "google_compute_instance" "droplet-node1" {
  name                      = "droplet-node1"
  machine_type              = "e2-standard-4"
  zone                      = "us-central1-b"
  tags                      = ["ssh"]
  allow_stopping_for_update = true

  metadata = {
    enable-oslogin            = "TRUE"
    gce-container-declaration = "spec:\n  containers:\n    - name: node\n      image: 'gcr.io/stride-nodes/testnet:droplet_node1'\n      stdin: false\n      tty: false\n  restartPolicy: Always\n"
  }
  boot_disk {
    initialize_params {
      image = "cos-cloud/cos-97-lts"
    }
  }

  network_interface {
    network = "default"
    access_config {
      nat_ip = google_compute_address.node1.address
    }
  }

  service_account {
    scopes = ["https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring.write",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/service.management.readonly",
    "https://www.googleapis.com/auth/trace.append"]
  }
}

resource "google_compute_instance" "droplet-node2" {
  name                      = "droplet-node2"
  machine_type              = "e2-standard-4"
  zone                      = "europe-west6-b"
  tags                      = ["ssh"]
  allow_stopping_for_update = true

  metadata = {
    enable-oslogin            = "TRUE"
    gce-container-declaration = "spec:\n  containers:\n    - name: node\n      image: 'gcr.io/stride-nodes/testnet:droplet_node2'\n      stdin: false\n      tty: false\n  restartPolicy: Always\n"
  }
  boot_disk {
    initialize_params {
      image = "cos-cloud/cos-97-lts"
    }
  }

  network_interface {
    network = "default"
    access_config {
      nat_ip = google_compute_address.node2.address
    }
  }

  service_account {
    scopes = ["https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring.write",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/service.management.readonly",
    "https://www.googleapis.com/auth/trace.append"]
  }
}

resource "google_compute_instance" "droplet-node3" {
  name                      = "droplet-node3"
  machine_type              = "e2-standard-4"
  zone                      = "us-east4-b"
  tags                      = ["ssh"]
  allow_stopping_for_update = true

  metadata = {
    enable-oslogin            = "TRUE"
    gce-container-declaration = "spec:\n  containers:\n    - name: node\n      image: 'gcr.io/stride-nodes/testnet:droplet_node3'\n      stdin: false\n      tty: false\n  restartPolicy: Always\n"
  }
  boot_disk {
    initialize_params {
      image = "cos-cloud/cos-97-lts"
    }
  }

  network_interface {
    network = "default"
    access_config {
      nat_ip = google_compute_address.node3.address
    }
  }

  service_account {
    scopes = ["https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring.write",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/service.management.readonly",
    "https://www.googleapis.com/auth/trace.append"]
  }
}

resource "google_compute_instance" "droplet-seed" {
  name                      = "droplet-seed"
  machine_type              = "e2-standard-4"
  zone                      = "us-west1-b"
  tags                      = ["ssh"]
  allow_stopping_for_update = true

  metadata = {
    enable-oslogin            = "TRUE"
    gce-container-declaration = "spec:\n  containers:\n    - name: node\n      image: 'gcr.io/stride-nodes/testnet:droplet_seed'\n      stdin: false\n      tty: false\n  restartPolicy: Always\n"
  }
  boot_disk {
    initialize_params {
      image = "cos-cloud/cos-97-lts"
    }
  }

  network_interface {
    network = "default"
    access_config {
      nat_ip = google_compute_address.seed.address
    }
  }

  service_account {
    scopes = ["https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring.write",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/service.management.readonly",
    "https://www.googleapis.com/auth/trace.append"]
  }
}


variable "regions" {
  type    = list(string)
  default = ["us-central1"]
}
variable "deployment_name" {
  type    = string
  default = "testnet"
}
variable "chain_name" {
  type    = string
  default = "stride"
}

variable "num_nodes" {
  type    = number
  default = 3
}

locals {
  node_names = [
    for i in range(1, var.num_nodes + 1) : "${var.chain_name}-node${i}"
  ]
}

module "node-containers" {
  source  = "terraform-google-modules/container-vm/google"
  version = "~> 2.0"

  count = length(local.node_names)
  container = {
    image = "gcr.io/stride-nodes/${var.deployment_name}:${local.node_names[count.index]}"
  }
  restart_policy = "Always"
}

resource "google_compute_address" "node-addresses" {
  count  = length(local.node_names)
  name   = local.node_names[count.index]
  region = var.regions[0]
}
resource "google_compute_instance" "stride-nodes" {
  count                     = length(local.node_names)
  name                      = local.node_names[count.index]
  machine_type              = "e2-standard-4"
  zone                      = "${element(var.regions, count.index)}-a"
  tags                      = ["ssh"]
  allow_stopping_for_update = true

  metadata = {
    enable-oslogin            = "TRUE"
    gce-container-declaration = module.node-containers[count.index].metadata_value
  }
  boot_disk {
    initialize_params {
      image = "cos-cloud/cos-97-lts"
    }
  }

  network_interface {
    network = "default"
    access_config {
      nat_ip = google_compute_address.node-addresses[count.index].address
    }
  }

  service_account {
    scopes = [
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring.write",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/service.management.readonly",
      "https://www.googleapis.com/auth/trace.append"
    ]
  }
}

resource "google_dns_managed_zone" "deployment-stridenet-zone" {
  name     = "${var.deployment_name}-stridenet"
  dns_name = "${var.deployment_name}.stridenet.co."
}
resource "google_dns_record_set" "parent-stridenet-name-service" {
  name = google_dns_managed_zone.deployment-stridenet-zone.dns_name
  type = "NS"
  ttl  = 300

  managed_zone = "stridenet"

  rrdatas = [
    "ns-cloud-a1.googledomains.com.", "ns-cloud-a2.googledomains.com.", "ns-cloud-a3.googledomains.com.", "ns-cloud-a4.googledomains.com."
  ]
}
resource "google_dns_record_set" "deployment-stridenet-name-service" {
  name = google_dns_managed_zone.deployment-stridenet-zone.dns_name
  type = "NS"
  ttl  = 300

  managed_zone = google_dns_managed_zone.deployment-stridenet-zone.name

  rrdatas = [
    "ns-cloud-a1.googledomains.com.", "ns-cloud-a2.googledomains.com.", "ns-cloud-a3.googledomains.com.", "ns-cloud-a4.googledomains.com."
  ]
}

resource "google_dns_record_set" "addresses" {
  count = length(local.node_names)
  name  = "${local.node_names[count.index]}.${google_dns_managed_zone.deployment-stridenet-zone.dns_name}"
  type  = "A"
  ttl   = 300

  managed_zone = google_dns_managed_zone.deployment-stridenet-zone.name

  rrdatas = [google_compute_instance.stride-nodes[count.index].network_interface[0].access_config[0].nat_ip]
}

resource "google_compute_firewall" "tendermint-firewall" {
  name    = "tendermint-firewall"
  network = "default"
  allow {
    protocol = "tcp"
    ports    = ["26656"]
  }

  source_tags = ["tendermint"]
}