$HomeGatewayMac = 'B0-38-6C-39-41-2B'
$Tailscale = 'C:\Program Files\Tailscale\tailscale.exe'

$mac = $null
foreach ($i in 1..10) {
  $route = Get-NetRoute -DestinationPrefix 0.0.0.0/0 -ErrorAction SilentlyContinue |
    Where-Object InterfaceAlias -ne 'Tailscale' |
    Sort-Object RouteMetric |
    Select-Object -First 1
  if ($route) {
    Test-Connection -ComputerName $route.NextHop -Count 1 -Quiet | Out-Null
    $mac = (Get-NetNeighbor -InterfaceIndex $route.InterfaceIndex -IPAddress $route.NextHop -ErrorAction SilentlyContinue).LinkLayerAddress
    if ($mac) { break }
  }
  Start-Sleep -Seconds 3
}

$accept = if ($mac -eq $HomeGatewayMac) { 'false' } else { 'true' }
& $Tailscale set "--accept-routes=$accept"
