#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""把后台侧边栏里**框架自带的演示 / 推广入口**收起来。

为什么是 `hidden=1` 而不是删行：
- 这些菜单挂在 `sys_authorities_menus` 上，删行还要连带清关联，删错了
  「菜单管理」页会显示一堆孤立记录；
- `hidden` 只是**不显示在侧边栏**，路由和权限都还在，随时能改回来；
- 判据只看一眼侧边栏就行，不需要重装数据。

收哪几个：
  5  示例文件      —— 媒体库 / 断点续传 / 客户列表，全是框架 demo
  7  官方网站      —— 外链到 gin-vue-admin.com，点出去就离开产品了
  9  插件系统      —— 插件市场 / 打包插件 / 邮件插件[示例]，框架自己的东西
  6  编程辅助      —— 代码生成器 / 表单生成器 / AI 工作流 / Mcp Tools /
                     Skills / AI 页面绘制，全是框架开发工具，对产品没用
  43 编程辅助      —— 同一批工具的插件版，和 id=6 重名；两个都收起来

顺手把首页从「仪表盘」改成「首页」：这一屏已经不是框架那个演示看板了，
标签页本来就写着「首页」，菜单还叫「仪表盘」会让人以为这是两个页面。

幂等：已经是 1 的直接跳过；第二次跑输出「已经收起来了，什么都没做」。
"""
import argparse
import os
import sqlite3
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_DB = os.path.join(HERE, "..", "..", "tf-platform-api", "data", "tf.db")

# (id, path, title) —— 三个字段一起校验，防的是「id 漂了但脚本照样改」
TARGETS = [
    (5, "example", "示例文件"),
    (7, "https://www.gin-vue-admin.com", "官方网站"),
    (9, "plugin", "插件系统"),
    (6, "systemTools", "编程辅助"),
    (43, "programmingAssistant", "编程辅助"),
]

# (id, path, 旧标题, 新标题)
RENAMES = [
    (1, "dashboard", "仪表盘", "首页"),
    # 框架原来写的是小写的「api管理」，看着像没填完的占位
    (13, "api", "api管理", "接口管理"),
]


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

    for mid, path, old_title, new_title in RENAMES:
        row = conn.execute(
            "select id, path, title from sys_base_menus where id = ?", (mid,)
        ).fetchone()
        if row is None:
            sys.exit("id=%d 不存在，脚本和数据对不上了" % mid)
        if row["path"] != path:
            sys.exit("id=%d 期望 path=%s，实际 %s" % (mid, path, row["path"]))
        if row["title"] == new_title:
            continue
        if row["title"] != old_title:
            sys.exit(
                "id=%d 期望标题「%s」，实际「%s」" % (mid, old_title, row["title"])
            )
        print("%s id=%-3d 「%s」→「%s」" % ("会改名" if args.dry_run else "改名", mid, old_title, new_title))
        if not args.dry_run:
            with conn:
                conn.execute(
                    "update sys_base_menus set title = ? where id = ?", (new_title, mid)
                )

    todo = []
    for mid, path, title in TARGETS:
        row = conn.execute(
            "select id, path, title, hidden from sys_base_menus where id = ?", (mid,)
        ).fetchone()
        if row is None:
            sys.exit("id=%d 不存在，脚本和数据对不上了" % mid)
        if row["path"] != path or row["title"] != title:
            sys.exit(
                "id=%d 期望 (%s / %s)，实际 (%s / %s)"
                % (mid, path, title, row["path"], row["title"])
            )
        if row["hidden"]:
            continue
        todo.append((mid, title))

    if not todo:
        print("已经收起来了，什么都没做。")
        return

    for mid, title in todo:
        print("%s id=%-3d %s" % ("会隐藏" if args.dry_run else "隐藏", mid, title))

    if args.dry_run:
        return

    with conn:
        for mid, _ in todo:
            conn.execute(
                "update sys_base_menus set hidden = 1 where id = ? and hidden = 0",
                (mid,),
            )

    left = conn.execute(
        "select count(*) from sys_base_menus where id in (%s) and hidden = 0"
        % ",".join(str(t[0]) for t in TARGETS)
    ).fetchone()[0]
    if left:
        sys.exit("还有 %d 条没改成隐藏" % left)
    print("完成。重新登录后台即可看到侧边栏变化。")


if __name__ == "__main__":
    main()
