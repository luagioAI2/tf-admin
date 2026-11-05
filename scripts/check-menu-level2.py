#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""#16 菜单重构的 DB / 源码断言（只读，不改库）。

    python scripts/check-menu-level2.py

和 `seed-menu-level2.py` 配对：那边负责写，这边负责验。
★ 界面层还要另跑 `_probe-brand.mjs` / `_probe-portal-ui.mjs` ——
  「库里没授权」不等于「界面上真的看不见」，两边都要钉。
"""

import argparse
import os
import sqlite3
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_DB = os.path.join(HERE, "..", "..", "tf-platform-api", "data", "tf.db")
WEB_SRC = os.path.join(HERE, "..", "web", "src")
WEB_VIEW = os.path.join(HERE, "..", "web", "src", "view")
GO_MENU = os.path.join(HERE, "..", "server", "source", "system", "menu.go")

PASSED, FAILED = [], []


def check(name, cond, detail=""):
    (PASSED if cond else FAILED).append(name)
    print(("  ✓ " if cond else "  ✗ ") + name + (("  → " + str(detail)) if detail else ""))


def menu(conn, name):
    return conn.execute(
        "SELECT * FROM sys_base_menus WHERE name=? AND deleted_at IS NULL", (name,)
    ).fetchone()


def vue_exists(component):
    rel = component[len("view/"):] if component.startswith("view/") else component
    return os.path.exists(os.path.join(WEB_VIEW, rel))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--db", default=DEFAULT_DB)
    args = ap.parse_args()

    db = os.path.abspath(args.db)
    if not os.path.exists(db):
        sys.exit("找不到库：%s" % db)
    conn = sqlite3.connect(db)
    conn.row_factory = sqlite3.Row

    print("== ① 四个平级的一级菜单 ==")
    mall = menu(conn, "mall")
    tpl = menu(conn, "templateRoot")
    wish = menu(conn, "wishMall")
    game = menu(conn, "gameRoot")
    for label, m in (
        ("商城运营", mall),
        ("模板配置", tpl),
        ("心愿商城", wish),
        ("游戏", game),
    ):
        check("一级菜单「%s」存在" % label, m is not None)
        if m:
            check(
                "「%s」真的是一级（parent_id=0 / menu_level=0 / hidden=0）" % label,
                m["parent_id"] == 0 and m["menu_level"] == 0 and m["hidden"] == 0,
                dict(id=m["id"], parent_id=m["parent_id"], menu_level=m["menu_level"]),
            )
            check(
                "「%s」用 routerHolder 分组组件" % label,
                m["component"] == "view/routerHolder.vue",
                m["component"],
            )
    if tpl and wish and game and mall:
        check("模板配置 / 心愿商城 / 游戏 与商城运营同层（四个 parent_id 都是 0）",
              tpl["parent_id"] == wish["parent_id"] == game["parent_id"] == mall["parent_id"] == 0)

    print("\n== ② 六个子页挂在正确的组下 ==")
    EXPECT = [
        ("templateTask", "任务模板", "task", "templateRoot", "view/mall/template/index.vue"),
        ("templateRedeem", "兑换模板", "redeem", "templateRoot", "view/mall/template/index.vue"),
        ("mallWish", "心愿商品", "item", "wishMall", "view/mall/wish/index.vue"),
        ("wishOrder", "心愿单", "order", "wishMall", "view/mall/wish/index.vue"),
        ("gameList", "游戏列表", "list", "gameRoot", "view/mall/game/index.vue"),
        ("gameCode", "游戏激活码", "code", "gameRoot", "view/mall/wish/index.vue"),
    ]
    ids = []
    for name, title, path, parent_name, comp in EXPECT:
        m = menu(conn, name)
        parent = menu(conn, parent_name)
        check("子页「%s」(name=%s) 存在" % (title, name), m is not None)
        if not m:
            continue
        ids.append(m["id"])
        check("「%s」挂在「%s」下" % (title, parent_name),
              parent is not None and m["parent_id"] == parent["id"],
              "parent_id=%s, 期望=%s" % (m["parent_id"], parent["id"] if parent else "?"))
        check("「%s」path=%s / 标题正确 / menu_level=1" % (title, path),
              m["path"] == path and m["title"] == title and m["menu_level"] == 1,
              dict(path=m["path"], title=m["title"]))
        check("「%s」component 指向 %s" % (title, comp), m["component"] == comp, m["component"])
        check("★「%s」的 .vue 文件真实存在（否则点进去空白页）" % title,
              vue_exists(m["component"]))

    print("\n== ③ 旧结构已拆干净 ==")
    if mall:
        leftovers = [dict(r) for r in conn.execute(
            "SELECT name, title FROM sys_base_menus WHERE parent_id=? AND deleted_at IS NULL"
            " AND name IN ('mallWish','wishOrder','gameList','gameCode','templateTask','templateRedeem')",
            (mall["id"],)
        )]
        check("商城运营下不再挂着这 6 个页面", not leftovers, leftovers)
    check("★ mallWish 这个路由名还活着（首页快捷入口 / 品牌探针都钉着它）",
          menu(conn, "mallWish") is not None)
    check("★ 老 seed-menu.py 没在商城运营下重建出重复的「游戏管理」(mallGame)",
          menu(conn, "mallGame") is None)

    print("\n== ③·5 全库没有重名菜单 / 一级菜单排序不并列 ==")
    # ★ 1003 是「家长 + 商家」两棵树，seed-portal.py 规划时同一个菜单会出现两次。
    #   已有的行看不出来（两次都「已存在，跳过」），但**新加一条**时会插两条，
    #   左侧就出现两个同名入口、各授权给一个角色（实测踩过一次）。
    dups = [dict(r) for r in conn.execute(
        "SELECT name, count(*) n FROM sys_base_menus WHERE deleted_at IS NULL"
        " GROUP BY name HAVING n > 1"
    )]
    check("★ 没有重名菜单（重名 = 左侧出现两个同名入口，且各只对一个角色可见）",
          not dups, dups)
    # ★ 一级菜单并列 sort：GVA 的 getMenu 只按 sort 排，并列时顺序由 SQLite 决定，
    #   刷新两次位置可能就不一样（新加的「模板配置」第一版正好和「超级管理员」撞在 3）。
    ties = [dict(r) for r in conn.execute(
        "SELECT sort, count(*) n, group_concat(title) titles FROM sys_base_menus"
        " WHERE parent_id=0 AND deleted_at IS NULL AND hidden=0"
        " GROUP BY sort HAVING n > 1"
    )]
    check("★ 一级菜单的 sort 不并列（并列时顺序不稳定）", not ties, ties)

    print("\n== ④ 菜单授权：只有 888，门户角色一条都没有 ==")
    all_ids = [m["id"] for m in (mall, tpl, wish, game) if m] + ids
    for mid in all_ids:
        has = conn.execute(
            "SELECT 1 FROM sys_authority_menus WHERE sys_base_menu_id=? AND sys_authority_authority_id=888",
            (mid,),
        ).fetchone()
        check("菜单 id=%d 已授权给 888" % mid, has is not None)
    if all_ids:
        q = ",".join("?" * len(all_ids))
        leaked = [dict(r) for r in conn.execute(
            "SELECT sys_base_menu_id, sys_authority_authority_id FROM sys_authority_menus"
            " WHERE sys_base_menu_id IN (%s) AND sys_authority_authority_id IN (1001,1002,1003)" % q,
            all_ids,
        )]
        check("★ 家长 1001 / 商家 1002 / 家长+商家 1003 对新菜单零授权",
              not leaked, leaked)

    print("\n== ⑤ 接口层：游戏 / 心愿只有 888；模板三家分权 ==")
    roles = [r[0] for r in conn.execute(
        "SELECT DISTINCT v0 FROM casbin_rule WHERE ptype='p'"
        " AND (v1 LIKE '/mall/game/%' OR v1 LIKE '/mall/wish/%') ORDER BY v0"
    )]
    check("★ /mall/game/* 与 /mall/wish/* 的 casbin 角色只有 888",
          roles == ["888"], roles)

    tpl_roles = sorted(r[0] for r in conn.execute(
        "SELECT DISTINCT v0 FROM casbin_rule WHERE ptype='p' AND v1 LIKE '/mall/template/%'"
    ))
    check("★ /mall/template/* 给 888 + 商家 1002 + 1003（商家要配自己的模板）",
          tpl_roles == ["1002", "1003", "888"], tpl_roles)
    check("★ 家长 1001 拿不到 /mall/template/*（它不是商家，配不了模板）",
          "1001" not in tpl_roles, tpl_roles)

    print("\n== ⑤·2 商家侧：商家中心 → 我的模板 ==")
    portal = menu(conn, "merchantPortal")
    mine = menu(conn, "merchantTemplates")
    check("商家菜单「我的模板」存在", mine is not None)
    if portal and mine:
        check("「我的模板」挂在商家中心下", mine["parent_id"] == portal["id"],
              "parent_id=%s" % mine["parent_id"])
        check("「我的模板」指向 template/index.vue",
              mine["component"] == "view/mall/template/index.vue", mine["component"])
        check("★「我的模板」的 .vue 真实存在", vue_exists(mine["component"]))
        check("商家中心里三页顺序：我的商品 → 我的模板 → 入驻状态",
              [r["name"] for r in conn.execute(
                  "SELECT name FROM sys_base_menus WHERE parent_id=? AND deleted_at IS NULL"
                  " ORDER BY sort", (portal["id"],)
              )] == ["merchantItems", "merchantTemplates", "merchantStatus"],
              [r["name"] for r in conn.execute(
                  "SELECT name FROM sys_base_menus WHERE parent_id=? AND deleted_at IS NULL"
                  " ORDER BY sort", (portal["id"],)
              )])
    if mine:
        roles_with = sorted(str(r[0]) for r in conn.execute(
            "SELECT sys_authority_authority_id FROM sys_authority_menus"
            " WHERE sys_base_menu_id=?", (mine["id"],)
        ))
        check("★「我的模板」只给 1002 / 1003（家长 1001 没有）",
              roles_with == ["1002", "1003"], roles_with)

    print("\n== ⑥ 源码双写（全新安装才有 initdb，共用库永远不跑） ==")
    go = open(GO_MENU, "r", encoding="utf-8").read()
    for name in ("templateRoot", "templateTask", "templateRedeem",
                 "wishMall", "gameRoot", "mallWish", "wishOrder", "gameList", "gameCode"):
        check("menu.go 源码 seed 里有 %s" % name, ('Name: "%s"' % name) in go)
    wish_vue = open(os.path.join(WEB_SRC, "view", "mall", "wish", "index.vue"),
                    "r", encoding="utf-8").read()
    check("wish/index.vue 有 MODE_BY_NAME（按路由 name 定默认页签）",
          "MODE_BY_NAME" in wish_vue)
    for name in ("mallWish", "wishOrder", "gameCode"):
        check("MODE_BY_NAME 里映射了 %s" % name, ("%s:" % name) in wish_vue)

    tpl_vue = open(os.path.join(WEB_SRC, "view", "mall", "template", "index.vue"),
                   "r", encoding="utf-8").read()
    for name in ("templateTask", "templateRedeem", "merchantTemplates"):
        check("template/index.vue 的 MODE_BY_NAME 里映射了 %s" % name,
              ("%s:" % name) in tpl_vue)
    portal_py = open(os.path.join(HERE, "seed-portal.py"), "r", encoding="utf-8").read()
    check("seed-portal.py 里也有 merchantTemplates（新装才有）",
          '"merchantTemplates"' in portal_py)

    total = len(PASSED) + len(FAILED)
    print("\n%s  %d/%d 通过" % ("全部通过" if not FAILED else "有失败", len(PASSED), total))
    if FAILED:
        for f in FAILED:
            print("  ✗ " + f)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
