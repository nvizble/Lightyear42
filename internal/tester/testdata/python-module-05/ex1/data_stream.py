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


class DataStream:
    def __init__(self) -> None:
        self._procs: list[DataProcessor] = []

    def register_processor(self, proc: DataProcessor) -> None:
        self._procs.append(proc)

    def process_stream(self, stream: list[Any]) -> None:
        for element in stream:
            for proc in self._procs:
                if proc.validate(element):
                    try:
                        proc.ingest(element)
                    except Exception as e:
                        print(f"DataStream error - {proc.name}: {e}")
                    break
            else:
                print("DataStream error - "
                      f"Can't process element in stream: {element}")

    def print_processors_stats(self) -> None:
        print("== DataStream statistics ==")
        if not self._procs:
            print("No processor found, no data")
        for proc in self._procs:
            print(f"{proc.name}: total {proc.total} items processed, "
                  f"remaining {proc.remaining()} on processor")


BATCH: list[Any] = [
    "Hello world",
    [3.14, -1, 2.71],
    [
        {"log_level": "WARNING",
         "log_message": "Telnet access! Use ssh instead"},
        {"log_level": "INFO", "log_message": "User wil is connected"},
    ],
    42,
    ["Hi", "five"],
]


def main() -> None:
    print("=== Code Nexus - Data Stream ===")
    print("Initialize Data Stream...")
    stream = DataStream()
    stream.print_processors_stats()
    print("Registering Numeric Processor")
    num = NumericProcessor()
    stream.register_processor(num)
    print(f"Send first batch of data on stream: {BATCH}")
    stream.process_stream(BATCH)
    stream.print_processors_stats()
    print("Registering other data processors")
    text = TextProcessor()
    log = LogProcessor()
    stream.register_processor(text)
    stream.register_processor(log)
    print("Send the same batch again")
    stream.process_stream(BATCH)
    stream.print_processors_stats()
    print("Consume some elements from the data processors: "
          "Numeric 3, Text 2, Log 1")
    for proc, n in ((num, 3), (text, 2), (log, 1)):
        for _ in range(n):
            proc.output()
    stream.print_processors_stats()


if __name__ == "__main__":
    main()
