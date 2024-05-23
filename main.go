package main

/*
https://dev.mysql.com/doc/refman/8.0/en/replication-administration-status.html

https://forum.golangbridge.org/t/how-to-work-with-mysql-replication-output-using-golang/6635/6

*/

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/gomail.v2"
	"log"
	"os"
	"strings"
)

var (
	mysqlIPs   []string
	user       string
	passwd     string
	sender     string
	toUsers    []string
	ccUsers    map[string]string
	EmailCode  string = os.Getenv("email_code")
	subStr     string
	smtpServer string
	smtpPort   int
)

func init() {
	cfile := pflag.String("config",
		"config/config.yaml", "配置文件路径")
	pflag.Parse()
	viper.SetConfigType("yaml")
	viper.SetConfigFile(*cfile)
	// 读取配置
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	mysqlIPs = viper.GetStringSlice("monitor.mysqlAddr")
	user = viper.GetString("monitor.user")
	passwd = viper.GetString("monitor.password")
	sender = viper.GetString("monitor.notify.sender")
	toUsers = viper.GetStringSlice("monitor.notify.toUser")
	ccUsers = viper.GetStringMapString("monitor.notify.ccUsers")
	subStr = viper.GetString("monitor.notify.sub")

	smtpServer = viper.GetString("monitor.notify.smtpServer")
	smtpPort = viper.GetInt("monitor.notify.smtpPort")

}

func main() {
	db0, err := sql.Open(`mysql`, user+`:`+passwd+`@tcp(`+mysqlIPs[0]+`)/`)
	if err != nil {
		panic(err)

	}
	defer db0.Close()

	db1, err := sql.Open(`mysql`, user+`:`+passwd+`@tcp(`+mysqlIPs[1]+`)/`)
	if err != nil {
		panic(err)
	}
	defer db1.Close()

	err = db0.Ping()
	if err != nil {
		Notify("健康检查失败数据库无法连接" + mysqlIPs[0])
		return
	}

	err = db1.Ping()
	if err != nil {
		Notify("健康检查失败数据库无法连接" + mysqlIPs[1])
		return
	}

	if isReadOnly(db0) && isReadOnly(db1) {
		Notify("健康检查失败 主从节点 都为read only")
		fmt.Println("健康检查失败 主从节点 都为read only")
		return
	}

	if !isReadOnly(db0) && !isReadOnly(db1) {
		Notify("健康检查失败 主从节点  都不为 read only")
		fmt.Println("健康检查失败 主从节点  都不为 read only")
		return
	}

	if !isReadOnly(db0) && isReadOnly(db1) {
		checkStatus(mysqlIPs[0], mysqlIPs[1])
	}

	if isReadOnly(db0) && !isReadOnly(db1) {
		checkStatus(mysqlIPs[1], mysqlIPs[0])
	}
}

func ShowReplicaStatus(db *sql.DB) ReplicaStatus {
	var replicaStatus ReplicaStatus

	rows, err := db.Query("SHOW REPLICA STATUS") //  返回 sql.Rows
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	columns, err := rows.Columns() // 获取所有 column
	if err != nil {
		log.Fatal(err)
	}

	values := make([]interface{}, len(columns)) // 创建一个切片 元素类型是 interface{}   长度是len(columns)

	// 为 每个 values 元素 赋值  new(sql.RawBytes)
	for key, _ := range columns {
		//fmt.Println(key)
		values[key] = new(sql.RawBytes)
		//fmt.Println(key)
	}
	for rows.Next() {
		//the "values..." tells Go to use every available slot to populate data
		// 填充  values 数据
		err = rows.Scan(values...)
		if err != nil {
			log.Fatal(err)
		}
	}

	for index, columnName := range columns {
		// convert sql.RawBytes to String using "fmt.Sprintf"
		columnValue := fmt.Sprintf("%s", values[index])

		// Remove "&" from row values
		columnValue = strings.Replace(columnValue, "&", "", -1)

		// Optional: Don't display values that are NULL
		// Remove "if" to return empty NULL values
		if len(columnValue) == 0 {
			continue
		}
		if columnName == "Source_Log_File" {
			replicaStatus.Source_Log_File = columnValue

		}
		if columnName == "Read_Source_Log_Pos" {
			replicaStatus.Read_Source_Log_Pos = columnValue

		}

		if columnName == "Source_UUID" {
			replicaStatus.Source_UUID = columnValue

		}

		if columnName == "Source_Server_Id" {
			replicaStatus.Source_Server_Id = columnValue

		}

		if columnName == "Exec_Source_Log_Pos" {
			replicaStatus.Exec_Source_Log_Pos = columnValue
		}

		if columnName == "Replica_IO_Running" {
			replicaStatus.Replica_IO_Running = columnValue
		}

		if columnName == "Replica_SQL_Running" {
			replicaStatus.Replica_SQL_Running = columnValue

		}

		if columnName == "Executed_Gtid_Set" {
			replicaStatus.Executed_Gtid_Set = columnValue
		}

		if columnName == "Last_IO_Error" {
			replicaStatus.Last_IO_Error = columnValue

		}

		if columnName == "Last_SQL_Error" {
			replicaStatus.Last_SQL_Error = columnValue

		}

		if columnName == "Seconds_Behind_Source" {
			replicaStatus.Seconds_Behind_Source = columnValue
		}
	}
	return replicaStatus

}

