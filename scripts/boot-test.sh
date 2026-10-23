#!/usr/bin/env bash
# Boots an h3xdOS ISO headless in QEMU, BIOS first and then UEFI, and passes
# when the serial console shows the login prompt.
set -euo pipefail

readonly host_out_dir=/mnt/a/h3xdOS/out
readonly login_prompt='h3xdos login:'
readonly kvm_timeout_s=240
readonly tcg_timeout_s=900
readonly poll_interval_s=2
readonly vm_memory_mb=4096
readonly vm_cpus=2
# Headless by default; BOOT_TEST_DISPLAY=gtk,zoom-to-fit=on shows the VM through WSLg.
readonly display=${BOOT_TEST_DISPLAY:-none}

iso=${1:?usage: boot-test.sh <iso>}
if [[ ! -f $iso ]]; then
    echo "boot-test.sh: no such ISO: $iso" >&2
    exit 1
fi

if [[ -w /dev/kvm ]]; then
    accel=(-accel kvm -cpu host)
    timeout_s=$kvm_timeout_s
else
    accel=(-accel tcg -cpu max)
    timeout_s=$tcg_timeout_s
fi

ovmf_code=$(pacman -Qlq edk2-ovmf | grep '/x64/OVMF_CODE\.4m\.fd$')
ovmf_vars=$(pacman -Qlq edk2-ovmf | grep '/x64/OVMF_VARS\.4m\.fd$')

scratch_dir=$(mktemp -d /var/tmp/h3xdos-boot-test.XXXXXX)
qemu_pid=

stop_qemu() {
    [[ -n $qemu_pid ]] || return 0
    # QEMU may already have exited, and a terminated QEMU exits non-zero;
    # the verdict comes from the serial log, not from its exit status.
    kill -- "$qemu_pid" 2>/dev/null || true
    wait "$qemu_pid" 2>/dev/null || true
    qemu_pid=
}
trap 'stop_qemu; rm -rf -- "$scratch_dir"' EXIT

boot() {
    local mode=$1
    shift
    local serial_log=$host_out_dir/boot-$mode.log
    local deadline=$(( SECONDS + timeout_s ))

    : > "$serial_log"
    qemu-system-x86_64 "${accel[@]}" -machine q35 -m "$vm_memory_mb" -smp "$vm_cpus" \
        -name "h3xdOS boot test $mode" -display "$display" -monitor none -no-reboot \
        -serial "file:$serial_log" \
        -drive "file=$iso,media=cdrom,readonly=on" -boot d "$@" &
    qemu_pid=$!

    while (( SECONDS < deadline )); do
        if grep -qF -- "$login_prompt" "$serial_log"; then
            stop_qemu
            echo "boot-test.sh: $mode PASS ($serial_log)"
            return 0
        fi
        if ! kill -0 "$qemu_pid" 2>/dev/null; then
            stop_qemu
            echo "boot-test.sh: $mode FAIL, QEMU exited before '$login_prompt' ($serial_log)" >&2
            return 1
        fi
        sleep "$poll_interval_s"
    done
    stop_qemu
    echo "boot-test.sh: $mode FAIL, no '$login_prompt' within ${timeout_s}s ($serial_log)" >&2
    return 1
}

failed=0
boot bios || failed=1

cp -- "$ovmf_vars" "$scratch_dir/OVMF_VARS.4m.fd"
boot uefi \
    -drive "if=pflash,format=raw,unit=0,readonly=on,file=$ovmf_code" \
    -drive "if=pflash,format=raw,unit=1,file=$scratch_dir/OVMF_VARS.4m.fd" \
    || failed=1

exit "$failed"
