def garden_operations(operation_number: int) -> None:
    if operation_number == 0:
        int("abc")
    elif operation_number == 1:
        print(10 / 0)
    elif operation_number == 2:
        open("/non/existent/file")
    elif operation_number == 3:
        print("abc" + 1)  # type: ignore[operator]


def test_error_types() -> None:
    print("=== Garden Error Types Demo ===")
    for op in [0, 1, 2, 3, 4]:
        print(f"Testing operation {op}...")
        try:
            garden_operations(op)
            print("Operation completed successfully")
        except ValueError as e:
            print(f"Caught ValueError: {e}")
        except ZeroDivisionError as e:
            print(f"Caught ZeroDivisionError: {e}")
        except FileNotFoundError as e:
            print(f"Caught FileNotFoundError: {e}")
        except TypeError as e:
            print(f"Caught TypeError: {e}")
    try:
        garden_operations(0)
    except (ValueError, ZeroDivisionError) as e:
        print(f"Caught one of several error types: {e}")
    print("All error types tested successfully!")


if __name__ == "__main__":
    test_error_types()
