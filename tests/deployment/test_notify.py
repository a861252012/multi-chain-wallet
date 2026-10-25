import hashlib
import hmac
from http.client import HTTPConnection
from http.server import BaseHTTPRequestHandler, HTTPServer
import importlib.util
import os
from pathlib import Path
import sys
import tempfile
import threading
import time
import unittest
from unittest.mock import patch


ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('wallet_notify', ROOT / 'scripts/deploy/notify-server.py')
notify = importlib.util.module_from_spec(spec)
spec.loader.exec_module(notify)
sender_spec = importlib.util.spec_from_file_location('wallet_notify_sender', ROOT / 'scripts/deploy/send-notify.py')
sender = importlib.util.module_from_spec(sender_spec)
sender_spec.loader.exec_module(sender)
SECRET = b'a' * 64
GATE = 'c' * 64
REVISION = b'b' * 40


class NotifyTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.trigger = Path(self.temp.name) / 'trigger'
        self.server = notify.LocalServer(('127.0.0.1', 0), notify.make_handler(SECRET, self.trigger))
        self.addCleanup(self.server.server_close)
        thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        thread.start()
        self.addCleanup(self.server.shutdown)
        self.port = self.server.server_port

    def request(self, method='POST', revision=REVISION, stamp=None, signature=None, path='/notify'):
        stamp = str(int(time.time())) if stamp is None else stamp
        if signature is None:
            signature = 'sha256=' + hmac.new(
                SECRET, stamp.encode() + b'\n' + revision, hashlib.sha256
            ).hexdigest()
        connection = HTTPConnection('127.0.0.1', self.port, timeout=3)
        try:
            connection.request(method, path, body=revision, headers={
                'X-Release-Time': stamp,
                'X-Release-Signature': signature,
            })
            response = connection.getresponse()
            response.read()
            return response.status
        finally:
            connection.close()

    def test_valid_notification_queues_only_a_fixed_trigger(self):
        stamp = str(int(time.time()))
        self.assertEqual(self.request(stamp=stamp), 202)
        self.assertTrue(self.trigger.exists())
        self.assertEqual(self.trigger.read_bytes(), b'')
        self.trigger.unlink()
        self.assertEqual(self.request(stamp=stamp), 202)
        self.assertFalse(self.trigger.exists(), 'replay must not queue again')

    def test_sender_signs_release_for_receiver(self):
        requests = []
        original_request = sender.Request

        def record_request(*args, **kwargs):
            request = original_request(*args, **kwargs)
            requests.append(request)
            return request

        with patch.object(sender, 'URL', f'http://127.0.0.1:{self.port}/notify'), \
                patch.object(sys, 'argv', ['send-notify.py', REVISION.decode()]), \
                patch.object(sender, 'Request', side_effect=record_request), \
                patch.dict(os.environ, {
                    'DEMO_NOTIFY_SECRET': SECRET.decode(),
                    'DEMO_NOTIFY_GATE': GATE,
                }, clear=True):
            sender.main()
        self.assertTrue(self.trigger.exists())
        self.assertEqual(requests[0].get_header('X-demo-notify-gate'), GATE)

    def test_sender_requires_waf_gate(self):
        with patch.object(sys, 'argv', ['send-notify.py', REVISION.decode()]), \
                patch.dict(os.environ, {'DEMO_NOTIFY_SECRET': SECRET.decode()}, clear=True):
            with self.assertRaisesRegex(SystemExit, 'Cloudflare WAF gate'):
                sender.main()
        self.assertFalse(self.trigger.exists())

    def test_sender_rejects_redirect_without_forwarding_credentials(self):
        received = []

        class Target(BaseHTTPRequestHandler):
            def do_GET(self):
                received.append(dict(self.headers))
                self.send_response(202)
                self.end_headers()

            def log_message(self, format, *args):
                pass

        target = HTTPServer(('127.0.0.1', 0), Target)
        target_thread = threading.Thread(target=target.serve_forever, daemon=True)
        target_thread.start()
        self.addCleanup(target.server_close)
        self.addCleanup(target.shutdown)

        class Redirect(BaseHTTPRequestHandler):
            def do_POST(self):
                self.send_response(302)
                self.send_header('Location', f'http://127.0.0.1:{target.server_port}/')
                self.end_headers()

            def log_message(self, format, *args):
                pass

        redirect = HTTPServer(('127.0.0.1', 0), Redirect)
        redirect_thread = threading.Thread(target=redirect.serve_forever, daemon=True)
        redirect_thread.start()
        self.addCleanup(redirect.server_close)
        self.addCleanup(redirect.shutdown)
        with patch.object(sender, 'URL', f'http://127.0.0.1:{redirect.server_port}/notify'), \
                patch.object(sys, 'argv', ['send-notify.py', REVISION.decode()]), \
                patch.object(sender.time, 'sleep'), \
                patch.dict(os.environ, {
                    'DEMO_NOTIFY_SECRET': SECRET.decode(),
                    'DEMO_NOTIFY_GATE': GATE,
                }, clear=True):
            with self.assertRaises(SystemExit):
                sender.main()
        self.assertEqual(received, [])

    def test_new_event_during_dispatch_is_queued(self):
        self.assertEqual(self.request(), 202)
        self.trigger.unlink()  # dispatcher consumed the first trigger
        self.assertEqual(self.request(revision=b'c' * 40), 202)
        self.assertTrue(self.trigger.exists())

    def test_invalid_requests_never_queue(self):
        with patch.object(notify.time, 'time', return_value=1_700_000_000):
            cases = [
                {'method': 'GET'},
                {'path': '/wrong'},
                {'revision': b'x' * 41},
                {'revision': b'not-a-commit'},
                {'stamp': str(int(time.time()) - 301)},
                {'stamp': str(int(time.time()) + 301)},
                {'signature': 'sha256=' + '0' * 64},
            ]
            for case in cases:
                with self.subTest(case=case):
                    self.assertNotEqual(self.request(**case), 202)
                    self.assertFalse(self.trigger.exists())

    def test_receiver_failure_does_not_acknowledge_event(self):
        with patch.object(notify.os, 'open', side_effect=OSError('disk full')):
            self.assertEqual(self.request(), 503)
        self.assertFalse(self.trigger.exists())
