# Ginctl

`ginctl`是快速构建go-gin项目的一个脚手架，旨在为了提高项目的开发效率，减少开发过程中重复的代码快的编写。

## 安装

你需要确认本地已经安装go1.15，为了更加方便操作，请先确认你本地已经添加如下配置：

```bash

# 如何确认是否添加 $GOPATH
echo $GOPATH

# 如果还未添加 $GOPATH 变量
echo 'export GOPATH="$HOME/go"' >> ~/.zshrc # 或者您所使用的其它bash的配置文件

# 如果已经添加了 $GOPATH 变量
echo 'export PATH="$GOPATH/bin:$PATH"' >> ~/.zshrc # 或者您所使用的其它bash的配置文件

# 添加GOPROXY相关
go env -w GO111MODULE="on"
go env -w GOPROXY="https://goproxy.cn,direct"
```

安装脚手架：

```bash
go get github.com/go-season/ginctl
```

## 使用

`ginctl`安装完成之后，执行`ginctl`可出现如下输出：

```text
Ginctl help you to build gin framework skeleton easily. Get started by running the new command in anywhere:

	ginctl new [project-name]

Usage:
  ginctl [command]

Available Commands:
    add         Convenience command: scaffold of quick generate some common code
    doc         Generate API router's document.
    help        Help about any command
    new         Create a gin framework skeleton
    route       Convenience command: relevant operation of route
    run         Run the application by starting a local development server
    tag         Add tag for your type struct

Flags:
  -h, --help   help for ginctl

Use "ginctl [command] --help" for more information about a command.
```

**更加详细使用，请参考:http://gof.piggy.xiaozhu.com/zh-CN/install/ginctl**

## 参与贡献

- fork仓库到本地，所有核心命令都在`cmd/`目录下，如有需求更改，可参与修改相应文件
- 本地修改完成之后，经过测试无问题，可提交`pull request`
