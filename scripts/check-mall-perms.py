#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""逐个探后台 mall 接口的权限状态：`code=7` + 「权限不足」= 没登记 sys_apis / 没授权 casbin。

为什么单独写一个：README 里那句「接口本来就有，缺的是权限」是**静默**的 ——
页面看着完全正常，只有点下去才报「权限不足」。所以判据必须是「逐个打一遍」。

★ 清单**不在这里手写**，而是从 `seed-mall.py` 的 `APIS` 读 ——
  两处各写一份的话，新加的接口会出现「登记了但没人探」，
  而「没人探」和「探了是绿的」在报告里长得一模一样。
"""
import importlib.util
import json
import os
import urllib.request

BASE = 'http://127.0.0.1:8890'
ADMIN = {'username': 'admin', 'password': 'TfAdmin@2026'}
HERE = os.path.dirname(os.path.abspath(__file__))


def load_seed_module():
    """`seed-mall.py` 名字里带连字符，普通 import 进不来 —— 用 importlib 按路径加载。"""
    path = os.path.join(HERE, 'seed-mall.py')
    spec = importlib.util.spec_from_file_location('seed_mall', path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def post(path, body=None, token=None):
    req = urllib.request.Request(
        BASE + path,
        data=json.dumps(body or {}).encode(),
        headers={'Content-Type': 'application/json', **({'x-token': token} if token else {})},
    )
    with urllib.request.urlopen(req, timeout=10) as r:
        return json.loads(r.read().decode())


def get(path, token):
    req = urllib.request.Request(BASE + path, headers={'x-token': token})
    with urllib.request.urlopen(req, timeout=10) as r:
        return json.loads(r.read().decode())


def delete(path, token):
    req = urllib.request.Request(
        BASE + path,
        data=b'{}',
        method='DELETE',
        headers={'Content-Type': 'application/json', 'x-token': token},
    )
    with urllib.request.urlopen(req, timeout=10) as r:
        return json.loads(r.read().decode())


def safe_body(method, path):
    """探针的请求体。

    判据只要「不是权限不足」就行，所以故意用**参数不全**的 body：
      · 列表接口给分页参数（真的能跑通）；
      · 写接口给 `{}` —— 参数校验必然失败，**碰不到任何真实数据**。
    这样既能证明 casbin 放行了，又不会改到谁的钱 / 密码 / 商品。
    """
    if method != 'POST':
        return None
    if path.endswith('/list'):
        return {'page': 1, 'pageSize': 1}
    return {}


tok = post('/base/login', ADMIN)['data']['token']
CASES = [(m, p, safe_body(m, p)) for m, p, _d, _g in load_seed_module().APIS]

bad = []
for method, path, body in CASES:
    try:
        if method == 'GET':
            d = get(path, tok)
        elif method == 'DELETE':
            d = delete(path, tok)
        else:
            d = post(path, body, tok)
        code = d.get('code')
        msg = d.get('msg', '')
        # 判据是 msg 里有没有「权限不足」，不是 code==0：
        # 参数校验失败也返回 code=7（Key: 'UpdateGameReq.GameID' Error:...）。
        # 只看 code 会把「参数没填」当成「没授权」，然后跑去改 casbin —— 改不对。
        ok = code == 0 or '权限不足' not in msg
        note = '' if ok else msg
        tag = '' if code == 0 else '（已放行，只是参数没填对）'
        print(('  ✓ ' if ok else '  ✗ ') + f'{method} {path}' + (f' — code={code} {note}' if not ok else tag))
        if not ok:
            bad.append((method, path, code, note))
    except Exception as e:  # noqa: BLE001
        print(f'  ✗ {method} {path} — 异常 {e}')
        bad.append((method, path, -1, str(e)))

print(f'\n{len(CASES) - len(bad)}/{len(CASES)} 通')
if bad:
    print('不通的：')
    for b in bad:
        print(' ', b)
