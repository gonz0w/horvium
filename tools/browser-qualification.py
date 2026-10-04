#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Real-browser checks against an explicitly synthetic loopback console fixture."""
import argparse
import json
import os
import platform
import signal
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ELEMENT = 'element-6066-11e4-a52e-4f735466cecf'


def contrast(a, b):
    def luminance(rgb):
        channels = [v / 255 for v in rgb]
        linear = [v / 12.92 if v <= .04045 else ((v + .055) / 1.055) ** 2.4 for v in channels]
        return sum(v * weight for v, weight in zip(linear, (.2126, .7152, .0722)))
    light, dark = sorted((luminance(a), luminance(b)), reverse=True)
    return (light + .05) / (dark + .05)


class Driver:
    def __init__(self, base):
        self.base = base
        self.session = None
        self.http = urllib.request.build_opener(urllib.request.ProxyHandler({}))

    def request(self, method, path, payload=None):
        body = None if payload is None else json.dumps(payload).encode()
        request = urllib.request.Request(self.base + path, body, {'Content-Type': 'application/json'}, method=method)
        try:
            with self.http.open(request, timeout=45) as response:
                result = json.load(response)
        except urllib.error.HTTPError as error:
            result = json.load(error)
        value = result.get('value')
        if isinstance(value, dict) and value.get('error'):
            raise RuntimeError('WebDriver operation failed: ' + value['error'] + (': ' + value.get('message', '') if path == '/session' else ''))
        return value

    def command(self, method, suffix, value=None):
        return self.request(method, '/session/' + self.session + suffix, value)

    def script(self, script, *args):
        return self.command('POST', '/execute/sync', {'script': script, 'args': list(args)})

    def element(self, selector):
        return self.command('POST', '/element', {'using': 'css selector', 'value': selector})[ELEMENT]

    def click(self, selector):
        self.command('POST', '/element/' + self.element(selector) + '/click', {})

    def type(self, selector, text):
        element = self.element(selector)
        self.command('POST', '/element/' + element + '/clear', {})
        self.command('POST', '/element/' + element + '/value', {'text': text, 'value': list(text)})

    def key(self, key):
        self.command('POST', '/actions', {'actions': [{'type': 'key', 'id': 'keyboard', 'actions': [{'type': 'keyDown', 'value': key}, {'type': 'keyUp', 'value': key}]}]})
        self.command('DELETE', '/actions')

    def wait(self, predicate):
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline:
            if predicate():
                return
            time.sleep(.05)
        raise AssertionError('browser condition did not complete')


def rgb(value):
    # Computed foreground/background colors for this CSS have opaque RGB values.
    values = value[value.index('(') + 1:value.index(')')].split(',')
    if len(values) == 4 and float(values[3]) != 1:
        raise AssertionError('unexpected transparent color in contrast fixture')
    return tuple(float(v.strip()) for v in values[:3])


def check_page(driver):
    result = driver.script('''
      const bg = el => { while(el) {const b=getComputedStyle(el).backgroundColor;
        if(b!=="rgba(0, 0, 0, 0)" && b!=="transparent") return b;el=el.parentElement;} return "rgb(255, 255, 255)"; };
      const selectors=["body",".muted",".eyebrow","caption",".generation",".sidebar nav a[aria-current]","button",".detail dt"];
      const text=selectors.map(s=>document.querySelector(s)).filter(Boolean).map(el=>({fg:getComputedStyle(el).color,bg:bg(el)}));
      const controls=[...document.querySelectorAll("input:not([type=hidden]),select,button")].map(el=>({border:getComputedStyle(el).borderTopColor,bg:bg(el),outside:bg(el.parentElement)}));
      return {text,controls,width:innerWidth,scroll:document.documentElement.scrollWidth,
        images:document.querySelectorAll("tbody img").length,
        labels:[...document.querySelectorAll("input:not([type=hidden]),select")].every(el=>el.id && document.querySelector('label[for="'+el.id+'"]')),
        live:document.querySelector("#collector-status")?.getAttribute("aria-live"),
        motion:getComputedStyle(document.body).animationName};''')
    assert result['scroll'] <= result['width'] + 1, 'page-level horizontal overflow'
    assert result['images'] == 0 and result['labels'] and result['live'] == 'polite', 'semantic or escaping contract'
    ratios = [contrast(rgb(row['fg']), rgb(row['bg'])) for row in result['text']]
    assert min(ratios) >= 4.5, 'normal text contrast below target'
    borders = [max(contrast(rgb(row['border']), rgb(row['outside'])), contrast(rgb(row['bg']), rgb(row['outside']))) for row in result['controls']]
    assert min(borders) >= 3, 'control boundary contrast below target'
    return {'minimum_sampled_text_contrast': min(ratios), 'minimum_control_boundary_contrast': min(borders), 'viewport_css_width': result['width']}


