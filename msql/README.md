# msql - SQL 数据库组件

基于 `sqlx` 和 `database/sql` 封装的数据库连接与事务管理工具，支持 MySQL 和 SQLite。

## 特性

- 🔌 **连接管理**：自动管理数据库连接池
- 📦 **sqlx 扩展**：支持 NamedQuery、NamedExec 等高级操作
- 🔄 **事务封装**：提供便捷的事务执行接口
- ⚙️ **灵活配置**：通过 Option 模式配置连接参数

## 快速开始

### 创建数据库连接

```go
package main

import (
    "github.com/quietking0312/component/msql"
)

func main() {
    db, err := msql.NewDB(
        msql.DriveName("mysql"),
        msql.DataSourceName("user:password@tcp(localhost:3306)/dbname?charset=utf8mb4"),
        msql.MaxOpenConnection(20),
        msql.MaxIdleConnection(5),
        msql.MaxQueryTime(30 * time.Second),
    )
    if err != nil {
        panic(err)
    }
    defer db.Close()

    if err := db.Ping(); err != nil {
        panic(err)
    }
}
```

### 使用 sqlx

```go
sqlxDB, err := msql.NewSqlxDB(
    msql.DriveName("mysql"),
    msql.DataSourceName("user:password@tcp(localhost:3306)/dbname"),
)
if err != nil {
    panic(err)
}
```

### 执行事务

```go
err := sqlxDB.SqlxBeginTx(msql.DefaultTxOptions(),
    func(tx *sqlx.Tx, ctx context.Context) error {
        _, err := tx.Exec("INSERT INTO users (name) VALUES (?)", "Alice")
        return err
    },
)
```

### 使用 BeginContext 的事务

```go
err := sqlxDB.SqlxBeginTx2(msql.LevelSerializable(),
    func(ctx *msql.BeginContext) error {
        tx := ctx.Tx()
        _, err := tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", 100, 1)
        if err != nil {
            return err
        }
        // 注册事务提交后的回调
        ctx.AppendAfter(func() {
            fmt.Println("事务已提交")
        })
        return nil
    },
)
```

### Named 查询

```go
err := sqlxDB.SqlxNameQuery("SELECT * FROM users WHERE name = :name",
    map[string]interface{}{"name": "Alice"},
    func(rows *sqlx.Rows) error {
        for rows.Next() {
            // 处理结果
        }
        return nil
    },
)
```

## API 文档

### 配置选项

| 函数 | 说明 |
|------|------|
| `DriveName(drivename string) Option` | 设置驱动名（mysql/sqlite3） |
| `DataSourceName(dsn string) Option` | 设置数据源连接串 |
| `MaxIdleConnection(idle int) Option` | 最大空闲连接数 |
| `MaxOpenConnection(open int) Option` | 最大打开连接数 |
| `MaxQueryTime(query time.Duration) Option` | 查询超时时间 |

### 事务选项

| 函数 | 说明 |
|------|------|
| `DefaultTxOptions() *sql.TxOptions` | 默认事务选项 |
| `LevelSerializable() TxOption` | 序列化隔离级别 |

### 客户端创建

| 函数 | 说明 |
|------|------|
| `NewDB(opts ...Option) (*DB, error)` | 创建标准 DB |
| `NewSqlxDB(opts ...Option) (*SqlxDB, error)` | 创建 sqlx DB |

### SqlxDB 方法

| 方法 | 说明 |
|------|------|
| `SqlxBeginTx(opts TxOption, cbs ...func(tx *sqlx.Tx, ctx context.Context) error) error` | 执行事务 |
| `SqlxBeginTx2(opts TxOption, cbs ...func(ctx *BeginContext) error) error` | 执行事务（增强版） |
| `SqlxNameExec(format string, arg interface{}) (sql.Result, error)` | NamedExec |
| `SqlxExec(format string, args ...interface{}) (sql.Result, error)` | 普通执行 |
| `SqlxNameQuery(format string, args interface{}, cb func(rows *sqlx.Rows) error) error` | NamedQuery |
| `SqlxQueryRow(format string, args []interface{}, cb func(row *sqlx.Row) error) error` | 查询单行 |

## 测试

```bash
cd msql
go test -v
```

## 注意事项

1. MySQL 驱动使用 `github.com/go-sql-driver/mysql`
2. SQLite 驱动使用 `github.com/mattn/go-sqlite3`，需要 CGO 支持
3. 生产环境建议设置合理的连接池参数
