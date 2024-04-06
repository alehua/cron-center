package local

import (
	"context"
	"fmt"
	"sort"
)

func Demo(ctx context.Context) error {
	var nums = []int{2, 7, 11, 15, -9}
	ans := threeSum(nums)
	for i := 0; i < len(ans); i++ {
		fmt.Println(ans[i])
	}
	return nil
}

func threeSum(nums []int) [][]int {
	sort.Ints(nums)
	var ans [][]int
	for i := 0; i < len(nums)-2; i++ {
		if i > 0 && nums[i] == nums[i-1] {
			continue // 去重
		}
		start, end := i+1, len(nums)-1
		if nums[i]+nums[start]+nums[end] > 0 {
			end--
		} else if nums[i]+nums[start]+nums[end] < 0 {
			start++
		} else {
			ans = append(ans, []int{nums[i], nums[start], nums[end]})
			for start < end {
				if nums[start] == nums[start+1] {
					start++
				} else if nums[end] == nums[end-1] {
					end--
				} else {
					break
				}
			}
		}
	}
	return ans
}
