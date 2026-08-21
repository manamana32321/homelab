#!/usr/bin/env bash
#
# r8169 TX 큐 행 자동 복구.
#
# 커널 netdev watchdog 이 "NETDEV WATCHDOG: transmit queue N timed out" 을 찍으면
# NIC 이 패킷을 물고 놓지 않는 상태다. r8169 는 자체 리셋(rtl_reset_work)에 실패하면
# 스스로 못 빠져나온다 (2026-08-21 인시던트: 21분간 153회 실패, 하드 리셋으로 종료).
#
# 3단계로 올라간다:
#   1. 링크 바운스   — netdev 를 유지한 채 칩만 재초기화. flannel.1 VXLAN 이 살아남는다.
#   2. 모듈 리로드   — 칩을 완전히 재적재. netdev 가 사라지므로 flannel 재구성용 k3s 재시작 동반.
#   3. clean reboot  — 손으로 전원을 끊는 것보다 안전 (Longhorn 언마운트 + kine SQLite 체크포인트).
#
set -uo pipefail

IFACE="${IFACE:-enp3s0}"
MODULE="${MODULE:-r8169}"
GATEWAY="${GATEWAY:-192.168.0.1}"

# 전체 복구 사이클 간 최소 간격. 행 중에는 watchdog 이 ~5초마다 찍히므로 필수.
COOLDOWN_SEC="${COOLDOWN_SEC:-120}"
# 최초 실패 후 이 시간이 지나도록 살아나지 않으면 재부팅. 0 이면 재부팅 비활성.
REBOOT_AFTER_SEC="${REBOOT_AFTER_SEC:-300}"

# /run 은 부팅 시 비워지므로 "부팅당 1회 재부팅" 가드로 그대로 쓸 수 있다.
STATE_DIR="/run/nic-tx-watchdog"
REBOOTED_MARKER="$STATE_DIR/rebooted"
FIRST_FAILURE_FILE="$STATE_DIR/first-failure"

log() { printf '%s\n' "$*"; }

is_watchdog_line() {
    local line="$1"
    [[ "$line" == *"NETDEV WATCHDOG"* && "$line" == *"$IFACE"* ]]
}

tx_packets() {
    cat "/sys/class/net/$IFACE/statistics/tx_packets" 2>/dev/null || echo 0
}

carrier_up() {
    [[ "$(cat "/sys/class/net/$IFACE/carrier" 2>/dev/null || echo 0)" == "1" ]]
}

# 캐리어가 올라왔고, 게이트웨이가 응답하거나 최소한 TX 카운터가 전진하면 살아난 것으로 본다.
# 게이트웨이 단독 판정은 공유기가 같이 죽었을 때 오탐이 나므로 TX 전진을 함께 본다.
link_healthy() {
    carrier_up || return 1
    ping -c1 -W2 -I "$IFACE" "$GATEWAY" >/dev/null 2>&1 && return 0
    local before after
    before="$(tx_packets)"
    sleep 5
    after="$(tx_packets)"
    (( after > before ))
}

wait_healthy() {
    local deadline=$(( $(date +%s) + $1 ))
    while (( $(date +%s) < deadline )); do
        link_healthy && return 0
        sleep 2
    done
    return 1
}

bounce_link() {
    log "[1/3] 링크 바운스: $IFACE down/up"
    ip link set dev "$IFACE" down || log "  ip link down 실패"
    sleep 3
    ip link set dev "$IFACE" up || log "  ip link up 실패"
}

reload_module() {
    log "[2/3] 모듈 리로드: $MODULE"
    if ! modprobe -r "$MODULE"; then
        log "  modprobe -r 실패 — 모듈이 점유 중일 수 있음"
        return 1
    fi
    sleep 3
    modprobe "$MODULE" || { log "  modprobe 재적재 실패"; return 1; }
    # netdev 가 새로 생기므로 systemd-networkd 가 DHCP 를 다시 물어야 한다.
    networkctl reconfigure "$IFACE" >/dev/null 2>&1 || true
    return 0
}

