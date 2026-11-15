locals {
  kiosk_j2_dashboards = [
    { name = "K8S Dashboard", uid = grafana_dashboard.community["k8s-dashboard"].uid },
    { name = "Node Exporter Full", uid = grafana_dashboard.community["node-exporter"].uid },
    { name = "OpenWRT", uid = grafana_dashboard.community["openwrt"].uid },
    { name = "Blackbox Exporter (HTTP prober)", uid = grafana_dashboard.community["blackbox-http-prober"].uid },
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