def flow(driver, base, javascript, width, poll=False, firefox=False):
    if firefox and width < 500:
        # Only the disposable browser chrome minimum is adjusted. Product CSS,
        # CSP, content and selected viewport remain unchanged.
        driver.command('POST', '/moz/context', {'context': 'chrome'})
        try:
            driver.script('document.documentElement.style.setProperty("min-width", "0px", "important")')
        finally:
            driver.command('POST', '/moz/context', {'context': 'content'})
    driver.command('POST', '/window/rect', {'width': width, 'height': 900})
    driver.command('POST', '/url', {'url': base + '/login'})
    driver.wait(lambda: driver.script('return !!document.querySelector("#token")'))
    actual_width = driver.script('return innerWidth')
    assert abs(actual_width - width) <= 1, f'requested CSS viewport {width} was not reached: {actual_width}'
    assert driver.script('return document.body.textContent.includes("Synthetic browser fixture")'), 'fixture provenance banner missing'
    driver.type('#token', 'a' * 64)  # Deliberately public synthetic fixture value.
    driver.click('form button[type=submit]')
    driver.wait(lambda: driver.script('return !!document.querySelector("#workspace")'))
    driver.click('nav a[href="/storage/datasets"]')
    driver.wait(lambda: driver.script('return !!document.querySelector("#filter")'))
    assert driver.script('return document.querySelector("tbody").textContent.includes("<img src=x onerror=alert(1)>") && !document.querySelector("tbody img")'), 'hostile resource escaping'
    if javascript:
        assert driver.script('return !!window.htmx && htmx.config.selfRequestsOnly && !htmx.config.allowEval && !htmx.config.allowScriptTags && htmx.config.historyCacheSize===0'), 'HTMX/CSP configuration'
    else:
        assert driver.script('return typeof window.htmx === "undefined"'), 'JavaScript was not actually disabled'
    # Keyboard navigation follows native focus behavior, never programmatic focus.
    driver.click('#filter')
    driver.key('\ue004')
    assert driver.script('return document.activeElement.id') == 'sort', 'keyboard order/focus'
    assert driver.script('return parseFloat(getComputedStyle(document.activeElement).outlineWidth) >= 3'), 'visible keyboard focus'
    driver.type('#filter', 'photos')
    if javascript:
        driver.script('window.horviumFixtureNavigation=1')
    driver.click('#apply-filters')
    driver.wait(lambda: driver.script('return document.querySelectorAll("tbody th a").length===1 && document.querySelector("tbody th a").textContent==="vault/photos"'))
    if javascript:
        assert driver.script('return window.horviumFixtureNavigation===1 && document.activeElement.id==="apply-filters"'), 'fragment navigation/focus preservation'
    driver.click('tbody th a')
    driver.wait(lambda: driver.script('return !!document.querySelector("#detail-title")'))
    if javascript:
        assert driver.script('return document.activeElement.id==="row-18446744073709551614"'), 'selection focus lost'
    if poll:
        driver.click('#filter')
        driver.script('window.horviumFixtureWorkspace=document.querySelector("#workspace");window.horviumFixtureStatus=document.querySelector("#collector-status")')
        deadline = time.monotonic() + 35
        while time.monotonic() < deadline:
            if driver.script('return window.horviumFixtureStatus!==document.querySelector("#collector-status")'):
                break
            time.sleep(1)
        else:
            raise AssertionError('actual HTMX status polling did not execute')
        assert driver.script('return document.activeElement.id==="filter" && window.horviumFixtureWorkspace===document.querySelector("#workspace") && !!document.querySelector("#detail-title")'), 'status polling disrupted focus/details'
    measurements = check_page(driver)
    driver.click('form[action="/refresh"] button')
    if javascript:
        driver.wait(lambda: driver.script('return document.querySelector("#refresh-status").textContent.includes("Collection requested")'))
    else:
        driver.wait(lambda: driver.command('GET', '/url').startswith(base + '/storage/datasets'))
    driver.click('form[action="/logout"] button')
    driver.wait(lambda: driver.script('return !!document.querySelector("#token")'))
    driver.command('POST', '/url', {'url': base + '/storage/datasets'})
    driver.wait(lambda: driver.script('return !!document.querySelector("#token")'))
    return measurements


def unused_port():
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        return sock.getsockname()[1]