func checkGTIDStatus(db *sql.DB) string {
	var gtidExecuted string
	rows, err := db.Query("SELECT @@GLOBAL.gtid_executed")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&gtidExecuted)
		if err != nil {
			panic(err.Error())
		}
	}
	return gtidExecuted
}

func getUUID(db *sql.DB) string {

	var serverUUID string
	err := db.QueryRow("SELECT @@server_uuid").Scan(&serverUUID)
	if err != nil {
		panic(err.Error())
	}

	return serverUUID
}

func getServerId(db *sql.DB) string {

	var serverID string
	err := db.QueryRow("SELECT @@server_id").Scan(&serverID)
	if err != nil {
		panic(err.Error())
	}

	return serverID
}

type MasterStatus struct {
	File              string
	Position          string
	Binlog_Do_DB      string
	Binlog_Ignore_DB  string
	Executed_Gtid_Set string
}

func showMasterStatus(db *sql.DB) MasterStatus {
	rows, err := db.Query("SHOW MASTER STATUS")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	var masterStatus MasterStatus

	for rows.Next() {
		err = rows.Scan(&masterStatus.File, &masterStatus.Position, &masterStatus.Binlog_Do_DB, &masterStatus.Binlog_Ignore_DB, &masterStatus.Executed_Gtid_Set)
		if err != nil {
			panic(err.Error())
		}
	}
	return masterStatus
}
func isReadOnly(db *sql.DB) bool {
	var readonly bool
	err := db.QueryRow("SELECT @@read_only").Scan(&readonly)
	if err != nil {
		panic(err.Error())
	}
	return readonly
}

func CheckStatus(master *sql.DB, slave *sql.DB) {

}

func checkStatus(masterIP, slaveIP string) {
	masterDb, err := sql.Open(`mysql`, user+`:`+passwd+`@tcp(`+masterIP+`)/`)
	if err != nil {
		panic(err.Error())
	}
	defer masterDb.Close()

	slaveDb, err := sql.Open(`mysql`, user+`:`+passwd+`@tcp(`+slaveIP+`)/`)
	if err != nil {
		panic(err.Error())
	}
	defer slaveDb.Close()

	//  获取  uuid  serverid
	masterUUID := getUUID(masterDb)
	slaveUUID := getUUID(slaveDb)
	masterServerID := getServerId(masterDb)
	slaveServerID := getServerId(slaveDb)

	// 主节点查看是否存在 Binlog Dump GTID进程
	if checkDumpBinlogProcess(masterDb) != 1 {
		Notify("健康检查失败：salve 进程不存在： 执行节点:   " + masterIP + "执行sql:   " + "SELECT count(*) FROM information_schema.processlist WHERE COMMAND = 'Binlog Dump GTID'")
		fmt.Println("健康检查失败：salve 进程不存在")
		return
	}

	// 主节点查看 show replicas 信息
	replicaInfo := checkShowReplicas(masterDb)

	// 比对  show replicas 和 实际的  uuid serverID 是否匹配
	if slaveUUID != replicaInfo.ReplicaUUID || slaveServerID != replicaInfo.ServerID || masterServerID != replicaInfo.SourceID {
		Notify("健康检查失败：salve 信息不匹配")
		fmt.Println("健康检查失败：salve 信息不匹配")
		return
	}

	// 主节点查看 show master status  信息
	masterStatus := showMasterStatus(masterDb)

	// 从节点 获取 SHOW REPLICA STATUS 信息
	replicaStatus := ShowReplicaStatus(slaveDb) // 检查从库的复制状态

	result :=
		"masterIP:                          &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;          " + masterIP + "<br>" +
			"slaveIP:                           &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;          " + "&nbsp&nbsp" + slaveIP + "<br>" +
			"master_Executed_Gtid_Set           &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;          " + masterStatus.Executed_Gtid_Set + "<br>" +
			"slave_Executed_Gtid_Set            &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;          " + replicaStatus.Executed_Gtid_Set + "<br>" +
			"master_File                        &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;          " + masterStatus.File + "<br>" +
			"slave_Source_Log_File              &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;          " + replicaStatus.Source_Log_File + "<br>" +
			"master_Position                    &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;          " + masterStatus.Position + "<br>" +
			"slave_Exec_Source_Log_Pos          &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;          " + replicaStatus.Exec_Source_Log_Pos + "<br>" +
			"slave_Read_Source_Log_Pos          &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;          " + replicaStatus.Read_Source_Log_Pos + "<br>" +
			"slave_Seconds_Behind_Source        &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;          " + replicaStatus.Seconds_Behind_Source + "<br>" +
			"slave_Replica_IO_Running           &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;          " + replicaStatus.Replica_IO_Running + "<br>" +
			"slave_Replica_Replica_SQL_Running  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;          " + replicaStatus.Replica_SQL_Running

	// 确认集群 拓扑
	if replicaStatus.Source_Server_Id != masterServerID || replicaStatus.Source_UUID != masterUUID {
		Notify("健康检查失败：集群 拓扑检查错误 uuid 或server id  不匹配")
		fmt.Println("健康检查失败：集群 拓扑检查错误 uuid 或server id  不匹配")
		return
	}

	if replicaStatus.Replica_IO_Running != "Yes" || replicaStatus.Replica_SQL_Running != "Yes" || replicaStatus.Last_SQL_Error != "" || replicaStatus.Last_IO_Error != "" {
		Notify("健康检查失败：salve 线程工作异常" + "Replica_IO_Running:" + replicaStatus.Replica_IO_Running + "Replica_SQL_Running:" + replicaStatus.Replica_SQL_Running + replicaStatus.Last_SQL_Error + replicaStatus.Last_IO_Error)
		fmt.Println("健康检查失败：salve 线程工作异常")
		return
	}

	if masterStatus.Executed_Gtid_Set != replicaStatus.Executed_Gtid_Set {
		Notify("健康检查失败：主从 Gtid_Set 不匹配:" + "<br>" + result)
		fmt.Println("主从 Gtid_Set 不匹配")
		return
	}

	if masterStatus.File != replicaStatus.Source_Log_File || masterStatus.Position != replicaStatus.Read_Source_Log_Pos || masterStatus.Position != replicaStatus.Exec_Source_Log_Pos {
		Notify("健康检查失败：主从 positon  不匹配:" + "<br>" + result)
		fmt.Println("主从 positon  不匹配")
		return
	}

	if checkGTIDStatus(masterDb) != checkGTIDStatus(slaveDb) {
		Notify("健康检查失败：主从  GTID  不一致")
		fmt.Println("主从  GTID  不一致")
		return
	}

	Notify("集群状态健康:" + "<br>" + result)
}

