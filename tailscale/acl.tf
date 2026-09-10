# Tailscale 정책 파일(ACL) 을 Terraform 이 소유한다.
#
# tailscale_acl 리소스는 ACL 섹션만이 아니라 policy file 전체를 덮어쓴다.
# tagOwners / autoApprovers 전용 속성이 없어서 정책 전체를 여기에 선언한다.
# 콘솔에서 직접 수정하면 다음 apply 에서 되돌아간다 (의도된 동작).
locals {
  policy = {
    # tailnet 기본 정책 그대로. 홈랩 접근을 붙이는 것이 목적이므로
    # 기존 접근 범위는 건드리지 않는다.
    grants = [
      {
        src = ["*"]
        dst = ["*"]
        ip  = ["*"]
      },
    ]

    ssh = [
      {
        action = "check"
        src    = ["autogroup:member"]
        dst    = ["autogroup:self"]
        users  = ["autogroup:nonroot", "root"]
      },
    ]

    # k8s operator 가 쓰는 태그.
    #   tag:k8s-operator = operator 파드 자신
    #   tag:k8s          = operator 가 만들어내는 proxy 파드 (= subnet router)
    # operator 가 tag:k8s 의 owner 여야 proxy 디바이스를 대신 등록할 수 있다.
    tagOwners = {
      "tag:k8s-operator" = []
      "tag:k8s"          = ["tag:k8s-operator"]
    }

    # subnet router 가 광고하는 홈랩 LAN 을 자동 승인한다.
    # 이게 없으면 Connector 가 라우트를 광고해도 콘솔에서 수동 승인하기 전까지
    # pending 상태로 멈춰서 홈랩에 도달하지 못한다.
    autoApprovers = {
      routes = {
        (var.homelab_subnet) = ["tag:k8s"]
      }
    }
  }
}

resource "tailscale_acl" "homelab" {
  acl = jsonencode(local.policy)

  # 위 local.policy 는 현재 tailnet 정책(기본 grants + ssh)을 그대로 재현한 뒤
  # tagOwners 와 autoApprovers 만 더한 것이다. 덮어써도 잃는 설정이 없다.
  overwrite_existing_content = true
}
