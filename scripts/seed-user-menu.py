#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""把「App 用户」这一页登记进**已经初始化过**的后台库（幂等）。

    python scripts/seed-user-menu.py                 # 用 ../tf-platform-api/data/tf.db
    python scripts/seed-user-menu.py --db X.db
    python scripts/seed-user-menu.py --dry-run       # 只打印要做什么

为什么需要这个脚本：GVA 的菜单 / 接口权限是**数据**，不是代码。
`source/system/*.go` 里那几份种子只在 `initdb` 时生效，而共用 tf.db 的后台
**永远没有机会再跑一次 initdb**（`GVA_DB != nil` 时它直接拒绝）。

官方路子是 `GET /api/syncApi` → `POST /api/enterSyncApi` → `POST /api/setApiRoles`
（见 README-TF §3.2），那条路要先用管理员密码登进去。这个脚本走的是
「拿不到管理员密码 / 想放进部署流程」时的那条：直接写同样四张表。

⚠️ 顺序不能反：菜单行要拿到自增 id 才能写 `sys_authority_menus`，
   否则菜单建了但谁都看不见（登录后左侧就是没有这一项，**不报错**）。
"""

import argparse
import os
import sqlite3
import sys
from datetime import datetime

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_DB = os.path.join(HERE, "..", "..", "tf-platform-api", "data", "tf.db")

PARENT_PATH = "mall"          # 「商城运营」这一组
MENU_NAME = "appUser"
MENU_TITLE = "App 用户"
MENU_PATH = "user"
MENU_COMPONENT = "view/mall/user/index.vue"
MENU_ICON = "user"
AUTHORITY = 888               # 超级管理员

APIS = [
    ("POST", "/mall/user/list", "App 用户列表"),
    ("GET", "/mall/user/detail", "App 用户详情"),
    ("GET", "/mall/user/ledger", "App 用户账本"),
    ("POST", "/mall/user/adjust", "调整用户余额"),
    ("POST", "/mall/user/reset-password", "重置用户密码"),
]
API_GROUP = "App用户"


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
    todo = []

    # ---------- ① 菜单行 ----------
    row = conn.execute(
        "SELECT id FROM sys_base_menus WHERE name = ? AND deleted_at IS NULL", (MENU_NAME,)
    ).fetchone()
    if row:
        menu_id = row["id"]
        print("· 菜单已存在：id=%d（跳过）" % menu_id)
    else:
        parent = conn.execute(
            "SELECT id FROM sys_base_menus WHERE path = ? AND parent_id = 0 AND deleted_at IS NULL",
            (PARENT_PATH,),
        ).fetchone()
        if not parent:
            sys.exit("找不到父菜单 path=%s —— 这个库不是本项目初始化出来的？" % PARENT_PATH)
        todo.append((
            "INSERT INTO sys_base_menus (created_at, updated_at, deleted_at, menu_level,"
            " parent_id, path, name, hidden, component, sort, active_name, keep_alive,"
            " default_menu, title, icon, close_tab, transition_type)"
            " VALUES (?,?,NULL,1,?,?,?,0,?,0,'',0,0,?,?,0,'')",
            (ts, ts, parent["id"], MENU_PATH, MENU_NAME, MENU_COMPONENT, MENU_TITLE, MENU_ICON),
        ))
        menu_id = None  # 插入后取

    # ---------- ② 菜单授权（少了这行 = 登录后看不见） ----------
    if menu_id is not None:
        has = conn.execute(
            "SELECT 1 FROM sys_authority_menus WHERE sys_base_menu_id=? AND sys_authority_authority_id=?",
            (menu_id, AUTHORITY),
        ).fetchone()
        if has:
            print("· 菜单已授权给 %d（跳过）" % AUTHORITY)
        else:
            todo.append((
                "INSERT INTO sys_authority_menus (sys_base_menu_id, sys_authority_authority_id) VALUES (?,?)",
                (menu_id, AUTHORITY),
            ))

    # ---------- ③ 接口登记 ----------
    for method, path, desc in APIS:
        has = conn.execute(
            "SELECT 1 FROM sys_apis WHERE path=? AND method=? AND deleted_at IS NULL", (path, method)
        ).fetchone()
        if has:
            print("· 接口已登记：%s %s（跳过）" % (method, path))
            continue
        todo.append((
            "INSERT INTO sys_apis (created_at, updated_at, deleted_at, path, description, api_group, method)"
            " VALUES (?,?,NULL,?,?,?,?)",
            (ts, ts, path, desc, API_GROUP, method),
        ))

    # ---------- ④ casbin（真正拦请求的那一层） ----------
    for method, path, _ in APIS:
        has = conn.execute(
            "SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0=? AND v1=? AND v2=?",
            (str(AUTHORITY), path, method),
        ).fetchone()
        if has:
            print("· casbin 已授权：%s %s（跳过）" % (method, path))
            continue
        todo.append((
            "INSERT INTO casbin_rule (ptype, v0, v1, v2, v3, v4, v5) VALUES ('p',?,?,?,'','','')",
            (str(AUTHORITY), path, method),
        ))

    if not todo:
        print("\n已经全部登记过了，什么都没做。")
        return 0

    if args.dry_run:
        print("\n[dry-run] 会执行 %d 条：" % len(todo))
        for sql, params in todo:
            print("  -", sql.split("(")[0].strip(), params)
        return 0

    done = 0
    for sql, params in todo:
        cur = conn.execute(sql, params)
        done += 1
        if sql.startswith("INSERT INTO sys_base_menus"):
            menu_id = cur.lastrowid
            # 菜单 id 拿到之后立刻补授权
            conn.execute(
                "INSERT INTO sys_authority_menus (sys_base_menu_id, sys_authority_authority_id) VALUES (?,?)",
                (menu_id, AUTHORITY),
            )
            done += 1  # 授权那一条也算
            print("· 菜单已建：id=%d，并授权给 %d" % (menu_id, AUTHORITY))
    conn.commit()
    # 报实际执行条数，不是 len(todo)：菜单那步会**额外**插一条授权
    print("\n写入 %d 条，完成。**重启后台**（或重新登录）才会看到菜单。" % done)
    return 0


if __name__ == "__main__":
    sys.exit(main())
