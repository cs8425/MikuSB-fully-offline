# MikuSB offline mode
a tool work with MikuSB, turn it into fully offline


## uasge

1. install MikuSB
2. start MikuSB once, do NOT need to install CA cert
3. change MikuSB's `Config.json`
	1. `Proxy` -> `Enabled`: false
	2. `Proxy` -> `InstallRootCertificate`: false
	3. `Proxy` -> `ManageSystemProxy`: false
4. add line to `host` file: `127.0.0.1 xgsdk.xoyo.games`
5. start MikuSB, and keep running in background
	1. be sure that MikuSB bind game server at `127.0.0.1:21000`
6. start `mikusb-proxy`
	1. be sure proxy bind at
		1. `127.0.0.1:18888`
		2. `127.0.0.1:13443` can not be changed
		3. `127.0.0.1:18443` can not be changed
		4. `127.0.0.1:31443` can not be changed
7. setup launch config
	1. steam launch options:
		1. windows: `cmd /C "set ALL_PROXY=socks5h://127.0.0.1:18888 && %command%"`
		2. linux: `ALL_PROXY="socks5h://127.0.0.1:18888" %command%`
	2. bat file: add env variable: `set ALL_PROXY=socks5h://127.0.0.1:18888`
8. launch the game


## build

1. clone the code
2. build: `go build -trimpath -ldflags="-s -w" .`


## Special Thanks

* [MikuLeaks/MikuSB](https://github.com/MikuLeaks/MikuSB) : the original work, this tool MUST use with it
* [Naruse](https://github.com/DevilProMT) : the author of MikuSB
* [Kei-Luna](https://github.com/Kei-Luna) : the author of MikuSB

