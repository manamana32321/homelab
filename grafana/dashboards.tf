locals {
  folders = {
    infrastructure = { title = "Infrastructure", uid = "bfipmlvctzu2od" }
    observability  = { title = "Observability", uid = "cfipmm0pt5fr4e" }
    apps           = { title = "Apps", uid = "dfipmm3rm6qyoa" }
    security       = { title = "Security", uid = "afipmw4jsrlz4a" }
  }

  community_dashboards = {
    k8s-dashboard        = { folder = "infrastructure", gnet_id = 15661, revision = 2, uid = "24049b2d-f6e0-484b-88d5-a932d67348db", time_from = "now-7d" }
    node-exporter        = { folder = "infrastructure", gnet_id = 1860, revision = 37, job = "node-exporter" }
    smartctl-exporter    = { folder = "infrastructure", gnet_id = 22604, revision = 2 }
    openwrt              = { folder = "infrastructure", gnet_id = 11147, revision = 1, job = "openwrt" }
    blackbox-http-prober = { folder = "observability", gnet_id = 13659, revision = 1 }
    loki-stack           = { folder = "observability", gnet_id = 14055, revision = 5 }
    argocd               = { folder = "apps", gnet_id = 14584, revision = 1 }
    minecraft            = { folder = "apps", url = "https://raw.githubusercontent.com/sladkoff/minecraft-prometheus-exporter/v3.1.2/dashboards/minecraft-server-dashboard.json", uid = "0bf8d9cd-4848-4701-9e57-03aa528e87a8" }
    crowdsec-overview = {
      folder = "security"
      url    = "https://raw.githubusercontent.com/crowdsecurity/grafana-dashboards/0d34f68/dashboards_v5/Crowdsec%20Overview.json"
      panel_query = {
        8 = {
          expr         = "sum(increase({__name__=~\"cs_.+source_hits_total|cs_cloudwatch_stream_hits_total\"}[$__interval])) by (datasource_type, acquis_type)"
          legendFormat = "{{datasource_type}}/{{acquis_type}}"
        }
      }
    }
  }

  dashboards = { for k, v in local.community_dashboards : k => {
    folder      = v.folder
    url         = try(v.url, "https://grafana.com/api/dashboards/${v.gnet_id}/revisions/${v.revision}/download")
    uid         = try(v.uid, null)
    time_from   = try(v.time_from, null)
    job         = try(v.job, null)
    panel_query = jsonencode(try(v.panel_query, {}))
  } }

  dashboard_json = { for k, v in local.dashboards : k => jsondecode(
    replace(replace(data.http.dashboard[k].response_body, "$${DS_LOKI}", "Loki"), "/\\$\\{DS_[A-Za-z0-9_-]+\\}/", "Prometheus")
  ) }
}

import {
  for_each = local.folders
  to       = grafana_folder.this[each.key]
  id       = each.value.uid
}

resource "grafana_folder" "this" {
  for_each = local.folders
  title    = each.value.title
  uid      = each.value.uid
}

data "http" "dashboard" {
  for_each = local.dashboards
  url      = each.value.url
}

resource "grafana_dashboard" "community" {
  for_each  = local.dashboards
  folder    = grafana_folder.this[each.value.folder].uid
  overwrite = true

  config_json = jsonencode(merge(local.dashboard_json[each.key], {
    uid  = each.value.uid != null ? each.value.uid : local.dashboard_json[each.key].uid
    time = each.value.time_from != null ? merge(local.dashboard_json[each.key].time, { from = each.value.time_from }) : local.dashboard_json[each.key].time
    panels = [for p in local.dashboard_json[each.key].panels : jsondecode(
      contains(keys(jsondecode(each.value.panel_query)), tostring(p.id))
      ? jsonencode(merge(p, { targets = [merge(p.targets[0], jsondecode(each.value.panel_query)[tostring(p.id)])] }))
      : jsonencode(p)
    )]
    templating = merge(local.dashboard_json[each.key].templating, {
      list = [for t in local.dashboard_json[each.key].templating.list : jsondecode(
        t.name == "job" && each.value.job != null
        ? jsonencode(merge(t, { current = { selected = true, text = each.value.job, value = each.value.job } }))
        : jsonencode(t)
      )]
    })
  }))
}
