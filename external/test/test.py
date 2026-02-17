#!/usr/bin/env python3
import json
import sys

from pydantic import BaseModel, field_validator


class Input(BaseModel):
    name: str

    @field_validator("name", mode="before")
    @classmethod
    def strip_name(cls, v: str) -> str:
        return v.strip() if isinstance(v, str) else v


class OutputData(BaseModel):
    message: str
    name_length: int


class Output(BaseModel):
    ok: bool
    data: OutputData | None = None
    error: str | None = None


def run(input: dict) -> dict:
    parsed = Input.model_validate(input)
    output = Output(
        ok=True,
        data=OutputData(
            message=f"Hello from Python, {parsed.name}!",
            name_length=len(parsed.name),
        ),
    )
    return output.model_dump()


if __name__ == "__main__":
    try:
        raw = sys.stdin.read()
        input_data = json.loads(raw) if raw.strip() else {}
        result = run(input_data)
    except Exception as e:
        result = Output(ok=False, error=str(e)).model_dump()

    print(json.dumps(result))
