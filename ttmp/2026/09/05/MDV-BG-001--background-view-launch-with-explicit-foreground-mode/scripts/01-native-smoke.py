#!/usr/bin/env python3
"""Linux/X11 native smoke. Owns only uniquely named tmux sessions/windows.
Run from repo root after make build; emits evidence JSON to stdout.
"""
import json
import os
from pathlib import Path
import re
import shlex
import subprocess
import tempfile
import time

binary = str(Path('build/bin/md-view').resolve())
evidence = {}
sessions = []
pids = []
windows = []


def run(*args):
    return subprocess.check_output(args, text=True).strip()


def wait_for(fn, label, timeout=15):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        value = fn()
        if value:
            return value
        time.sleep(.1)
    raise RuntimeError('timeout: ' + label)


def alive(pid):
    stat = Path(f'/proc/{pid}/stat')
    return stat.exists() and stat.read_text().split(') ')[1][0] != 'Z'


def window_for(title):
    for line in run('wmctrl', '-lp').splitlines():
        if line.endswith(title):
            return line.split()[0]
    return None


def close_window(window):
    subprocess.run(['wmctrl', '-ic', window], check=True)
    windows.remove(window)


with tempfile.TemporaryDirectory(prefix='mdv-bg-smoke-') as temp:
    root = Path(temp)
    fixture = root / 'relative file.md'
    fixture.write_text('---\nTitle: MDV-BG-001 native smoke\n---\n\n# Background smoke\n\nRendered from a relative path.\n')
    env = f'XDG_CACHE_HOME={shlex.quote(temp)}/cache XDG_CONFIG_HOME={shlex.quote(temp)}/config'
    try:
        # Shell capture must reach EOF while the detached desktop is still alive.
        session = f'mdv-bg-001-{os.getpid()}'
        sessions.append(session)
        command = f'cd {shlex.quote(temp)}; {env} {shlex.quote(binary)} view --dark "relative file.md" > output; echo $? > returned; sleep 120'
        subprocess.run(['tmux', 'new-session', '-d', '-s', session, command], check=True)
        wait_for(lambda: (root / 'returned').exists(), 'background parent return')
        output = (root / 'output').read_text().strip()
        assert (root / 'returned').read_text().strip() == '0', output
        match = re.fullmatch(r'Started md-view process (\d+); log: (.+)', output)
        assert match, output
        pid, log = int(match[1]), match[2]
        pids.append(pid)
        assert alive(pid)
        assert os.getsid(pid) == pid
        args = Path(f'/proc/{pid}/cmdline').read_bytes().split(b'\0')
        assert b'--foregruond' in args and b'--dark' in args
        assert str(fixture).encode() in args
        fd = {str(i): os.readlink(f'/proc/{pid}/fd/{i}') for i in range(3)}
        assert fd == {'0': '/dev/null', '1': log, '2': log}, fd
        window = wait_for(lambda: window_for('md-view: MDV-BG-001 native smoke'), 'rendered document title')
        windows.append(window)
        evidence['background'] = {'output': output, 'pid': pid, 'sid': os.getsid(pid), 'args': [x.decode() for x in args if x], 'fd': fd, 'window': window, 'title': run('xdotool', 'getwindowname', window)}
        subprocess.run(['tmux', 'kill-session', '-t', session], check=True)
        sessions.remove(session)
        time.sleep(.5)
        assert alive(pid), 'terminal closure killed child'
        evidence['background']['survived_terminal_close'] = True
        evidence['background']['log'] = Path(log).read_text()
        close_window(window)
        wait_for(lambda: not alive(pid), 'background window shutdown')

        # Foreground stays in the tmux shell until its native window is closed.
        (root / 'returned').unlink()
        session = f'mdv-fg-001-{os.getpid()}'
        sessions.append(session)
        command = f'cd {shlex.quote(temp)}; {env} {shlex.quote(binary)} view --foregruond "relative file.md"; echo $? > returned; sleep 120'
        subprocess.run(['tmux', 'new-session', '-d', '-s', session, command], check=True)
        window = wait_for(lambda: window_for('md-view: MDV-BG-001 native smoke'), 'foreground rendered title')
        windows.append(window)
        time.sleep(.5)
        assert not (root / 'returned').exists(), 'foreground returned before window close'
        evidence['foreground'] = {'blocked_with_window_open': True, 'window': window, 'title': run('xdotool', 'getwindowname', window)}
        close_window(window)
        wait_for(lambda: (root / 'returned').exists(), 'foreground command return')
        evidence['foreground']['exit_code'] = (root / 'returned').read_text().strip()
        evidence['foreground']['pane'] = run('tmux', 'capture-pane', '-pt', session)
        assert evidence['foreground']['exit_code'] == '0'
        subprocess.run(['tmux', 'kill-session', '-t', session], check=True)
        sessions.remove(session)

        # Help and malformed arguments are synchronous and must not create logs.
        before = set((root / 'cache' / 'md-view').glob('*'))
        for args, expected in [(['view', '--help'], 0), (['view', 'a', 'b'], 1), (['view', '--unknown'], 1)]:
            result = subprocess.run([binary, *args], env={**os.environ, 'XDG_CACHE_HOME': str(root / 'cache')}, capture_output=True, text=True, timeout=5)
            assert result.returncode == expected, result
        assert set((root / 'cache' / 'md-view').glob('*')) == before
        evidence['validation_does_not_spawn'] = True
        print(json.dumps(evidence, indent=2))
    finally:
        for window in windows:
            subprocess.run(['wmctrl', '-ic', window], check=False)
        for session in sessions:
            subprocess.run(['tmux', 'kill-session', '-t', session], check=False)
        for pid in pids:
            if alive(pid):
                os.kill(pid, 15)
