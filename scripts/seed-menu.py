#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""把 mall 的页面登记进**已经初始化过**的后台库（幂等），菜单 + 菜单授权。

    python scripts/seed-menu.py                 # 用 ../tf-platform-api/data/tf.db
    python scripts/seed-menu.py --dry-run
    python scripts/seed-menu.py --only dashboard

## 为什么需要它

GVA 的菜单是**数据**（`sys_base_menus` + `sys_authority_menus`）：
`source/system/*.go` 那几份种子只在 `initdb` 时生效，而共用 tf.db 的后台
永远没有机会再跑一次 initdb（`GVA_DB != nil` 时它直接拒绝）。

## ★ 最重要的一条：**页面不存在就不建菜单**

菜单建了、`component` 指向的 `.vue` 不存在 ⇒ 点进去是**空白页**，
前端只报一个 404 chunk，而人的第一反应是「权限配错了」，会去查 casbin。
所以这个脚本**先看文件在不在**，不在就跳过并打印出来 —— 它比人可靠。

## 顺序陷阱

菜单行要拿到自增 id 才能写 `sys_authority_menus`，否则菜单建了但**谁都看不见**
（登录后左侧就是没有这一项，不报错）。下面用 `lastrowid` 立刻补。

## 接口权限不放这里

菜单只管「能不能看见入口」，接口还另需 `sys_apis` + `casbin_rule` 两处，
见 `scripts/seed-mall.py`。只加菜单 ⇒ 点进去第一屏就 `{"code":7,"权限不足"}`。
"""

import argparse
import os
import sqlite3
import sys
from datetime import datetime

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_DB = os.path.join(HERE, "..", "..", "tf-platform-api", "data", "tf.db")
WEB_VIEW = os.path.join(HERE, "..", "web", "src", "view")
AUTHORITY = 888  # 超级管理员
PARENT_PATH = "mall"  # 「商城运营」这一组

# (name, title, path, component 相对 web/src, icon)
PAGES = [
    ("mallDashboard", "经营看板", "dashboard", "mall/dashboard/index.vue", "data-analysis"),
    ("appUser", "App 用户", "user", "mall/user/index.vue", "user"),
    ("merchant", "招商审核", "merchant", "mall/merchant/index.vue", "message"),
    # 下面这些**页面还没写**，脚本会自动跳过（并提示），别硬建菜单。
    ("mallRelease", "游戏版本", "release", "mall/release/index.vue", "upload"),
    ("mallRedeem", "兑换商城", "redeem", "mall/redeem/index.vue", "shopping-cart"),
    ("mallTask", "任务记录", "task", "mall/task/index.vue", "list"),
    # ★ #16 起「心愿商品 / 心愿单 / 游戏列表 / 游戏激活码」由
    #   scripts/seed-menu-level2.py 管（升成和商城运营平级的一级菜单）。
    #   这里再列 mallWish / mallGame 会怎样：mallWish 已存在会跳过，但
    #   mallGame 这名字库里没有、而 mall/game/index.vue 现在**真的存在**
    #   ⇒ 本脚本会在「商城运营」下再建一个重复的「游戏管理」并授权 888。
    #   别把这两条加回来。
]


def now() -> str:
    """GVA 存进 SQLite 的时间格式（照抄现有行的形状，别用 isoformat）。"""
    d = datetime.now().astimezone()
    return d.strftime("%Y-%m-%d %H:%M:%S.%f") + d.strftime("%z")[:3] + ":" + d.strftime("%z")[3:]


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--db", default=DEFAULT_DB)
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--only", default="", help="只处理某一页（菜单 name）")
    args = ap.parse_args()

    db = os.path.abspath(args.db)
    if not os.path.exists(db):
        sys.exit("找不到库：%s" % db)
    conn = sqlite3.connect(db)
    conn.row_factory = sqlite3.Row
    ts = now()

    parent = conn.execute(
        "SELECT id FROM sys_base_menus WHERE path = ? AND parent_id = 0 AND deleted_at IS NULL",
        (PARENT_PATH,),
    ).fetchone()
    if not parent:
        sys.exit("找不到父菜单 path=%s —— 这个库不是本项目初始化出来的？" % PARENT_PATH)

    added, skipped_no_file, already = [], [], []
    for name, title, path, component, icon in PAGES:
        if args.only and name != args.only:
            continue
        vue = os.path.join(WEB_VIEW, component)
        if not os.path.exists(vue):
            skipped_no_file.append("%s（缺 %s）" % (title, component))
            continue
        row = conn.execute(
            "SELECT id FROM sys_base_menus WHERE name = ? AND deleted_at IS NULL", (name,)
        ).fetchone()
        if row:
            already.append(title)
            continue
        if args.dry_run:
            added.append("%s → %s" % (title, component))
            continue
        cur = conn.execute(
            "INSERT INTO sys_base_menus (created_at, updated_at, deleted_at, menu_level,"
            " parent_id, path, name, hidden, component, sort, active_name, keep_alive,"
            " default_menu, title, icon, close_tab, transition_type)"
            " VALUES (?,?,NULL,1,?,?,?,0,?,0,'',0,0,?,?,0,'')",
            (ts, ts, parent["id"], path, name, "view/" + component, title, icon),
        )
        menu_id = cur.lastrowid
        # 拿到 id 立刻授权，否则菜单建了但谁都看不见（静默）
        conn.execute(
            "INSERT INTO sys_authority_menus (sys_base_menu_id, sys_authority_authority_id)"
            " VALUES (?,?)",
            (menu_id, AUTHORITY),
        )
        added.append("%s（id=%d）" % (title, menu_id))

    if not args.dry_run:
        conn.commit()

    for t in already:
        print("· 已有菜单：%s（跳过）" % t)
    for t in skipped_no_file:
        print("· 跳过（页面不存在，建了也是空白页）：%s" % t)
    if args.dry_run:
        print("\n[dry-run] 会新增 %d 个菜单：" % len(added))
    for t in added:
        print("  + %s" % t)
    if not added and not args.dry_run:
        print("\n没有需要新增的菜单。")
    else:
        print("\n完成。**重新登录**（或刷新页面）才会看到新菜单。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
