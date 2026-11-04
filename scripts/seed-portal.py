#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""把「家长门户 / 商家门户」登记进**已经初始化过**的后台库（幂等）。

    python scripts/seed-portal.py                 # 用 ../tf-platform-api/data/tf.db
    python scripts/seed-portal.py --db X.db
    python scripts/seed-portal.py --dry-run       # 只打印要做什么

# 它到底写了几张表

| 表 | 作用 | 少写会怎样 |
| --- | --- | --- |
| `sys_authorities` | 角色 1001 家长 / 1002 商家 | 影子账号挂到不存在的角色 ⇒ 登录成功、进去一片空白 |
| `sys_base_menus` | 左侧菜单 | 登录成功、左侧什么都没有（**没有任何报错**） |
| `sys_authority_menus` | 菜单给谁看 | 菜单建了但没人看得见 |
| `sys_apis` | 接口登记（后台「API 管理」页展示用） | 只影响展示，不影响鉴权 |
| `casbin_rule` | **真正拦请求的那一层** | 菜单点得开，每个请求 403 `权限不足` |

# 为什么不能靠 initdb

`source/system/*.go` 里那几份种子只在 `initdb` 时生效，而共用 tf.db 的后台
**永远没有机会再跑一次 initdb**（`GVA_DB != nil` 时 `/init/initdb` 直接拒绝）。
官方路子 `syncApi → enterSyncApi → setApiRoles` 要先用管理员密码登进去；
这个脚本走「放进部署流程」的那条：直接写同样几张表。

# ⚠️ 写完要重载 casbin

casbin 的策略在进程启动时从库里读进内存。直接写库之后：
  - 重启后台，或
  - 用管理员 token 调 `POST /api/freshCasbin`

