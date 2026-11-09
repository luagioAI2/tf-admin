#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""「模板配置」「心愿商城」「游戏」升成一级菜单，并拆出 6 个子页面（幂等）。

    python scripts/seed-menu-level2.py --dry-run
    python scripts/seed-menu-level2.py

## 这轮到底改了什么

原来 `商城运营`(id=10) 下面挂着一个「心愿商城」大页（三个页签挤在一起）。
现在：

    商城运营          (id=10，保留，少一项)
    模板配置          (新建一级，跟商城运营同级)
      ├ 任务模板       ← 新用户首次启动铺的那套（每日 / 签到 / 长期）
      └ 兑换模板       ← 官方基础兑换（现在 App 里那 15 条）
    心愿商城          (新建一级，跟商城运营同级)
      ├ 心愿商品       ← 复用旧 id=56 / name=mallWish
      └ 心愿单         ← 新建
    游戏              (新建一级，跟商城运营同级)
      ├ 游戏列表       ← 新建页面 view/mall/game/index.vue
      └ 游戏激活码     ← 复用心愿大页的「激活码」页签

★ 商家那一侧另有一个「商家中心 → 我的模板」（seed-portal.py 建），
  指向同一个 `view/mall/template/index.vue` —— 商家只配自己的那份，
  数据范围由后端按 token 收敛，菜单和接口都不是本脚本的事。

## ★ 为什么「心愿商品」复用旧的 mallWish 名字

首页快捷入口 `to: 'mallWish'` 和 `_probe-brand.mjs` 的路由名断言都钉着
`mallWish`。改名字 ⇒ 首页点进去是死链、探针恒绿（hasRoute 为 false 直接
return，什么都不做）。所以**只挪 parent / 改标题 / 改 path**，name 不动。

## ★ 六个页面指向两个 .vue，靠路由 name 决定默认页签

`view/mall/wish/index.vue` 里 `MODE_BY_NAME` 把 mallWish / wishOrder /
gameCode 映射到 item / order / code 页签；`view/mall/template/index.vue`
把 templateTask / templateRedeem / merchantTemplates 映射到 task / redeem。
Vue Router 复用同一组件实例时不重跑 setup，所以两边都用
`watch(() => route.name)` 而不是 ref 初值。
**别把 template 里的 el-tab-pane 删掉**：现有 wish / wishcode 探针
还要点页签切换，删了大片断言会红。

## ★「游戏对家长和商家不可见」怎么保证

