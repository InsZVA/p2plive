# P2PLive
forked from InsZVA 

## forward 前线阵地
负责不断推进，普天之下，莫非王土
forwardd.go
    PushHandler: PushHandler handle the udp to push stream to forward
    AsyncBroadCast: 异步广播
    /load LoadHandler: 从forward获取stream
    / WSHandler: 用户上线和下线

## coordinate 战场指挥中心
协调服务器
p2plived.go
    协调服务器入口，路由功能
    /stream StreamHandler
    /tracker TrackerHandler
    /debug DebugHandler
forward.go
    UpdateForwards: 遍历前线阵地，临时存储在ForwardsUpdating
    ForwardStream: 向前线阵地提供stream
    ApplyUpdate: 遍历结束，更新ForwardsUpdating到前线阵地列表Forwards
stream.go
    StreamHandler: 源源不断的为前线阵地提供stream，同时更新前线阵地列表
debug.go
    DebugHandler: 显示前线阵地列表，咸鱼列表
tracker.go
    TrackerHandler: 负责处理咸鱼们的请求
	POST: 注册一个咸鱼
	PUT: 咸鱼要时不时的报告信息并且更新信息
	GET: 获取一个附近的咸鱼
	DELETE: 咸鱼请求删除自己

## tracker 跑来跑去的咸鱼
咸鱼要负责ob啊
tracker.go
    咸鱼生成程序，向战场指挥中心注册，之后时不时的报告信息并且更新信息
    /debug DebugHandler
	/resource ResourceHandler
forward.go
    CollectInfo: 检测可用的前线阵地
    ForwardsUpdate: 从文件FORWARDS中读取前线阵地列表
debug.go
    显示咸鱼知道的前线阵地列表
resource.go
    ResourceHandler
        getSource: 为小兵寻求前线阵地的支持，如果附近没有，小兵就地建立一个前线阵地
        update: 更新小兵的pullnum, pushnum
    PeekSourceClient: 查看附近有没有前线阵地