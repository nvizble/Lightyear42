def array_rotation_detector(arr1: list, arr2: list) -> bool:
    if len(arr1) != len(arr2):
        return False
    n = len(arr1)
    if n == 0:
        return True
    doubled = arr1 + arr1
    return any(doubled[i:i + n] == arr2 for i in range(n))