def stop_process(process):
    if process.poll() is None:
        os.killpg(process.pid, signal.SIGTERM)
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGKILL)
            process.wait(timeout=5)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--browser', choices=('firefox', 'safari'), required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if args.browser == 'safari' and platform.system() != 'Darwin':
        raise SystemExit('Safari requires macOS')
    if args.browser == 'firefox' and platform.system() != 'Linux':
        raise SystemExit('Pinned Firefox requires Linux')
    records = []
    with tempfile.TemporaryDirectory(prefix='horvium-browser-') as temporary:
        path = Path(temporary)
        with (path / 'fixture.log').open('w+') as log:
            env = os.environ.copy()
            env.update(HORVIUM_UI_PREVIEW='1', GOENV='off', GOTOOLCHAIN='local', CGO_ENABLED='0', GOCACHE=str(ROOT / '.tools/go-cache'), GOMODCACHE=str(ROOT / '.tools/go-modules'))
            fixture = subprocess.Popen([str(ROOT / '.tools/go/bin/go'), 'test', '-v', './internal/console', '-run', '^TestBrowserPreview$', '-count=1', '-timeout=25m'], cwd=ROOT, env=env, stdout=log, stderr=log, start_new_session=True)
            try:
                base = None
                for _ in range(600):
                    log.seek(0)
                    for line in log:
                        if line.startswith('Synthetic browser fixture: '):
                            base = line.split()[-1]
                    if base:
                        break
                    if fixture.poll() is not None:
                        raise RuntimeError('synthetic browser fixture failed to start')
                    time.sleep(.1)
                if not base or not base.startswith('http://127.0.0.1:'):
                    raise RuntimeError('synthetic loopback fixture URL missing')
                scenarios = [(True, 'light', 1280), (True, 'dark', 390), (False, 'light', 1280), (False, 'dark', 320)] if args.browser == 'firefox' else [(True, 'system', 1280), (True, 'system', 390)]
                for javascript, theme, width in scenarios:
                    port = unused_port()
                    executable = ROOT / '.tools/browser/geckodriver/geckodriver' if args.browser == 'firefox' else Path('/usr/bin/safaridriver')
                    with (path / 'driver.log').open('w') as driver_log:
                        driver_args = [str(executable), '--port', str(port)]
                        if args.browser == 'firefox':
                            driver_args.append('--allow-system-access')
                        process = subprocess.Popen(driver_args, stdout=driver_log, stderr=driver_log, start_new_session=True)
                        driver = Driver('http://127.0.0.1:' + str(port))
                        try:
                            for _ in range(100):
                                try:
                                    driver.request('GET', '/status')
                                    break
                                except (urllib.error.URLError, ConnectionError):
                                    time.sleep(.1)
                            options = {'browserName': args.browser}
                            if args.browser == 'firefox':
                                options['moz:firefoxOptions'] = {'binary': str(ROOT / '.tools/browser/firefox/firefox/firefox'), 'args': ['-headless'], 'prefs': {'javascript.enabled': javascript, 'ui.systemUsesDarkTheme': 1 if theme == 'dark' else 0, 'layout.css.prefers-color-scheme.content-override': 0 if theme == 'dark' else 1, 'ui.prefersReducedMotion': 1}}
                            value = driver.request('POST', '/session', {'capabilities': {'alwaysMatch': options}})
                            driver.session = value['sessionId']
                            capabilities = value['capabilities']
                            if args.browser == 'firefox':
                                assert capabilities['browserVersion'] == '157.0', 'unpinned Firefox version'
                            media = driver.script('return {dark:matchMedia("(prefers-color-scheme:dark)").matches,reduce:matchMedia("(prefers-reduced-motion:reduce)").matches}')
                            if args.browser == 'firefox':
                                assert media['dark'] == (theme == 'dark') and media['reduce'], 'requested media preferences not observed'
                            measurements = flow(driver, base, javascript, width, javascript and theme == 'light' and width == 1280, args.browser == 'firefox')
                            measurements['observed_media'] = media
                            measurements['viewport_method'] = 'owned browser chrome minimum override' if args.browser == 'firefox' and width < 500 else 'browser window'
                            record = {'browser': args.browser, 'browser_version': capabilities['browserVersion'], 'platform': capabilities['platformName'], 'host_platform': platform.platform(), 'javascript': javascript, 'theme': theme, 'requested_width': width, 'status': 'pass', 'measurements': measurements, 'manual_screen_reader_and_native_zoom': 'pending'}
                            records.append(record)
                            print('BROWSER_RESULT ' + json.dumps(record), flush=True)
                        finally:
                            if driver.session:
                                driver.command('DELETE', '')
                            stop_process(process)
            finally:
                stop_process(fixture)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(records, indent=2) + '\n')


if __name__ == '__main__':
    main()
