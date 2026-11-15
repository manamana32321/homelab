output "kiosk_j2_playlist_url" {
  value = "${var.grafana_url}/playlists/play/${split(":", grafana_playlist.kiosk_j2.id)[1]}"
}
