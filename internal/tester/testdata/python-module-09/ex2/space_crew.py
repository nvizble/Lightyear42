from datetime import datetime
from enum import Enum
from typing import List

from pydantic import BaseModel, Field, ValidationError, model_validator


class Rank(str, Enum):
    CADET = "cadet"
    OFFICER = "officer"
    LIEUTENANT = "lieutenant"
    CAPTAIN = "captain"
    COMMANDER = "commander"


class CrewMember(BaseModel):
    """One member of a mission crew."""

    member_id: str = Field(min_length=3, max_length=10)
    name: str = Field(min_length=2, max_length=50)
    rank: Rank
    age: int = Field(ge=18, le=80)
    specialization: str = Field(min_length=3, max_length=30)
    years_experience: int = Field(ge=0, le=50)
    is_active: bool = True


class SpaceMission(BaseModel):
    """A mission and the crew that flies it."""

    mission_id: str = Field(min_length=5, max_length=15)
    mission_name: str = Field(min_length=3, max_length=100)
    destination: str = Field(min_length=3, max_length=50)
    launch_date: datetime
    duration_days: int = Field(ge=1, le=3650)
    crew: List[CrewMember] = Field(min_length=1, max_length=12)
    mission_status: str = "planned"
    budget_millions: float = Field(ge=1.0, le=10000.0)

    @model_validator(mode="after")
    def check_safety(self) -> "SpaceMission":
        if not self.mission_id.startswith("M"):
            raise ValueError("Mission ID must start with 'M'")
        leaders = (Rank.COMMANDER, Rank.CAPTAIN)
        if not any(member.rank in leaders for member in self.crew):
            raise ValueError(
                "Mission must have at least one Commander or Captain")
        if self.duration_days > 365:
            veterans = [m for m in self.crew if m.years_experience >= 5]
            if len(veterans) * 2 < len(self.crew):
                raise ValueError("Long missions (> 365 days) need 50% "
                                 "experienced crew (5+ years)")
        if not all(member.is_active for member in self.crew):
            raise ValueError("All crew members must be active")
        return self


def main() -> None:
    print("Space Mission Crew Validation")
    print("=" * 41)
    crew = [
        CrewMember(member_id="CM001", name="Sarah Connor",
                   rank=Rank.COMMANDER, age=45,
                   specialization="Mission Command", years_experience=20),
        CrewMember(member_id="CM002", name="John Smith",
                   rank=Rank.LIEUTENANT, age=35,
                   specialization="Navigation", years_experience=8),
        CrewMember(member_id="CM003", name="Alice Johnson",
                   rank=Rank.OFFICER, age=29,
                   specialization="Engineering", years_experience=3),
    ]
    mission = SpaceMission(
        mission_id="M2024_MARS",
        mission_name="Mars Colony Establishment",
        destination="Mars",
        launch_date=datetime(2024, 9, 1),
        duration_days=900,
        crew=crew,
        budget_millions=2500.0,
    )
    print("Valid mission created:")
    print(f"Mission: {mission.mission_name}")
    print(f"ID: {mission.mission_id}")
    print(f"Destination: {mission.destination}")
    print(f"Duration: {mission.duration_days} days")
    print(f"Budget: ${mission.budget_millions}M")
    print(f"Crew size: {len(mission.crew)}")
    print("Crew members:")
    for member in mission.crew:
        print(f"- {member.name} ({member.rank.value}) - "
              f"{member.specialization}")
    print()
    print("=" * 41)
    print("Expected validation error:")
    try:
        SpaceMission(
            mission_id="M2024_MOON",
            mission_name="Lunar Survey",
            destination="Moon",
            launch_date=datetime(2025, 3, 1),
            duration_days=30,
            crew=crew[1:],
            budget_millions=300.0,
        )
    except ValidationError as e:
        for item in e.errors():
            print(str(item["msg"]).removeprefix("Value error, "))


if __name__ == "__main__":
    main()
