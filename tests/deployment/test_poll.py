import os
from pathlib import Path
import shlex
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SHA = 'a' * 40
REF = 'ghcr.io/a861252012/multi-chain-wallet@sha256:' + 'b' * 64

class PollTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='wallet poll fixture-')
        self.addCleanup(self.tmp.cleanup)
        self.base = Path(self.tmp.name)
        self.bin = self.base / 'bin'
        self.bin.mkdir()
        self.env = {**os.environ, 'FIXTURE': str(self.base), 'HEAD': SHA, 'REF': REF}
        source = (ROOT / 'scripts/deploy/poll.sh').read_text()
        source = source.replace('readonly base=/opt/multi-chain-wallet', 'readonly base=' + shlex.quote(str(self.base)))
        source = source.replace('export PATH=/usr/sbin:/usr/bin:/sbin:/bin', 'export PATH=' + shlex.quote(str(self.bin) + os.pathsep + os.environ['PATH']))
        source = source.replace('/usr/local/sbin/wallet-deploy', shlex.quote(str(self.bin / 'deploy')))
        self.script = self.base / 'poll.sh'
        self.script.write_text(source)
        fake = '''#!/usr/bin/env python3
import os,sys,json
from pathlib import Path
name=Path(sys.argv[0]).name
with (Path(os.environ['FIXTURE'])/'calls').open('a') as f: f.write(name+' '+ ' '.join(sys.argv[1:])+'\\n')
if name=='curl': print(json.dumps({'sha':os.environ['HEAD']}))
if name=='docker':
 if sys.argv[1]=='pull' and os.getenv('PULL_FAIL'): sys.exit(1)
 if sys.argv[1:3]==['image','inspect']:
  print(os.environ.get('CURRENT', 'c'*40) if 'revision' in sys.argv[4] else os.environ['REF'])
'''
        for name in ('curl', 'docker', 'flock', 'deploy'):
            p = self.bin/name
            p.write_text(fake)
            p.chmod(0o755)

    def run_poll(self, **env):
        result = subprocess.run(['bash', str(self.script)], env={**self.env, **env}, capture_output=True, text=True)
        calls = (self.base/'calls').read_text()
        return result, calls

    def test_release_passes_only_digest_and_sha(self):
        result,calls = self.run_poll()
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertIn('deploy sha256:'+'b'*64+' '+SHA,calls)

    def test_failed_ci_or_registry_keeps_old_app(self):
        result,calls = self.run_poll(PULL_FAIL='1')
        self.assertNotEqual(result.returncode,0)
        self.assertNotIn('deploy ',calls)

    def test_current_release_is_not_pulled_again(self):
        (self.base/'current-image').write_text(REF)
        result,calls = self.run_poll(CURRENT=SHA)
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertNotIn('docker pull',calls)
        self.assertNotIn('deploy ',calls)

    def test_invalid_head_and_foreign_registry_are_rejected(self):
        for env in ({'HEAD':'bad; command'}, {'REF':'ghcr.io/attacker/app@sha256:'+'b'*64}):
            with self.subTest(env=env):
                result,calls=self.run_poll(**env)
                self.assertNotEqual(result.returncode,0)
                self.assertNotIn('deploy ',calls)
