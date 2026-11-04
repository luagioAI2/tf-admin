#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""家长 / 商家门户的权限与数据范围探针。

    python scripts/check-portal-perms.py
    python scripts/check-portal-perms.py --keep    # 保留探针账号（排查用）

# 它到底在证明什么

这不是「接口通不通」，而是**四条互相独立、缺一条就出事**的判据：

1. **闸门**：没通过审核的账号，登录这一步就被拒（不是进去以后空白）
2. **角色**：家长拿不到商家接口，商家拿不到家长接口（casbin 那一层）
3. **身份**：token 必须能反查成 `app_{uid}`，解析不出来就拒（不是回落成 uid=0）
4. **数据范围**：家长 A 看不到家长 B 的订单；商家 B 改不动商家 A 的商品

第 4 条是最容易漏的 —— casbin 只回答「这个角色能不能调这个地址」，
**从不回答「能看谁的数据」**。只测前三条的话，A 拿自己的 token 翻 B 的订单
会一路绿灯。

# 为什么用真账号而不是造 token

造 token 只测到中间件；真账号走完 `/base/appLogin` 全程，
包括 bcrypt 校验、`merchant_application` 查审核、影子账号生成。
这条链上任何一环断了，症状都是「点了一下什么都没发生」，**不报错**。

# 依赖

