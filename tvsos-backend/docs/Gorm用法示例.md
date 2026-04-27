# Gorm用法示例

## 插入语句

### 插入一条数据

```go
// 已经有一个要插入的user对象
results := DB.Create(&user)
fmt.Println(result.RowsAffected, user.Id)
```

- user自动获得自增主键
- `result.RowsAffected`：影响行数

## 查询语句

### 根据id查询一条数据详情

```go
var user User
// 查询id为3的user
DB.First(&user, 3)
// 也可以用Last，First是查询匹配的第一条数据，Last是查询匹配的最后一条数据
// 顺序为主键排序
```

- 查询到的结果放入参数user中
- 也有返回值result，用法和上面一样，记录修改行数，错误信息等

### 根据条件查询多条数据

```go
var users []User
DB.where("age > ? and grads > ?", 18, 90).Find(&users)
```

- 查询到的结果放入users中
- 切片也要传指针

### 动态条件查询多条数据

```go
func GetUsers(username string, minAge, maxAge int) []User {
    db := DB.Model(&User{})

    if username != "" {
        db = db.Where("username LIKE ?", "%"+username+"%")
    }
    if minAge > 0 {
        db = db.Where("age >= ?", minAge)
    }
    if maxAge > 0 {
        db = db.Where("age <= ?", maxAge)
    }

    var users []User
    db.Find(&users)
    return users
}
```

- `db.Where()`会自动拼接查询条件
- 查询语句比较复杂，需要`db := DB.Model(&User{})`提前指明要查询的表

 ## 更新语句

### 更新某一条数据（依赖主键）

```go
user := User{
  Id: 1,
  Name: "Tom",
  Age: 18,
}
// DB.Model()的参数如果有id，这里会自动插入"where id = 1"的条件，如果没有，需要手动指定
DB.Model(&user).Updates(user)
//上面的写法与下面等价（updates方法不会更新主键 放心）
db := DB.Model(&User{})
db.Where("id = ?", user.Id).Updates(user)
```

- `Updates()`只会更新参数中的非零字段，如果age = 0，则不会更新age
- 补充`DB.Model(&user).Select("age").Updates(user)`：只更新“age”字段，其它字段不更新，并且即使age字段为0也会强制更新
- 补充`DB.Model(&user).Omit("Name").Updates(user)`：正常执行`Updates()`，但排除“Name”字段，不会更新Name

### 更新单个字段

```go
db := DB.Model(&User{})
// 注意区分Update和Updates，分别用于更新单个字段和多个字段
// 把大于18的user都改名为微光娘
db.Where("age > 18").Update("name", "微光娘")
```

### 更新多个字段

```go
db := DB.Model(&User{})
db.Where("id = ?", 1).Updates(map[string]interface{}{
    "name": "Bob",
    "age":  0,
})
```

- `Updates()`的参数可以为结构体或者map，区别在于结构体不会更新零值字段，map会更新所有字段包括零值字段

## 删除语句

### 根据id删除数据

```go
// 删除id为6的数据
DB.Delete(&User{}, 6)
// 删除id为1，2，3的数据
DB.Delete(&User{}, []int{1, 2, 3})
```

- 补充：`DB.Delete(&user)`也合法，会自动根据user的id属性删除数据库中对应的该条数据

### 根据条件删除数据

```go
DB.Where("age < ?", 18).Delete(&User{})
```

- `Where`中可以写多个条件，也可以用map汇总条件

## 高级查询

### 多表连查

```go
type UserVehicleDTO struct {
    ID       uint
    Username string
    Age      int
    License  string
}

var list []UserVehicleDTO

// 下面的查询相当于：
// select u.id, u.username, u.age, v.license from user u left outer join vehicle v
// on u.id = v.user_id where u.age > 18 and v.status = 1
DB.Table("user u")
.Select("u.id, u.username, u.age, v.license")
.Joins("left outer join vehicle v on u.id = v.user_id")
.Where("u.age > ? and v.status = ?", 18, 1)
.Scan(&list)
```

- `Scan()`用于接受没有和数据库表字段一一对应的结构体，如果是entity一般用`Find()`
- `Select()`参数处允许用`u.*`，查询所有数据

### 分页查询

```go
// 先说明原生分页语句：
// select * from user order by id limit 20 10
// 表示从索引20开始查询 往下查十条 即[20, 21 ... 29] 共十条
// 因此公式为 limit ?, ?, (page - 1) * pageSize, pageSize
var total int
var users []User

DB.Model(&User{}).Count(&total)

DB.Model(&User{})
.Order("id")
.Offset((page - 1) * pageSize)
.Limit(pageSize)
.Find(&users)
```

- `Order`，`Offset`，`Limit`的顺序不重要，gorm会为你拼接成合法顺序的sql语句
