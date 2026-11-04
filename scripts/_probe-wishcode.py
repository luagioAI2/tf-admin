#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""激活码**全链路**探针：后台生成 → App 核销。

    python scripts/_probe-wishcode.py
    python scripts/_probe-wishcode.py --item wish-game-cloudfarm --count 3

## 这个探针存在的唯一理由

激活码的哈希口径写在**三个地方**：

    · C 端   `src/domain/gameCode.ts`      normalizeGameCode
    · 平台   `wish_srv.go`                 normalizeWishCode / hashWishCode
    · 后台   `service/mall/wishcode.go`    normalizeWishCode / hashWishCode

三处任何一处不一样，症状都是**「后台生成的码，App 说没有这个码」**，
而且**两边都不报错** —— 后台生成成功、App 只是回一句「码无效」。
读代码看不出来（三份实现长得几乎一样），只有「生成一个真的拿去核销」
才能证明它们一致。

## 断言清单

  1. 生成接口返回明文，长度 6、字符在 Crockford Base32 里
  2. 列表里 `code_tail` == 明文后 4 位（家长拿卡来问「是不是这张」就靠它）
  3. ★ 明文码在 App 侧**核销成功**，并且那款游戏真的进了库
  4. 同一个码再核销一次 → 「已经用过」（不能一码两用）
  5. 作废未用的码 → 数量对；作废后核销 → 「码无效」
  6. 已经用掉的那张**不许作废**（孩子的游戏已经到账）

