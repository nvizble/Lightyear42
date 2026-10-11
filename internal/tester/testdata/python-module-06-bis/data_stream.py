from typing import Any

from data_processor import (DataProcessor, LogProcessor, NumericProcessor,
                            TextProcessor)


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