// 查看 mster 上是否存在 salve 的dump 进程
func checkDumpBinlogProcess(masterDB *sql.DB) int {
	rows, err := masterDB.Query("SELECT count(*) FROM information_schema.processlist WHERE COMMAND = 'Binlog Dump GTID'")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	var count int

	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			panic(err.Error())
		}
		// 如果预期为一行结果，可以直接跳出循环
		break
	}

	//fmt.Println("checkDumpBinlogProcess count:", count)
	return count
}

// 在主节点上执行
func checkShowReplicas(masterDB *sql.DB) Replica {
	rows, err := masterDB.Query("show replicas")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()
	var replicaData Replica
	for rows.Next() {
		err = rows.Scan(&replicaData.ServerID, &replicaData.Host, &replicaData.Port, &replicaData.SourceID, &replicaData.ReplicaUUID)
		if err != nil {
			panic(err.Error())
		}
	}
	return replicaData
}

type Replica struct {
	ServerID    string
	Host        string
	Port        int
	SourceID    string
	ReplicaUUID string
}

type ReplicaStatus struct {
	Source_Log_File     string
	Source_UUID         string
	Source_Server_Id    string
	Read_Source_Log_Pos string
	Exec_Source_Log_Pos string

	Replica_IO_Running  string
	Replica_SQL_Running string

	Executed_Gtid_Set     string
	Last_IO_Error         string
	Last_SQL_Error        string
	Seconds_Behind_Source string
}

func SendMail(fromUser string, toUsers []string, ccUsers map[string]string, subjectStr string, bodyStr string) {
	m := gomail.NewMessage()
	m.SetHeader("From", fromUser)
	m.SetHeader("To", toUsers...)

	// 设置抄送
	var cclist []string
	for userEmail, userName := range ccUsers {
		cclist = append(cclist, m.FormatAddress(userEmail, userName))
	}
	m.SetHeader("Cc", cclist...)

	// 设置主题
	m.SetHeader("Subject", subjectStr)
	m.SetBody("text/html", bodyStr)
	d := gomail.NewDialer(smtpServer, smtpPort, fromUser, EmailCode)
	if err := d.DialAndSend(m); err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func Notify(bodyStr string) {
	SendMail(sender, // 发送者
		toUsers,
		ccUsers,
		subStr, //邮件标题
		bodyStr,
	)
}