否则「库里明明有那一行，请求还是 403」—— 这个坑排查起来很费时间。
"""

import argparse
import os
import sqlite3
import sys
from datetime import datetime

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_DB = os.path.join(HERE, "..", "..", "tf-platform-api", "data", "tf.db")

# ============================ 角色 ============================

# (authority_id, authority_name, default_router)
#
# ⚠️ `default_router` 填的是**路由 name**（不是 path）。前端登录后
# `router.replace({name: authority.defaultRouter})` —— 名字对不上时
# 不会报错，只会停在空白页。
AUTHORITIES = [
    (1001, "家长", "parentOverview"),
    (1002, "商家", "merchantItems"),
]

# ============================ 菜单 ============================
#
# 每个门户一棵两层的树：`routerHolder.vue` 分组 + 具体页面。
# `name` 必须全局唯一 —— 商家分组不能叫 `merchant`，那个名字已经被
# 「招商审核」（id=42）占了，重名会让路由互相覆盖。
#
# 元组：(path, name, title, component, icon, sort)

MENUS = {
    "parent": [
        ("parent", "parent", "家长中心", "view/routerHolder.vue", "user", 20),
        ("overview", "parentOverview", "概览", "view/portal/parent/overview.vue", "data-line", 1),
        ("wishes", "parentWishes", "心愿单", "view/portal/parent/wishes.vue", "present", 2),
        ("records", "parentRecords", "兑换记录", "view/portal/parent/records.vue", "tickets", 3),
    ],
    "merchant": [
        ("merchantPortal", "merchantPortal", "商家中心", "view/routerHolder.vue", "shop", 21),
        ("items", "merchantItems", "我的商品", "view/portal/merchant/items.vue", "goods", 1),
        ("status", "merchantStatus", "入驻状态", "view/portal/merchant/status.vue", "info-filled", 2),
    ],
}

# ============================ 接口 ============================
#
# `public=True` 的只登记、不写 casbin：它在公开路由组里，压根不走 casbin。
# 登记只是为了让运营在「API 管理」页里能看到它存在。

APIS = [
    # (method, path, description, group, public)
    ("POST", "/base/appLogin", "家长/商家用 App 账号登录", "门户", True),
    ("GET", "/mall/portal/parent/overview", "家长概览", "门户·家长", False),
    ("POST", "/mall/portal/parent/wishes", "家长心愿单", "门户·家长", False),
    ("POST", "/mall/portal/parent/records", "家长兑换记录", "门户·家长", False),
    ("GET", "/mall/portal/merchant/status", "商家入驻状态", "门户·商家", False),
    ("POST", "/mall/portal/merchant/items", "商家我的商品", "门户·商家", False),
    ("POST", "/mall/portal/merchant/item/save", "商家保存商品", "门户·商家", False),
]

# 哪个角色能调哪些接口。**故意分开写**：
# 家长拿不到商家接口，商家也拿不到家长接口 —— 两者数据互不可见。
ROLE_APIS = {
    1001: [
        ("GET", "/mall/portal/parent/overview"),
        ("POST", "/mall/portal/parent/wishes"),
        ("POST", "/mall/portal/parent/records"),
    ],
    1002: [
        ("GET", "/mall/portal/merchant/status"),
        ("POST", "/mall/portal/merchant/items"),
        ("POST", "/mall/portal/merchant/item/save"),
    ],
}

# 所有登录后都要调的「外壳接口」。两个角色都要。
#
# ⚠️ 这一组最容易漏，而且症状是**最不像权限问题的**：
#   - `/menu/getMenu`    登录后拿动态路由。没有它 ⇒ 左侧空白，控制台只有一行 403
#   - `/user/getUserInfo` 刷新页面时取回自己。没有它 ⇒ 一按 F5 就被踢回登录页
#   - `/jwt/jsonInBlacklist` 退出登录。没有它 ⇒ 点退出没反应，token 还在
#   - `/user/setSelfInfo`  保存界面偏好（主题 / 侧边栏折叠）
#
# 它们只碰「调用者自己」，给 1001 / 1002 开出来不扩大数据面。
SHELL_APIS = [
    ("POST", "/menu/getMenu"),
    ("GET", "/user/getUserInfo"),
    ("POST", "/jwt/jsonInBlacklist"),
    ("PUT", "/user/setSelfInfo"),
]

AUTHORITY_APIS = {aid: pairs + SHELL_APIS for aid, pairs in ROLE_APIS.items()}


def now() -> str:
    """GVA 存进 SQLite 的时间格式：`2026-10-30 00:54:51.2136132+08:00`。

    照抄现有行的形状，别用 `isoformat()`（`T` 分隔）—— 驱动读回时按固定
    几种 layout 解析，猜错的表现是**整棵菜单接口报错**，而不是这一行没了。
    """
    d = datetime.now().astimezone()
    return d.strftime("%Y-%m-%d %H:%M:%S.%f") + d.strftime("%z")[:3] + ":" + d.strftime("%z")[3:]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--db", default=DEFAULT_DB)
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()

    db = os.path.abspath(args.db)
    if not os.path.exists(db):
        sys.exit("找不到库：%s" % db)
    conn = sqlite3.connect(db)
    conn.row_factory = sqlite3.Row
    ts = now()
    actions = []

    # ---------- ① 角色 ----------
    for aid, name, default_router in AUTHORITIES:
        row = conn.execute(
            "SELECT authority_id FROM sys_authorities WHERE authority_id = ? AND deleted_at IS NULL",
            (aid,),
        ).fetchone()
        if row:
            print("· 角色已存在：%d %s（跳过）" % (aid, name))
            continue
        actions.append({
            "kind": "authority", "aid": aid, "name": name,
            "sql": "INSERT INTO sys_authorities (created_at, updated_at, deleted_at,"
                   " authority_id, authority_name, parent_id, default_router)"
                   " VALUES (?,?,NULL,?,?,0,?)",
            "params": (ts, ts, aid, name, default_router),
        })

    # ---------- ② 菜单（父 id 用**名字**记，插完才知道） ----------
    plan = []          # (role, aid, level, parent_name, path, name, title, comp, icon, sort)
    for role, items in MENUS.items():
        aid = 1001 if role == "parent" else 1002
        parent_name = None
        for idx, (path, name, title, comp, icon, sort) in enumerate(items):
            plan.append((role, aid, 0 if idx == 0 else 1,
                         None if idx == 0 else parent_name,
                         path, name, title, comp, icon, sort))
            if idx == 0:
                parent_name = name

    for _role, aid, level, parent_name, path, name, title, comp, icon, sort in plan:
        row = conn.execute(
            "SELECT id FROM sys_base_menus WHERE name = ? AND deleted_at IS NULL", (name,)
        ).fetchone()
        if row:
            print("· 菜单已存在：%s（id=%d，跳过）" % (name, row["id"]))
        else:
            actions.append({
                "kind": "menu", "name": name, "parent_name": parent_name,
                "sql": "INSERT INTO sys_base_menus (created_at, updated_at, deleted_at, menu_level,"
                       " parent_id, path, name, hidden, component, sort, active_name, keep_alive,"
                       " default_menu, title, icon, close_tab, transition_type)"
                       " VALUES (?,?,NULL,?,?,?,?,0,?,?,'',0,0,?,?,0,'')",
                "params": (ts, ts, level, None, path, name, comp, sort, title, icon),
            })
        actions.append({
            "kind": "menu_auth", "name": name, "aid": aid,
        })

    # ---------- ③ 接口登记 ----------
    for method, path, desc, group, _public in APIS:
        has = conn.execute(
            "SELECT 1 FROM sys_apis WHERE path=? AND method=? AND deleted_at IS NULL", (path, method)
        ).fetchone()
        if has:
            print("· 接口已登记：%s %s（跳过）" % (method, path))
            continue
        actions.append({
            "kind": "api",
            "sql": "INSERT INTO sys_apis (created_at, updated_at, deleted_at, path, description, api_group, method)"
                   " VALUES (?,?,NULL,?,?,?,?)",
            "params": (ts, ts, path, desc, group, method),
        })

    # ---------- ④ casbin（真正拦请求的那一层） ----------
    for aid, pairs in AUTHORITY_APIS.items():
        for method, path in pairs:
            has = conn.execute(
                "SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0=? AND v1=? AND v2=?",
                (str(aid), path, method),
            ).fetchone()
            if has:
                print("· casbin 已授权：%d %s %s（跳过）" % (aid, method, path))
                continue
            actions.append({
                "kind": "casbin", "aid": aid, "method": method, "path": path,
                "sql": "INSERT INTO casbin_rule (ptype, v0, v1, v2, v3, v4, v5) VALUES ('p',?,?,?,'','','')",
                "params": (str(aid), path, method),
            })

    # 已经存在的菜单也要保证授权 —— 否则「菜单建过一次、授权那步失败过」
    # 之后再跑这个脚本会一直跳过，左侧永远空着。
    existing_menu_ids = {}
    for _role, aid, _lvl, _pn, _p, name, _t, _c, _i, _s in plan:
        row = conn.execute(
            "SELECT id FROM sys_base_menus WHERE name = ? AND deleted_at IS NULL", (name,)
        ).fetchone()
        if row:
            existing_menu_ids[name] = row["id"]

    filtered = []
    for act in actions:
        if act["kind"] != "menu_auth":
            filtered.append(act)
            continue
        mid = existing_menu_ids.get(act["name"])
        if mid is None:
            filtered.append(act)      # 本轮才建，执行时再取 id
            continue
        has = conn.execute(
            "SELECT 1 FROM sys_authority_menus WHERE sys_base_menu_id=? AND sys_authority_authority_id=?",
            (mid, act["aid"]),
        ).fetchone()
        if has:
            print("· 菜单已授权：%s -> %d（跳过）" % (act["name"], act["aid"]))
            continue
        filtered.append(act)
    actions = filtered

    if not actions:
        print("\n已经全部登记过了，什么都没做。")
        return 0

    if args.dry_run:
        print("\n[dry-run] 会执行 %d 条：" % len(actions))
        for act in actions:
            if act["kind"] == "menu_auth":
                print("  - menu_auth %s -> %d" % (act["name"], act["aid"]))
            else:
                print("  -", act["sql"].split("(")[0].strip(), act["params"])
        return 0

    # ---------- 执行 ----------
    created = {}       # name -> id（本轮新建的）
    done = 0
    for act in actions:
        kind = act["kind"]
        if kind == "menu":
            params = list(act["params"])
            if act["parent_name"] is not None:
                pid = created.get(act["parent_name"])
                if pid is None:
                    # 父菜单是以前建的（本轮跳过），回库里查
                    row = conn.execute(
                        "SELECT id FROM sys_base_menus WHERE name=? AND deleted_at IS NULL",
                        (act["parent_name"],),
                    ).fetchone()
                    pid = row["id"] if row else None
                if pid is None:
                    sys.exit("父菜单 %s 不存在，中止（不要留下孤儿菜单）" % act["parent_name"])
                params[3] = pid
            cur = conn.execute(act["sql"], tuple(params))
            created[act["name"]] = cur.lastrowid
            done += 1
            print("· 菜单已建：%s（id=%d，parent=%s）" % (act["name"], cur.lastrowid, params[3]))
            continue

        if kind == "menu_auth":
            mid = created.get(act["name"])
            if mid is None:
                mid = existing_menu_ids.get(act["name"])
            if mid is None:
                row = conn.execute(
                    "SELECT id FROM sys_base_menus WHERE name=? AND deleted_at IS NULL",
                    (act["name"],),
                ).fetchone()
                mid = row["id"] if row else None
            if mid is None:
                print("  ! 菜单 %s 找不到，跳过授权" % act["name"])
                continue
            conn.execute(
                "INSERT INTO sys_authority_menus (sys_base_menu_id, sys_authority_authority_id) VALUES (?,?)",
                (mid, act["aid"]),
            )
            done += 1
            print("· 菜单已授权：%s(id=%d) -> %d" % (act["name"], mid, act["aid"]))
            continue

        conn.execute(act["sql"], act["params"])
        done += 1

    conn.commit()

    print("\n写入 %d 条，完成。" % done)
    print("⚠️ 还要重载 casbin 才生效：重启后台，或管理员 token 调 POST /api/freshCasbin")
    print("⚠️ 登录页「家长 / 商家」两个 tab 用的是 App 的手机号 + 密码；")
    print("   没在 App 里申请过（或还没通过审核）的账号会在登录这一步被拒。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
