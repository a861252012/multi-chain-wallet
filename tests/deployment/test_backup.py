import io
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import signal
import subprocess
import tarfile
import tempfile
import time
import unittest


ROOT = Path(__file__).resolve().parents[2]
CONTAINER = "multi-chain-wallet-demo-app-1"


class BackupTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="wallet backup fixture-")
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name).resolve()
        self.new = self.base / "new"
        self.old = self.base / "old"
        self.bin = self.base / "bin"
        for path in (self.new / "wallet", self.old / "wallet", self.bin):
            path.mkdir(parents=True)
        for name in (".env", "current-image", "compose.demo.yaml", "verify-demo.py"):
            (self.new / name).write_text("private-fixture-value\n")
        for label, path in (("old", self.old), ("new", self.new)):
            (path / "wallet" / "journal.json").write_text(label + "-journal")
            (path / "wallet" / ".lock").touch()
        (self.base / "running").write_text("true")
        self.set_mounts([self.mount(self.old / "wallet")])
        self.env = {
            **os.environ,
            "FIXTURE": str(self.base),
            "PATH": str(self.bin) + os.pathsep + os.environ["PATH"],
            "REAL_TAR": shutil.which("tar"),
        }
        source = (ROOT / "scripts/deploy/backup-to-mac.sh").read_text()
        source = source.replace("cd /opt/multi-chain-wallet", "cd " + shlex.quote(str(self.new)))
        source = source.replace("/opt/multi-chain-wallet/wallet", str(self.new / "wallet"))
        source = source.replace("/opt/testnet-wallet-lab/wallet", str(self.old / "wallet"))
        source = source.replace('keys="$HOME/.ssh/wallet-demo-secrets"', "keys=" + shlex.quote(str(self.base / "keys")))
        source = source.replace('backups="$HOME/Documents/wallet-demo-backups"', "backups=" + shlex.quote(str(self.base / "backups")))
        self.local = self.base / "backup.sh"
        self.local.write_text(source)
        remote = re.search(r"<<'REMOTE'[^\n]*\n(.*?)\nREMOTE\n", source, re.S).group(1)
        self.remote = self.base / "remote.sh"
        self.remote.write_text(remote)
        fake = r'''#!/usr/bin/env python3
import json, os, signal, subprocess, sys, time
from pathlib import Path
base = Path(os.environ["FIXTURE"])
command, args = Path(sys.argv[0]).name, sys.argv[1:]
with (base / "calls").open("a") as log:
    log.write(json.dumps([command, args]) + "\n")
if command == "docker":
    assert args[-1] == "multi-chain-wallet-demo-app-1", args
    if args[0] == "inspect":
        print((base / ("mounts" if args[2] == "{{json .Mounts}}" else "running")).read_text())
    elif args[0] == "stop":
        if os.getenv("FAIL_STOP"): sys.exit(17)
        (base / "running").write_text("false")
    elif args[0] == "start":
        (base / "restart-attempted").touch()
        os.write(2, b"restart diagnostic\n")
        if os.getenv("FAIL_START"): sys.exit(18)
        (base / "running").write_text("true")
elif command == "tar":
    if "-czf" in args and os.getenv("FAIL_ARCHIVE"): sys.exit(19)
    if "-czf" in args and os.getenv("BROKEN_PIPE_ARCHIVE"):
        (base / "archive-started").touch()
        signal.signal(signal.SIGPIPE, signal.SIG_DFL)
        while True: os.write(1, b"x" * 65536)
    if "-czf" in args and os.getenv("WAIT_FOR_HANGUP"):
        (base / "archive-started").touch()
        time.sleep(20)
    os.execv(os.environ["REAL_TAR"], [os.environ["REAL_TAR"], *args])
elif command == "timeout":
    assert args[:3] == ["--signal=TERM", "--kill-after=10s", "300s"]
    os.execvp(args[3], args[3:])
elif command == "ssh":
    sys.exit(subprocess.run(["bash"], input=sys.stdin.buffer.read()).returncode)
elif command == "age-keygen":
    if "-o" in args: Path(args[args.index("-o") + 1]).write_text("mock-key")
    else: print("mock-recipient")
elif command == "age":
    # Transport fixture only; this does not test the age cryptography.
    if "-d" in args:
        sys.stdout.buffer.write(Path(args[-1]).read_bytes())
    else:
        Path(args[args.index("-o") + 1]).write_bytes(sys.stdin.buffer.read())
'''
        for name in ("docker", "flock", "tar", "timeout", "ssh", "age", "age-keygen"):
            path = self.bin / name
            path.write_text(fake)
            path.chmod(0o755)

    @staticmethod
    def mount(source, **changes):
        return {"Type": "bind", "Source": str(source), "Destination": "/data/wallet", "RW": True, **changes}

    def set_mounts(self, mounts):
        (self.base / "mounts").write_text(json.dumps(mounts))

    def run_backup(self, local=False, **env):
        command = ["bash", str(self.local if local else self.remote)]
        if local:
            command.append("192.0.2.1")
        return subprocess.run(command, env={**self.env, **env}, capture_output=True)

    def calls(self):
        path = self.base / "calls"
        return [json.loads(line) for line in path.read_text().splitlines()] if path.exists() else []

    def mutations(self):
        return [args[0] for command, args in self.calls() if command == "docker" and args[0] in ("stop", "start")]

    def test_archives_actual_legacy_bind_and_current_configuration(self):
        result = self.run_backup()
        self.assertEqual(result.returncode, 0, result.stderr)
        with tarfile.open(fileobj=io.BytesIO(result.stdout), mode="r:gz") as archive:
            self.assertEqual(archive.extractfile("wallet/journal.json").read(), b"old-journal")
            self.assertEqual(archive.extractfile(".env").read(), b"private-fixture-value\n")
            self.assertNotIn("wallet/.lock", archive.getnames())
        self.assertEqual(self.mutations(), ["stop", "start"])
        self.assertEqual((self.base / "running").read_text(), "true")
        self.assertNotIn(b"private-fixture-value", result.stderr)

    def test_default_new_bind_is_supported(self):
        self.set_mounts([self.mount(self.new / "wallet")])
        result = self.run_backup()
        self.assertEqual(result.returncode, 0, result.stderr)
        with tarfile.open(fileobj=io.BytesIO(result.stdout), mode="r:gz") as archive:
            self.assertEqual(archive.extractfile("wallet/journal.json").read(), b"new-journal")

    def test_invalid_mounts_fail_before_stopping(self):
        valid = self.mount(self.old / "wallet")
        cases = [[], [valid, valid], [self.mount(self.old / "wallet", Type="volume")],
                 [self.mount(self.old / "wallet", RW=False)], [self.mount(self.base)],
                 [self.mount(self.old / "wallet", Destination="/another")], None]
        for mounts in cases:
            with self.subTest(mounts=mounts):
                self.set_mounts(mounts)
                result = self.run_backup()
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.mutations(), [])
                self.assertEqual(result.stdout, b"")
                self.assertNotIn(b"private-fixture-value", result.stderr)

    def test_symlink_or_missing_bind_fails_before_stopping(self):
        wallet = self.old / "wallet"
        wallet.rename(self.old / "saved-wallet")
        for symlink in (False, True):
            with self.subTest(symlink=symlink):
                if symlink:
                    wallet.symlink_to(self.old / "saved-wallet", target_is_directory=True)
                self.assertNotEqual(self.run_backup().returncode, 0)
                self.assertEqual(self.mutations(), [])

    def test_nested_mount_is_rejected_before_incomplete_backup(self):
        self.set_mounts([
            self.mount(self.old / "wallet"),
            self.mount(self.base / "other-keys", Destination="/data/wallet/keys"),
        ])
        result = self.run_backup()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.mutations(), [])
        self.assertEqual(result.stdout, b"")

    def test_stopped_application_is_not_started_by_backup(self):
        (self.base / "running").write_text("false")
        self.assertNotEqual(self.run_backup().returncode, 0)
        self.assertEqual(self.mutations(), [])

    def test_missing_configuration_fails_before_stopping(self):
        (self.new / "current-image").unlink()
        self.assertNotEqual(self.run_backup().returncode, 0)
        self.assertEqual(self.mutations(), [])

    def test_stop_failure_attempts_recovery_without_archiving(self):
        result = self.run_backup(FAIL_STOP="1")
        self.assertEqual(result.returncode, 17)
        self.assertEqual(self.mutations(), ["stop", "start"])
        self.assertFalse(any(command == "tar" for command, _ in self.calls()))

    def test_archive_failure_restarts_application_and_stays_failed(self):
        result = self.run_backup(FAIL_ARCHIVE="1")
        self.assertEqual(result.returncode, 19)
        self.assertEqual(self.mutations(), ["stop", "start"])
        self.assertEqual((self.base / "running").read_text(), "true")

    def test_restart_failure_is_explicit(self):
        result = self.run_backup(FAIL_START="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("restart=failed", (self.new / "backup-recovery.log").read_text())

    def test_recovery_log_failure_does_not_prevent_restart(self):
        (self.new / "backup-recovery.log").mkdir()
        result = self.run_backup(FAIL_ARCHIVE="1")
        self.assertEqual(result.returncode, 19)
        self.assertTrue((self.base / "restart-attempted").exists())
        self.assertEqual((self.base / "running").read_text(), "true")

    def test_regular_file_output_is_rejected_before_stopping(self):
        with (self.base / "unsafe-output").open("wb") as output:
            result = subprocess.run(["bash", str(self.remote)], env=self.env,
                                    stdout=output, stderr=subprocess.PIPE)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.mutations(), [])
        self.assertEqual((self.base / "unsafe-output").read_bytes(), b"")

    def interrupted_backup(self, hangup=False):
        env = {**self.env, "WAIT_FOR_HANGUP" if hangup else "BROKEN_PIPE_ARCHIVE": "1"}
        process = subprocess.Popen(["bash", str(self.remote)], env=env,
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                   start_new_session=True)
        try:
            deadline = time.monotonic() + 5
            while not (self.base / "archive-started").exists():
                self.assertIsNone(process.poll(), "backup exited before archive interruption")
                self.assertLess(time.monotonic(), deadline, "archive did not start")
                time.sleep(0.01)
            process.stdout.close()
            process.stderr.close()
            if hangup:
                os.killpg(process.pid, signal.SIGHUP)
            return process.wait(timeout=10)
        finally:
            if process.poll() is None:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait(timeout=5)
            process.stdout.close()
            process.stderr.close()

    def test_real_broken_pipe_restarts_after_both_parent_streams_close(self):
        self.assertNotEqual(self.interrupted_backup(), 0)
        self.assertTrue((self.base / "restart-attempted").exists())
        self.assertEqual((self.base / "running").read_text(), "true")
        self.assertIn("restart=ok", (self.new / "backup-recovery.log").read_text())

    def test_real_hangup_restarts_after_both_parent_streams_close(self):
        self.assertEqual(self.interrupted_backup(hangup=True), 129)
        self.assertTrue((self.base / "restart-attempted").exists())
        self.assertEqual((self.base / "running").read_text(), "true")
        self.assertIn("archive_exit=129 restart=ok", (self.new / "backup-recovery.log").read_text())

    def test_local_transport_verifies_before_reporting_without_secret_output(self):
        result = self.run_backup(local=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn(b"Encrypted backup verified", result.stdout)
        self.assertNotIn(b"private-fixture-value", result.stdout + result.stderr)
        self.assertEqual(len(list((self.base / "backups").glob("*.tar.gz.age"))), 1)
        self.assertEqual(list((self.base / "backups").glob("*.partial")), [])

    def test_local_restart_failure_does_not_publish_backup_or_success(self):
        result = self.run_backup(local=True, FAIL_START="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn(b"Encrypted backup verified", result.stdout)
        self.assertEqual(list((self.base / "backups").iterdir()), [])


if __name__ == "__main__":
    unittest.main()
