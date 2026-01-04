Go 的 interface 在 64 位系统上占多少字节？

---

**16 字节**（两个指针）

```go
type iface struct {
    tab  *itab  // 8 字节：类型信息 + 方法表
    data *void  // 8 字节：指向实际数据
}
```