退出码：0 = 全绿；1 = 有断言红了（直接拿去当 CI 判据）。
"""
import argparse
import json
import sys
import urllib.error
import urllib.request

import os

# 两个地址都能用环境变量覆盖。
# ⚠️ 平台的监听端口来自它自己的 `config.yaml`（本机是 **8080**，不是 8090），
#    写死 8090 时症状是「游客登录拿到 token」那条红、后面全断，
#    而看日志像「平台没起来」—— 其实起来了，只是端口不一样。
ADMIN = os.environ.get('TF_ADMIN_API', 'http://127.0.0.1:8890')
API = os.environ.get('TF_PLATFORM_API', 'http://127.0.0.1:8090')
ADMIN_USER = {'username': 'admin', 'password': 'TfAdmin@2026'}
ALPHABET = '0123456789ABCDEFGHJKMNPQRSTVWXYZ'  # Crockford Base32（无 I L O U）

FAILS = []

# ⚠️ 两个服务的「成功码」不是同一个数：
#   · GVA 后台  `{"code":0}`      （response.Ok*）
#   · 平台 API  `{"code":200}`    （errors.SUCCESS）
# 混用的话「核销成功」这条会永远判红，而 data 里明明有 game_id ——
# 很容易被当成「哈希不一致」，跑去改三处 normalize（改不对）。
OK_ADMIN = 0
OK_API = 200


def ok(cond, label, detail=''):
    print(('  ✓ ' if cond else '  ✗ ') + label + ('' if cond else ('  ← ' + str(detail))))
    if not cond:
        FAILS.append(label)


def http(url, body=None, token=None, method=None):
    data = None if body is None else json.dumps(body).encode()
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['x-token' if '8890' in url else 'Authorization'] = (
            token if '8890' in url else 'Bearer ' + token
        )
    req = urllib.request.Request(url, data=data, headers=headers, method=method or ('POST' if data else 'GET'))
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            return json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        raw = e.read().decode(errors='replace')
        return {'code': e.code, 'msg': raw[:200], 'error': raw[:200]}


def main() -> int:
    ap = argparse.ArgumentParser()
    # 默认挑一款「游戏」类商品：码只能挂在有 game_id 的商品上
    ap.add_argument('--item', default='wish-game-cloudfarm')
    ap.add_argument('--count', type=int, default=3)
    ap.add_argument('--batch', default='PROBE-' + __import__('datetime').datetime.now().strftime('%H%M%S'))
    args = ap.parse_args()

    print('== 0. 登录 ==')
    r = http(ADMIN + '/base/login', ADMIN_USER)
    tok = (r.get('data') or {}).get('token')
    ok(bool(tok), '后台登录拿到 token', r.get('msg'))
    if not tok:
        return 1

    print('== 1. 生成 %d 个码（批次 %s）==' % (args.count, args.batch))
    r = http(
        ADMIN + '/mall/wish/code/generate',
        {'item_id': args.item, 'count': args.count, 'batch': args.batch, 'expire_days': 0},
        tok,
    )
    ok(r.get('code') == OK_ADMIN, '生成接口返回成功', r.get('msg'))
    codes = (r.get('data') or {}).get('codes') or []
    ok(len(codes) == args.count, '返回 %d 个明文码' % args.count, codes)
    if not codes:
        return 1
    ok(all(len(c) == 6 for c in codes), '每个码都是 6 位', codes)
    bad = [c for c in codes if any(ch not in ALPHABET for ch in c)]
    ok(not bad, '字符都在 Crockford Base32 里（不会有 I/L/O/U）', bad)
    ok(len(set(codes)) == len(codes), '本批内不重复', codes)

    print('== 2. 列表里的码尾要对得上明文 ==')
    r = http(ADMIN + '/mall/wish/code/list', {'page': 1, 'pageSize': 50, 'batch': args.batch}, tok)
    rows = (r.get('data') or {}).get('list') or []
    ok(len(rows) == args.count, '列表能按批次查到 %d 条' % args.count, len(rows))
    tails = {x.get('code_tail') for x in rows}
    ok(tails == {c[-4:] for c in codes}, 'code_tail == 明文后 4 位', sorted(tails))
    ok(all(x.get('used') is False for x in rows), '新生成的码都是「未使用」')

    print('== 3. ★ App 侧核销（这一步才证明三处哈希一致）==')
    def guest(tag):
        """新建一个游客账号，返回 token。

        ★ 每个码都用**新的**游客：同一批码解锁的是**同一款游戏**，
          同一个账号核销第二张必然回「你已经拥有这个游戏了」——
          那是正确的产品行为，但会把「小写/短横线也能用」这条测成红的。
        """
        dev = 'probe-wishcode-%s-%d' % (tag, __import__('time').time())
        rr = http(API + '/api/auth/guest', {'device_id': dev})
        return (rr.get('data') or {}).get('access_token')

    api_tok = guest('a')
    ok(bool(api_tok), '游客登录拿到 token')
    if not api_tok:
        return 1

    first = codes[0]
    r = http(API + '/api/wish/redeem-code', {'code': first}, api_tok)
    ok(r.get('code') == OK_API, '明文码核销成功（三处 normalize/hash 一致）', r.get('msg') or r.get('error'))
    game_id = (r.get('data') or {}).get('game_id')
    ok(bool(game_id), '核销结果里带 game_id', r.get('data'))

    r = http(API + '/api/game/list', {}, api_tok)
    owned = [g.get('game_id') for g in ((r.get('data') or {}).get('list') or [])]
    ok(game_id in owned, '那款游戏真的进了用户的库（%s）' % game_id, owned)

    print('== 4. 同一个码不能再用第二次 ==')
    r = http(API + '/api/wish/redeem-code', {'code': first}, api_tok)
    ok(r.get('code') != 0, '第二次核销被拒', r.get('data'))
    print('     （错误码 %s：%s）' % (r.get('code'), r.get('msg') or r.get('error')))

    print('== 5. 大小写 / 分隔符不该影响核销 ==')
    # 家长手抄成 `abc-def` 也要能用 —— normalize 就是为这个
    second = codes[1]
    messy = (second[:3] + '-' + second[3:]).lower()
    r = http(API + '/api/wish/redeem-code', {'code': messy}, guest('b'))
    ok(r.get('code') == OK_API, '「%s」也能核销（小写 + 短横线）' % messy, r.get('msg') or r.get('error'))

    print('== 6. 作废 ==')
    r = http(ADMIN + '/mall/wish/code/void', {'batch': args.batch}, tok)
    ok(r.get('code') == OK_ADMIN, '按批次作废成功', r.get('msg'))
    # 3 个码里 2 个已用 ⇒ 只能作废剩下的那 1 个
    ok((r.get('data') or {}).get('voided') == args.count - 2,
       '只作废了没用过的 %d 个（已用的不动）' % (args.count - 2), r.get('data'))

    last = codes[-1]
    r = http(API + '/api/wish/redeem-code', {'code': last}, api_tok)
    ok(r.get('code') != 0, '作废后的码核销被拒', r.get('data'))

    print('== 7. 已经用掉的码不许作废 ==')
    r = http(ADMIN + '/mall/wish/code/list', {'page': 1, 'pageSize': 50, 'batch': args.batch, 'used': True}, tok)
    used_rows = (r.get('data') or {}).get('list') or []
    ok(len(used_rows) == 2, '按「已使用」筛出 2 条', len(used_rows))
    if used_rows:
        r = http(ADMIN + '/mall/wish/code/void', {'id': used_rows[0].get('id')}, tok)
        ok(r.get('code') != 0, '单张作废已用的码被拒（游戏已经到账）', r.get('data'))
        print('     （提示：%s）' % (r.get('msg') or ''))

    print()
    if FAILS:
        print('✗ %d 条断言红了：' % len(FAILS))
        for f in FAILS:
            print('  -', f)
        return 1
    print('✓ 全绿')
    return 0


if __name__ == '__main__':
    sys.exit(main())
