这是一个空项目，目的是构建一个迁移脚本，将游戏的安卓存档，迁移到电脑上，如 Windows / macOS / Linux，当前仅考虑 Windows 平台。

游戏全称为「我在地府打麻将」，简称「wodima」

电脑上的存档地址为`{Steam安装目录}/userdata/{用户steamID}/3444020/remote/Slot[012].json` ， 游戏最多支持三个存档，默认下只会有 `Slot0.json`。


游戏的安卓存档为  `Android/data/com.itaotuo.wodima/files/game.db`，数据库为 SQLite，表名在 slot，表结构为这样：

```sql
CREATE TABLE slot (
    id INTEGER PRIMARY KEY NOT NULL
    , slotIndex INTEGER NOT NULL
    , userAccount TEXT NOT NULL DEFAULT ''
    , jsonString TEXT NOT NULL DEFAULT ''
    , UNIQUE(slotIndex, userAccount)
);
```

`jsonString` 字段的内容对应着 Windows 下的 `Slot[012].json`。

现在要做的是创建一个迁移脚本，最好有图形界面。

如果能访问到安卓手机时，自动获取安卓端的存档文件的地址，并将地址显示在界面上。同样也要允许用户选择存档文件的地址，因为用户可以自己将存档文件拷贝到电脑上。

steam 安装目录，可以通过查询系统获取，比如 Windows 上注册表。

用户steamID，不好查询，但可以遍历 `{Steam安装目录}/userdata` 下哪个用户下有 `3444020` 文件夹，这个ID 是游戏在 steam 上的ID。



准备迁移时，将用户的旧存档重命名为：`Slot[012].json.[时间戳].bak`



我已经连接了手机，尝试通过代码访问安卓端的存档文件。如果访问有困难，可以先用项目内的存档文件。

项目的后端应当使用编译型语言，避免用户安装解释器，如 python.

目前我知道的选型有 taui wails，如果有更容易的实现，且编译产物的空间不超过 100MB，也可以考虑其他的框架。