菜单只写 `sys_authority_menus` 给 888；1001 / 1002 / 1003 一条都不给。
接口那层本来就已经只有 888 有 casbin（/mall/game/* 与 /mall/wish/*），
本脚本不动它，但结束时会反查一遍并打印，防止以后有人误加。

## ★ 页面不存在就别建菜单

`component` 指向的 .vue 不在 ⇒ 点进去空白页，前端只报一个 404 chunk，
看着像权限配错。脚本先查文件在不在（和 seed-menu.py 同一条判据）。
"""

import argparse
import os
import sqlite3
import sys
from datetime import datetime

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_DB = os.path.join(HERE, "..", "..", "tf-platform-api", "data", "tf.db")
WEB_VIEW = os.path.join(HERE, "..", "web", "src", "view")
AUTHORITY = 888  # 只有超级管理员

# (path, name, title, component 相对 web/src, icon, sort)
# ★ sort 别和「超级管理员」撞（它也是 3）：GVA 的 getMenu 只按 sort 排，
#   并列时顺序由 SQLite 决定 —— 刷新两次位置可能就不一样。
GROUPS = [
    ("templateRoot", "templateRoot", "模板配置", "view/routerHolder.vue", "files", 6),
    ("wishMall", "wishMall", "心愿商城", "view/routerHolder.vue", "present", 4),
    ("gameRoot", "gameRoot", "游戏", "view/routerHolder.vue", "platform", 5),
]

# (path, name, title, component, icon, sort, 父菜单 name)
PAGES = [
    ("task", "templateTask", "任务模板", "view/mall/template/index.vue", "document", 0, "templateRoot"),
    ("redeem", "templateRedeem", "兑换模板", "view/mall/template/index.vue", "shopping-cart", 1, "templateRoot"),
    ("item", "mallWish", "心愿商品", "view/mall/wish/index.vue", "goods", 0, "wishMall"),
    ("order", "wishOrder", "心愿单", "view/mall/wish/index.vue", "tickets", 1, "wishMall"),
    ("list", "gameList", "游戏列表", "view/mall/game/index.vue", "grid", 0, "gameRoot"),
    ("code", "gameCode", "游戏激活码", "view/mall/wish/index.vue", "key", 1, "gameRoot"),
    # 版本发布（2026-11-07）：上传包 → 平台落盘 → 签进游戏目录。
    # ★ 放在「游戏」组下而不是「商城运营」：它是发游戏包，不是商城运营动作；
    #   而且 `seed-menu.py` 里原来那条 `mallRelease` 挂在商城运营下，
    #   两处都留会建出**两个同名菜单**（name 相同但 parent 不同）。
    ("release", "gameRelease", "版本发布", "view/mall/release/index.vue", "upload", 2, "gameRoot"),
]

# 商家中心那一列的排序。
# ★ seed-portal.py **只插不改**（`菜单已存在 → 跳过`），所以「我的模板」插进去之后
#   会和「入驻状态」撞在 sort=2 上，谁在前看 id —— 老库上就是「入驻状态在前」。
#   这里补一条对齐，两个脚本都幂等。
REORDER = [("merchantItems", 1), ("merchantTemplates", 2), ("merchantStatus", 3)]

PORTAL_ROLES = (1001, 1002, 1003)


def now() -> str:
    """GVA 存进 SQLite 的时间格式，照抄现有行的形状，别用 isoformat。"""
    d = datetime.now().astimezone()
    return d.strftime("%Y-%m-%d %H:%M:%S.%f") + d.strftime("%z")[:3] + ":" + d.strftime("%z")[3:]


def vue_exists(component: str) -> bool:
    rel = component[len("view/"):] if component.startswith("view/") else component
    return os.path.exists(os.path.join(WEB_VIEW, rel))


def main() -> int:
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

    # ---------- ⓪ 前置：页面文件都在 ----------
    missing = [c for _, _, _, c, _, _ in GROUPS if not vue_exists(c)]
    missing += [c for _, _, _, c, _, _, _ in PAGES if not vue_exists(c)]
    if missing:
        sys.exit("以下组件文件不存在，先别建菜单：\n  " + "\n  ".join(missing))

    # ---------- ① 一级分组 ----------
    group_ids = {}
    for path, name, title, comp, icon, sort in GROUPS:
        row = conn.execute(
            "SELECT id FROM sys_base_menus WHERE name=? AND deleted_at IS NULL", (name,)
        ).fetchone()
        if row:
            group_ids[name] = row["id"]
            print("· 一级菜单已存在：%s（id=%d）" % (title, row["id"]))
            continue
        print("· %s建一级菜单：%s (path=%s)" % ("[dry-run] " if args.dry_run else "", title, path))
        if args.dry_run:
            continue
        cur = conn.execute(
            "INSERT INTO sys_base_menus (created_at, updated_at, deleted_at, menu_level,"
            " parent_id, path, name, hidden, component, sort, active_name, keep_alive,"
            " default_menu, title, icon, close_tab, transition_type)"
            " VALUES (?,?,NULL,0,0,?,?,0,?,?,'',0,0,?,?,0,'')",
            (ts, ts, path, name, comp, sort, title, icon),
        )
        group_ids[name] = cur.lastrowid

    # ---------- ② 子页面：存在则「挪位置」，不存在则新建 ----------
    # 幂等的关键：不删不重建（id 一变，sys_authority_menus 就成孤儿），
    # 只把已有行 UPDATE 到目标位置。mallWish 的 name 保持不变。
    menu_ids = {}
    for path, name, title, comp, icon, sort, parent_name in PAGES:
        parent_id = group_ids.get(parent_name)
        if parent_id is None and not args.dry_run:
            sys.exit("父菜单 %s 没建出来，中止（不要留下孤儿菜单）" % parent_name)
        row = conn.execute(
            "SELECT id, parent_id, path, title FROM sys_base_menus"
            " WHERE name=? AND deleted_at IS NULL", (name,)
        ).fetchone()
        if row:
            menu_ids[name] = row["id"]
            need = (row["parent_id"] != parent_id or row["path"] != path
                    or row["title"] != title)
            print("· 子菜单已存在：%s（id=%d，%s）" %
                  (name, row["id"], "需要挪位置/改标题" if need else "位置正确"))
            if need and not args.dry_run:
                conn.execute(
                    "UPDATE sys_base_menus SET updated_at=?, menu_level=1, parent_id=?,"
                    " path=?, title=?, icon=?, component=?, sort=? WHERE id=?",
                    (ts, parent_id, path, title, icon, comp, sort, row["id"]),
                )
            continue
        print("· %s建子菜单：%s (name=%s → %s)" %
              ("[dry-run] " if args.dry_run else "", title, name, parent_name))
        if args.dry_run:
            continue
        cur = conn.execute(
            "INSERT INTO sys_base_menus (created_at, updated_at, deleted_at, menu_level,"
            " parent_id, path, name, hidden, component, sort, active_name, keep_alive,"
            " default_menu, title, icon, close_tab, transition_type)"
            " VALUES (?,?,NULL,1,?,?,?,0,?,?,'',0,0,?,?,0,'')",
            (ts, ts, parent_id, path, name, comp, sort, title, icon),
        )
        menu_ids[name] = cur.lastrowid

    # ---------- ③ 菜单授权：只给 888 ----------
    all_ids = list(group_ids.values()) + list(menu_ids.values())
    if not args.dry_run:
        for mid in all_ids:
            conn.execute(
                "INSERT INTO sys_authority_menus (sys_base_menu_id, sys_authority_authority_id)"
                " SELECT ?,? WHERE NOT EXISTS ("
                "   SELECT 1 FROM sys_authority_menus"
                "   WHERE sys_base_menu_id=? AND sys_authority_authority_id=?)",
                (mid, AUTHORITY, mid, AUTHORITY),
            )
        conn.commit()
        print("\n· %d 个菜单已确保授权给 888（幂等）。" % len(all_ids))

    # ---------- ③·5 商家中心那一列的顺序 ----------
    # seed-portal.py 只插不改 ⇒ 新插的「我的模板」和「入驻状态」会撞 sort。
    if not args.dry_run:
        for name, sort in REORDER:
            conn.execute(
                "UPDATE sys_base_menus SET sort=? WHERE name=? AND deleted_at IS NULL"
                " AND sort<>?", (sort, name, sort),
            )
        conn.commit()
        print("· 商家中心顺序已对齐：%s" % ", ".join("%s=%d" % (n, s) for n, s in REORDER))

    # ---------- ④ 反向断言：家长 / 商家一个都不该看得见 ----------
    if not args.dry_run:
        qmarks = ",".join("?" * len(all_ids))
        leaked = list(conn.execute(
            "SELECT sys_base_menu_id, sys_authority_authority_id FROM sys_authority_menus"
            " WHERE sys_base_menu_id IN (%s) AND sys_authority_authority_id IN (1001,1002,1003)"
            % qmarks, all_ids
        ))
        if leaked:
            print("!! 警告：门户角色拿到了新菜单的授权：%s" % [dict(r) for r in leaked])
        else:
            print("· 反向断言通过：1001/1002/1003 没有这 %d 个菜单的任何授权。" % len(all_ids))

        # 「商城运营」下不该再留着心愿商城
        leftovers = list(conn.execute(
            "SELECT id, name, title FROM sys_base_menus"
            " WHERE parent_id=(SELECT id FROM sys_base_menus WHERE name='mall' AND deleted_at IS NULL)"
            " AND deleted_at IS NULL AND name IN ('mallWish','wishOrder','gameList','gameCode')"
        ))
        print("· 商城运营下的旧心愿入口：%s" % ([dict(r) for r in leftovers] or "无"))

        # 游戏 / 心愿接口的 casbin 仍应只有 888
        cas = list(conn.execute(
            "SELECT DISTINCT v0 FROM casbin_rule WHERE ptype='p'"
            " AND (v1 LIKE '/mall/game/%' OR v1 LIKE '/mall/wish/%')"
        ))
        print("· /mall/game/* 与 /mall/wish/* 的 casbin 角色：%s" % [r[0] for r in cas])

    if args.dry_run:
        print("\n[dry-run] 没写库。去掉 --dry-run 才生效。")
    else:
        print("\n完成。**重新登录**（或刷新页面）才会看到新菜单。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