- 后台 8890（`tf-admin/server`，需先 `scripts/seed-portal.py`）
- 平台 8080（`tf-platform-api`，用来注册探针账号）
- 两者共用 `tf-platform-api/data/tf.db`（探针直接写审核表 / 造一条商品和订单）
"""

import argparse
import json
import os
import sqlite3
import sys
import time
import urllib.error
import urllib.request

BASE = "http://127.0.0.1:8890"
API = "http://127.0.0.1:8080"
HERE = os.path.dirname(os.path.abspath(__file__))
DB = os.path.abspath(os.path.join(HERE, "..", "..", "tf-platform-api", "data", "tf.db"))

ADMIN = {"username": "admin", "password": "TfAdmin@2026"}

# 探针账号。手机号用 `199 0000 xxxx` 这种明显不会和真用户撞的段，
# 清理时按前缀 `+861990000` 一刀切。
PW = "Probe@123456"
PARENT_A = "+8619900000101"
PARENT_B = "+8619900000102"
MERCH_A = "+8619900000201"
MERCH_B = "+8619900000202"
PENDING = "+8619900000301"     # 只申请了家长、状态 pending
PROBE_MOBILE_PREFIX = "+861990000"

ITEM_A = "probe_goods_merchant_a"
ORDER_MARK = "探针心愿-只有家长A看得到"
BUY_URL = "https://example.com/probe-item"

results = []      # (ok, label, detail)


def check(ok, label, detail=""):
    results.append((bool(ok), label, detail))
    print(("  ✓ " if ok else "  ✗ ") + label + (("  — " + detail) if detail and not ok else ""))
    return ok


def http(method, url, body=None, token=None, timeout=15):
    """返回 (http_status, parsed_json_or_text)。HTTP 非 200 也照样解析。"""
    data = None
    headers = {}
    if body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    if token:
        headers["x-token"] = token
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            raw = r.read().decode("utf-8", "replace")
            status = r.status
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", "replace")
        status = e.code
    try:
        return status, json.loads(raw)
    except ValueError:
        return status, raw


def admin_token():
    _s, d = http("POST", BASE + "/base/login", ADMIN)
    if not isinstance(d, dict) or d.get("code") != 0:
        sys.exit("管理员登录失败：%s" % (d,))
    return d["data"]["token"]


# ============================ 造数据 ============================

def ensure_platform_user(mobile, password):
    """注册（或复用）一个 App 账号，返回 uid。

    先注册后登录：注册成功说明是新号；失败（已存在）就用同一个密码登录。
    **不靠解析错误文案判断「已存在」** —— 换个措辞就误判了。
    """
    status, d = http("POST", API + "/api/auth/register",
                     {"mobile": mobile, "password": password, "nickname": "探针"})
    if isinstance(d, dict) and d.get("code") == 200 and d.get("data", {}).get("user"):
        return d["data"]["user"]["uid"]
    status, d = http("POST", API + "/api/auth/login", {"mobile": mobile, "password": password})
    if isinstance(d, dict) and d.get("code") == 200 and d.get("data", {}).get("user"):
        return d["data"]["user"]["uid"]
    sys.exit("建探针账号失败：%s %s" % (mobile, d))


def set_application(conn, uid, role, status):
    """写（或改）一条入驻申请。`uk_merchant_uid_role` 是唯一索引，所以先查后写。"""
    ts = int(time.time())
    row = conn.execute(
        "SELECT id FROM merchant_application WHERE uid=? AND role=?", (uid, role)
    ).fetchone()
    if row:
        conn.execute(
            "UPDATE merchant_application SET status=?, deleted_at=NULL, updated_at=? WHERE id=?",
            (status, ts, row[0]),
        )
    else:
        conn.execute(
            "INSERT INTO merchant_application (created_at, updated_at, deleted_at, uid, role,"
            " status, name, contact, note, admin_url, reviewed_at, reviewed_by, review_note)"
            " VALUES (?,?,NULL,?,?,?,?,?,?,'',?,0,'')",
            (ts, ts, uid, role, status, "探针", "probe", "探针", ts),
        )


def wipe_probe_rows(conn):
    """只删探针自己的行（按手机号前缀 / 固定 item_id / 固定订单标题）。"""
    uids = [r[0] for r in conn.execute(
        "SELECT id FROM user WHERE mobile LIKE ?", (PROBE_MOBILE_PREFIX + "%",))]
    if uids:
        q = ",".join("?" * len(uids))
        conn.execute("DELETE FROM merchant_application WHERE uid IN (%s)" % q, uids)
        conn.execute("DELETE FROM wallet WHERE uid IN (%s)" % q, uids)
        conn.execute("DELETE FROM wish_order WHERE uid IN (%s)" % q, uids)
        conn.execute("DELETE FROM redeem_record WHERE uid IN (%s)" % q, uids)
        conn.execute("DELETE FROM sys_users WHERE username IN (%s)"
                     % ",".join("?" * len(uids)),
                     ["app_%d" % u for u in uids])
        conn.execute("DELETE FROM user WHERE id IN (%s)" % q, uids)
    conn.execute("DELETE FROM wish_item WHERE item_id LIKE 'probe_goods_%'")
    conn.execute("DELETE FROM wish_order WHERE item_name = ?", (ORDER_MARK,))


def build_fixtures(conn, parent_a, parent_b, merch_a, merch_b):
    ts = int(time.time())
    # 先清掉这两个探针家长上一轮可能留下的心愿（进程被中断时清理不会执行）
    conn.execute("DELETE FROM wish_order WHERE uid IN (?,?)", (parent_a, parent_b))
    # 家长 A 的钱包给**非零**的余额。
    #
    # ★ 为什么必须非零：界面探针要断言「卡片上的数字 == 接口的数字」，
    #   而钱包没有行时两边都是 0 —— `0 == 0` 恒真，卡片根本没渲染也能通过。
    #   用 777 / 13 / 5 这种不会和别处撞的数，一眼能看出是不是真的画上去了。
    conn.execute("DELETE FROM wallet WHERE uid = ?", (parent_a,))
    conn.execute(
        "INSERT INTO wallet (created_at, updated_at, deleted_at, uid, points, coupon, diamond, version)"
        " VALUES (?,?,NULL,?,777,13,5,1)",
        (ts, ts, parent_a),
    )
    # 商家 A 名下的一件商品
    conn.execute(
        "INSERT INTO wish_item (created_at, updated_at, deleted_at, item_id, kind, name, emoji,"
        " tint, partner_name, pool, game_id, price_cents, age_range, note, notice, buy_url,"
        " buy_label, archived, sort_order, unlock_count, merchant_uid)"
        " VALUES (?,?,NULL,?,'goods',?,'🎁','#FFB4A2','探针店','','',19900,'','','',?,'去下单',0,0,0,?)",
        (ts, ts, ITEM_A, "探针商品A", BUY_URL, merch_a),
    )
    # 家长 A 的一条心愿（家长 B 不该看见）
    conn.execute(
        "INSERT INTO wish_order (created_at, updated_at, deleted_at, uid, order_id, item_id,"
        " kind, item_name, item_emoji, partner_name, voucher_price, status, vouchers_held,"
        " points_held, opened_game_id, code_tail, code_hash, ship, tracking_no,"
        " committed_at, confirmed_at, done_at, cancelled_at, cancel_reason, opened_game_ids)"
        " VALUES (?,?,NULL,?,?,?,'goods',?,'🎁','探针店',1,'committed',1,0,'','','','','',0,0,0,0,'','')",
        (ts, ts, parent_a, "probe_order_%d" % ts, ITEM_A, ORDER_MARK),
    )


# ============================ 判据 ============================

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--keep", action="store_true", help="保留探针账号（默认清理）")
    args = ap.parse_args()

    if not os.path.exists(DB):
        sys.exit("找不到库：%s" % DB)
    conn = sqlite3.connect(DB)

    print("== 准备探针账号与数据 ==")
    pa = ensure_platform_user(PARENT_A, PW)
    pb = ensure_platform_user(PARENT_B, PW)
    ma = ensure_platform_user(MERCH_A, PW)
    mb = ensure_platform_user(MERCH_B, PW)
    pend = ensure_platform_user(PENDING, PW)
    print("  parentA=%d parentB=%d merchA=%d merchB=%d pending=%d" % (pa, pb, ma, mb, pend))

    # ⚠️ 清理放在 finally 里、**连造数据一起包住**：
    # 造数据的 SQL 写错时（列数不对这种）异常会在这里抛出，
    # 如果 finally 只包住 run_checks，探针账号就留在库里了 ——
    # 下次跑还会复用那几个 uid，出问题很难看出是上一轮的残留。
    try:
        set_application(conn, pa, "parent", "approved")
        set_application(conn, pb, "parent", "approved")
        set_application(conn, ma, "merchant", "approved")
        set_application(conn, mb, "merchant", "approved")
        set_application(conn, pend, "parent", "pending")
        conn.execute("DELETE FROM wish_item WHERE item_id = ?", (ITEM_A,))
        conn.execute("DELETE FROM wish_order WHERE item_name = ?", (ORDER_MARK,))
        build_fixtures(conn, pa, pb, ma, mb)
        conn.commit()

        run_checks(conn, pa, pb, ma, mb, pend)
    finally:
        if not args.keep:
            wipe_probe_rows(conn)
            conn.commit()
            print("\n（探针数据已清理；要保留加 --keep）")
        conn.close()

    bad = [r for r in results if not r[0]]
    print("\n%d/%d 通过" % (len(results) - len(bad), len(results)))
    if bad:
        print("没通过：")
        for _ok, label, detail in bad:
            print("  -", label, "|", detail)
        return 1
    return 0


def flatten_menu(nodes, out=None):
    """把菜单树摊平成 [(name, path)]。

    前端 `permission.js` 会递归注册路由，判「左侧有没有这一项」等价于
    「摊平后的 name 集合里有没有它」。
    """
    if out is None:
        out = []
    for n in nodes or []:
        if isinstance(n, dict):
            out.append((n.get("name"), n.get("path")))
            flatten_menu(n.get("children"), out)
    return out


def run_checks(conn, pa, pb, ma, mb, pend):
    # ---------- 1. 闸门：没审核过的不许进 ----------
    print("\n== 1. 审核闸门 ==")
    _s, d = http("POST", BASE + "/base/appLogin",
                 {"mobile": PENDING, "password": PW, "role": "parent"})
    check(isinstance(d, dict) and d.get("code") != 0, "pending 账号登录被拒", str(d))
    check("审核" in str(d.get("msg", "")), "拒绝理由是「还没通过审核」", str(d.get("msg")))

    # 家长通过了，但拿家长身份登**商家**门户 —— 两个角色各审一次
    _s, d = http("POST", BASE + "/base/appLogin",
                 {"mobile": PARENT_A, "password": PW, "role": "merchant"})
    check(isinstance(d, dict) and d.get("code") != 0, "家长账号登商家门户被拒", str(d))

    # 密码错
    _s, d = http("POST", BASE + "/base/appLogin",
                 {"mobile": PARENT_A, "password": "wrong-pass", "role": "parent"})
    check(isinstance(d, dict) and d.get("code") != 0, "密码错被拒", str(d))
    # 措辞不能泄露「这个手机号存在」
    check("不存在" not in str(d.get("msg", "")), "错误措辞不泄露账号是否存在", str(d.get("msg")))

    # ---------- 2. 登录成功 + 影子账号 ----------
    print("\n== 2. 登录与影子账号 ==")
    _s, d = http("POST", BASE + "/base/appLogin",
                 {"mobile": PARENT_A, "password": PW, "role": "parent"})
    ok = isinstance(d, dict) and d.get("code") == 0
    check(ok, "家长登录成功", str(d))
    if not ok:
        return
    p_tok = d["data"]["token"]
    user = d["data"]["user"]
    check(user.get("authorityId") == 1001, "家长角色 = 1001", str(user.get("authorityId")))
    check(user.get("userName") == "app_%d" % pa, "影子账号名 = app_%d" % pa, str(user.get("userName")))
    # 不带 +86 也要能登（家长只会输 11 位）
    # 用切片而不是 lstrip('+86')：lstrip 是「去掉这些字符的任意组合」，
    # 号码里出现 8 / 6 时会被多啃掉几位，而那种错看起来像「账号不存在」。
    _s, d2 = http("POST", BASE + "/base/appLogin",
                  {"mobile": PARENT_A[3:], "password": PW, "role": "parent"})
    check(isinstance(d2, dict) and d2.get("code") == 0, "手机号不带 +86 也能登", str(d2))

    _s, d = http("POST", BASE + "/base/appLogin",
                 {"mobile": MERCH_A, "password": PW, "role": "merchant"})
    ok = isinstance(d, dict) and d.get("code") == 0
    check(ok, "商家登录成功", str(d))
    if not ok:
        return
    m_tok = d["data"]["token"]
    check(d["data"]["user"].get("authorityId") == 1002, "商家角色 = 1002",
          str(d["data"]["user"].get("authorityId")))

    _s, d = http("POST", BASE + "/base/appLogin",
                 {"mobile": MERCH_B, "password": PW, "role": "merchant"})
    mb_tok = d["data"]["token"] if isinstance(d, dict) and d.get("code") == 0 else None
    check(bool(mb_tok), "商家 B 登录成功", str(d))

    _s, d = http("POST", BASE + "/base/appLogin",
                 {"mobile": PARENT_B, "password": PW, "role": "parent"})
    pb_tok = d["data"]["token"] if isinstance(d, dict) and d.get("code") == 0 else None
    check(bool(pb_tok), "家长 B 登录成功", str(d))

    # ---------- 3. 角色隔离（casbin） ----------
    print("\n== 3. 角色隔离 ==")
    _s, d = http("GET", BASE + "/mall/portal/parent/overview", None, p_tok)
    check(isinstance(d, dict) and d.get("code") == 0, "家长调家长接口：放行", str(d))
    _s, d = http("POST", BASE + "/mall/portal/parent/wishes", {"page": 1, "pageSize": 5}, p_tok)
    check(isinstance(d, dict) and d.get("code") == 0, "家长心愿单：放行", str(d))
    _s, d = http("POST", BASE + "/mall/portal/parent/records", {"page": 1, "pageSize": 5}, p_tok)
    check(isinstance(d, dict) and d.get("code") == 0, "家长兑换记录：放行", str(d))

    for method, path, body in [
        ("GET", "/mall/portal/merchant/status", None),
        ("POST", "/mall/portal/merchant/items", {"page": 1, "pageSize": 5}),
        ("POST", "/mall/portal/merchant/item/save", {"name": "x"}),
    ]:
        _s, d = http(method, BASE + path, body, p_tok)
        check("权限不足" in str(d.get("msg", "")), "家长调商家接口被拒 %s %s" % (method, path), str(d))

    _s, d = http("GET", BASE + "/mall/portal/parent/overview", None, m_tok)
    check("权限不足" in str(d.get("msg", "")), "商家调家长接口被拒", str(d))

    # 管理员不是 App 账号 —— 即使 casbin 放行也必须被身份那一层挡住
    a_tok = admin_token()
    _s, d = http("GET", BASE + "/mall/portal/parent/overview", None, a_tok)
    check(isinstance(d, dict) and d.get("code") != 0, "管理员 token 进不了门户接口", str(d))

    # ---------- 4. 数据范围：家长 ----------
    print("\n== 4. 家长数据范围 ==")
    _s, d = http("POST", BASE + "/mall/portal/parent/wishes",
                 {"page": 1, "pageSize": 50}, p_tok)
    names = [x.get("item_name") for x in (d.get("data", {}).get("list") or [])] if isinstance(d, dict) else []
    check(ORDER_MARK in names, "家长 A 能看到自己的心愿", str(names[:5]))
    check(all(x == ORDER_MARK for x in names), "家长 A 的列表里没有别人的心愿", str(names[:5]))

    if pb_tok:
        _s, d = http("POST", BASE + "/mall/portal/parent/wishes",
                     {"page": 1, "pageSize": 50}, pb_tok)
        names_b = [x.get("item_name") for x in (d.get("data", {}).get("list") or [])]
        check(ORDER_MARK not in names_b, "家长 B 看不到家长 A 的心愿", str(names_b[:5]))

        _s, d = http("GET", BASE + "/mall/portal/parent/overview", None, pb_tok)
        check(isinstance(d, dict) and d.get("data", {}).get("uid") == pb,
              "家长 B 的概览 uid 是自己", str(d.get("data", {}).get("uid")))

    # ---------- 5. 数据范围：商家 ----------
    print("\n== 5. 商家数据范围 ==")
    _s, d = http("POST", BASE + "/mall/portal/merchant/items",
                 {"page": 1, "pageSize": 50}, m_tok)
    ids = [x.get("item_id") for x in (d.get("data", {}).get("list") or [])] if isinstance(d, dict) else []
    check(ITEM_A in ids, "商家 A 能看到自己的商品", str(ids[:5]))
    check(all(x.startswith("probe_") for x in ids), "商家 A 只看到自己名下的商品", str(ids[:5]))

    if mb_tok:
        _s, d = http("POST", BASE + "/mall/portal/merchant/items",
                     {"page": 1, "pageSize": 50}, mb_tok)
        ids_b = [x.get("item_id") for x in (d.get("data", {}).get("list") or [])]
        check(ITEM_A not in ids_b, "商家 B 看不到商家 A 的商品", str(ids_b[:5]))

        # 越权改价：必须失败，而且不能把商品搬走
        _s, d = http("POST", BASE + "/mall/portal/merchant/item/save",
                     {"itemId": ITEM_A, "name": "被B改了", "priceCents": 1}, mb_tok)
        check(isinstance(d, dict) and d.get("code") != 0, "商家 B 改不动商家 A 的商品", str(d))
        row = conn.execute("SELECT name, price_cents FROM wish_item WHERE item_id=?", (ITEM_A,)).fetchone()
        check(row and row[0] == "探针商品A" and row[1] == 19900,
              "越权尝试后商品没被改动", str(row))

    # 商家 A 改自己的：应该成功
    #
    # ⚠️ 这个接口是**整条覆盖**（表单每次提交全部字段），所以这里必须把
    # `buyUrl` 一起带上 —— 少带一个字段就等于把它清空。
    # 第一版就是只传了 name/priceCents，结果 buy_url 被抹掉，
    # 界面探针的「去下单」链接红在 null 上，而接口本身一切正常。
    _s, d = http("POST", BASE + "/mall/portal/merchant/item/save",
                 {"itemId": ITEM_A, "name": "探针商品A", "priceCents": 20900,
                  "buyUrl": BUY_URL, "buyLabel": "去下单"}, m_tok)
    check(isinstance(d, dict) and d.get("code") == 0, "商家 A 能改自己的商品", str(d))
    row = conn.execute("SELECT price_cents, buy_url FROM wish_item WHERE item_id=?", (ITEM_A,)).fetchone()
    check(row and row[0] == 20900, "改价真的落库了（19900 -> 20900）", str(row))
    check(row and row[1] == BUY_URL, "整条覆盖时没带上的字段会被清空（这次带上了）", str(row))

    # 商家不能靠传 merchant_uid 把商品搬到别人名下（字段根本不在请求体里）
    _s, d = http("POST", BASE + "/mall/portal/merchant/item/save",
                 {"itemId": ITEM_A, "name": "探针商品A", "priceCents": 20900,
                  "buyUrl": BUY_URL, "buyLabel": "去下单", "merchantUid": mb},
                 m_tok)
    row = conn.execute("SELECT merchant_uid FROM wish_item WHERE item_id=?", (ITEM_A,)).fetchone()
    check(row and row[0] == ma, "请求体里塞 merchantUid 也搬不走商品", str(row))

    # ---------- 6. 登录之后的外壳（菜单 / 自己） ----------
    #
    # 这一段测的是「进去以后有没有东西」。前面 5 段全绿也可能这一屏是空白的：
    # casbin 少开 `/menu/getMenu` 时，前端只会在控制台打一行 403，
    # 界面上什么都不说。
    print("\n== 6. 登录后的外壳 ==")
    _s, d = http("POST", BASE + "/menu/getMenu", None, p_tok)
    ok = isinstance(d, dict) and d.get("code") == 0
    check(ok, "家长能拿到动态路由（/menu/getMenu）", str(d))
    p_names = [n for n, _p in flatten_menu(d.get("data", {}).get("menus") if ok else [])]
    check("parentOverview" in p_names, "家长菜单里有 parentOverview", str(p_names))
    check("merchantItems" not in p_names, "家长菜单里没有商家页面", str(p_names))

    _s, d = http("POST", BASE + "/menu/getMenu", None, m_tok)
    ok = isinstance(d, dict) and d.get("code") == 0
    check(ok, "商家能拿到动态路由（/menu/getMenu）", str(d))
    m_names = [n for n, _p in flatten_menu(d.get("data", {}).get("menus") if ok else [])]
    check("merchantItems" in m_names, "商家菜单里有 merchantItems", str(m_names))
    check("parentOverview" not in m_names, "商家菜单里没有家长页面", str(m_names))

    # 前端登录后做的是 `router.hasRoute(authority.defaultRouter)` ——
    # defaultRouter 和菜单 name 对不上时只弹一句「不存在可以登陆的首页」，
    # 而角色本身是好的，很容易查错方向。
    _s, d = http("POST", BASE + "/base/appLogin",
                 {"mobile": PARENT_A, "password": PW, "role": "parent"})
    dr = d.get("data", {}).get("user", {}).get("authority", {}).get("defaultRouter")
    check(dr in p_names, "家长 defaultRouter 落在自己的菜单里（%s）" % dr, str(p_names))

    _s, d = http("POST", BASE + "/base/appLogin",
                 {"mobile": MERCH_A, "password": PW, "role": "merchant"})
    dr = d.get("data", {}).get("user", {}).get("authority", {}).get("defaultRouter")
    check(dr in m_names, "商家 defaultRouter 落在自己的菜单里（%s）" % dr, str(m_names))

    _s, d = http("GET", BASE + "/user/getUserInfo", None, p_tok)
    check(isinstance(d, dict) and d.get("code") == 0, "家长刷新页面能取回自己", str(d))
    check(str(d.get("data", {}).get("userInfo", {}).get("userName")) == "app_%d" % pa,
          "取回的仍是影子账号", str(d.get("data", {}).get("userInfo", {}).get("userName")))


if __name__ == "__main__":
    sys.exit(main())
