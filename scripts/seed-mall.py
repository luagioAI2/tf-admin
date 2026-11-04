#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""把 **mall 模块全部接口** 登记进后台库（幂等），补 `sys_apis` + `casbin_rule`。

    python scripts/seed-mall.py                 # 用 ../tf-platform-api/data/tf.db
    python scripts/seed-mall.py --db X.db
    python scripts/seed-mall.py --dry-run       # 只打印要做什么
    python scripts/seed-mall.py --no-refresh    # 不尝试热加载 casbin

## 为什么需要它

GVA 的接口权限是**数据**：路由注册了、handler 写好了、页面也画了，
少 `sys_apis` 一行或 `casbin_rule` 一行，请求就返回
`{"code":7,"msg":"权限不足"}` —— 而**页面看着完全正常**。
2026-11-03 实测：14 个 mall 接口只有 3 个通（`user/list`、`merchant/list`、
`merchant/pending-count`），其余 11 个全是「权限不足」。

## 两个必须知道的坑

1. **写完库不等于立刻生效**：casbin 的策略是进程启动时加载的。
   直接改库之后必须让后台重新加载 —— `GET /api/freshCasbin`（公开接口），
   或者重启后台。否则「库里有行、请求还是 403」，看着像写错了表。
   这个脚本默认会试着调一次，调不到就只打印提示（离线跑也不报错）。

2. **`casbin_rule` 的表名不是 `sys_casbin_rule`**（GVA 里就叫 `casbin_rule`）。
   写成后者会「新建一张空表 + 权限一点没变」，不报错。

## 菜单不放这里

