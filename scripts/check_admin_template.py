#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""后台「任务 / 兑换模板」CRUD + 数据隔离 探针（打真接口，不是源码 grep）。

    python scripts/check_admin_template.py

## 它在证明什么

后台模板配置的准入是**三层**，缺任何一层都不报错、只是「点一下没反应」：

1. **casbin** —— 这个角色能不能调这个地址（`1001` 家长不给）
2. **templateActor** —— 这个 token 到底是谁（从 `app_{uid}` 解析，失败直接拒）
3. **service** —— 每条 SQL 都带 `scope` / `owner_uid` 范围

只测「超管能增删改查」等于只测了第 1 层的一半；本探针还钉：

- 商家请求体里**谎报** `scope=official / owner_uid=0` 时，服务端强制收敛成
  `merchant + 自己`（`normalizeScope`）—— 这是「商家自己配自己的」唯一落地处
- 商家 B 拿自己的 token 改商家 A 的模板 id ⇒ 拒（`loadOwned`）
- 商家碰官方模板 ⇒ 拒；家长碰任意模板接口 ⇒ casbin 拒
- 模板条目是**硬删**（软删会占着 `uk_tpl_item` 让「删掉再加回来」撞索引）
- `Extra` 里 typed 列的键被剔除（否则下发时会把运营改的 price 盖回旧值）

## 依赖

- 后台 8890（`tf-admin/server`，需先跑 `seed-mall.py` + `seed-portal.py`）
- 平台 8080（注册探针账号用）
- 两者共用 `tf-platform-api/data/tf.db`

