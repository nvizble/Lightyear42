from typing import Any


def py_room_scheduler(meetings: list[list[int]]) -> dict[str, Any]:
    rooms = []
    for start, end in sorted(meetings):
        for room in rooms:
            if room[-1][1] <= start:
                room.append([start, end])
                break
        else:
            rooms.append([[start, end]])
    return {"total_rooms": len(rooms), "schedule": rooms}
