#!/usr/bin/env python3
"""Bounded host diagnostics. Never read wallet files, environment or HTTP bodies."""
import json
import logging
from logging.handlers import RotatingFileHandler
from pathlib import Path
import shutil
import subprocess
import time
import urllib.error
import urllib.request


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def health(url, host=None):
    started = time.monotonic()
    headers = {"User-Agent": "multi-chain-wallet-health/1.0"}
    if host:
        headers["Host"] = host
    request = urllib.request.Request(url, headers=headers)
    try:
        response = urllib.request.build_opener(
            urllib.request.ProxyHandler({}), NoRedirect()).open(request, timeout=4)
    except urllib.error.HTTPError as error:
        response = error
    except Exception as error:
        # Exception messages may contain URLs or other sensitive data.
        return {"error": type(error).__name__, "ms": round((time.monotonic() - started) * 1000)}
    with response:
        version = response.headers.get("X-App-Version", "")
        ray = response.headers.get("CF-Ray", "")
        return {"status": response.code,
                "revision": version if len(version) == 40 and all(c in "0123456789abcdef" for c in version) else None,
                "cf_ray": ray if len(ray) <= 80 and all(c.isalnum() or c == "-" for c in ray) else None,
                "ms": round((time.monotonic() - started) * 1000)}


def command(args):
    try:
        result = subprocess.run(args, capture_output=True, text=True, timeout=5, check=False)
        if result.returncode:
            return {"error": "exit", "code": result.returncode}
        return {"output": result.stdout[:16384]}
    except (OSError, subprocess.TimeoutExpired) as error:
        return {"error": type(error).__name__}


def snapshot():
    memory = {}
    for line in Path("/proc/meminfo").read_text().splitlines():
        key, value = line.split(":", 1)
        if key in ("MemTotal", "MemAvailable", "SwapTotal", "SwapFree"):
            memory[key + "KiB"] = int(value.split()[0])
    disk = shutil.disk_usage("/var/log")
    now = int(time.time())
    result = {"time": now, "boot_id": Path("/proc/sys/kernel/random/boot_id").read_text().strip(),
              "uptime_seconds": float(Path("/proc/uptime").read_text().split()[0]),
              "load": Path("/proc/loadavg").read_text().split()[:3],
              "memory": memory, "log_disk_free_bytes": disk.free,
              "services": command(["systemctl", "show", "cloudflared.service", "docker.service",
                                    "wallet-demo-update.service", "-p", "Id", "-p", "ActiveState", "-p", "SubState",
                                    "-p", "Result", "-p", "NRestarts", "-p", "MemoryCurrent", "-p", "MemoryPeak"])}
    containers = command(["docker", "ps", "-aq", "--filter", "label=com.docker.compose.project=multi-chain-wallet-demo",
                          "--filter", "label=com.docker.compose.service=app"])
    result["containers"] = []
    for container in containers.get("output", "").splitlines()[:4]:
        if not container or any(c not in "0123456789abcdef" for c in container):
            continue
        result["containers"].append({"id": container, "state": command([
            "docker", "inspect", "--format",
            '{"status":{{json .State.Status}},"oom":{{.State.OOMKilled}},"exit":{{.State.ExitCode}},'
            '"restarts":{{.RestartCount}},"started":{{json .State.StartedAt}},"finished":{{json .State.FinishedAt}}}', container]),
            "resources": command(["docker", "stats", "--no-stream", "--format",
                                   '{{.CPUPerc}} {{.MemUsage}} {{.MemPerc}} {{.PIDs}}', container])})
    if "error" in containers:
        result["container_error"] = containers
    # Bounded time window catches OOM/die/restart even after automatic recovery.
    events = command(["docker", "events", "--since", str(now - 120), "--until", str(now),
                      "--filter", "type=container", "--filter", "label=com.docker.compose.project=multi-chain-wallet-demo",
                      "--format", '{{.Time}} {{.Action}} {{.Actor.ID}}'])
    result["events"] = []
    if "error" in events:
        result["event_error"] = events
    for line in events.get("output", "").splitlines()[:64]:
        parts = line.split()
        if (len(parts) == 3 and parts[0].isdigit() and parts[1] in ("oom", "die", "restart", "start", "stop", "kill")
                and all(c in "0123456789abcdef" for c in parts[2])):
            result["events"].append({"time": int(parts[0]), "action": parts[1], "id": parts[2][:12]})
    result["local"] = health("http://127.0.0.1:8090/healthz", "wallet.tedlin.fyi")
    result["public"] = health("https://wallet.tedlin.fyi/healthz")
    return result


def record(value, path="/var/log/wallet-demo/health.jsonl", max_bytes=2 * 1024 * 1024):
    handler = RotatingFileHandler(path, maxBytes=max_bytes, backupCount=3, encoding="utf-8")
    try:
        handler.emit(logging.LogRecord("health", logging.INFO, "", 0, json.dumps(value), (), None))
    finally:
        handler.close()


if __name__ == "__main__":
    record({"time": int(time.time()), "phase": "started"})
    try:
        record(snapshot())
    except Exception as error:
        record({"time": int(time.time()), "phase": "failed", "error": type(error).__name__})
        raise SystemExit(1)
