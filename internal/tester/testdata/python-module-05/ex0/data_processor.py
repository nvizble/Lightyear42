from abc import ABC, abstractmethod
from typing import Any


class DataProcessor(ABC):
    name = "Data Processor"

    def __init__(self) -> None:
        self._data: list[str] = []
        self.total = 0

    @abstractmethod
    def validate(self, data: Any) -> bool:
        pass

    @abstractmethod
    def ingest(self, data: Any) -> None:
        pass

    def _store(self, items: list[str]) -> None:
        self._data.extend(items)
        self.total += len(items)

    def remaining(self) -> int:
        return len(self._data)

    def output(self) -> tuple[int, str]:
        if not self._data:
            raise IndexError(f"{self.name}: no data to output")
        rank = self.total - len(self._data)
        return rank, self._data.pop(0)


def _is_num(x: Any) -> bool:
    return isinstance(x, (int, float)) and not isinstance(x, bool)


def _is_log(x: Any) -> bool:
    return (isinstance(x, dict)
            and "log_level" in x and "log_message" in x
            and all(isinstance(k, str) and isinstance(v, str)
                    for k, v in x.items()))


class NumericProcessor(DataProcessor):
    name = "Numeric Processor"

    def validate(self, data: Any) -> bool:
        if isinstance(data, list):
            return all(_is_num(x) for x in data)
        return _is_num(data)

    def ingest(self, data: int | float | list[int | float]) -> None:
        if not self.validate(data):
            raise ValueError("Improper numeric data")
        items = data if isinstance(data, list) else [data]
        self._store([str(x) for x in items])


class TextProcessor(DataProcessor):
    name = "Text Processor"

    def validate(self, data: Any) -> bool:
        if isinstance(data, list):
            return all(isinstance(x, str) for x in data)
        return isinstance(data, str)

    def ingest(self, data: str | list[str]) -> None:
        if not self.validate(data):
            raise ValueError("Improper text data")
        self._store(data if isinstance(data, list) else [data])


class LogProcessor(DataProcessor):
    name = "Log Processor"

    def validate(self, data: Any) -> bool:
        if isinstance(data, list):
            return all(_is_log(x) for x in data)
        return _is_log(data)

    def ingest(self, data: dict[str, str] | list[dict[str, str]]) -> None:
        if not self.validate(data):
            raise ValueError("Improper log data")
        items = data if isinstance(data, list) else [data]
        self._store([f"{d['log_level']}: {d['log_message']}" for d in items])


def extract(proc: DataProcessor, label: str, n: int) -> None:
    print(f"Extracting {n} value{'s' if n > 1 else ''}...")
    for _ in range(n):
        rank, value = proc.output()
        print(f"{label} {rank}: {value}")


def main() -> None:
    print("=== Code Nexus - Data Processor ===")
    print("Testing Numeric Processor...")
    num = NumericProcessor()
    print(f"Trying to validate input '42': {num.validate(42)}")
    print(f"Trying to validate input 'Hello': {num.validate('Hello')}")
    print("Test invalid ingestion of string 'foo' without prior validation:")
    try:
        num.ingest("foo")
    except ValueError as e:
        print(f"Got exception: {e}")
    data: list[int | float] = [1, 2, 3, 4, 5]
    print(f"Processing data: {data}")
    num.ingest(data)
    extract(num, "Numeric value", 3)

    print("Testing Text Processor...")
    text = TextProcessor()
    print(f"Trying to validate input '42': {text.validate(42)}")
    words = ["Hello", "Nexus", "World"]
    print(f"Processing data: {words}")
    text.ingest(words)
    extract(text, "Text value", 1)

    print("Testing Log Processor...")
    log = LogProcessor()
    print(f"Trying to validate input 'Hello': {log.validate('Hello')}")
    logs = [
        {"log_level": "NOTICE", "log_message": "Connection to server"},
        {"log_level": "ERROR", "log_message": "Unauthorized access!!"},
    ]
    print(f"Processing data: {logs}")
    log.ingest(logs)
    extract(log, "Log entry", 2)


if __name__ == "__main__":
    main()
