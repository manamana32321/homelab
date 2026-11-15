locals {
  kiosk_j2_dashboards = [
    { name = "K8S Dashboard", uid = "24049b2d-f6e0-484b-88d5-a932d67348db" },
    { name = "Node Exporter Full", uid = "rYdddlPWk" },
    { name = "OpenWRT", uid = "fLi0yXAWk" },
    { name = "Blackbox Exporter (HTTP prober)", uid = "NEzutrbMk" },
    { name = "Crowdsec Overview", uid = "hjmZdB4nk" },
  ]
}

resource "grafana_playlist" "kiosk_j2" {
  name     = "kiosk-json-server-2"
  interval = "1m"

  dynamic "item" {
    for_each = local.kiosk_j2_dashboards
    content {
      order = item.key + 1
      type  = "dashboard_by_uid"
      value = item.value.uid
    }
  }
}
