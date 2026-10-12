from collections import deque


def sliding_window_maximum(nums: list[int], k: int) -> list[int]:
    if k <= 0 or k > len(nums):
        return []
    window = deque()
    result = []
    for i, value in enumerate(nums):
        while window and nums[window[-1]] <= value:
            window.pop()
        window.append(i)
        if window[0] <= i - k:
            window.popleft()
        if i >= k - 1:
            result.append(nums[window[0]])
    return result