探针**自清理**：开头和结尾各清一次，中断了下次跑也能自己收干净。
"""

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
PW = "Probe@123456"
PROBE_PREFIX = "+86199000007"
MERCH_A = PROBE_PREFIX + "01"
MERCH_B = PROBE_PREFIX + "02"
PARENT = PROBE_PREFIX + "03"
TPL_NAME_PREFIX = "探针模板-"

results = []


def check(ok, label, detail=""):
    results.append((bool(ok), label, detail))
    print(("  OK   " if ok else "  FAIL ") + label +
          (("  -- " + str(detail)[:400]) if detail and not ok else ""))
    return bool(ok)


def http(method, url, body=None, token=None, timeout=15):
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
            raw, status = r.read().decode("utf-8", "replace"), r.status
    except urllib.error.HTTPError as e:
        raw, status = e.read().decode("utf-8", "replace"), e.code
    try:
        return status, json.loads(raw)
    except ValueError:
        return status, raw


def is_succ(d):
    """GVA（后台 8890）：成功 code=0。"""
    return isinstance(d, dict) and d.get("code") == 0


def api_succ(d):
    """tf-platform-api（8080）：成功 code=200，失败也是 HTTP 200。
    ★ 照抄 GVA 的 code=0 会把「注册成功」当成失败，探针白跑。"""
    return isinstance(d, dict) and d.get("code") == 200


def is_reject(d):
    """业务拒绝 = 没成功，而且 data 是空的（`FailWithMessage` 给空 map）。"""
    return isinstance(d, dict) and d.get("code") != 0 and d.get("data") in ({}, None, [], "")


def tpl(path, body, token):
    return http("POST", BASE + "/mall/template/" + path, body, token)


def admin_token():
    _s, d = http("POST", BASE + "/base/login", ADMIN)
    if not is_succ(d):
        sys.exit("管理员登录失败：%s" % (d,))
    return d["data"]["token"]


def ensure_app_user(mobile):
    """注册（或复用）一个 App 账号，返回 uid。"""
    _s, d = http("POST", API + "/api/auth/register",
                 {"mobile": mobile, "password": PW, "nickname": "模板探针"})
    if api_succ(d) and d.get("data", {}).get("user"):
        return d["data"]["user"]["uid"]
    _s, d = http("POST", API + "/api/auth/login", {"mobile": mobile, "password": PW})
    if api_succ(d) and d.get("data", {}).get("user"):
        return d["data"]["user"]["uid"]
    sys.exit("建探针账号失败：%s %s" % (mobile, d))


def set_application(conn, uid, role, status):
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
            (ts, ts, uid, role, status, "探针", "probe", "模板探针", ts),
        )


def portal_token(mobile):
    _s, d = http("POST", BASE + "/base/appLogin", {"mobile": mobile, "password": PW})
    if not is_succ(d):
        sys.exit("门户登录失败 %s：%s" % (mobile, d))
    return d["data"]["token"], d["data"]["user"]["authorityId"]


def invite_code(uid):
    """和 Go 的 MakeInviteCode 独立实现一遍，交叉校验。"""
    import hashlib
    digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    n, b36 = uid, ""
    while n:
        b36 = digits[n % 36] + b36
        n //= 36
    h = hashlib.sha256(("tf-invite-v1:%d" % uid).encode()).hexdigest()[:4].upper()
    return "TF" + b36 + "-" + h


def cleanup(conn):
    """清掉上一轮（可能中断）留下的探针数据。"""
    rows = conn.execute(
        "SELECT id FROM template WHERE name LIKE ?", (TPL_NAME_PREFIX + "%",)
    ).fetchall()
    ids = [r[0] for r in rows]
    if ids:
        q = ",".join("?" * len(ids))
        conn.execute("DELETE FROM template_item WHERE template_id IN (%s)" % q, ids)
        conn.execute("DELETE FROM template WHERE id IN (%s)" % q, ids)
    mobiles = [MERCH_A, MERCH_B, PARENT]
    uids = [r[0] for r in conn.execute(
        "SELECT id FROM user WHERE mobile IN (?,?,?)", mobiles
    ).fetchall()]
    if uids:
        q = ",".join("?" * len(uids))
        conn.execute("DELETE FROM merchant_application WHERE uid IN (%s)" % q, uids)
        conn.execute("DELETE FROM wallet WHERE uid IN (%s)" % q, uids)
        conn.execute("DELETE FROM sys_users WHERE username IN (%s)"
                     % ",".join("?" * len(uids)), ["app_%d" % u for u in uids])
        conn.execute("DELETE FROM user WHERE id IN (%s)" % q, uids)
    conn.commit()


def main():
    conn = sqlite3.connect(DB)
    cleanup(conn)

    print("\n== 0. 造探针账号（真走 App 注册 + 审核 + 门户登录）==")
    admin_tok = admin_token()
    ua, ub, up = (ensure_app_user(m) for m in (MERCH_A, MERCH_B, PARENT))
    set_application(conn, ua, "merchant", "approved")
    set_application(conn, ub, "merchant", "approved")
    set_application(conn, up, "parent", "approved")
    conn.commit()
    ma_tok, ma_role = portal_token(MERCH_A)
    mb_tok, mb_role = portal_token(MERCH_B)
    pa_tok, pa_role = portal_token(PARENT)
    check(ma_role == 1002 and mb_role == 1002, "两个商家都是角色 1002",
          (ma_role, mb_role))
    check(pa_role == 1001, "家长是角色 1001", pa_role)

    print("\n== 1. 超管读：官方种子必须完整（不是「有返回就算过」）==")
    _s, d = tpl("list", {"page": 1, "pageSize": 20}, admin_tok)
    check(is_succ(d), "超管 list 通了", d)
    rows = d.get("data", {}).get("list", []) if isinstance(d, dict) else []
    by_kind = {r.get("kind"): r for r in rows if r.get("scope") == "official"}
    task_id = by_kind.get("task", {}).get("id")
    redeem_id = by_kind.get("redeem", {}).get("id")
    check(bool(task_id and redeem_id), "task / redeem 两份官方模板都在",
          [(r.get("kind"), r.get("id")) for r in rows])
    check(by_kind.get("task", {}).get("item_count") == 16,
          "官方任务模板 16 条（种子没被截断）", by_kind.get("task", {}).get("item_count"))
    check(by_kind.get("redeem", {}).get("item_count") == 15,
          "官方兑换模板 15 条", by_kind.get("redeem", {}).get("item_count"))
    check(by_kind.get("task", {}).get("is_default") == 1 and
          by_kind.get("redeem", {}).get("is_default") == 1,
          "两份官方模板都是默认", [(r.get("kind"), r.get("is_default")) for r in rows])

    _s, d = tpl("detail", {"id": task_id}, admin_tok)
    items = d.get("data", {}).get("items", []) if isinstance(d, dict) else []
    check(is_succ(d) and len(items) == 16, "detail 返回 16 条", len(items))
    extras = []
    for it in items:
        try:
            extras.append(json.loads(it.get("extra") or "{}"))
        except ValueError:
            pass
    check(any(e.get("allowOvertime") is True for e in extras),
          "Extra 里保留了 App 原生 camelCase 玩法参数（allowOvertime）", str(extras[:2]))
    check(all(k not in e for e in extras for k in ("title", "base_points", "planned_minutes")),
          "Extra 里没有 typed 列的键（否则下发会盖回旧值）", str(extras[:2]))

    _s, d = tpl("invite-code", {}, admin_tok)
    check(is_succ(d) and d["data"]["uid"] == 0 and d["data"]["invite_code"] == "",
          "超管没有「自己的」商家邀请码（返回空串）", d)

    print("\n== 2. 超管 CRUD 全链路（create → item/save → update → delete）==")
    _s, d = tpl("create", {
        "name": TPL_NAME_PREFIX + "官方-超管", "kind": "task",
        "scope": "official", "owner_uid": 0, "note": "probe", "sort": 99,
    }, admin_tok)
    check(is_succ(d) and d.get("data", {}).get("id", 0) > 0, "新建官方模板", d)
    probe_id = d["data"]["id"] if is_succ(d) else 0
    if not probe_id:
        cleanup(conn)
        return finish()

    _s, d = tpl("item/save", {
        "template_id": probe_id, "item_id": "probe_item_1",
        "name": "探针条目", "base_points": 7, "planned_minutes": 25,
        "extra": '{"price":30,"allowOvertime":true,"base_points":99}',
    }, admin_tok)
    check(is_succ(d) and d.get("data", {}).get("id", 0) > 0, "保存条目", d)
    _s, d = tpl("detail", {"id": probe_id}, admin_tok)
    it = (d.get("data", {}).get("items") or [{}])[0] if isinstance(d, dict) else {}
    check(is_succ(d) and it.get("name") == "探针条目" and it.get("base_points") == 7,
          "条目落库且值正确", it)
    check(it.get("extra") == '{"allowOvertime":true}',
          "Extra 的 typed 键被剔除，只留玩法参数", it.get("extra"))

    _s, d = tpl("item/save", {
        "id": it.get("id"), "template_id": probe_id, "item_id": "probe_item_1",
        "name": "探针条目-改", "base_points": 9, "extra": "{}",
    }, admin_tok)
    check(is_succ(d), "改条目成功", d)
    _s, d = tpl("detail", {"id": probe_id}, admin_tok)
    items2 = d.get("data", {}).get("items", []) if isinstance(d, dict) else []
    check(len(items2) == 1 and items2[0].get("base_points") == 9,
          "改条目是覆盖而不是又插一条", items2)

    _s, d = tpl("update", {"id": probe_id, "name": TPL_NAME_PREFIX + "官方-改过", "sort": 98}, admin_tok)
    check(is_succ(d), "改模板头成功", d)
    _s, d = tpl("detail", {"id": probe_id}, admin_tok)
    check(is_succ(d) and d["data"].get("name") == TPL_NAME_PREFIX + "官方-改过" and
          d["data"].get("sort") == 98, "改的是头（名字 + 排序），条目没被清空",
          d.get("data", {}).get("name"))

    print("\n== 3. 家长：casbin 那一层就不给 ==")
    _s, d = tpl("list", {"page": 1, "pageSize": 20}, pa_tok)
    check(isinstance(d, dict) and d.get("code") == 7,
          "家长调模板列表被 casbin 拒（code=7 权限不足）", d)
    check(not is_succ(d), "家长没拿到任何模板数据", d)

    print("\n== 4. 商家：谎报 scope/owner 被收敛，且只能看自己的 ==")
    _s, d = tpl("create", {
        "name": TPL_NAME_PREFIX + "商家A-谎报", "kind": "task",
        "scope": "official", "owner_uid": 0, "is_default": True,
    }, ma_tok)
    check(is_succ(d) and d.get("data", {}).get("id", 0) > 0,
          "商家能建自己的模板（谎报 scope 也不报错）", d)
    ma_tpl1 = d["data"]["id"] if is_succ(d) else 0

    if ma_tpl1:
        _s, d = tpl("detail", {"id": ma_tpl1}, admin_tok)
        row = d.get("data", {}) if isinstance(d, dict) else {}
        check(row.get("scope") == "merchant" and row.get("owner_uid") == ua,
              "谎报的 scope/owner_uid 被强制收敛成 merchant + 自己",
              (row.get("scope"), row.get("owner_uid"), ua))
        check(row.get("is_default") == 1, "商家自己的默认模板能设上", row.get("is_default"))

    _s, d = tpl("create", {
        "name": TPL_NAME_PREFIX + "商家A-第二份", "kind": "task",
        "scope": "merchant", "owner_uid": ua,
    }, ma_tok)
    ma_tpl2 = d["data"]["id"] if is_succ(d) else 0
    check(ma_tpl2 > 0, "商家建第二份模板", d)

    if ma_tpl2:
        _s, d = tpl("set-default", {"id": ma_tpl2}, ma_tok)
        check(is_succ(d), "商家把第二份设为默认", d)
        _s, d = tpl("list", {"page": 1, "pageSize": 20}, ma_tok)
        mine = d.get("data", {}).get("list", []) if isinstance(d, dict) else []
        by_id = {r.get("id"): r for r in mine}
        check(by_id.get(ma_tpl1, {}).get("is_default") == 0 and
              by_id.get(ma_tpl2, {}).get("is_default") == 1,
              "同组旧默认被清掉、只有一份默认（setDefaultInGroup）",
              [(r.get("id"), r.get("is_default")) for r in mine])
        check(all(r.get("scope") == "merchant" and r.get("owner_uid") == ua for r in mine),
              "商家 list 只返回自己的模板（看不到官方 / 商家 B）",
              [(r.get("id"), r.get("scope"), r.get("owner_uid")) for r in mine])

    print("\n== 5. 隔离：跨商家 / 碰官方 / 家长，三种都要拒 ==")
    if ma_tpl1:
        for label, tok, body in [
            ("商家 B 改商家 A 的模板头", mb_tok, {"id": ma_tpl1, "name": "越权改"}),
            ("商家 B 删商家 A 的模板", mb_tok, {"id": ma_tpl1}),
            ("商家 B 看商家 A 的详情", mb_tok, {"id": ma_tpl1}),
            ("商家 A 看官方模板", ma_tok, {"id": task_id}),
            ("商家 A 删官方模板", ma_tok, {"id": task_id}),
        ]:
            path = "update" if "name" in body else ("detail" if "看" in label else "delete")
            _s, d = tpl(path, body, tok)
            check(is_reject(d), label + " 被拒", d)

        _s, d = tpl("item/save", {
            "template_id": task_id, "name": "越权条目", "extra": "{}",
        }, ma_tok)
        check(is_reject(d), "商家往官方模板里塞条目被拒", d)

    _s, d = tpl("invite-code", {}, ma_tok)
    check(is_succ(d) and d["data"]["uid"] == ua and
          d["data"]["invite_code"] == invite_code(ua),
          "商家邀请码 = 独立实现的公式（和平台端逐字节一致）",
          (d.get("data"), invite_code(ua)))

    print("\n== 6. 硬删：不留软删行占着 uk_tpl_item ==")
    for tid in [x for x in (probe_id, ma_tpl1, ma_tpl2) if x]:
        _s, d = tpl("delete", {"id": tid}, admin_tok)
        check(is_succ(d), "删除模板 id=%d" % tid, d)
    left = conn.execute(
        "SELECT COUNT(*) FROM template_item WHERE template_id IN (?,?,?)",
        (probe_id, ma_tpl1, ma_tpl2),
    ).fetchone()[0]
    check(left == 0, "条目是硬删（连软删行都不剩，不会撞唯一索引）", left)

    print("\n== 7. 收尾：官方模板没被动过 ==")
    _s, d = tpl("list", {"page": 1, "pageSize": 20}, admin_tok)
    rows = d.get("data", {}).get("list", []) if isinstance(d, dict) else []
    check(sorted((r.get("kind"), r.get("item_count"), r.get("is_default"))
                 for r in rows if r.get("scope") == "official") ==
          [("redeem", 15, 1), ("task", 16, 1)],
          "官方模板仍是 2 份、条目数 16/15、都是默认",
          [(r.get("kind"), r.get("item_count"), r.get("is_default")) for r in rows])
    check(len(rows) == 2, "探针模板全清干净了", len(rows))

    cleanup(conn)
    conn.close()
    return finish()


def finish():
    ok = sum(1 for r in results if r[0])
    bad = [r for r in results if not r[0]]
    print("\n" + "=" * 60)
    print("PASS %d / FAIL %d" % (ok, len(bad)))
    for _ok, label, detail in bad:
        print("  ✗ %s -- %s" % (label, str(detail)[:200]))
    print("=" * 60)
    return 0 if not bad else 1


if __name__ == "__main__":
    sys.exit(main())
