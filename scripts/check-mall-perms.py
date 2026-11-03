#!/usr/bin/env python
"""逐个探后台 mall 接口的权限状态：`code=7` = 没登记 sys_apis / 没授权 casbin。

为什么单独写一个：README 里那句「接口本来就有，缺的是权限」是**静默**的 ——
页面看着完全正常，只有点下去才报「权限不足」。所以判据必须是「逐个打一遍」。
"""
import json
import urllib.request

BASE = 'http://127.0.0.1:8890'
ADMIN = {'username': 'admin', 'password': 'TfAdmin@2026'}


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


tok = post('/base/login', ADMIN)['data']['token']

CASES = [
    ('GET', '/mall/dashboard/overview', None),
    ('POST', '/mall/game/update', {'id':0}),
    ('POST', '/mall/game/list', {'page': 1, 'pageSize': 1}),
    ('POST', '/mall/game/options', {}),
    ('POST', '/mall/release/list', {'page': 1, 'pageSize': 1}),
    ('POST', '/mall/user/list', {'page': 1, 'pageSize': 1}),
    # 这三个之前一直没覆盖（清单只到 list）。detail / ledger 是 GET，
    # 少了 id 会「参数校验失败」——那也算**放行了**，正是这条要判的。
    ('GET', '/mall/user/detail', None),
    ('GET', '/mall/user/ledger', None),
    # 两个写接口都用「参数不全」的方式探：能证明 casbin 没拦，
    # 又不会真的改到谁的钱 / 密码（uid=0 根本不存在）。
    ('POST', '/mall/user/adjust', {'uid': 0}),
    ('POST', '/mall/user/reset-password', {}),
    ('POST', '/mall/redeem/item/list', {'page': 1, 'pageSize': 1}),
    ('POST', '/mall/redeem/record/list', {'page': 1, 'pageSize': 1}),
    ('POST', '/mall/wish/item/list', {'page': 1, 'pageSize': 1}),
    ('POST', '/mall/wish/order/list', {'page': 1, 'pageSize': 1}),
    ('POST', '/mall/task/list', {'page': 1, 'pageSize': 1}),
    ('POST', '/mall/task/instance/list', {'page': 1, 'pageSize': 1}),
    ('POST', '/mall/merchant/list', {'page': 1, 'pageSize': 1}),
    ('GET', '/mall/merchant/pending-count', None),
]

bad = []
for method, path, body in CASES:
    try:
        d = get(path, tok) if method == 'GET' else post(path, body, tok)
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
