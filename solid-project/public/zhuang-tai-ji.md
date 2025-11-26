# 状态机

[最大子数组和](https://leetcode.cn/problems/maximum-subarray/)

状态:

- ~~--在子数组前--~~

- 在子数组中

- 在子数组后

[跳跃游戏](https://leetcode.cn/problems/jump-game/)

状态:

- 可达

- 不可达

[买卖股票的最佳时机](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock)

状态:

- 持有

- 不持有:持有前

- 不持有:已卖出

[买卖股票的最佳时机 II](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock-ii/)

状态:

- 持有

- 不持有

[买卖股票的最佳时机 III](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock-iii/)

状态:

- 持有:第一次买入

- 持有:第二次买入

- 不持有:第一次卖出

- 不持有:第二次卖出

[买卖股票的最佳时机含手续费](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock-with-transaction-fee)

状态:

- 持有

- 不持有

[309. 买卖股票的最佳时机含冷冻期](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock-with-cooldown)

状态:

- 持有

- 不持有

- 冷冻期



## 思考题

[188. 买卖股票的最佳时机 IV](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock-iv/)

```go
import "math"
func maxProfit(k int, prices []int) int {
	have := make([]int,k)
    // have[i] = have_i+1
	not_have := make([]int,k+1)
    // not_have[i+1] = not_have_i+1

    for i,_ := range have{
        have[i] = math.MinInt
    }
    // not_have_0(0)  not_have_1 not_have_2 ... not_have_k 
    // have_1 have_2 have_3 .... have_k

    // have_k = max(have_k,not_have_k-1 - p)
    // not_have_k = max(not_have_k,have_k + p)
    // ans = max(not_have_k)
    var ans = 0
    for _, p := range prices{
        have_cp := make([]int,k)
	    not_have_cp := make([]int,k+1)
        for i := range k{
            have_cp[i] = max(have[i], not_have[i] - p)  // k <- i+1
            not_have_cp[i+1] = max(not_have[i+1], have[i] + p)
            ans = max(not_have_cp[i+1])
        }
        have = have_cp
        not_have = not_have_cp
    }
    return ans
}
```

### 背包问题

状态数量由变量决定的状态机

[416. 分割等和子集](https://leetcode.cn/problems/partition-equal-subset-sum/)

状态:

- 选:当前容量为j

- 选:当前容量为j-1

- 选:当前容量为j-2

- ...

- 选:当前容量为j-k

- 不选:当前容量为j-k-1

- ...

[322. 零钱兑换](https://leetcode.cn/problems/coin-change/)

状态:

- 选:当前金额为j

- 选:当前金额为j-1

- 选:当前金额为j-2

- ...

- 不选:当前金额为j

- 不选:当前金额为j-1

- ...