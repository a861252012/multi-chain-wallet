import json
import os
import stat
from pathlib import Path
import shlex
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SHA = "a" * 40
DIGEST = "sha256:" + "b" * 64
OLD = "ghcr.io/a861252012/multi-chain-wallet@sha256:" + "d" * 64


class DeployTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="wallet deploy fixture-")
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name)
        self.bin = self.base / "bin"
        self.bin.mkdir()
        self.env = {**os.environ, "FIXTURE": str(self.base), "FIXTURE_MAIN": SHA}
        source = (ROOT / "scripts/deploy/deploy.sh").read_text()
        source = source.replace("readonly base=/opt/multi-chain-wallet", "readonly base=" + shlex.quote(str(self.base)))
        source = source.replace("export PATH=/usr/sbin:/usr/bin:/sbin:/bin", "export PATH=" + shlex.quote(str(self.bin) + os.pathsep + os.environ["PATH"]))
        source = source.replace("/usr/local/bin/cosign", shlex.quote(str(self.bin / "cosign")))
        self.script = self.base / "deploy.sh"
        self.script.write_text(source)
        (self.base / "current-image").write_text(OLD + "\n")
        (self.base / ".env").write_text("PUBLIC_ORIGIN=https://wallet.example\n")
        (self.base / "compose.demo.yaml").write_text("services: {}\n")
        (self.base / "verify-demo.py").write_text('import os,sys\nfrom pathlib import Path\nactive=Path(os.environ["FIXTURE"]+"/active").read_text()\nsys.exit(1 if os.getenv("FAIL_SMOKE") and "'+ DIGEST +'" in active else 0)\n')
        fake = '''#!/usr/bin/env python3
import json,os,sys,subprocess
from pathlib import Path
base=Path(os.environ["FIXTURE"])
command=Path(sys.argv[0]).name
args=sys.argv[1:]
with (base/"calls").open("a") as f: f.write(json.dumps([command,args,os.getenv("APP_IMAGE","")])+"\\n")
if command=="timeout":
 assert args[:2]==["--signal=KILL", "60"]
 sys.exit(subprocess.run(args[2:], timeout=60).returncode)
if command=="curl":
 if os.getenv("FAIL_MAIN_AFTER_PULL") and (base/"pulled").exists(): sys.exit(22)
 print(json.dumps({"sha":(base/"moved-main").read_text() if (base/"moved-main").exists() else os.environ["FIXTURE_MAIN"]}))
if command=="cosign":
 if os.environ.get("TUF_ROOT") != str(base/".sigstore"): sys.exit(2)
 expected={"--certificate-identity":"https://github.com/a861252012/multi-chain-wallet/.github/workflows/verify.yml@refs/heads/main", "--certificate-oidc-issuer":"https://token.actions.githubusercontent.com", "--certificate-github-workflow-repository":"a861252012/multi-chain-wallet", "--certificate-github-workflow-ref":"refs/heads/main", "--certificate-github-workflow-trigger":"push", "--certificate-github-workflow-sha":os.environ["FIXTURE_MAIN"]}
 if args[0]!="verify" or any(k not in args or args[args.index(k)+1]!=v for k,v in expected.items()): sys.exit(2)
 if args[-1]!="ghcr.io/a861252012/multi-chain-wallet@sha256:"+"b"*64: sys.exit(2)
 if os.getenv("BAD_SIGNATURE"): sys.exit(1)
if command=="docker":
 if args[0]=="pull": (base/"pulled").touch()
 if args[0]=="pull" and os.getenv("MOVE_MAIN"): (base/"moved-main").write_text("f"*40)
 if args[0]=="inspect": print("2"*64 if (base/"active").exists() else "1"*64)
 if args[0]=="logs":
  print("failed-release" if args[-1]=="2"*64 else "previous-release", file=sys.stderr)
  if os.getenv("FAIL_LOGS") or (os.getenv("FAIL_NEW_LOGS") and args[-1]=="2"*64): sys.exit(1)
 if args[:2]==["image","inspect"]:
  print("c"*40 if args[-1].endswith("d"*64) else ("e"*40 if os.getenv("BAD_LABEL") else "a"*40))
 if args[0]=="compose":
  if "config" in args and os.getenv("FAIL_CONFIG"): sys.exit(1)
  if "up" in args:
   (base/"active").write_text(os.environ["APP_IMAGE"])
   old=os.environ["APP_IMAGE"].endswith("d"*64)
   if os.getenv("FAIL_UP") and not old: sys.exit(1)
   if os.getenv("FAIL_ROLLBACK") and old: sys.exit(1)
  if "ps" in args: print("fixture-container")
'''
        for name in ["docker", "curl", "flock", "cosign", "timeout"]:
            p = self.bin / name
            p.write_text(fake)
            p.chmod(0o755)

    def run_deploy(self, *args, **env):
        return subprocess.run(["bash", str(self.script), *(args or (DIGEST, SHA))], env={**self.env, **env}, text=True, capture_output=True)

    def calls(self):
        p = self.base / "calls"
        return [json.loads(x) for x in p.read_text().splitlines()] if p.exists() else []

    def test_success_records_exact_image(self):
        result = self.run_deploy()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.base/"current-image").read_text().strip(), "ghcr.io/a861252012/multi-chain-wallet@"+DIGEST)

    def test_signature_failure_never_pulls_or_stops(self):
        result = self.run_deploy(BAD_SIGNATURE="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(x[0] == "docker" for x in self.calls()))
        self.assertEqual((self.base/"current-image").read_text().strip(), OLD)

    def test_missing_verifier_fails_closed(self):
        (self.bin/"cosign").unlink()
        self.assertNotEqual(self.run_deploy().returncode, 0)
        self.assertFalse(any(x[0] == "docker" for x in self.calls()))

    def test_stale_commit_never_stops_service(self):
        self.assertNotEqual(self.run_deploy(FIXTURE_MAIN="f"*40).returncode, 0)
        self.assertFalse(any(x[0]=="docker" for x in self.calls()))

    def test_bad_image_label_never_stops_service(self):
        self.assertNotEqual(self.run_deploy(BAD_LABEL="1").returncode, 0)
        self.assertFalse(any("stop" in x[1] for x in self.calls()))

    def test_invalid_config_never_stops_service(self):
        self.assertNotEqual(self.run_deploy(FAIL_CONFIG="1").returncode, 0)
        self.assertFalse(any("stop" in x[1] for x in self.calls()))

    def test_failed_start_rolls_back_but_reports_failure(self):
        result = self.run_deploy(FAIL_UP="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Previous image restored", result.stderr)
        self.assertEqual((self.base/"active").read_text(), OLD)
        self.assertEqual((self.base/"current-image").read_text().strip(), OLD)
        self.assertFalse(any("down" in x[1] for x in self.calls()))

    def test_failed_smoke_rolls_back(self):
        result = self.run_deploy(FAIL_SMOKE="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Previous image restored", result.stderr)
        self.assertEqual((self.base/"active").read_text(), OLD)

    def test_failed_rollback_is_explicit(self):
        result = self.run_deploy(FAIL_UP="1", FAIL_ROLLBACK="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("ROLLBACK FAILED", result.stderr)

    def test_first_failure_stops_failed_release(self):
        (self.base/"current-image").unlink()
        result = self.run_deploy(FAIL_UP="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("no previous image", result.stderr)
        self.assertFalse((self.base/"current-image").exists())

    def test_injection_rejected_before_commands(self):
        for digest in [DIGEST+";touch /tmp/never", "latest", "sha256:"+"g"*64]:
            self.assertEqual(self.run_deploy(digest, SHA).returncode, 2)
        self.assertEqual(self.calls(), [])

    def test_main_moves_during_pull_without_stopping_or_rollback(self):
        result = self.run_deploy(MOVE_MAIN="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any("stop" in c[1] or "up" in c[1] for c in self.calls()))
        self.assertEqual((self.base/"current-image").read_text().strip(), OLD)

    def test_previous_log_is_private_and_captured_before_recreate(self):
        result = self.run_deploy()
        self.assertEqual(result.returncode, 0, result.stderr)
        logs = list(self.base.glob("container-" + "1"*64 + ".*.log"))
        self.assertEqual(len(logs), 1)
        self.assertEqual(logs[0].read_text(), "previous-release\n")
        self.assertEqual(stat.S_IMODE(logs[0].stat().st_mode), 0o600)
        calls = self.calls()
        capture = next(i for i,c in enumerate(calls) if c[1][0]=="logs")
        recreate = next(i for i,c in enumerate(calls) if "up" in c[1])
        stopped = next(i for i,c in enumerate(calls) if "stop" in c[1])
        self.assertLess(stopped, capture)
        self.assertLess(capture, recreate)
        self.assertEqual(calls[capture][1][1:4], ["--timestamps", "--tail", "1000"])

    def test_log_capture_failure_restores_previous_app(self):
        result = self.run_deploy(FAIL_LOGS="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Previous image restored", result.stderr)
        self.assertEqual((self.base/"active").read_text(), OLD)
        self.assertFalse(any("up" in c[1] and c[2].endswith(DIGEST) for c in self.calls()))

    def test_rollback_preserves_failed_log(self):
        result = self.run_deploy(FAIL_SMOKE="1")
        self.assertNotEqual(result.returncode, 0)
        logs = list(self.base.glob("container-" + "2"*64 + ".*.log"))
        self.assertEqual(len(logs), 1)
        self.assertEqual(logs[0].read_text(), "failed-release\n")
        calls = self.calls()
        capture = next(i for i,c in enumerate(calls) if c[1][0]=="logs" and c[1][-1]=="2"*64)
        restore = next(i for i,c in enumerate(calls) if "up" in c[1] and c[2]==OLD)
        self.assertLess(capture, restore)
        self.assertEqual((self.base/"active").read_text(), OLD)

    def test_rollback_continues_when_failed_log_capture_fails(self):
        result = self.run_deploy(FAIL_SMOKE="1", FAIL_NEW_LOGS="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Previous image restored", result.stderr)
        self.assertIn("Could not preserve failed container logs", result.stderr)
        self.assertEqual((self.base/"active").read_text(), OLD)

    def test_main_query_failure_after_pull_leaves_app_untouched(self):
        result = self.run_deploy(FAIL_MAIN_AFTER_PULL="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any("stop" in c[1] or "up" in c[1] or c[1][0]=="logs" for c in self.calls()))
        self.assertNotIn("Deployment failed", result.stderr)
        self.assertEqual((self.base/"current-image").read_text().strip(), OLD)