# flannel.1 은 vxlan 언더레이로 enp3s0 을 직접 참조한다. 모듈 리로드로 netdev 가
# 사라지면 터널도 무너지므로, NIC 이 돌아와도 k3s 를 재시작해야 파드 네트워킹이 산다.
rebuild_overlay() {
    log "  flannel VXLAN 재구성을 위해 k3s 재시작"
    systemctl restart k3s || log "  k3s 재시작 실패"
}

record_first_failure() {
    [[ -f "$FIRST_FAILURE_FILE" ]] || date +%s > "$FIRST_FAILURE_FILE"
}

clear_first_failure() {
    rm -f "$FIRST_FAILURE_FILE"
}

maybe_reboot() {
    (( REBOOT_AFTER_SEC > 0 )) || { log "  재부팅 폴백 비활성 (REBOOT_AFTER_SEC=0)"; return; }
    if [[ -f "$REBOOTED_MARKER" ]]; then
        log "  이번 부팅에서 이미 자동 재부팅함 — 반복 방지를 위해 중단. 수동 개입 필요"
        return
    fi
    local first now
    first="$(cat "$FIRST_FAILURE_FILE" 2>/dev/null || echo 0)"
    now="$(date +%s)"
    if (( first > 0 && now - first >= REBOOT_AFTER_SEC )); then
        log "[3/3] $(( now - first ))초간 복구 실패 — clean reboot 실행"
        : > "$REBOOTED_MARKER"
        systemctl reboot
    else
        log "  누적 실패 $(( now - first ))초 < ${REBOOT_AFTER_SEC}초 — 재부팅 보류"
    fi
}

recover() {
    log "=== $IFACE TX 큐 행 감지 — 복구 시작 ==="
    record_first_failure

    bounce_link
    if wait_healthy 20; then
        log "링크 바운스로 복구됨"
        clear_first_failure
        return
    fi

    if reload_module && wait_healthy 60; then
        log "모듈 리로드로 복구됨"
        rebuild_overlay
        clear_first_failure
        return
    fi

    log "1·2단계 복구 실패"
    maybe_reboot
}

self_test() {
    local sample="r8169 0000:03:00.0 $IFACE: NETDEV WATCHDOG: CPU: 7: transmit queue 0 timed out 5567 ms"
    local benign="r8169 0000:03:00.0 $IFACE: rtl_counters_cond == 1 (loop: 1000, delay: 10)."
    local other="r8169 0000:03:00.0 eth9: NETDEV WATCHDOG: CPU: 1: transmit queue 0 timed out 5000 ms"
    local rc=0

    is_watchdog_line "$sample"  && log "OK   watchdog 라인 매칭"          || { log "FAIL watchdog 라인 미매칭"; rc=1; }
    is_watchdog_line "$benign"  && { log "FAIL 무해한 라인을 오탐"; rc=1; } || log "OK   무해한 라인 무시"
    is_watchdog_line "$other"   && { log "FAIL 다른 인터페이스를 오탐"; rc=1; } || log "OK   다른 인터페이스 무시"

    carrier_up   && log "OK   $IFACE 캐리어 up"        || { log "FAIL $IFACE 캐리어 down"; rc=1; }
    link_healthy && log "OK   $GATEWAY 연결 정상"      || { log "FAIL $GATEWAY 연결 실패"; rc=1; }

    log "self-test 종료 (rc=$rc) — 실제 복구 동작은 수행하지 않음"
    return $rc
}

main() {
    if [[ "${1:-}" == "--self-test" ]]; then
        self_test
        exit $?
    fi

    mkdir -p "$STATE_DIR"
    log "$IFACE TX 큐 행 감시 시작 (모듈=$MODULE, 게이트웨이=$GATEWAY, 재부팅 임계=${REBOOT_AFTER_SEC}초)"

    local last_cycle=0
    while read -r line; do
        is_watchdog_line "$line" || continue
        now="$(date +%s)"
        if (( now - last_cycle < COOLDOWN_SEC )); then
            continue
        fi
        last_cycle="$now"
        recover
    done < <(journalctl -k -f -n0 -o cat)

    log "커널 저널 스트림 종료 — 서비스 재시작 필요"
    exit 1
}

main "$@"
