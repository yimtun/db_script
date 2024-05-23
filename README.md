#  主从健康状态检查 脚本


## 一 部署

###  1.1  编辑yaml配置文件

```yaml  config.yaml
monitor:
  user: www # 检查数据库状态的用户
  password: xxx # 检查数据库状态的用户密码
  mysqlAddr:
    - x.x.x.x:3306 #数据库节点 不区分集群角色 仅支持2个节点
    - y.y.y.y:3306 #数据库节点 不区分集群角色 仅支持2个节点
  notify:
    smtpServer: "xxx" #  smtp 邮件服务器地址
    smtpPort: "xxx" #  smtp 邮件服务器端口
    sub: "xxx" # 邮件主题
    sender: "ss@ss.com" # 邮件发送者
    toUser: #邮件接收者
      - "zz@zz.com"
    ccUsers: #邮件抄送者
      xx@xx.com: "xx"
      uu@uu.com: "uu"
```



### 1.2 环境变量加载敏感信息

```bash
export export email_code=xxx
```


### 1.3 指定配置文件路径启动

```bash
./db_script  --config ./config.yaml 
```





## 二 脚本逻辑


### 2.1 主 从 节点都需要获取的信息

#### 1） 获取 uuid

```mysql
SELECT UUID();
```


#### 2） 获取 server id

```mysql
SELECT @@server_id;
```


#### 3） 获取  read_only 状态


```mysql
SELECT @@read_only;
```





### 2.2 主 节点获取信息


#### 1） 主节点 执行 show replicas

``
+-----------+------+------+-----------+--------------------------------------+
| Server_Id | Host | Port | Source_Id | Replica_UUID                         |
+-----------+------+------+-----------+--------------------------------------+
|       101 |      | 3306 |       102 | 630c117f-1742-11ef-b16f-525400d89c6b |
+-----------+------+------+-----------+--------------------------------------+
``



#### 2) 主节点 查看是否有 Binlog Dump 进程

```
SELECT count(*) FROM information_schema.processlist WHERE COMMAND = 'Binlog Dump GTID';
```


#### 3)  主节点 执行 show master status;

```mysql
show master  status;
```

```
+------------------+----------+----------------+------------------+------------------------------------------+
| File             | Position | Binlog_Do_DB   | Binlog_Ignore_DB | Executed_Gtid_Set                        |
+------------------+----------+----------------+------------------+------------------------------------------+
| xx |      935 | xx |                  | 6424ed16-xx1742-11ef-b073-5254002dbeb8:1-9 |
+------------------+----------+----------------+------------------+------------------------------------------+
```





### 2.3  从 节点获取信息

#### 1） 从节点执行  show replica status；

````mysql
show replica status\G
````


```
Source_Log_File: mysql-bin.000002
Read_Source_Log_Pos: 935
Exec_Source_Log_Pos: 935
Executed_Gtid_Set:6424ed16-1742-11ef-b073-5254002dbeb8:1-9
Seconds_Behind_Source:0

Last_IO_Error 空
Last_SQL_Error 空

Replica_IO_Running: Yes
Replica_SQL_Running: Yes
```



## 三 编译

```mysql
GOOS=linux  GOARCH=amd64 go build   -o  db_script  main.go
```