菜单加了但 `component` 指向的 `.vue` 不存在 ⇒ 点进去是空白页（前端只报一个
404 chunk），而人的第一反应是「权限配错了」。所以**先写页面，再加菜单**；
菜单由各自的 `seed-*-menu.py` 负责。
"""

import argparse
import json
import os
import sqlite3
import sys
import urllib.request
from datetime import datetime

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_DB = os.path.join(HERE, "..", "..", "tf-platform-api", "data", "tf.db")
AUTHORITY = 888  # 超级管理员

# (method, path, 说明, 分组) —— 与 server/router/mall/mall.go 一一对应
APIS = [
    # 看板
    ("GET", "/mall/dashboard/overview", "经营看板总览", "看板"),
    # 游戏
    ("POST", "/mall/game/list", "游戏列表", "游戏"),
    ("POST", "/mall/game/options", "游戏下拉选项", "游戏"),
    ("POST", "/mall/game/update", "更新游戏", "游戏"),
    # 版本
    ("POST", "/mall/release/list", "游戏版本列表", "游戏版本"),
    ("POST", "/mall/release/publish", "发布版本", "游戏版本"),
    ("POST", "/mall/release/deactivate", "下架版本", "游戏版本"),
    # ⚠️ `delete` 是 DELETE 方法。漏了它，页面上「删除版本」按钮会 403，
    #    而列表 / 发布 / 下架都正常 —— 看着像按钮写错了。
    ("DELETE", "/mall/release/delete", "删除版本", "游戏版本"),
    # App 用户
    ("POST", "/mall/user/list", "App 用户列表", "App用户"),
    ("GET", "/mall/user/detail", "App 用户详情", "App用户"),
    ("GET", "/mall/user/ledger", "App 用户账本", "App用户"),
    ("POST", "/mall/user/adjust", "调整用户余额", "App用户"),
    ("POST", "/mall/user/reset-password", "重置用户密码", "App用户"),
    # 兑换
    ("POST", "/mall/redeem/item/list", "兑换商品列表", "兑换"),
    ("POST", "/mall/redeem/item/update", "更新兑换商品", "兑换"),
    ("POST", "/mall/redeem/record/list", "兑换记录", "兑换"),
    # 心愿
    ("POST", "/mall/wish/item/list", "心愿商品列表", "心愿"),
    ("POST", "/mall/wish/item/update", "更新心愿商品", "心愿"),
    ("POST", "/mall/wish/item/create", "新建心愿商品", "心愿"),
    ("POST", "/mall/wish/item/delete", "删除心愿商品", "心愿"),
    ("POST", "/mall/wish/order/list", "心愿订单", "心愿"),
    # 激活码（家长线下买卡 → 孩子输码解锁一款游戏）
    ("POST", "/mall/wish/code/list", "激活码列表", "心愿"),
    ("POST", "/mall/wish/code/generate", "批量生成激活码", "心愿"),
    ("POST", "/mall/wish/code/void", "作废激活码", "心愿"),
    ("POST", "/mall/wish/code/batches", "激活码批次号", "心愿"),
    # 任务
    ("POST", "/mall/task/list", "任务列表", "任务"),
    ("POST", "/mall/task/instance/list", "任务实例", "任务"),
    # 招商
    ("POST", "/mall/merchant/list", "招商申请列表", "招商"),
    ("POST", "/mall/merchant/review", "审核招商申请", "招商"),
    ("GET", "/mall/merchant/pending-count", "待审核数", "招商"),
]


def now() -> str:
    """GVA 存进 SQLite 的时间格式（照抄现有行的形状，别用 isoformat）。"""
    d = datetime.now().astimezone()
    return d.strftime("%Y-%m-%d %H:%M:%S.%f") + d.strftime("%z")[:3] + ":" + d.strftime("%z")[3:]


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--db", default=DEFAULT_DB)
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--no-refresh", action="store_true")
    ap.add_argument("--refresh-url", default="http://127.0.0.1:8890/api/freshCasbin")
    args = ap.parse_args()

    db = os.path.abspath(args.db)
    if not os.path.exists(db):
        sys.exit("找不到库：%s" % db)
    conn = sqlite3.connect(db)
    conn.row_factory = sqlite3.Row
    ts = now()
    todo = []

    # ① sys_apis
    for method, path, desc, group in APIS:
        has = conn.execute(
            "SELECT 1 FROM sys_apis WHERE path=? AND method=? AND deleted_at IS NULL", (path, method)
        ).fetchone()
        if has:
            continue
        todo.append((
            "INSERT INTO sys_apis (created_at, updated_at, deleted_at, path, description, api_group, method)"
            " VALUES (?,?,NULL,?,?,?,?)",
            (ts, ts, path, desc, group, method),
        ))

    # ② casbin（真正拦请求的那一层）
    for method, path, _desc, _group in APIS:
        has = conn.execute(
            "SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0=? AND v1=? AND v2=?",
            (str(AUTHORITY), path, method),
        ).fetchone()
        if has:
            continue
        todo.append((
            "INSERT INTO casbin_rule (ptype, v0, v1, v2, v3, v4, v5) VALUES ('p',?,?,?,'','','')",
            (str(AUTHORITY), path, method),
        ))

    if not todo:
        print("已经全部登记过了，什么都没做。")
        _refresh(args)
        return 0

    if args.dry_run:
        print("[dry-run] 会执行 %d 条：" % len(todo))
        for sql, params in todo:
            print("  -", sql.split("(")[0].strip(), params[-2:])
        return 0

    for sql, params in todo:
        conn.execute(sql, params)
    conn.commit()
    print("写入 %d 条（接口 %d 个，已登记的一律跳过）。" % (len(todo), len(APIS)))
    _refresh(args)
    return 0


def _refresh(args) -> None:
    """让运行中的后台重新加载 casbin 策略 —— 不刷的话「库里有行、请求还是 403」。"""
    if args.no_refresh:
        print("· 跳过 casbin 热加载（--no-refresh）。重启后台或调一次 GET /api/freshCasbin 才生效。")
        return
    try:
        with urllib.request.urlopen(args.refresh_url, timeout=5) as r:
            body = json.loads(r.read().decode())
        print("· casbin 已热加载：%s" % body.get("msg", body))
    except Exception as e:  # noqa: BLE001
        print("· 没能热加载 casbin（%s）—— 后台没起或地址不对。" % e)
        print("  改完重启后台，或者手动：curl %s" % args.refresh_url)


if __name__ == "__main__":
    sys.exit(main())
